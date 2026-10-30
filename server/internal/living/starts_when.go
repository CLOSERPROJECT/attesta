package living

// IsStartsWhenSatisfied: absent gate always allows. `some` = ≥1 matching Submission;
// `all` = every Job instance of the referenced Job has that Task Submission
// (and at least one instance exists). Completeness `some` Tasks never satisfy
// (fail closed) — they must not be startsWhen targets.
func IsStartsWhenSatisfied(startsWhen *StartsWhen, submissions []Submission, jobInstances []JobInstance, referencedTask any) bool {
	if startsWhen == nil {
		return true
	}
	if referencedTask == nil || !taskMayBeUnlockGate(referencedTask) {
		return false
	}

	jobID := startsWhen.Task.JobID
	taskID := startsWhen.Task.TaskID
	matching := make([]Submission, 0)
	for _, s := range submissions {
		if s.JobID == jobID && s.TaskID == taskID {
			matching = append(matching, s)
		}
	}

	if startsWhen.Quantifier == "all" {
		instances := make([]JobInstance, 0)
		for _, inst := range jobInstances {
			if inst.JobID == jobID {
				instances = append(instances, inst)
			}
		}
		if len(instances) == 0 {
			return false
		}
		for _, inst := range instances {
			found := false
			for _, s := range matching {
				if s.JobInstanceID == inst.ID {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}

	// default / "some"
	return len(matching) > 0
}

// CanStartJob: start affordance when startsWhen holds and (repeatable or no instance yet).
func CanStartJob(job Job, jobID string, submissions []Submission, jobInstances []JobInstance, referencedTask any) bool {
	n := 0
	for _, inst := range jobInstances {
		if inst.JobID == jobID {
			n++
		}
	}
	if !(job.Repeatable || n == 0) {
		return false
	}
	return IsStartsWhenSatisfied(job.StartsWhen, submissions, jobInstances, referencedTask)
}

func referencedStartsWhenTask(cfg OrgConfig, sw *StartsWhen) any {
	if sw == nil {
		return nil
	}
	jobVal, ok := cfg.Jobs.Get(sw.Task.JobID)
	if !ok {
		return nil
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return nil
	}
	task, ok := TaskByID(job.Tasks, sw.Task.TaskID)
	if !ok {
		return nil
	}
	return task
}
