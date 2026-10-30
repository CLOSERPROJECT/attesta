package living

// TaskStatus is the Worklist chrome vocabulary for one Task on a Job instance.
type TaskStatus string

const (
	TaskStatusLocked    TaskStatus = "locked"
	TaskStatusAvailable TaskStatus = "available"
	TaskStatusDone      TaskStatus = "done"
)

// Worklist is the server-projected operator board for one org.
type Worklist struct {
	Jobs []WorklistJob `json:"jobs"`
}

// WorklistJob is one Job's start affordance and per-instance Task statuses.
// TaskIDs / StartLabel / Repeatable are chrome-only (not a second gate body).
type WorklistJob struct {
	JobID      string             `json:"jobId"`
	CanStart   bool               `json:"canStart"`
	Repeatable bool               `json:"repeatable"`
	StartLabel string             `json:"startLabel,omitempty"`
	TaskIDs    []string           `json:"taskIds"`
	Instances  []WorklistInstance `json:"instances"`
}

// WorklistInstance is one Job instance's Task status map.
type WorklistInstance struct {
	ID    string                `json:"id"`
	Label string                `json:"label"`
	Tasks map[string]TaskStatus `json:"tasks"`
}

// ProjectWorklist evaluates Unlock order, startsWhen, and Leftovers openness
// into canStart + Task statuses. Living code stays generic (no plant Task paths).
func ProjectWorklist(cfg OrgConfig, domain Domain) Worklist {
	out := Worklist{Jobs: make([]WorklistJob, 0, cfg.Jobs.Len())}
	for _, jobID := range cfg.Jobs.Keys {
		jobVal, ok := cfg.Jobs.Get(jobID)
		if !ok {
			continue
		}
		job, ok := AsJob(jobVal)
		if !ok {
			continue
		}
		out.Jobs = append(out.Jobs, projectWorklistJob(cfg, domain, jobID, job))
	}
	return out
}

func projectWorklistJob(cfg OrgConfig, domain Domain, jobID string, job Job) WorklistJob {
	refTask := referencedStartsWhenTask(cfg, job.StartsWhen)
	canStart := CanStartJob(job, jobID, domain.Submissions, domain.JobInstances, refTask)

	ordered := OrderedTaskIDs(job.Tasks)
	instances := make([]WorklistInstance, 0)
	for _, inst := range domain.JobInstances {
		if inst.JobID != jobID {
			continue
		}
		tasks := make(map[string]TaskStatus, len(ordered))
		for _, taskID := range ordered {
			tasks[taskID] = projectTaskStatus(domain, jobID, job.Tasks, ordered, taskID, inst.ID)
		}
		instances = append(instances, WorklistInstance{
			ID:    inst.ID,
			Label: inst.Label,
			Tasks: tasks,
		})
	}

	return WorklistJob{
		JobID:      jobID,
		CanStart:   canStart,
		Repeatable: job.Repeatable,
		StartLabel: job.StartLabel,
		TaskIDs:    ordered,
		Instances:  instances,
	}
}

func projectTaskStatus(
	domain Domain,
	jobID string,
	tasks []TaskConfig,
	ordered []string,
	taskID string,
	jobInstanceID string,
) TaskStatus {
	if !IsTaskUnlocked(UnlockQuery{
		OrderedIDs:    ordered,
		JobID:         jobID,
		TaskID:        taskID,
		JobInstanceID: jobInstanceID,
		Submissions:   domain.Submissions,
		Tasks:         tasks,
	}) {
		return TaskStatusLocked
	}

	task, ok := TaskByID(tasks, taskID)
	if !ok {
		return TaskStatusLocked
	}
	payload := taskPayload(task)
	if needsLeftoverProjection(payload) {
		_, completeness, ok := primaryDynamicProperty(payload)
		if ok && completeness == "some" {
			ids, err := leftoverIDsForTask(domain, jobID, payload, jobInstanceID)
			if err != nil {
				return TaskStatusLocked
			}
			if len(ids) > 0 {
				return TaskStatusAvailable
			}
			return TaskStatusDone
		}
	}

	if hasSubmissionOnInstance(domain.Submissions, jobID, taskID, jobInstanceID) {
		return TaskStatusDone
	}
	return TaskStatusAvailable
}
