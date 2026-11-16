package living

// UnlockQuery asks whether a Task is unlocked on a Job instance (in-Job Unlock order).
// For Unlock among children of an open group instance, set GroupID + GroupInstanceID
// and pass that group's child OrderedIDs/Tasks.
type UnlockQuery struct {
	OrderedIDs      []string
	JobID           string
	TaskID          string
	JobInstanceID   string // empty only valid for the first Task
	GroupID         string
	GroupInstanceID string
	Submissions     []Submission
	ClosedGroups    []ClosedGroup
	Tasks           []TaskConfig
}

// IsTaskUnlocked: Task unlocks when its Unlock predecessor is satisfied on this
// Job instance. Default predecessor is the nearest earlier once/group sibling
// (submit: many leaves are skipped so peer many Tasks share one once bookend;
// ADR-0008). Optional unlockFrom names an earlier sibling instead. Leaf
// predecessor → one Submission. Group predecessor → group closed (not one child
// Submission). First Task is always unlocked. Distinct from startsWhen /
// readyWhen. Predecessor primitive inside EvaluatePathUnlock.
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
	prev, ok := TaskByID(q.Tasks, previousID)
	if !ok {
		return false
	}

	if q.GroupID != "" {
		if q.GroupInstanceID == "" {
			return false
		}
		return hasGroupChildSubmission(q.Submissions, q.JobID, q.GroupID, q.GroupInstanceID, previousID, q.JobInstanceID)
	}

	if prev.IsGroup() {
		return IsGroupClosed(q.ClosedGroups, q.JobID, q.JobInstanceID, previousID)
	}
	return hasFlatSubmissionOnInstance(q.Submissions, q.JobID, previousID, q.JobInstanceID)
}

// unlockPredecessorID resolves the Unlock predecessor for OrderedIDs[index].
// unlockFrom (when present) must be a non-empty string Task id; omit ⇒ walk
// backward skipping submit:many leaves until an once leaf or group.
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
		if prev.IsGroup() || !taskSubmitMany(prev) {
			return id, true
		}
	}
	return "", false
}

func hasFlatSubmissionOnInstance(subs []Submission, jobID, taskID, jobInstanceID string) bool {
	for _, s := range subs {
		if s.JobID == jobID && s.TaskID == taskID && s.JobInstanceID == jobInstanceID && s.GroupID == "" {
			return true
		}
	}
	return false
}

func hasGroupChildSubmission(subs []Submission, jobID, groupID, groupInstanceID, taskID, jobInstanceID string) bool {
	for _, s := range subs {
		if s.JobID == jobID && s.TaskID == taskID && s.JobInstanceID == jobInstanceID &&
			s.GroupID == groupID && s.GroupInstanceID == groupInstanceID {
			return true
		}
	}
	return false
}

// PathUnlock is Unlock-path availability for one TaskPathRef on a Job instance:
// nested scaffolding + predecessor Unlock → Unlocked; submit: once Submission /
// group-closed → Done (submit: many never Done). startsWhen / readyWhen stay outside.
type PathUnlock struct {
	Unlocked bool
	Done     bool
}

// EvaluatePathUnlock is the single Unlock/path availability law (Org Config + Domain).
// Provisional / ProvisionalGroupInstanceID paths unlock the first child only.
// Group-node chrome uses a flat ref with TaskID = group id.
func EvaluatePathUnlock(cfg OrgConfig, domain Domain, ref TaskPathRef, jobInstanceID string) PathUnlock {
	jobVal, ok := cfg.Jobs.Get(ref.JobID)
	if !ok {
		return PathUnlock{}
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return PathUnlock{}
	}

	if ref.Nested() {
		return evaluateNestedPathUnlock(job, domain, ref, jobInstanceID)
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
		ClosedGroups:  domain.ClosedGroups,
		Tasks:         job.Tasks,
	}) {
		return PathUnlock{}
	}
	if tc.IsGroup() {
		return PathUnlock{
			Unlocked: true,
			Done:     IsGroupClosed(domain.ClosedGroups, ref.JobID, jobInstanceID, ref.TaskID),
		}
	}
	// submit: many never becomes done from Submission (ADR-0008); once (default) does.
	done := !taskSubmitMany(tc) &&
		hasFlatSubmissionOnInstance(domain.Submissions, ref.JobID, ref.TaskID, jobInstanceID)
	return PathUnlock{
		Unlocked: true,
		Done:     done,
	}
}

func evaluateNestedPathUnlock(job Job, domain Domain, ref TaskPathRef, jobInstanceID string) PathUnlock {
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs:    OrderedTaskIDs(job.Tasks),
		JobID:         ref.JobID,
		TaskID:        ref.GroupID,
		JobInstanceID: jobInstanceID,
		Submissions:   domain.Submissions,
		ClosedGroups:  domain.ClosedGroups,
		Tasks:         job.Tasks,
	}) {
		return PathUnlock{}
	}
	group, ok := TaskByID(job.Tasks, ref.GroupID)
	if !ok || !group.IsGroup() {
		return PathUnlock{}
	}
	children := group.ChildTasks()
	childDone := false
	if ref.GroupInstanceID != "" && ref.GroupInstanceID != ProvisionalGroupInstanceID {
		if leaf, ok := TaskByID(children, ref.TaskID); ok && !taskSubmitMany(leaf) {
			childDone = hasGroupChildSubmission(
				domain.Submissions, ref.JobID, ref.GroupID, ref.GroupInstanceID, ref.TaskID, jobInstanceID,
			)
		}
	}
	if IsGroupClosed(domain.ClosedGroups, ref.JobID, jobInstanceID, ref.GroupID) {
		return PathUnlock{Unlocked: false, Done: childDone}
	}

	groupInstanceID := ref.GroupInstanceID
	if groupInstanceID == "" || groupInstanceID == ProvisionalGroupInstanceID {
		// Spawn-mint: only the first child leaf may open a new group instance.
		ids := OrderedTaskIDs(children)
		if len(ids) == 0 || ids[0] != ref.TaskID {
			return PathUnlock{}
		}
		return PathUnlock{Unlocked: true}
	}
	if !hasGroupInstance(domain, ref.JobID, jobInstanceID, ref.GroupID, groupInstanceID) {
		return PathUnlock{}
	}
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs:      OrderedTaskIDs(children),
		JobID:           ref.JobID,
		TaskID:          ref.TaskID,
		JobInstanceID:   jobInstanceID,
		GroupID:         ref.GroupID,
		GroupInstanceID: groupInstanceID,
		Submissions:     domain.Submissions,
		ClosedGroups:    domain.ClosedGroups,
		Tasks:           children,
	}) {
		return PathUnlock{Done: childDone}
	}
	return PathUnlock{Unlocked: true, Done: childDone}
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
