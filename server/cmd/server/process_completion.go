// process_completion.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

var (
	ErrProgressUpdate = errors.New("process: progress update failed")
	ErrNotarization   = errors.New("process: notarization failed")
)

type ProcessService struct {
	store Store
	now   func() time.Time
}

type CompleteSubstepCmd struct {
	Process     *Process
	WorkflowKey string
	SubstepID   string
	Substep     WorkflowSub
	Actor       Actor
	Payload     map[string]interface{}
	Config      RuntimeConfig
	Now         time.Time
}

func (p *ProcessService) serviceNow(fallback time.Time) time.Time {
	if p != nil && p.now != nil {
		return p.now().UTC()
	}
	if !fallback.IsZero() {
		return fallback.UTC()
	}
	return time.Now().UTC()
}

// CompleteSubstep selects the Stream blueprint completion strategy, then delegates.
// Classic Streams keep today's Complete→done+notarize body via classicCompleteSubstep.
func (p *ProcessService) CompleteSubstep(ctx context.Context, cmd CompleteSubstepCmd) (*Process, error) {
	return p.completionStrategy(cmd.Config.Workflow).CompleteSubstep(ctx, cmd)
}

// SaveDraft selects the Stream blueprint completion strategy, then delegates.
func (p *ProcessService) SaveDraft(ctx context.Context, cmd SaveDraftCmd) (*Process, error) {
	return p.completionStrategy(cmd.Config.Workflow).SaveDraft(ctx, cmd)
}

// SealStep selects the Stream blueprint completion strategy, then delegates.
func (p *ProcessService) SealStep(ctx context.Context, cmd SealStepCmd) (*Process, error) {
	return p.completionStrategy(cmd.Config.Workflow).SealStep(ctx, cmd)
}

// persistDoneSubstep marks a Substep done and inserts its notarization record.
// Callers prepare the actor (classic DoneBy vs sealActor) and keep finalize/reload.
func (p *ProcessService) persistDoneSubstep(
	ctx context.Context,
	processID primitive.ObjectID,
	workflowKey string,
	substepID string,
	description string,
	actor Actor,
	payload map[string]interface{},
	now time.Time,
) error {
	progressUpdate := ProcessStep{
		State:       "done",
		Description: &description,
		DoneAt:      &now,
		DoneBy:      &actor,
		Data:        payload,
	}
	if err := p.store.UpdateProcessProgress(ctx, processID, workflowKey, substepID, progressUpdate); err != nil {
		return fmt.Errorf("%w: %v", ErrProgressUpdate, err)
	}
	notary := Notarization{
		ProcessID: processID,
		SubstepID: substepID,
		Payload:   payload,
		Actor:     actor,
		CreatedAt: now,
		FakeNotary: FakeNotary{
			Method: "sha256",
			Digest: digestPayload(payload),
		},
	}
	if err := p.store.InsertNotarization(ctx, notary); err != nil {
		return fmt.Errorf("%w: %v", ErrNotarization, err)
	}
	return nil
}

func (p *ProcessService) EnsureCompletionArtifacts(ctx context.Context, cfg RuntimeConfig, workflowKey string, process *Process) *Process {
	if process == nil || !isProcessClosed(cfg.Workflow, process) {
		return process
	}
	return p.finalizeProcessIfDone(ctx, cfg, workflowKey, process, p.serviceNow(time.Time{}))
}

func (p *ProcessService) finalizeProcessIfDone(ctx context.Context, cfg RuntimeConfig, workflowKey string, process *Process, generatedAt time.Time) *Process {
	if process == nil {
		return process
	}

	updated := false
	if process.Termination == nil && strings.TrimSpace(process.Status) != "done" && isProcessDone(cfg.Workflow, process) {
		if err := p.store.UpdateProcessStatus(ctx, process.ID, workflowKey, "done"); err != nil {
			log.Printf("failed to persist process status for %s: %v", process.ID.Hex(), err)
		} else {
			updated = true
		}
	}

	if cfg.DPP.Enabled && process.DPP == nil {
		dpp, err := buildProcessDPP(cfg.Workflow, cfg.DPP, process, generatedAt)
		if err != nil {
			log.Printf("failed to build dpp for process %s: %v", process.ID.Hex(), err)
		} else if err := p.store.UpdateProcessDPP(ctx, process.ID, workflowKey, dpp); err != nil {
			log.Printf("failed to persist dpp for process %s: %v", process.ID.Hex(), err)
		} else {
			updated = true
		}
	}

	if !updated {
		return process
	}
	reloaded, err := p.reloadProcess(ctx, process.ID)
	if err != nil {
		log.Printf("failed to reload process %s after completion artifact update: %v", process.ID.Hex(), err)
		return process
	}
	return reloaded
}

func (p *ProcessService) reloadProcess(ctx context.Context, processID primitive.ObjectID) (*Process, error) {
	reloaded, err := p.store.LoadProcessByID(ctx, processID)
	if err != nil {
		return nil, err
	}
	reloaded.Progress = normalizeProgressKeys(reloaded.Progress)
	return reloaded, nil
}
