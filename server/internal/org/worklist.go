package org

import "strings"

// TaskStatus is the Worklist chrome vocabulary for one Task on a Job instance.
// Path Done (submit: once Submission) yields done; otherwise available then
// readyWhen (false ⇒ unavailable) and role eligibility (available vs actionable).
// submit: many never done from one Commit.
type TaskStatus string

const (
	TaskStatusUnavailable TaskStatus = "unavailable"
	TaskStatusAvailable   TaskStatus = "available"
	TaskStatusActionable  TaskStatus = "actionable"
	TaskStatusDone        TaskStatus = "done"
)

// Worklist is the server-projected operator board for one org.
type Worklist struct {
	Jobs []WorklistJob `json:"jobs"`
}

// WorklistTask is Config-free Task chrome (title + roles + submit) for board display.
type WorklistTask struct {
	ID     string   `json:"id"`
	Title  string   `json:"title"`
	Roles  []string `json:"roles"`
	Submit string   `json:"submit"` // once (default) | many — SPA completion ignores many
}

// WorklistJob is one Job's start affordance and per-instance Task statuses.
// TaskIDs / Title / StartLabel / Color / Repeatable / Tasks are chrome-only (not a second gate body).
type WorklistJob struct {
	JobID      string             `json:"jobId"`
	CanStart   bool               `json:"canStart"`
	Repeatable bool               `json:"repeatable"`
	Title      string             `json:"title,omitempty"`
	StartLabel string             `json:"startLabel,omitempty"`
	Color      string             `json:"color,omitempty"`
	TaskIDs    []string           `json:"taskIds"`
	Tasks      []WorklistTask     `json:"tasks"`
	Instances  []WorklistInstance `json:"instances"`
}

// WorklistInstance is one Job instance's viewer-scoped Task status map.
// Leaf statuses are unavailable | available | actionable | done.
// SubmissionCounts is Domain Submission N per flat leaf taskId;
// omitted when empty. Overview uses this for many-submit chrome without recap.
// Finished is platform law: every submit: once Task is done (many ignored).
// BlockedReasons carries blockedReason text for unavailable leaves only.
type WorklistInstance struct {
	ID               string                `json:"id"`
	Label            string                `json:"label"`
	Tasks            map[string]TaskStatus `json:"tasks"`
	SubmissionCounts map[string]int        `json:"submissionCounts,omitempty"`
	Finished         bool                  `json:"finished"`
	BlockedReasons   map[string]string     `json:"blockedReasons,omitempty"`
}

// ProjectWorklist evaluates startsWhen into canStart + Task statuses for one Org
// stream instance, keeping only member Job instances (§3 / §4). canStart first-Task
// readyWhen and leaf readyWhen/blockedReason bind $stream. submit: once (default)
// is done after Submission; submit: many never becomes done from one Commit (ADR-0008).
// Not-done leaves fold readyWhen (false ⇒ unavailable), else split by role
// eligibility into available vs actionable (ADR-0005).
func ProjectWorklist(cfg OrgConfig, domain Domain, op OperatorIdentity, streamInstanceID string) Worklist {
	return projectWorklist(cfg, domain, op, streamBindingFrom(cfg, streamInstanceID), true)
}

// ProjectWorklistAll is the unfiltered board (QR resolve, stream list, Play).
// Prefer ProjectWorklist when an instance id is known.
func ProjectWorklistAll(cfg OrgConfig, domain Domain, op OperatorIdentity) Worklist {
	return projectWorklistUnfiltered(cfg, domain, op)
}

func projectWorklistUnfiltered(cfg OrgConfig, domain Domain, op OperatorIdentity) Worklist {
	// Bind $stream.key (no id) so Play / plant-wide projection can still read the stream slug.
	return projectWorklist(cfg, domain, op, streamBindingFrom(cfg, ""), false)
}

func projectWorklist(cfg OrgConfig, domain Domain, op OperatorIdentity, stream *streamBinding, filterMembers bool) Worklist {
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
		out.Jobs = append(out.Jobs, projectWorklistJob(cfg, domain, jobID, job, op, stream, filterMembers))
	}
	return out
}

func projectWorklistJob(cfg OrgConfig, domain Domain, jobID string, job Job, op OperatorIdentity, stream *streamBinding, filterMembers bool) WorklistJob {
	refTask := referencedStartsWhenTask(cfg, job.StartsWhen)
	canStart := CanStartJob(job, jobID, domain.Submissions, domain.JobInstances, refTask)

	ordered := OrderedTaskIDs(job.Tasks)
	chrome := projectWorklistTasks(job.Tasks, ordered)
	// Provisional start has no instance yet; evaluate first-Task readyWhen over Domain (+ $stream).
	if canStart && len(ordered) > 0 {
		if tc, ok := TaskByID(job.Tasks, ordered[0]); ok &&
			!readyWhenOkForCommit(domain, taskPayload(tc), "", cfg.Catalogs, stream) {
			canStart = false
		}
	}
	if canStart && (len(chrome) == 0 || !rolesIntersect(op.Roles, chrome[0].Roles)) {
		canStart = false
	}
	// Instance-scoped Worklist: anchor canStart would open a sibling stream
	// instance; Play owns that affordance (ADR-0014 / #361).
	if canStart && filterMembers && stream != nil && stream.ID != "" &&
		cfg.Stream != nil && jobID == cfg.Stream.Instance.Job {
		canStart = false
	}

	instances := make([]WorklistInstance, 0)
	for _, inst := range domain.JobInstances {
		if inst.JobID != jobID {
			continue
		}
		if filterMembers {
			streamID := ""
			if stream != nil {
				streamID = stream.ID
			}
			if !JobInstanceBelongsToStream(cfg, domain, jobID, inst.ID, streamID) {
				continue
			}
		}
		tasks := make(map[string]TaskStatus, len(ordered))
		blocked := map[string]string{}
		for i, taskID := range ordered {
			tc, ok := TaskByID(job.Tasks, taskID)
			path := projectTaskStatus(cfg, domain, jobID, taskID, inst.ID)
			status := path
			if ok {
				status = foldReadyWhen(domain, path, tc, inst.ID, cfg.Catalogs, stream)
				if status == TaskStatusUnavailable {
					if reason := evaluateBlockedReason(domain, taskPayload(tc), inst.ID, cfg.Catalogs, stream); reason != "" {
						blocked[taskID] = reason
					}
				}
			}
			eligible := i < len(chrome) && rolesIntersect(op.Roles, chrome[i].Roles)
			tasks[taskID] = viewerScopedStatus(status, eligible)
		}
		wlInst := WorklistInstance{
			ID:               inst.ID,
			Label:            inst.Label,
			Tasks:            tasks,
			SubmissionCounts: countFlatLeafSubmissions(domain.Submissions, jobID, inst.ID),
		}
		if len(blocked) > 0 {
			wlInst.BlockedReasons = blocked
		}
		wlInst.Finished = worklistInstanceIsFinished(job, wlInst)
		instances = append(instances, wlInst)
	}

	return WorklistJob{
		JobID:      jobID,
		CanStart:   canStart,
		Repeatable: job.Repeatable,
		Title:      job.Title,
		StartLabel: job.StartLabel,
		Color:      job.Color,
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
			out = append(out, WorklistTask{
				ID: taskID, Title: taskID, Roles: []string{}, Submit: submitOnce,
			})
			continue
		}
		payload := taskPayload(task)
		roles := rolesFromTask(task)
		if roles == nil {
			roles = []string{}
		}
		out = append(out, WorklistTask{
			ID:     taskID,
			Title:  taskTitle(payload, taskID),
			Roles:  roles,
			Submit: taskSubmit(task),
		})
	}
	return out
}

func pathAvailabilityStatus(ev PathAvailability) TaskStatus {
	if ev.Done {
		return TaskStatusDone
	}
	return TaskStatusAvailable
}

// foldReadyWhen applies Task readyWhen after path says available-not-done.
// Same semantics as Commit (readyWhenOkForCommit): omit/blank ⇒ true; fail ⇒ false.
// False ⇒ unavailable (caller must not role-split).
func foldReadyWhen(domain Domain, path TaskStatus, task TaskConfig, jobInstanceID string, catalogs map[string]any, stream ...*streamBinding) TaskStatus {
	if path != TaskStatusAvailable {
		return path
	}
	if !readyWhenOkForCommit(domain, taskPayload(task), jobInstanceID, catalogs, stream...) {
		return TaskStatusUnavailable
	}
	return path
}

// worklistInstanceIsFinished matches SPA worklist-overview: every submit: once
// Task is done; submit: many is ignored; no Tasks ⇒ not finished.
func worklistInstanceIsFinished(job Job, inst WorklistInstance) bool {
	ordered := OrderedTaskIDs(job.Tasks)
	if len(ordered) == 0 {
		return false
	}
	for _, tid := range ordered {
		tc, ok := TaskByID(job.Tasks, tid)
		if ok && taskSubmitMany(tc) {
			continue
		}
		if inst.Tasks[tid] != TaskStatusDone {
			return false
		}
	}
	return true
}

func evaluateBlockedReason(domain Domain, task map[string]any, jobInstanceID string, catalogs map[string]any, stream *streamBinding) string {
	if task == nil {
		return ""
	}
	raw, ok := task["blockedReason"]
	if !ok || raw == nil {
		return ""
	}
	expr, ok := raw.(string)
	if !ok || strings.TrimSpace(expr) == "" {
		return ""
	}
	value, err := evaluateExpr(expr, nil, domainBindingsWith(domain, jobInstanceID, catalogs, "", stream))
	if err != nil || value == nil {
		return ""
	}
	s, ok := value.(string)
	if !ok {
		return ""
	}
	return s
}

// viewerScopedStatus maps path Available into actionable when the viewer
// intersects Config roles; otherwise leaves available for handoff.
// unavailable / done pass through (no role split).
func viewerScopedStatus(path TaskStatus, eligible bool) TaskStatus {
	if path != TaskStatusAvailable {
		return path
	}
	if eligible {
		return TaskStatusActionable
	}
	return TaskStatusAvailable
}

func projectTaskStatus(cfg OrgConfig, domain Domain, jobID, taskID, jobInstanceID string) TaskStatus {
	return pathAvailabilityStatus(EvaluatePathAvailability(cfg, domain, TaskPathRef{JobID: jobID, TaskID: taskID}, jobInstanceID))
}

func countFlatLeafSubmissions(subs []Submission, jobID, jobInstanceID string) map[string]int {
	var out map[string]int
	for _, s := range subs {
		if s.JobID != jobID || s.JobInstanceID != jobInstanceID {
			continue
		}
		if out == nil {
			out = map[string]int{}
		}
		out[s.TaskID]++
	}
	return out
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

// worklistInstanceHasRemainingWork is true when any submit: once Task on the
// instance is not done. submit: many leaves are ignored — they never become
// done and must not block Completed / home navigation (ADR-0008).
func worklistInstanceHasRemainingWork(cfg OrgConfig, wl Worklist, jobID, jobInstanceID string) bool {
	job, ok := worklistJobByID(wl, jobID)
	if !ok {
		return false
	}
	inst, ok := worklistInstance(job, jobInstanceID)
	if !ok {
		return false
	}
	var cfgTasks []TaskConfig
	if jobVal, ok := cfg.Jobs.Get(jobID); ok {
		if j, ok := AsJob(jobVal); ok {
			cfgTasks = j.Tasks
		}
	}
	for _, tid := range job.TaskIDs {
		if tc, ok := TaskByID(cfgTasks, tid); ok && taskSubmitMany(tc) {
			continue
		}
		if inst.Tasks[tid] != TaskStatusDone {
			return true
		}
	}
	return false
}

// ActionableTaskRef addresses one Worklist-actionable Task on an instance.
type ActionableTaskRef struct {
	JobID         string
	TaskID        string
	JobInstanceID string
}

// CollectActionableTaskRefs walks Worklist in Org Config Job order: provisional
// first Task when canStart (empty jobInstanceId), then every actionable leaf on
// each instance. canStart / actionable statuses are already viewer-scoped by
// ProjectWorklist.
func CollectActionableTaskRefs(cfg OrgConfig, wl Worklist) []ActionableTaskRef {
	byJob := make(map[string]WorklistJob, len(wl.Jobs))
	for _, j := range wl.Jobs {
		byJob[j.JobID] = j
	}
	out := make([]ActionableTaskRef, 0)
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
			first := ordered[0]
			if _, ok := TaskByID(job.Tasks, first); ok {
				out = append(out, ActionableTaskRef{
					JobID:         jobID,
					TaskID:        first,
					JobInstanceID: "",
				})
			}
		}
		for _, inst := range wlJob.Instances {
			for _, taskID := range ordered {
				if _, ok := TaskByID(job.Tasks, taskID); !ok {
					continue
				}
				if inst.Tasks[taskID] != TaskStatusActionable {
					continue
				}
				out = append(out, ActionableTaskRef{
					JobID:         jobID,
					TaskID:        taskID,
					JobInstanceID: inst.ID,
				})
			}
		}
	}
	return out
}

// FirstActionableTask returns the first CollectActionableTaskRefs entry.
func FirstActionableTask(cfg OrgConfig, wl Worklist) (ActionableTaskRef, bool) {
	refs := CollectActionableTaskRefs(cfg, wl)
	if len(refs) == 0 {
		return ActionableTaskRef{}, false
	}
	return refs[0], true
}

// NormalizeJobInstanceID maps SPA provisional "new" to the empty instance id
// used by CollectActionableTaskRefs / BuildTaskFormBundle.
func NormalizeJobInstanceID(jobInstanceID string) string {
	id := strings.TrimSpace(jobInstanceID)
	if id == "new" {
		return ""
	}
	return id
}

// NextAdvanceTaskRef picks the first projected flat actionable Task on
// jobInstanceID that allows advanceTarget (ADR-0006 auto-advance). Empty
// jobInstanceID never advances.
func NextAdvanceTaskRef(cfg OrgConfig, wl Worklist, jobID, jobInstanceID string) (ActionableTaskRef, bool) {
	jobID = strings.TrimSpace(jobID)
	jobInstanceID = strings.TrimSpace(jobInstanceID)
	if jobID == "" || jobInstanceID == "" {
		return ActionableTaskRef{}, false
	}
	job, ok := worklistJobByID(wl, jobID)
	if !ok {
		return ActionableTaskRef{}, false
	}
	inst, ok := worklistInstance(job, jobInstanceID)
	if !ok {
		return ActionableTaskRef{}, false
	}
	var cfgTasks []TaskConfig
	if jobVal, ok := cfg.Jobs.Get(jobID); ok {
		if j, ok := AsJob(jobVal); ok {
			cfgTasks = j.Tasks
		}
	}
	for _, taskID := range job.TaskIDs {
		if inst.Tasks[taskID] != TaskStatusActionable {
			continue
		}
		if leaf, ok := TaskByID(cfgTasks, taskID); ok && !taskAdvanceTarget(map[string]any(leaf)) {
			continue
		}
		return ActionableTaskRef{
			JobID:         jobID,
			TaskID:        taskID,
			JobInstanceID: jobInstanceID,
		}, true
	}
	return ActionableTaskRef{}, false
}

// ViewerMayMountTask reports whether addressed Task form hydrate is in board
// scope for the viewer (same CollectActionableTaskRefs law as actionable chrome).
func ViewerMayMountTask(cfg OrgConfig, wl Worklist, ref ActionableTaskRef) bool {
	ref.JobInstanceID = NormalizeJobInstanceID(ref.JobInstanceID)
	for _, got := range CollectActionableTaskRefs(cfg, wl) {
		if got.JobID == ref.JobID && got.TaskID == ref.TaskID && got.JobInstanceID == ref.JobInstanceID {
			return true
		}
	}
	return false
}

func containsString(ss []string, want string) bool {
	for _, s := range ss {
		if s == want {
			return true
		}
	}
	return false
}
