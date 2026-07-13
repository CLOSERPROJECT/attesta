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

func (p *ProcessService) CompleteSubstep(ctx context.Context, cmd CompleteSubstepCmd) (*Process, error) {
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
