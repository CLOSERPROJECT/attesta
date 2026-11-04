package living

import "strings"

// TaskStatus is the Worklist chrome vocabulary for one Task on a Job instance.
// Unlock/Submission yields locked / unlocked / done; unlocked-not-done folds
// Task readyWhen (false ⇒ unavailable); else role eligibility splits into
// available (not for this viewer) vs actionable.
type TaskStatus string

const (
	TaskStatusLocked      TaskStatus = "locked"
	TaskStatusUnavailable TaskStatus = "unavailable"
	TaskStatusAvailable   TaskStatus = "available"
	TaskStatusActionable  TaskStatus = "actionable"
	TaskStatusDone        TaskStatus = "done"
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
// TaskIDs / Title / StartLabel / Color / Repeatable / Tasks are chrome-only (not a second gate body).
// Tasks lists top-level chrome in TaskIDs order, then nested group-child chrome
// (lookup by id; TaskIDs stays top-level only).
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
// Leaf statuses are locked | unavailable | available | actionable | done;
// Groups is omitted on flat Jobs (backward-compatible wire).
type WorklistInstance struct {
	ID     string                   `json:"id"`
	Label  string                   `json:"label"`
	Tasks  map[string]TaskStatus    `json:"tasks"`
	Groups map[string]WorklistGroup `json:"groups,omitempty"`
}

// WorklistGroup is server-projected chrome for one Task group on a Job instance.
// SPA reads canSpawn / canClose and must not re-implement Unlock or close gates.
type WorklistGroup struct {
	GroupID      string                  `json:"groupId"`
	StartLabel   string                  `json:"startLabel,omitempty"`
	CloseLabel   string                  `json:"closeLabel,omitempty"`
	Repeatable   bool                    `json:"repeatable"`
	CanSpawn     bool                    `json:"canSpawn"`
	CanClose     bool                    `json:"canClose"`
	ChildTaskIDs []string                `json:"childTaskIds"`
	Instances    []WorklistGroupInstance `json:"instances"`
}

// WorklistGroupInstance is one open group instance's viewer-scoped child leaf statuses.
// Label is optional overview chrome: first nested Submission IDsTouched id
// under Config instanceLabelEntity; omit when unset or no match.
type WorklistGroupInstance struct {
	ID     string                `json:"id"`
	Label  string                `json:"label,omitempty"`
	Tasks  map[string]TaskStatus `json:"tasks"`
}

// ProjectWorklist evaluates Unlock order and startsWhen into canStart + Task
// statuses, then scopes canStart to first-Task readyWhen (omit ⇒ true; same
// Domain grammar as Commit) and OperatorIdentity Config-role intersection with
// the Job's first Task (board chrome; not Cerbos). A Task is done after
// Submission on the instance (Submission law). Unlocked leaves fold readyWhen
// (false ⇒ unavailable), else split by role eligibility into available vs
// actionable (ADR-0005).
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
	// Nested group children are not in TaskIDs; append their title/roles chrome
	// so overview can resolve leaf titles without a Config walk.
	for _, taskID := range ordered {
		tc, ok := TaskByID(job.Tasks, taskID)
		if !ok || !tc.IsGroup() {
			continue
		}
		children := tc.ChildTasks()
		chrome = append(chrome, projectWorklistTasks(children, OrderedTaskIDs(children))...)
	}
	// Provisional start has no instance yet; evaluate first-Task readyWhen over Domain only.
	if canStart && len(ordered) > 0 {
		if tc, ok := TaskByID(job.Tasks, ordered[0]); ok &&
			!readyWhenOkForCommit(domain, taskPayload(tc), "", cfg.Catalogs) {
			canStart = false
		}
	}
	if canStart && (len(chrome) == 0 || !rolesIntersect(op.Roles, chrome[0].Roles)) {
		canStart = false
	}

	instances := make([]WorklistInstance, 0)
	for _, inst := range domain.JobInstances {
		if inst.JobID != jobID {
			continue
		}
		tasks := make(map[string]TaskStatus, len(ordered))
		var groups map[string]WorklistGroup
		for i, taskID := range ordered {
			tc, ok := TaskByID(job.Tasks, taskID)
			if ok && tc.IsGroup() {
				// Group-node chrome stays Unlock/Close unless the group authors readyWhen.
				tasks[taskID] = foldReadyWhen(domain, projectGroupStatus(cfg, domain, jobID, taskID, inst.ID), tc, inst.ID, cfg.Catalogs)
				g := projectWorklistGroup(cfg, domain, jobID, inst.ID, tc, op)
				if groups == nil {
					groups = map[string]WorklistGroup{}
				}
				groups[taskID] = g
				continue
			}
			unlock := projectTaskStatus(cfg, domain, jobID, taskID, inst.ID)
			status := unlock
			if ok {
				status = foldReadyWhen(domain, unlock, tc, inst.ID, cfg.Catalogs)
			}
			eligible := i < len(chrome) && rolesIntersect(op.Roles, chrome[i].Roles)
			tasks[taskID] = viewerScopedStatus(status, eligible)
		}
		instances = append(instances, WorklistInstance{
			ID:     inst.ID,
			Label:  inst.Label,
			Tasks:  tasks,
			Groups: groups,
		})
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

func pathUnlockStatus(ev PathUnlock) TaskStatus {
	if ev.Done {
		return TaskStatusDone
	}
	if ev.Unlocked {
		return TaskStatusAvailable
	}
	return TaskStatusLocked
}

// foldReadyWhen applies Task readyWhen after path Unlock says unlocked-not-done.
// Same semantics as Commit (readyWhenOkForCommit): omit/blank ⇒ true; fail ⇒ false.
// False ⇒ unavailable (caller must not role-split).
func foldReadyWhen(domain Domain, unlock TaskStatus, task TaskConfig, jobInstanceID string, catalogs map[string]any) TaskStatus {
	if unlock != TaskStatusAvailable {
		return unlock
	}
	if !readyWhenOkForCommit(domain, taskPayload(task), jobInstanceID, catalogs) {
		return TaskStatusUnavailable
	}
	return unlock
}

// viewerScopedStatus maps unlock Available into actionable when the viewer
// intersects Config roles; otherwise leaves available for handoff.
// unavailable / locked / done pass through (no role split).
func viewerScopedStatus(unlock TaskStatus, eligible bool) TaskStatus {
	if unlock != TaskStatusAvailable {
		return unlock
	}
	if eligible {
		return TaskStatusActionable
	}
	return TaskStatusAvailable
}

func projectGroupStatus(cfg OrgConfig, domain Domain, jobID, groupID, jobInstanceID string) TaskStatus {
	return pathUnlockStatus(EvaluatePathUnlock(cfg, domain, TaskPathRef{JobID: jobID, TaskID: groupID}, jobInstanceID))
}

func projectWorklistGroup(
	cfg OrgConfig,
	domain Domain,
	jobID, jobInstanceID string,
	group TaskConfig,
	op OperatorIdentity,
) WorklistGroup {
	groupID := group.ID()
	children := group.ChildTasks()
	childIDs := OrderedTaskIDs(children)
	repeatable := taskBool(group, "repeatable")

	instances := make([]WorklistGroupInstance, 0)
	for _, gi := range domain.GroupInstances {
		if gi.JobID != jobID || gi.JobInstanceID != jobInstanceID || gi.GroupID != groupID {
			continue
		}
		childStatus := make(map[string]TaskStatus, len(childIDs))
		for _, childID := range childIDs {
			unlock := projectGroupChildStatus(cfg, domain, jobID, jobInstanceID, groupID, gi.ID, childID)
			child, ok := TaskByID(children, childID)
			status := unlock
			if ok {
				status = foldReadyWhen(domain, unlock, child, jobInstanceID, cfg.Catalogs)
			}
			eligible := ok && rolesIntersect(op.Roles, rolesFromTask(child))
			childStatus[childID] = viewerScopedStatus(status, eligible)
		}
		instances = append(instances, WorklistGroupInstance{
			ID:    gi.ID,
			Label: groupInstanceEntityLabel(domain, jobID, jobInstanceID, groupID, gi.ID, taskString(group, "instanceLabelEntity")),
			Tasks: childStatus,
		})
	}

	lc := EvaluateGroupLifecycle(GroupLifecycleQuery{
		Group:         group,
		JobID:         jobID,
		JobInstanceID: jobInstanceID,
		Cfg:           cfg,
		Domain:        domain,
	})
	canSpawn := lc.CanSpawn
	if canSpawn && len(childIDs) > 0 {
		if child, ok := TaskByID(children, childIDs[0]); ok {
			if !rolesIntersect(op.Roles, rolesFromTask(child)) {
				canSpawn = false
			}
		}
	}

	return WorklistGroup{
		GroupID:      groupID,
		StartLabel:   taskString(group, "startLabel"),
		CloseLabel:   taskString(group, "closeLabel"),
		Repeatable:   repeatable,
		CanSpawn:     canSpawn,
		CanClose:     lc.CanClose,
		ChildTaskIDs: childIDs,
		Instances:    instances,
	}
}

func projectGroupChildStatus(
	cfg OrgConfig,
	domain Domain,
	jobID, jobInstanceID, groupID, groupInstanceID, taskID string,
) TaskStatus {
	return pathUnlockStatus(EvaluatePathUnlock(cfg, domain, TaskPathRef{
		JobID: jobID, GroupID: groupID, GroupInstanceID: groupInstanceID, TaskID: taskID,
	}, jobInstanceID))
}

// groupInstanceEntityLabel returns the first IDsTouched id present under
// domain.Kind(entityKind) for a matching nested Submission; empty when
// entityKind is unset or nothing intersects.
func groupInstanceEntityLabel(domain Domain, jobID, jobInstanceID, groupID, groupInstanceID, entityKind string) string {
	if entityKind == "" {
		return ""
	}
	entities := domain.Kind(entityKind)
	if len(entities) == 0 {
		return ""
	}
	for _, sub := range domain.Submissions {
		if sub.JobID != jobID || sub.JobInstanceID != jobInstanceID ||
			sub.GroupID != groupID || sub.GroupInstanceID != groupInstanceID {
			continue
		}
		for _, id := range sub.IDsTouched {
			if _, ok := entities[id]; ok {
				return id
			}
		}
	}
	return ""
}

func projectTaskStatus(cfg OrgConfig, domain Domain, jobID, taskID, jobInstanceID string) TaskStatus {
	return pathUnlockStatus(EvaluatePathUnlock(cfg, domain, TaskPathRef{JobID: jobID, TaskID: taskID}, jobInstanceID))
}

func taskString(t TaskConfig, key string) string {
	if t == nil {
		return ""
	}
	s, _ := t[key].(string)
	return s
}

func taskBool(t TaskConfig, key string) bool {
	if t == nil {
		return false
	}
	v, _ := t[key].(bool)
	return v
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

// ActionableTaskRef addresses one Worklist-actionable Task on an instance.
// Nested leaves set GroupID; provisional spawn uses empty GroupInstanceID
// (Commit path uses ProvisionalGroupInstanceID with mint.spawningNew).
type ActionableTaskRef struct {
	JobID           string
	GroupID         string
	GroupInstanceID string
	TaskID          string
	JobInstanceID   string
}

// CollectActionableTaskRefs walks Worklist in Org Config Job order: provisional
// first Task when canStart (empty jobInstanceId), then every actionable leaf on
// each instance (flat + nested group children + provisional when canSpawn).
// canStart / actionable statuses / group chrome are already viewer-scoped by
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
			if tc, ok := TaskByID(job.Tasks, first); ok && !tc.IsGroup() {
				out = append(out, ActionableTaskRef{
					JobID:         jobID,
					TaskID:        first,
					JobInstanceID: "",
				})
			}
		}
		for _, inst := range wlJob.Instances {
			for _, taskID := range ordered {
				tc, ok := TaskByID(job.Tasks, taskID)
				if !ok {
					continue
				}
				if tc.IsGroup() {
					g, hit := inst.Groups[taskID]
					if !hit {
						continue
					}
					childIDs := g.ChildTaskIDs
					if len(childIDs) == 0 {
						childIDs = OrderedTaskIDs(tc.ChildTasks())
					}
					if g.CanSpawn && len(childIDs) > 0 {
						out = append(out, ActionableTaskRef{
							JobID:         jobID,
							GroupID:       taskID,
							TaskID:        childIDs[0],
							JobInstanceID: inst.ID,
						})
					}
					for _, gi := range g.Instances {
						for _, childID := range childIDs {
							if gi.Tasks[childID] != TaskStatusActionable {
								continue
							}
							out = append(out, ActionableTaskRef{
								JobID:           jobID,
								GroupID:         taskID,
								GroupInstanceID: gi.ID,
								TaskID:          childID,
								JobInstanceID:   inst.ID,
							})
						}
					}
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

// NormalizeGroupInstanceID maps SPA provisional "new" to empty group instance id
// used by CollectActionableTaskRefs provisional spawn refs.
func NormalizeGroupInstanceID(groupInstanceID string) string {
	id := strings.TrimSpace(groupInstanceID)
	if id == ProvisionalGroupInstanceID {
		return ""
	}
	return id
}

// NextAdvanceTaskRef picks the first projected flat actionable Task on
// jobInstanceID (ADR-0006 auto-advance). Nested group leaves stay overview
// while the group is open. Empty jobInstanceID never advances.
func NextAdvanceTaskRef(wl Worklist, jobID, jobInstanceID string) (ActionableTaskRef, bool) {
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
	for _, taskID := range job.TaskIDs {
		if _, isGroup := inst.Groups[taskID]; isGroup {
			continue
		}
		if inst.Tasks[taskID] != TaskStatusActionable {
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
	ref.GroupInstanceID = NormalizeGroupInstanceID(ref.GroupInstanceID)
	for _, got := range CollectActionableTaskRefs(cfg, wl) {
		if got.JobID == ref.JobID && got.TaskID == ref.TaskID && got.JobInstanceID == ref.JobInstanceID &&
			got.GroupID == ref.GroupID && got.GroupInstanceID == ref.GroupInstanceID {
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
