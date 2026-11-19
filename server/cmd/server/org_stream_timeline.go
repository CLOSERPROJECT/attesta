package main

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
)

// TimelineRowGroup is one each-step row (collapsible lot) under a TimelineStep.
type TimelineRowGroup struct {
	ID       string
	Label    string
	Expanded bool
	Substeps []TimelineSubstep
}

func appOrgInstancePath(streamKey, instanceID string) string {
	return "/app/" + strings.TrimSpace(streamKey) + "/i/" + strings.TrimSpace(instanceID)
}

func appOrgTaskPath(streamKey, streamInstanceID, jobID, jobInstanceID, taskID string) string {
	return appOrgInstancePath(streamKey, streamInstanceID) + "/" +
		strings.TrimSpace(jobID) + "/" +
		strings.TrimSpace(jobInstanceID) + "/" +
		strings.TrimSpace(taskID)
}

func appOrgSeededStartTaskPath(streamKey, streamInstanceID, jobID, taskID, seedField, seedValue string) string {
	path := appOrgTaskPath(streamKey, streamInstanceID, jobID, "new", taskID)
	field := strings.TrimSpace(seedField)
	if field == "" {
		return path
	}
	q := url.Values{}
	q.Set("seed."+field, seedValue)
	return path + "?" + q.Encode()
}

// orgJobSeedField derives the start-Task seed query key from Job.instanceId (e.g. $.lotId → lotId).
func orgJobSeedField(job org.Job) string {
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

func orgJobInstanceExists(domain org.Domain, jobID, instanceID string) bool {
	jobID = strings.TrimSpace(jobID)
	instanceID = strings.TrimSpace(instanceID)
	for _, inst := range domain.JobInstances {
		if inst.JobID == jobID && inst.ID == instanceID {
			return true
		}
	}
	return false
}

func buildOrgTimelineSteps(cfg org.OrgConfig, domain org.Domain, timeline org.StreamTimeline, streamKey, streamInstanceID string) []TimelineStep {
	out := make([]TimelineStep, 0, len(timeline.Steps))
	for i, step := range timeline.Steps {
		jobVal, _ := cfg.Jobs.Get(step.JobID)
		job, _ := org.AsJob(jobVal)
		title := strings.TrimSpace(step.Title)
		if title == "" {
			title = step.JobID
		}
		tlStep := TimelineStep{
			Summary: StepSummaryView{
				StepID:       fmt.Sprintf("%d", i+1),
				Title:        title,
				HideOrgMark:  true,
				SubstepCount: len(step.Tasks),
			},
			Expanded: i == 0,
		}
		if len(step.Rows) > 0 {
			tlStep.Summary.Title = fmt.Sprintf("%s ×%d", title, len(step.Rows))
			tlStep.Summary.SubstepCount = len(step.Rows)
			tlStep.Rows = buildOrgTimelineRows(job, step, domain, streamKey, streamInstanceID)
			if i == 0 || anyOrgRowExpanded(tlStep.Rows) {
				tlStep.Expanded = true
			}
		} else {
			tlStep.Substeps = buildOrgTimelineTaskSubsteps(job, step.JobID, streamInstanceID, step.Tasks, streamKey, streamInstanceID, false, "")
		}
		out = append(out, tlStep)
	}
	return out
}

func anyOrgRowExpanded(rows []TimelineRowGroup) bool {
	for _, row := range rows {
		if row.Expanded {
			return true
		}
	}
	return false
}

func buildOrgTimelineRows(job org.Job, step org.StreamTimelineStep, domain org.Domain, streamKey, streamInstanceID string) []TimelineRowGroup {
	seedField := orgJobSeedField(job)
	rows := make([]TimelineRowGroup, 0, len(step.Rows))
	for _, row := range step.Rows {
		exists := orgJobInstanceExists(domain, step.JobID, row.ID)
		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = row.ID
		}
		group := TimelineRowGroup{
			ID:       row.ID,
			Label:    label,
			Expanded: !row.Finished,
			Substeps: buildOrgTimelineTaskSubsteps(job, step.JobID, row.ID, row.Tasks, streamKey, streamInstanceID, !exists, seedField),
		}
		rows = append(rows, group)
	}
	return rows
}

func buildOrgTimelineTaskSubsteps(job org.Job, jobID, jobInstanceID string, tasks []org.StreamTimelineTask, streamKey, streamInstanceID string, provisional bool, seedField string) []TimelineSubstep {
	out := make([]TimelineSubstep, 0, len(tasks))
	ordered := org.OrderedTaskIDs(job.Tasks)
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
		sub := TimelineSubstep{
			SubstepID:     jobID + "." + task.ID,
			Title:         title,
			Status:        status,
			StatusLabel:   processStatusLabel(status),
			BlockedReason: strings.TrimSpace(task.BlockedReason),
		}
		switch {
		case provisional && task.ID == startTaskID:
			sub.Href = appOrgSeededStartTaskPath(streamKey, streamInstanceID, jobID, task.ID, seedField, jobInstanceID)
		case !provisional:
			sub.Href = appOrgTaskPath(streamKey, streamInstanceID, jobID, jobInstanceID, task.ID)
		}
		out = append(out, sub)
	}
	return out
}
