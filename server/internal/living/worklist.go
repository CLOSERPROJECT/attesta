package living

import "strings"

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

// WorklistTask is Config-free Task chrome (title + roles) for board display.
type WorklistTask struct {
	ID    string   `json:"id"`
	Title string   `json:"title"`
	Roles []string `json:"roles"`
}

// WorklistJob is one Job's start affordance and per-instance Task statuses.
// TaskIDs / Title / StartLabel / Repeatable / Tasks are chrome-only (not a second gate body).
type WorklistJob struct {
	JobID      string             `json:"jobId"`
	CanStart   bool               `json:"canStart"`
	Repeatable bool               `json:"repeatable"`
	Title      string             `json:"title,omitempty"`
	StartLabel string             `json:"startLabel,omitempty"`
	TaskIDs    []string           `json:"taskIds"`
	Tasks      []WorklistTask     `json:"tasks"`
	Instances  []WorklistInstance `json:"instances"`
}

// WorklistInstance is one Job instance's Task status map plus viewer Available refs.
// Tasks stay role-blind (handoff); ViewerAvailable is status-available ∩ slug∩roles.
type WorklistInstance struct {
	ID              string                `json:"id"`
	Label           string                `json:"label"`
	Tasks           map[string]TaskStatus `json:"tasks"`
	ViewerAvailable []string              `json:"viewerAvailable"`
}

// ProjectWorklist evaluates Unlock order and startsWhen into canStart + Task
// statuses, then scopes canStart to OperatorIdentity Config-role intersection
// with the Job's first Task (board chrome; not Cerbos). A Task is done after
// Submission on the instance (Submission law). Status vocabulary stays unlock /
// Submission based so overview can hand off role-gated available work.
func ProjectWorklist(cfg OrgConfig, domain Domain, op OperatorIdentity) Worklist {
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
		out.Jobs = append(out.Jobs, projectWorklistJob(cfg, domain, jobID, job, op))
	}
	return out
}

func projectWorklistJob(cfg OrgConfig, domain Domain, jobID string, job Job, op OperatorIdentity) WorklistJob {
	refTask := referencedStartsWhenTask(cfg, job.StartsWhen)
	canStart := CanStartJob(job, jobID, domain.Submissions, domain.JobInstances, refTask)

	ordered := OrderedTaskIDs(job.Tasks)
	chrome := projectWorklistTasks(job.Tasks, ordered)
	if canStart && (len(chrome) == 0 || !rolesIntersect(op.Roles, chrome[0].Roles)) {
		canStart = false
	}

	instances := make([]WorklistInstance, 0)
	for _, inst := range domain.JobInstances {
		if inst.JobID != jobID {
			continue
		}
		tasks := make(map[string]TaskStatus, len(ordered))
		viewerAvailable := make([]string, 0)
		for i, taskID := range ordered {
			status := projectTaskStatus(domain, jobID, job.Tasks, ordered, taskID, inst.ID)
			tasks[taskID] = status
			if status != TaskStatusAvailable {
				continue
			}
			if i >= len(chrome) || !rolesIntersect(op.Roles, chrome[i].Roles) {
				continue
			}
			viewerAvailable = append(viewerAvailable, taskID)
		}
		instances = append(instances, WorklistInstance{
			ID:              inst.ID,
			Label:           inst.Label,
			Tasks:           tasks,
			ViewerAvailable: viewerAvailable,
		})
	}

	return WorklistJob{
		JobID:      jobID,
		CanStart:   canStart,
		Repeatable: job.Repeatable,
		Title:      job.Title,
		StartLabel: job.StartLabel,
		TaskIDs:    ordered,
		Tasks:      chrome,
		Instances:  instances,
	}
}

func projectWorklistTasks(tasks []TaskConfig, ordered []string) []WorklistTask {
	out := make([]WorklistTask, 0, len(ordered))
	for _, taskID := range ordered {
		task, ok := TaskByID(tasks, taskID)
		if !ok {
			out = append(out, WorklistTask{ID: taskID, Title: taskID, Roles: []string{}})
			continue
		}
		payload := taskPayload(task)
		roles := rolesFromTask(task)
		if roles == nil {
			roles = []string{}
		}
		out = append(out, WorklistTask{
			ID:    taskID,
			Title: taskTitle(payload, taskID),
			Roles: roles,
		})
	}
	return out
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

	if _, ok := TaskByID(tasks, taskID); !ok {
		return TaskStatusLocked
	}
	if hasSubmissionOnInstance(domain.Submissions, jobID, taskID, jobInstanceID) {
		return TaskStatusDone
	}
	return TaskStatusAvailable
}

// rolesIntersect is slug∩roles board chrome (not Cerbos).
func rolesIntersect(viewer, allowed []string) bool {
	if len(viewer) == 0 || len(allowed) == 0 {
		return false
	}
	have := make(map[string]struct{}, len(viewer))
	for _, r := range viewer {
		if r == "" {
			continue
		}
		have[r] = struct{}{}
	}
	for _, r := range allowed {
		if r == "" {
			continue
		}
		if _, ok := have[r]; ok {
			return true
		}
	}
	return false
}

func worklistTaskByID(job WorklistJob, taskID string) (WorklistTask, bool) {
	for _, t := range job.Tasks {
		if t.ID == taskID {
			return t, true
		}
	}
	return WorklistTask{}, false
}

func worklistJobByID(wl Worklist, jobID string) (WorklistJob, bool) {
	for _, j := range wl.Jobs {
		if j.JobID == jobID {
			return j, true
		}
	}
	return WorklistJob{}, false
}

func worklistInstance(job WorklistJob, id string) (WorklistInstance, bool) {
	for _, inst := range job.Instances {
		if inst.ID == id {
			return inst, true
		}
	}
	return WorklistInstance{}, false
}

// worklistInstanceHasRemainingWork is true when any Task on the instance is not done
// (ADR-0006 overview vs home after Commit when auto-advance does not apply).
func worklistInstanceHasRemainingWork(wl Worklist, jobID, jobInstanceID string) bool {
	job, ok := worklistJobByID(wl, jobID)
	if !ok {
		return false
	}
	inst, ok := worklistInstance(job, jobInstanceID)
	if !ok {
		return false
	}
	for _, tid := range job.TaskIDs {
		if inst.Tasks[tid] != TaskStatusDone {
			return true
		}
	}
	return false
}

// AvailableTaskRef addresses one Worklist-available Task on an instance.
type AvailableTaskRef struct {
	JobID         string
	TaskID        string
	JobInstanceID string
}

// CollectAvailableTaskRefs walks Worklist in Org Config Job order: provisional
// first Task when canStart (empty jobInstanceId), then each instance's projected
// ViewerAvailable refs (authored Task order). canStart / ViewerAvailable are
// already viewer-scoped by ProjectWorklist.
func CollectAvailableTaskRefs(cfg OrgConfig, wl Worklist) []AvailableTaskRef {
	byJob := make(map[string]WorklistJob, len(wl.Jobs))
	for _, j := range wl.Jobs {
		byJob[j.JobID] = j
	}
	out := make([]AvailableTaskRef, 0)
	for _, jobID := range cfg.Jobs.Keys {
		jobVal, ok := cfg.Jobs.Get(jobID)
		if !ok {
			continue
		}
		job, ok := AsJob(jobVal)
		if !ok {
			continue
		}
		wlJob, ok := byJob[jobID]
		if !ok {
			continue
		}
		ordered := OrderedTaskIDs(job.Tasks)
		if wlJob.CanStart && len(ordered) > 0 {
			out = append(out, AvailableTaskRef{
				JobID:         jobID,
				TaskID:        ordered[0],
				JobInstanceID: "",
			})
		}
		for _, inst := range wlJob.Instances {
			for _, taskID := range inst.ViewerAvailable {
				out = append(out, AvailableTaskRef{
					JobID:         jobID,
					TaskID:        taskID,
					JobInstanceID: inst.ID,
				})
			}
		}
	}
	return out
}

// FirstAvailableTask returns the first CollectAvailableTaskRefs entry.
func FirstAvailableTask(cfg OrgConfig, wl Worklist) (AvailableTaskRef, bool) {
	refs := CollectAvailableTaskRefs(cfg, wl)
	if len(refs) == 0 {
		return AvailableTaskRef{}, false
	}
	return refs[0], true
}

// NormalizeJobInstanceID maps SPA provisional "new" to the empty instance id
// used by CollectAvailableTaskRefs / BuildTaskFormBundle.
func NormalizeJobInstanceID(jobInstanceID string) string {
	id := strings.TrimSpace(jobInstanceID)
	if id == "new" {
		return ""
	}
	return id
}

// NextAdvanceTaskRef picks the first projected ViewerAvailable Task on
// jobInstanceID (ADR-0006 auto-advance). Empty jobInstanceID never advances.
func NextAdvanceTaskRef(wl Worklist, jobID, jobInstanceID string) (AvailableTaskRef, bool) {
	jobID = strings.TrimSpace(jobID)
	jobInstanceID = strings.TrimSpace(jobInstanceID)
	if jobID == "" || jobInstanceID == "" {
		return AvailableTaskRef{}, false
	}
	job, ok := worklistJobByID(wl, jobID)
	if !ok {
		return AvailableTaskRef{}, false
	}
	inst, ok := worklistInstance(job, jobInstanceID)
	if !ok || len(inst.ViewerAvailable) == 0 {
		return AvailableTaskRef{}, false
	}
	return AvailableTaskRef{
		JobID:         jobID,
		TaskID:        inst.ViewerAvailable[0],
		JobInstanceID: jobInstanceID,
	}, true
}

// ViewerMayMountTask reports whether addressed Task form hydrate is in board
// scope for the viewer (same CollectAvailableTaskRefs law as Available chrome).
func ViewerMayMountTask(cfg OrgConfig, wl Worklist, ref AvailableTaskRef) bool {
	ref.JobInstanceID = NormalizeJobInstanceID(ref.JobInstanceID)
	for _, got := range CollectAvailableTaskRefs(cfg, wl) {
		if got.JobID == ref.JobID && got.TaskID == ref.TaskID && got.JobInstanceID == ref.JobInstanceID {
			return true
		}
	}
	return false
}
