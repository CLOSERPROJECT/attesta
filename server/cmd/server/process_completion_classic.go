// process_completion_classic.go
package main

import (
	"context"
	"fmt"
	"time"
)

// classicCompletionStrategy is today's sequential Complete + availability path.
// Divergent draft-and-seal behavior must not be added here.
type classicCompletionStrategy struct {
	svc *ProcessService
}

func (classicCompletionStrategy) Name() string { return CompletionStrategyClassic }

func (s classicCompletionStrategy) CompleteSubstep(ctx context.Context, cmd CompleteSubstepCmd) (*Process, error) {
	if s.svc == nil {
		return nil, fmt.Errorf("missing process service")
	}
	return s.svc.classicCompleteSubstep(ctx, cmd)
}

func (classicCompletionStrategy) ComputeAvailability(def WorkflowDef, process *Process) map[string]bool {
	return computeAvailability(def, process)
}

func (classicCompletionStrategy) IsSequenceOK(def WorkflowDef, process *Process, substepID string) bool {
	return isSequenceOK(def, process, substepID)
}

func (classicCompletionStrategy) SaveDraft(context.Context, SaveDraftCmd) (*Process, error) {
	return nil, fmt.Errorf("%w: SaveDraft is not available for classic completion strategy", ErrCompletionUnsupported)
}

func (classicCompletionStrategy) SealStep(context.Context, SealStepCmd) (*Process, error) {
	return nil, fmt.Errorf("%w: SealStep is not available for classic completion strategy", ErrCompletionUnsupported)
}

// classicCompleteSubstep is the classic strategy body: mark done, notarize, maybe finalize.
func (p *ProcessService) classicCompleteSubstep(ctx context.Context, cmd CompleteSubstepCmd) (*Process, error) {
	if cmd.Process == nil {
		return nil, fmt.Errorf("missing process")
	}
	now := cmd.Now
	if now.IsZero() {
		now = p.serviceNow(time.Time{})
	}

	description := cmd.Substep.InputKey
	progressUpdate := ProcessStep{
		State:       "done",
		Description: &description,
		DoneAt:      &now,
		DoneBy:      &cmd.Actor,
		Data:        cmd.Payload,
	}
	if err := p.store.UpdateProcessProgress(ctx, cmd.Process.ID, cmd.WorkflowKey, cmd.SubstepID, progressUpdate); err != nil {
		return cmd.Process, fmt.Errorf("%w: %v", ErrProgressUpdate, err)
	}

	notary := Notarization{
		ProcessID: cmd.Process.ID,
		SubstepID: cmd.SubstepID,
		Payload:   cmd.Payload,
		Actor:     cmd.Actor,
		CreatedAt: now,
		FakeNotary: FakeNotary{
			Method: "sha256",
			Digest: digestPayload(cmd.Payload),
		},
	}
	if err := p.store.InsertNotarization(ctx, notary); err != nil {
		return cmd.Process, fmt.Errorf("%w: %v", ErrNotarization, err)
	}

	reloaded, err := p.reloadProcess(ctx, cmd.Process.ID)
	if err != nil {
		return cmd.Process, err
	}

	if isProcessDone(cmd.Config.Workflow, reloaded) {
		return p.finalizeProcessIfDone(ctx, cmd.Config, cmd.WorkflowKey, reloaded, now), nil
	}
	return reloaded, nil
}

func computeAvailability(def WorkflowDef, process *Process) map[string]bool {
	available := map[string]bool{}
	if isProcessClosed(def, process) {
		for _, sub := range orderedSubsteps(def) {
			available[sub.SubstepID] = false
		}
		return available
	}
	ordered := orderedSubsteps(def)
	allPrevDone := true
	for _, sub := range ordered {
		done := false
		if process != nil {
			if entry, ok := process.Progress[sub.SubstepID]; ok && entry.State == "done" {
				done = true
			}
		}
		if done {
			available[sub.SubstepID] = false
			continue
		}
		if allPrevDone {
			available[sub.SubstepID] = true
			allPrevDone = false
		} else {
			available[sub.SubstepID] = false
		}
	}
	return available
}

func isSequenceOK(def WorkflowDef, process *Process, substepID string) bool {
	ordered := orderedSubsteps(def)
	for _, sub := range ordered {
		if sub.SubstepID == substepID {
			return true
		}
		if process == nil {
			return false
		}
		if entry, ok := process.Progress[sub.SubstepID]; !ok || entry.State != "done" {
			return false
		}
	}
	return false
}
