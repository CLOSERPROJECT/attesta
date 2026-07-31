package main

import (
	"context"
	"strings"
	"time"
)

const publicStreamRecentRunLimit = 8

func processCompletedAt(process *Process) time.Time {
	if process == nil {
		return time.Time{}
	}
	var latest time.Time
	for _, step := range process.Progress {
		if step.DoneAt != nil && step.DoneAt.After(latest) {
			latest = *step.DoneAt
		}
	}
	if latest.IsZero() && process.DPP != nil && !process.DPP.GeneratedAt.IsZero() {
		return process.DPP.GeneratedAt.UTC()
	}
	return latest
}

func buildPublicStreamRunCards(def WorkflowDef, processes []Process) []PublicStreamRunCardView {
	out := make([]PublicStreamRunCardView, 0, publicStreamRecentRunLimit)
	for i := range processes {
		p := processes[i]
		p.Progress = normalizeProgressKeys(p.Progress)
		if deriveProcessStatus(def, &p) != processStatusDone {
			continue
		}
		card := PublicStreamRunCardView{
			StatusLabel: "Completed",
			CompletedAt: humanReadableTraceabilityTime(processCompletedAt(&p)),
		}
		if p.DPP != nil {
			gtin := strings.TrimSpace(p.DPP.GTIN)
			lot := strings.TrimSpace(p.DPP.Lot)
			serial := strings.TrimSpace(p.DPP.Serial)
			if gtin != "" && lot != "" && serial != "" {
				card.DigitalLink = digitalLinkURL(gtin, lot, serial)
				card.PassportChip = true
			}
		}
		out = append(out, card)
		if len(out) >= publicStreamRecentRunLimit {
			break
		}
	}
	return out
}

func (s *Server) buildPublicStreamBlueprint(ctx context.Context, cfg RuntimeConfig, workflowKey string) StreamInstanceDetailView {
	preview := makeStreamInstanceDetailReadOnly(
		s.buildStreamInstanceDetailView(
			ctx,
			cfg,
			workflowKey,
			buildWorkflowPreviewProcess(cfg.Workflow, workflowKey),
			Actor{},
			"",
			"",
			false,
		),
		"Public preview.",
	)
	preview.HideStatus = true
	preview.WorkflowPath = publicStreamPath(workflowKey)
	return preview
}
