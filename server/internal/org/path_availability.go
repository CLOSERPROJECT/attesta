package org

// PathAvailability is submit:once Done law for one TaskPathRef on a Job instance
// (submit: many never Done). startsWhen / readyWhen stay outside.
type PathAvailability struct {
	Done bool
}

// EvaluatePathAvailability is path Done law (Org Config + Domain Submissions).
// Sibling Unlock / unlockFrom are not platform gates — plants use readyWhen.
func EvaluatePathAvailability(cfg OrgConfig, domain Domain, ref TaskPathRef, jobInstanceID string) PathAvailability {
	jobVal, ok := cfg.Jobs.Get(ref.JobID)
	if !ok {
		return PathAvailability{}
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return PathAvailability{}
	}
	return evaluateFlatPathAvailability(job, domain, ref, jobInstanceID)
}

func evaluateFlatPathAvailability(job Job, domain Domain, ref TaskPathRef, jobInstanceID string) PathAvailability {
	tc, ok := TaskByID(job.Tasks, ref.TaskID)
	if !ok {
		return PathAvailability{}
	}
	// submit: many never becomes done from Submission (ADR-0008); once (default) does.
	done := !taskSubmitMany(tc) &&
		hasFlatSubmissionOnInstance(domain.Submissions, ref.JobID, ref.TaskID, jobInstanceID)
	return PathAvailability{Done: done}
}

func hasFlatSubmissionOnInstance(subs []Submission, jobID, taskID, jobInstanceID string) bool {
	for _, s := range subs {
		if s.JobID == jobID && s.TaskID == taskID && s.JobInstanceID == jobInstanceID {
			return true
		}
	}
	return false
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
	if _, ok := TaskByID(job.Tasks, ref.TaskID); !ok {
		return false
	}
	// Done (submit: once after Submission) blocks Commit; many never Done ⇒ repeats OK.
	return !EvaluatePathAvailability(cfg, domain, ref, jobInstanceID).Done && startsWhenOk(cfg, job, domain)
}

func startsWhenOk(cfg OrgConfig, job Job, domain Domain) bool {
	if job.StartsWhen == nil {
		return true
	}
	return IsStartsWhenSatisfied(job.StartsWhen, domain.Submissions, domain.JobInstances, referencedStartsWhenTask(cfg, job.StartsWhen))
}
