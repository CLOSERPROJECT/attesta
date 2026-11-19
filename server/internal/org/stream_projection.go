package org

import (
	"strings"
)

// StreamSubstepState is the /my timeline vocabulary for one Task (§4 mapping).
type StreamSubstepState string

const (
	StreamSubstepAvailable StreamSubstepState = "available"
	StreamSubstepPending   StreamSubstepState = "pending"
	StreamSubstepLocked    StreamSubstepState = "locked"
	StreamSubstepDone      StreamSubstepState = "done"
)

const (
	StreamInstanceActive = "active"
	StreamInstanceDone   = "done"
)

// StreamInstanceSummary is one Org stream instance for the /my dashboard.
type StreamInstanceSummary struct {
	ID        string
	Label     string
	CreatedAt string
	Status    string // active | done
	Steps     []StreamStepProgress
}

// StreamStepProgress is per-step done/total counts for a dashboard row.
type StreamStepProgress struct {
	JobID string
	Done  int
	Total int
}

// StreamTimeline is the /my instance timeline projection.
type StreamTimeline struct {
	Steps []StreamTimelineStep
}

// StreamTimelineStep is one stream.steps entry: once Tasks, or each Rows.
type StreamTimelineStep struct {
	JobID string
	Title string
	Tasks []StreamTimelineTask // once steps
	Rows  []StreamTimelineRow  // each steps
}

// StreamTimelineRow is one each repetition (entity id = Job instance id).
type StreamTimelineRow struct {
	ID       string
	Label    string
	Finished bool
	Tasks    []StreamTimelineTask
}

// StreamTimelineTask is one Task on a timeline step/row with mapped state.
type StreamTimelineTask struct {
	ID            string
	Title         string
	State         StreamSubstepState
	BlockedReason string
}

// MapWorklistStatusToStreamState maps Worklist TaskStatus → Stream substep state (§4).
func MapWorklistStatusToStreamState(status TaskStatus) StreamSubstepState {
	switch status {
	case TaskStatusActionable:
		return StreamSubstepAvailable
	case TaskStatusAvailable:
		return StreamSubstepPending
	case TaskStatusUnavailable:
		return StreamSubstepLocked
	case TaskStatusDone:
		return StreamSubstepDone
	default:
		return StreamSubstepLocked
	}
}

// ListStreamInstances returns one summary per anchor Job instance.
func ListStreamInstances(cfg OrgConfig, domain Domain, op OperatorIdentity) []StreamInstanceSummary {
	if cfg.Stream == nil {
		return nil
	}
	anchorJob := cfg.Stream.Instance.Job
	out := make([]StreamInstanceSummary, 0)
	for _, inst := range domain.JobInstances {
		if inst.JobID != anchorJob {
			continue
		}
		wl := ProjectWorklist(cfg, domain, op, inst.ID)
		steps := projectStepProgress(cfg, domain, wl, inst.ID)
		status := StreamInstanceActive
		if streamInstanceIsDone(cfg, domain, wl, inst.ID) {
			status = StreamInstanceDone
		}
		out = append(out, StreamInstanceSummary{
			ID:        inst.ID,
			Label:     inst.Label,
			CreatedAt: firstAnchorSubmissionTime(domain, anchorJob, inst.ID),
			Status:    status,
			Steps:     steps,
		})
	}
	return out
}

// ProjectStreamTimeline returns steps/rows/tasks for one Org stream instance.
func ProjectStreamTimeline(cfg OrgConfig, domain Domain, op OperatorIdentity, instanceID string) StreamTimeline {
	if cfg.Stream == nil || instanceID == "" {
		return StreamTimeline{}
	}
	wl := ProjectWorklist(cfg, domain, op, instanceID)
	stream := streamBindingFrom(cfg, instanceID)
	out := StreamTimeline{Steps: make([]StreamTimelineStep, 0, len(cfg.Stream.Steps))}
	for _, step := range cfg.Stream.Steps {
		jobVal, ok := cfg.Jobs.Get(step.Job)
		if !ok {
			continue
		}
		job, ok := AsJob(jobVal)
		if !ok {
			continue
		}
		wlJob, _ := worklistJobByID(wl, step.Job)
		tlStep := StreamTimelineStep{
			JobID: step.Job,
			Title: job.Title,
		}
		if step.Each != nil {
			tlStep.Rows = projectEachRows(cfg, domain, job, step, wlJob, op, stream, instanceID)
		} else {
			tlStep.Tasks = projectOnceTasks(job, wlJob, instanceID)
		}
		out.Steps = append(out.Steps, tlStep)
	}
	return out
}

func projectStepProgress(cfg OrgConfig, domain Domain, wl Worklist, streamInstanceID string) []StreamStepProgress {
	out := make([]StreamStepProgress, 0, len(cfg.Stream.Steps))
	for _, step := range cfg.Stream.Steps {
		prog := StreamStepProgress{JobID: step.Job}
		if step.Each != nil {
			ids, err := selectStreamEachIDs(cfg, domain, step.Each, streamInstanceID)
			if err != nil {
				out = append(out, prog)
				continue
			}
			prog.Total = len(ids)
			wlJob, _ := worklistJobByID(wl, step.Job)
			for _, id := range ids {
				if inst, ok := worklistInstance(wlJob, id); ok && inst.Finished {
					prog.Done++
				}
			}
		} else {
			prog.Total = 1
			wlJob, ok := worklistJobByID(wl, step.Job)
			if ok {
				if inst, ok := worklistInstance(wlJob, streamInstanceID); ok && inst.Finished {
					prog.Done = 1
				}
			}
		}
		out = append(out, prog)
	}
	return out
}

func streamInstanceIsDone(cfg OrgConfig, domain Domain, wl Worklist, streamInstanceID string) bool {
	anchor := cfg.Stream.Instance.Job
	wlJob, ok := worklistJobByID(wl, anchor)
	if !ok {
		return false
	}
	inst, ok := worklistInstance(wlJob, streamInstanceID)
	if !ok || !inst.Finished {
		return false
	}
	for _, step := range cfg.Stream.Steps {
		if step.Each == nil {
			continue
		}
		ids, err := selectStreamEachIDs(cfg, domain, step.Each, streamInstanceID)
		if err != nil {
			return false
		}
		eachJob, ok := worklistJobByID(wl, step.Job)
		if !ok {
			if len(ids) > 0 {
				return false
			}
			continue
		}
		for _, id := range ids {
			row, ok := worklistInstance(eachJob, id)
			if !ok || !row.Finished {
				return false
			}
		}
	}
	return true
}

func firstAnchorSubmissionTime(domain Domain, jobID, instanceID string) string {
	earliest := ""
	for _, s := range domain.Submissions {
		if s.JobID != jobID || s.JobInstanceID != instanceID {
			continue
		}
		if earliest == "" || s.Timestamp < earliest {
			earliest = s.Timestamp
		}
	}
	return earliest
}

func projectOnceTasks(job Job, wlJob WorklistJob, instanceID string) []StreamTimelineTask {
	inst, ok := worklistInstance(wlJob, instanceID)
	if !ok {
		return timelineTasksFromChrome(job, wlJob, nil, nil)
	}
	return timelineTasksFromChrome(job, wlJob, inst.Tasks, inst.BlockedReasons)
}

func projectEachRows(cfg OrgConfig, domain Domain, job Job, step StreamStep, wlJob WorklistJob, op OperatorIdentity, stream *streamBinding, streamInstanceID string) []StreamTimelineRow {
	ids, err := selectStreamEachIDs(cfg, domain, step.Each, streamInstanceID)
	if err != nil {
		return nil
	}
	ordered := OrderedTaskIDs(job.Tasks)
	rows := make([]StreamTimelineRow, 0, len(ids))
	for _, id := range ids {
		if inst, ok := worklistInstance(wlJob, id); ok {
			rows = append(rows, StreamTimelineRow{
				ID:       id,
				Label:    inst.Label,
				Finished: inst.Finished,
				Tasks:    timelineTasksFromChrome(job, wlJob, inst.Tasks, inst.BlockedReasons),
			})
			continue
		}
		rows = append(rows, StreamTimelineRow{
			ID:       id,
			Label:    labelForPendingEachRow(cfg, domain, job, step.Each.Entity, id, stream),
			Finished: false,
			Tasks:    provisionalStartTasks(cfg, domain, job, ordered, wlJob, op, stream),
		})
	}
	return rows
}

func labelForPendingEachRow(cfg OrgConfig, domain Domain, job Job, entity, rowID string, stream *streamBinding) string {
	facts := map[string]any{"id": rowID, "lotId": rowID}
	if m := domain.Kind(entity); m != nil {
		if entityFacts, ok := m[rowID]; ok {
			for k, v := range entityFacts {
				facts[k] = v
			}
		}
	}
	facts["id"] = rowID
	facts["lotId"] = rowID
	labelRaw, err := evaluateExpr(job.Label, facts, domainBindingsWith(domain, "", cfg.Catalogs, "", stream))
	if err != nil || labelRaw == nil {
		return rowID
	}
	s, ok := labelRaw.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return rowID
	}
	return s
}

func provisionalStartTasks(cfg OrgConfig, domain Domain, job Job, ordered []string, wlJob WorklistJob, op OperatorIdentity, stream *streamBinding) []StreamTimelineTask {
	out := make([]StreamTimelineTask, 0, len(ordered))
	for i, taskID := range ordered {
		chrome, _ := worklistTaskByID(wlJob, taskID)
		title := chrome.Title
		if title == "" {
			title = taskID
		}
		task := StreamTimelineTask{ID: taskID, Title: title, State: StreamSubstepLocked}
		if i == 0 {
			tc, ok := TaskByID(job.Tasks, taskID)
			status := TaskStatusAvailable
			if ok {
				status = foldReadyWhen(domain, TaskStatusAvailable, tc, "", cfg.Catalogs, stream)
				if status == TaskStatusUnavailable {
					if reason := evaluateBlockedReason(domain, taskPayload(tc), "", cfg.Catalogs, stream); reason != "" {
						task.BlockedReason = reason
					}
				}
			}
			eligible := rolesIntersect(op.Roles, chrome.Roles)
			task.State = MapWorklistStatusToStreamState(viewerScopedStatus(status, eligible))
		}
		out = append(out, task)
	}
	return out
}

func timelineTasksFromChrome(job Job, wlJob WorklistJob, statuses map[string]TaskStatus, blocked map[string]string) []StreamTimelineTask {
	ordered := OrderedTaskIDs(job.Tasks)
	out := make([]StreamTimelineTask, 0, len(ordered))
	for _, taskID := range ordered {
		chrome, _ := worklistTaskByID(wlJob, taskID)
		title := chrome.Title
		if title == "" {
			title = taskID
		}
		task := StreamTimelineTask{ID: taskID, Title: title, State: StreamSubstepLocked}
		if statuses != nil {
			if st, ok := statuses[taskID]; ok {
				task.State = MapWorklistStatusToStreamState(st)
			}
			if task.State == StreamSubstepLocked && blocked != nil {
				task.BlockedReason = blocked[taskID]
			}
		}
		out = append(out, task)
	}
	return out
}
