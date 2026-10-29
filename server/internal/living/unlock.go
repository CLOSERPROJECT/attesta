package living

// UnlockQuery asks whether a Task is unlocked on a Job instance (in-Job Unlock order).
type UnlockQuery struct {
	OrderedIDs    []string
	JobID         string
	TaskID        string
	JobInstanceID string // empty only valid for the first Task
	Submissions   []Submission
	Tasks         OrderedObject
}

// OrderedTaskIDs returns Task ids in authored (Unlock) order.
func OrderedTaskIDs(tasks OrderedObject) []string {
	out := make([]string, len(tasks.Keys))
	copy(out, tasks.Keys)
	return out
}

// IsTaskUnlocked: Task k+1 unlocks when predecessor k has a Submission on this
// Job instance. First Task is always unlocked. Distinct from startsWhen / readyWhen.
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

	previousID := q.OrderedIDs[index-1]
	previous, ok := q.Tasks.Get(previousID)
	if !ok {
		return false
	}
	if !taskMayBeUnlockGate(previous) {
		current, _ := q.Tasks.Get(q.TaskID)
		return taskUnlocksWithInstance(current) && q.JobInstanceID != ""
	}
	return hasSubmissionOnInstance(q.Submissions, q.JobID, previousID, q.JobInstanceID)
}

func hasSubmissionOnInstance(subs []Submission, jobID, taskID, jobInstanceID string) bool {
	for _, s := range subs {
		if s.JobID == jobID && s.TaskID == taskID && s.JobInstanceID == jobInstanceID {
			return true
		}
	}
	return false
}

// Predecessor with Leftovers completeness "some" must not unlock the next Task.
func taskMayBeUnlockGate(task any) bool {
	m, ok := task.(map[string]any)
	if !ok {
		return true
	}
	dyn, ok := m["dynamicProperties"].(map[string]any)
	if !ok {
		return true
	}
	for _, raw := range dyn {
		def, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if def["completeness"] == "some" {
			return false
		}
	}
	return true
}

func taskUnlocksWithInstance(task any) bool {
	m, ok := task.(map[string]any)
	if !ok {
		return false
	}
	return m["unlockFrom"] == "instance"
}

func sequenceOkForCommit(cfg OrgConfig, domain Domain, jobID, taskID, jobInstanceID string) bool {
	jobVal, ok := cfg.Jobs.Get(jobID)
	if !ok {
		return false
	}
	job, ok := jobVal.(Job)
	if !ok {
		return false
	}
	return IsTaskUnlocked(UnlockQuery{
		OrderedIDs:    OrderedTaskIDs(job.Tasks),
		JobID:         jobID,
		TaskID:        taskID,
		JobInstanceID: jobInstanceID,
		Submissions:   domain.Submissions,
		Tasks:         job.Tasks,
	})
}
