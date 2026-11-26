package main

import (
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
	return org.AppOrgInstancePath(streamKey, instanceID)
}

func appOrgTaskPath(streamKey, streamInstanceID, jobID, jobInstanceID, taskID string) string {
	return org.AppOrgTaskPath(streamKey, streamInstanceID, jobID, jobInstanceID, taskID)
}

func appOrgSeededStartTaskPath(streamKey, streamInstanceID, jobID, taskID, seedField, seedValue string) string {
	return org.AppOrgSeededStartTaskPath(streamKey, streamInstanceID, jobID, taskID, seedField, seedValue)
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

func buildOrgTimelineSteps(cfg org.OrgConfig, domain org.Domain, timeline org.StreamTimeline, streamKey, streamInstanceID string) []TimelineStep {
	focus := org.TimelineBoot()
	if id := strings.TrimSpace(streamInstanceID); id != "" {
		var err error
		focus, err = org.TimelineInstance(id)
		if err != nil {
			focus = org.TimelineBoot()
		}
	}
	hrefs := org.TimelineHrefPolicyFromFocus(streamKey, focus)
	return timelineStepsFromDTO(org.BuildOrgTimelineSteps(cfg, domain, timeline, hrefs, ""))
}

func orgTimelineSubmissions(domain org.Domain, jobID, jobInstanceID, taskID string) []TimelineSubmissionView {
	return timelineSubmissionsFromDTO(org.OrgTimelineSubmissions(domain, jobID, jobInstanceID, taskID))
}

func timelineStepsFromDTO(steps []org.TimelineStepDTO) []TimelineStep {
	out := make([]TimelineStep, 0, len(steps))
	for _, step := range steps {
		tl := TimelineStep{
			Summary: StepSummaryView{
				StepID:       step.StepID,
				Title:        step.Title,
				HideOrgMark:  true,
				SubstepCount: step.SubstepCount,
			},
			Expanded: step.Expanded,
			Substeps: timelineSubstepsFromDTO(step.Substeps),
			Rows:     timelineRowsFromDTO(step.Rows),
		}
		out = append(out, tl)
	}
	return out
}

func timelineRowsFromDTO(rows []org.TimelineRowDTO) []TimelineRowGroup {
	if len(rows) == 0 {
		return nil
	}
	out := make([]TimelineRowGroup, 0, len(rows))
	for _, row := range rows {
		out = append(out, TimelineRowGroup{
			ID:       row.ID,
			Label:    row.Label,
			Expanded: row.Expanded,
			Substeps: timelineSubstepsFromDTO(row.Substeps),
		})
	}
	return out
}

func timelineSubstepsFromDTO(subs []org.TimelineSubstepDTO) []TimelineSubstep {
	if len(subs) == 0 {
		return nil
	}
	out := make([]TimelineSubstep, 0, len(subs))
	for _, sub := range subs {
		out = append(out, TimelineSubstep{
			SubstepID:     sub.SubstepID,
			Title:         sub.Title,
			Status:        sub.Status,
			StatusLabel:   sub.StatusLabel,
			BlockedReason: sub.BlockedReason,
			OrgActionable: sub.OrgActionable,
			SubmitMany:    sub.SubmitMany,
			Href:          sub.Href,
			Submissions:   timelineSubmissionsFromDTO(sub.Submissions),
		})
	}
	return out
}

func timelineSubmissionsFromDTO(subs []org.TimelineSubmissionDTO) []TimelineSubmissionView {
	if len(subs) == 0 {
		return nil
	}
	out := make([]TimelineSubmissionView, 0, len(subs))
	for _, s := range subs {
		out = append(out, TimelineSubmissionView{
			Timestamp:      s.Timestamp,
			TimestampHuman: s.TimestampHuman,
			UserID:         s.UserID,
		})
	}
	return out
}
