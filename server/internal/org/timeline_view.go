package org

import (
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	TimelineModeBoot     = "boot"
	TimelineModeInstance = "instance"
)

// TimelineSubmissionDTO is one Domain Submission summary for the SPA / JSON timeline.
type TimelineSubmissionDTO struct {
	Timestamp      string `json:"timestamp,omitempty"`
	TimestampHuman string `json:"timestampHuman,omitempty"`
	UserID         string `json:"userId,omitempty"`
}

// TimelineSubstepDTO is one Org timeline card / accordion row.
type TimelineSubstepDTO struct {
	SubstepID     string                  `json:"substepId"`
	Title         string                  `json:"title"`
	Status        string                  `json:"status"`
	StatusLabel   string                  `json:"statusLabel,omitempty"`
	OrgActionable bool                    `json:"orgActionable,omitempty"`
	SubmitMany    bool                    `json:"submitMany,omitempty"`
	BlockedReason string                  `json:"blockedReason,omitempty"`
	Href          string                  `json:"href,omitempty"`
	Submissions   []TimelineSubmissionDTO `json:"submissions,omitempty"`
}

// TimelineRowDTO is one each-step row under a timeline step.
type TimelineRowDTO struct {
	ID          string               `json:"id"`
	Label       string               `json:"label"`
	Status      string               `json:"status,omitempty"`
	StatusLabel string               `json:"statusLabel,omitempty"`
	Expanded    bool                 `json:"expanded"`
	Substeps    []TimelineSubstepDTO `json:"substeps"`
}

// TimelineStepDTO is one blueprint step in the enriched Org timeline.
type TimelineStepDTO struct {
	StepID       string               `json:"stepId"`
	Title        string               `json:"title"`
	Status       string               `json:"status,omitempty"`
	StatusLabel  string               `json:"statusLabel,omitempty"`
	SubstepCount int                  `json:"substepCount"`
	Expanded     bool                 `json:"expanded"`
	Substeps     []TimelineSubstepDTO `json:"substeps,omitempty"`
	Rows         []TimelineRowDTO     `json:"rows,omitempty"`
}

// TimelineView is GET /api/org/timeline for Boot (stream-only) or one stream instance.
type TimelineView struct {
	Mode     string            `json:"mode"`
	Stream   string            `json:"stream"`
	Instance string            `json:"instance"`
	Label    string            `json:"label"`
	Status   string            `json:"status"`
	Steps    []TimelineStepDTO `json:"steps"`
	Revision int64             `json:"revision"`
}

// TimelineHrefPolicy builds SPA Task hrefs for timeline enrichment.
// Derived once from TimelineFocus at BuildOrgTimelineView — no boot bool in steps.
type TimelineHrefPolicy struct {
	streamKey        string
	streamInstanceID string // empty ⇒ Play / boot paths
}

// TimelineHrefPolicyFromFocus maps Focus to instance vs Play href builders.
func TimelineHrefPolicyFromFocus(streamKey string, focus TimelineFocus) TimelineHrefPolicy {
	streamKey = strings.TrimSpace(streamKey)
	if id, ok := focus.InstanceID(); ok {
		return TimelineHrefPolicy{streamKey: streamKey, streamInstanceID: id}
	}
	return TimelineHrefPolicy{streamKey: streamKey}
}

// TaskHref is a Task form path under instance home or Play board.
func (p TimelineHrefPolicy) TaskHref(jobID, jobInstanceID, taskID string) string {
	if p.streamInstanceID == "" {
		return AppOrgPlayTaskPath(p.streamKey, jobID, jobInstanceID, taskID)
	}
	return AppOrgTaskPath(p.streamKey, p.streamInstanceID, jobID, jobInstanceID, taskID)
}

// SeededStartHref seeds a provisional each-Job start Task.
func (p TimelineHrefPolicy) SeededStartHref(jobID, taskID, seedField, seedValue string) string {
	if p.streamInstanceID == "" {
		path := AppOrgPlayTaskPath(p.streamKey, jobID, "new", taskID)
		field := strings.TrimSpace(seedField)
		if field == "" {
			return path
		}
		q := url.Values{}
		q.Set("seed."+field, seedValue)
		return path + "?" + q.Encode()
	}
	return AppOrgSeededStartTaskPath(p.streamKey, p.streamInstanceID, jobID, taskID, seedField, seedValue)
}

// BuildOrgTimelineView projects + enriches the operator timeline for one Focus.
func BuildOrgTimelineView(cfg OrgConfig, domain Domain, op OperatorIdentity, streamKey string, focus TimelineFocus) TimelineView {
	streamKey = strings.TrimSpace(streamKey)
	hrefs := TimelineHrefPolicyFromFocus(streamKey, focus)
	timeline := ProjectStreamTimeline(cfg, domain, op, focus)

	if focus.IsBoot() {
		label := streamKey
		if cfg.Stream != nil {
			if title := strings.TrimSpace(cfg.Stream.Title); title != "" {
				label = title
			}
		}
		return TimelineView{
			Mode:     TimelineModeBoot,
			Stream:   streamKey,
			Instance: "",
			Label:    label,
			Status:   "",
			Steps:    BuildOrgTimelineSteps(cfg, domain, timeline, hrefs),
		}
	}

	instanceID, _ := focus.InstanceID()
	label := instanceID
	status := StreamInstanceActive
	for _, summary := range ListStreamInstances(cfg, domain, op) {
		if summary.ID == instanceID {
			if strings.TrimSpace(summary.Label) != "" {
				label = summary.Label
			}
			if strings.TrimSpace(summary.Status) != "" {
				status = summary.Status
			}
			break
		}
	}
	return TimelineView{
		Mode:     TimelineModeInstance,
		Stream:   streamKey,
		Instance: instanceID,
		Label:    label,
		Status:   status,
		Steps:    BuildOrgTimelineSteps(cfg, domain, timeline, hrefs),
	}
}

// BuildOrgTimelineSteps enriches a ProjectStreamTimeline with Href / submissions / flags.
func BuildOrgTimelineSteps(cfg OrgConfig, domain Domain, timeline StreamTimeline, hrefs TimelineHrefPolicy) []TimelineStepDTO {
	out := make([]TimelineStepDTO, 0, len(timeline.Steps))
	for i, step := range timeline.Steps {
		jobVal, _ := cfg.Jobs.Get(step.JobID)
		job, _ := AsJob(jobVal)
		title := strings.TrimSpace(step.Title)
		if title == "" {
			title = step.JobID
		}
		tlStep := TimelineStepDTO{
			StepID:       fmt.Sprintf("%d", i+1),
			Title:        title,
			SubstepCount: len(step.Tasks),
			// Board load: expand every step/row; substep bodies stay client-collapsed.
			Expanded: true,
		}
		if len(step.Rows) > 0 {
			tlStep.Title = fmt.Sprintf("%s ×%d", title, len(step.Rows))
			tlStep.SubstepCount = len(step.Rows)
			tlStep.Rows = buildOrgTimelineRows(job, step, domain, hrefs)
			if status, ok := aggregateStreamSubstepStates(timelineChildStatusesFromRows(tlStep.Rows)); ok {
				tlStep.Status = string(status)
				tlStep.StatusLabel = timelineStatusLabel(tlStep.Status)
			}
		} else {
			jobInstanceID := hrefs.streamInstanceID
			tlStep.Substeps = buildOrgTimelineTaskSubsteps(job, step.JobID, jobInstanceID, step.Tasks, hrefs, false, "", domain)
			if status, ok := aggregateStreamSubstepStates(timelineChildStatusesFromSubsteps(tlStep.Substeps)); ok {
				tlStep.Status = string(status)
				tlStep.StatusLabel = timelineStatusLabel(tlStep.Status)
			}
		}
		out = append(out, tlStep)
	}
	return out
}

func buildOrgTimelineRows(job Job, step StreamTimelineStep, domain Domain, hrefs TimelineHrefPolicy) []TimelineRowDTO {
	seedField := orgJobSeedField(job)
	rows := make([]TimelineRowDTO, 0, len(step.Rows))
	for _, row := range step.Rows {
		exists := orgJobInstanceExists(domain, step.JobID, row.ID)
		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = row.ID
		}
		group := TimelineRowDTO{
			ID:       row.ID,
			Label:    label,
			Expanded: true,
			Substeps: buildOrgTimelineTaskSubsteps(job, step.JobID, row.ID, row.Tasks, hrefs, !exists, seedField, domain),
		}
		if status, ok := aggregateStreamSubstepStates(timelineChildStatusesFromSubsteps(group.Substeps)); ok {
			group.Status = string(status)
			group.StatusLabel = timelineStatusLabel(group.Status)
		}
		rows = append(rows, group)
	}
	return rows
}

// aggregateStreamSubstepStates rolls up child states attention-first: available → pending → locked → done.
// ok is false when states is empty (no status / omit chip).
func aggregateStreamSubstepStates(states []StreamSubstepState) (StreamSubstepState, bool) {
	if len(states) == 0 {
		return "", false
	}
	hasPending := false
	hasLocked := false
	allDone := true
	for _, s := range states {
		switch s {
		case StreamSubstepAvailable:
			return StreamSubstepAvailable, true
		case StreamSubstepPending:
			hasPending = true
			allDone = false
		case StreamSubstepLocked:
			hasLocked = true
			allDone = false
		case StreamSubstepDone:
		default:
			allDone = false
		}
	}
	switch {
	case hasPending:
		return StreamSubstepPending, true
	case hasLocked:
		return StreamSubstepLocked, true
	case allDone:
		return StreamSubstepDone, true
	default:
		return "", false
	}
}

func timelineChildStatusesFromSubsteps(substeps []TimelineSubstepDTO) []StreamSubstepState {
	out := make([]StreamSubstepState, 0, len(substeps))
	for _, sub := range substeps {
		if sub.Status == "" {
			continue
		}
		out = append(out, StreamSubstepState(sub.Status))
	}
	return out
}

func timelineChildStatusesFromRows(rows []TimelineRowDTO) []StreamSubstepState {
	out := make([]StreamSubstepState, 0, len(rows))
	for _, row := range rows {
		if row.Status == "" {
			continue
		}
		out = append(out, StreamSubstepState(row.Status))
	}
	return out
}

func buildOrgTimelineTaskSubsteps(job Job, jobID, jobInstanceID string, tasks []StreamTimelineTask, hrefs TimelineHrefPolicy, provisional bool, seedField string, domain Domain) []TimelineSubstepDTO {
	out := make([]TimelineSubstepDTO, 0, len(tasks))
	ordered := OrderedTaskIDs(job.Tasks)
	startTaskID := ""
	if len(ordered) > 0 {
		startTaskID = ordered[0]
	}
	for _, task := range tasks {
		status := string(task.State)
		title := strings.TrimSpace(task.Title)
		if title == "" {
			title = task.ID
		}
		sub := TimelineSubstepDTO{
			SubstepID:     jobID + "." + task.ID,
			Title:         title,
			Status:        status,
			StatusLabel:   timelineStatusLabel(status),
			BlockedReason: strings.TrimSpace(task.BlockedReason),
			OrgActionable: task.State == StreamSubstepAvailable,
		}
		if tc, ok := TaskByID(job.Tasks, task.ID); ok {
			sub.SubmitMany = TaskSubmitMany(tc)
		}
		switch {
		case provisional && task.ID == startTaskID:
			sub.Href = hrefs.SeededStartHref(jobID, task.ID, seedField, jobInstanceID)
		case !provisional:
			sub.Href = hrefs.TaskHref(jobID, jobInstanceID, task.ID)
			sub.Submissions = OrgTimelineSubmissions(domain, jobID, jobInstanceID, task.ID)
		}
		out = append(out, sub)
	}
	return out
}

// OrgTimelineSubmissions lists Domain submissions for a Task address (oldest first).
func OrgTimelineSubmissions(domain Domain, jobID, jobInstanceID, taskID string) []TimelineSubmissionDTO {
	jobID = strings.TrimSpace(jobID)
	jobInstanceID = strings.TrimSpace(jobInstanceID)
	taskID = strings.TrimSpace(taskID)
	matched := make([]Submission, 0)
	for _, s := range domain.Submissions {
		if s.JobID == jobID && s.TaskID == taskID && s.JobInstanceID == jobInstanceID {
			matched = append(matched, s)
		}
	}
	if len(matched) == 0 {
		return nil
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Timestamp < matched[j].Timestamp
	})
	out := make([]TimelineSubmissionDTO, 0, len(matched))
	for _, s := range matched {
		view := TimelineSubmissionDTO{
			Timestamp: strings.TrimSpace(s.Timestamp),
			UserID:    strings.TrimSpace(s.UserID),
		}
		if view.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339Nano, view.Timestamp); err == nil {
				view.TimestampHuman = humanReadableTimelineTime(parsed)
				view.Timestamp = rfc3339UTC(parsed)
			} else if parsed, err := time.Parse(time.RFC3339, view.Timestamp); err == nil {
				view.TimestampHuman = humanReadableTimelineTime(parsed)
				view.Timestamp = rfc3339UTC(parsed)
			} else {
				view.TimestampHuman = view.Timestamp
			}
		}
		out = append(out, view)
	}
	return out
}

func orgJobSeedField(job Job) string {
	expr := strings.TrimSpace(job.InstanceID)
	if !strings.HasPrefix(expr, "$.") {
		return ""
	}
	field := strings.TrimPrefix(expr, "$.")
	if field == "" || strings.ContainsAny(field, ".([") {
		return ""
	}
	return field
}

func orgJobInstanceExists(domain Domain, jobID, instanceID string) bool {
	jobID = strings.TrimSpace(jobID)
	instanceID = strings.TrimSpace(instanceID)
	for _, inst := range domain.JobInstances {
		if inst.JobID == jobID && inst.ID == instanceID {
			return true
		}
	}
	return false
}

// AppOrgInstancePath is the SPA instance home (/app/{stream}/i/{instance}).
func AppOrgInstancePath(streamKey, instanceID string) string {
	return "/app/" + strings.TrimSpace(streamKey) + "/i/" + strings.TrimSpace(instanceID)
}

// AppOrgPlayPath is the SPA Play board (/app/{stream}/new).
func AppOrgPlayPath(streamKey string) string {
	return "/app/" + strings.TrimSpace(streamKey) + "/new"
}

// AppOrgTaskPath is the SPA Task form path under instance home.
func AppOrgTaskPath(streamKey, streamInstanceID, jobID, jobInstanceID, taskID string) string {
	return AppOrgInstancePath(streamKey, streamInstanceID) + "/" +
		strings.TrimSpace(jobID) + "/" +
		strings.TrimSpace(jobInstanceID) + "/" +
		strings.TrimSpace(taskID)
}

// AppOrgPlayTaskPath is a Play-board Task href (no stream instance yet).
// Shape: /app/{stream}/new/{job}/{jobInstance}/{task}; blank jobInstance → "new".
func AppOrgPlayTaskPath(streamKey, jobID, jobInstanceID, taskID string) string {
	ji := strings.TrimSpace(jobInstanceID)
	if ji == "" {
		ji = "new"
	}
	return AppOrgPlayPath(streamKey) + "/" +
		strings.TrimSpace(jobID) + "/" +
		ji + "/" +
		strings.TrimSpace(taskID)
}

// AppOrgSeededStartTaskPath seeds a provisional each-Job start Task.
func AppOrgSeededStartTaskPath(streamKey, streamInstanceID, jobID, taskID, seedField, seedValue string) string {
	path := AppOrgTaskPath(streamKey, streamInstanceID, jobID, "new", taskID)
	field := strings.TrimSpace(seedField)
	if field == "" {
		return path
	}
	q := url.Values{}
	q.Set("seed."+field, seedValue)
	return path + "?" + q.Encode()
}

func timelineStatusLabel(status string) string {
	switch strings.TrimSpace(status) {
	case "all":
		return "All"
	case "terminated":
		return "TERMINATED"
	default:
		return strings.TrimSpace(status)
	}
}

func humanReadableTimelineTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format("2 Jan 2006 at 15:04 MST")
}

func rfc3339UTC(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
