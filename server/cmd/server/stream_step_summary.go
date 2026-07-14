package main

import (
	"time"
)

func buildStepSummary(step WorkflowStep, substeps []WorkflowSub, process *Process, orgNames map[string]string) StepSummaryView {
	summary := StepSummaryView{
		StepID:           step.StepID,
		Title:            step.Title,
		OrganizationName: organizationDisplayName(step.OrganizationSlug, orgNames),
		SubstepCount:     len(substeps),
	}
	if process == nil || len(substeps) == 0 {
		return summary
	}

	allDone := true
	var latestDoneAt time.Time
	for _, sub := range substeps {
		progress, ok := process.Progress[sub.SubstepID]
		if !ok || progress.State != "done" {
			allDone = false
			continue
		}
		if progress.DoneAt != nil && progress.DoneAt.After(latestDoneAt) {
			latestDoneAt = *progress.DoneAt
		}
	}
	if allDone && !latestDoneAt.IsZero() {
		summary.CompletedAt = latestDoneAt.UTC().Format(time.RFC3339)
		summary.CompletedAtHuman = humanReadableTraceabilityTime(latestDoneAt)
	}
	return summary
}
