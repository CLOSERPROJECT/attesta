// process_completion.go
package main

import (
	"context"
	"errors"
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
