package org

// UnlockQuery asks whether a Task is unlocked on a Job instance (in-Job Unlock order).
type UnlockQuery struct {
	OrderedIDs    []string
	JobID         string
	TaskID        string
	JobInstanceID string // empty only valid for the first Task
	Submissions   []Submission
	Tasks         []TaskConfig
}

// IsTaskUnlocked: Task unlocks when its Unlock predecessor is satisfied on this
// Job instance. Default predecessor is the nearest earlier once sibling
// (submit: many leaves are skipped so peer many Tasks share one once bookend;
// ADR-0008). Optional unlockFrom names an earlier sibling instead. Leaf
// predecessor → one Submission. First Task is always unlocked. Distinct from
// startsWhen / readyWhen. Predecessor primitive inside EvaluatePathUnlock.
func IsTaskUnlocked(q UnlockQuery) bool {
	index := -1
	for i, id := range q.OrderedIDs {
		if id == q.TaskID {
			index = i
			break
		}
	}
	if index < 0 {
		return false
	}
	if index == 0 {
		return true
	}
	if q.JobInstanceID == "" {
		return false
	}

	previousID, ok := unlockPredecessorID(q, index)
	if !ok {
		// Only submit:many siblings precede this Task — unlock with the instance.
		return true
	}
	if _, ok := TaskByID(q.Tasks, previousID); !ok {
		return false
	}
	return hasFlatSubmissionOnInstance(q.Submissions, q.JobID, previousID, q.JobInstanceID)
}

// unlockPredecessorID resolves the Unlock predecessor for OrderedIDs[index].
// unlockFrom (when present) must be a non-empty string Task id; omit ⇒ walk
// backward skipping submit:many leaves until an once leaf.
func unlockPredecessorID(q UnlockQuery, index int) (string, bool) {
	current, ok := TaskByID(q.Tasks, q.TaskID)
	if !ok {
		return "", false
	}
	if raw, present := current["unlockFrom"]; present {
		from, ok := raw.(string)
		if !ok || from == "" {
			return "", false
		}
		return from, true
	}
	if index < 1 || index > len(q.OrderedIDs) {
		return "", false
	}
	for i := index - 1; i >= 0; i-- {
		id := q.OrderedIDs[i]
		prev, ok := TaskByID(q.Tasks, id)
		if !ok {
			continue
		}
		if !taskSubmitMany(prev) {
			return id, true
		}
	}
	return "", false
}

func hasFlatSubmissionOnInstance(subs []Submission, jobID, taskID, jobInstanceID string) bool {
	for _, s := range subs {
		if s.JobID == jobID && s.TaskID == taskID && s.JobInstanceID == jobInstanceID {
			return true
		}
	}
	return false
}

// PathUnlock is Unlock-path availability for one TaskPathRef on a Job instance:
// predecessor Unlock → Unlocked; submit: once Submission → Done (submit: many
// never Done). startsWhen / readyWhen stay outside.
type PathUnlock struct {
	Unlocked bool
	Done     bool
}

// EvaluatePathUnlock is the single Unlock/path availability law (Org Config + Domain).
func EvaluatePathUnlock(cfg OrgConfig, domain Domain, ref TaskPathRef, jobInstanceID string) PathUnlock {
	jobVal, ok := cfg.Jobs.Get(ref.JobID)
	if !ok {
		return PathUnlock{}
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return PathUnlock{}
	}
	return evaluateFlatPathUnlock(job, domain, ref, jobInstanceID)
}

func evaluateFlatPathUnlock(job Job, domain Domain, ref TaskPathRef, jobInstanceID string) PathUnlock {
	tc, ok := TaskByID(job.Tasks, ref.TaskID)
	if !ok {
		return PathUnlock{}
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs:    OrderedTaskIDs(job.Tasks),
		JobID:         ref.JobID,
		TaskID:        ref.TaskID,
		JobInstanceID: jobInstanceID,
		Submissions:   domain.Submissions,
		Tasks:         job.Tasks,
	}) {
		return PathUnlock{}
	}
	// submit: many never becomes done from Submission (ADR-0008); once (default) does.
	done := !taskSubmitMany(tc) &&
		hasFlatSubmissionOnInstance(domain.Submissions, ref.JobID, ref.TaskID, jobInstanceID)
	return PathUnlock{
		Unlocked: true,
		Done:     done,
	}
}

func sequenceOkForCommit(cfg OrgConfig, domain Domain, jobID, taskID, jobInstanceID string) bool {
	return sequenceOkForPath(cfg, domain, TaskPathRef{JobID: jobID, TaskID: taskID}, jobInstanceID)
}

func sequenceOkForPath(cfg OrgConfig, domain Domain, ref TaskPathRef, jobInstanceID string) bool {
	jobVal, ok := cfg.Jobs.Get(ref.JobID)
	if !ok {
		return false
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return false
	}
	// Done (submit: once after Submission) blocks Commit; many never Done ⇒ repeats OK.
	ev := EvaluatePathUnlock(cfg, domain, ref, jobInstanceID)
	return ev.Unlocked && !ev.Done && startsWhenOk(cfg, job, domain)
}

func startsWhenOk(cfg OrgConfig, job Job, domain Domain) bool {
	if job.StartsWhen == nil {
		return true
	}
	return IsStartsWhenSatisfied(job.StartsWhen, domain.Submissions, domain.JobInstances, referencedStartsWhenTask(cfg, job.StartsWhen))
}
