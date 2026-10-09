// process_completion_strategy.go
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	CompletionStrategyClassic      = "classic"
	CompletionStrategyDraftAndSeal = "draft-and-seal"
)

var ErrCompletionUnsupported = errors.New("process: operation unsupported for completion strategy")

type SaveDraftCmd struct {
	Process     *Process
	WorkflowKey string
	SubstepID   string
	Substep     WorkflowSub
	Actor       Actor
	Payload     map[string]interface{}
	Config      RuntimeConfig
	Now         time.Time
}

type SealStepCmd struct {
	Process     *Process
	WorkflowKey string
	StepID      string
	Actor       Actor
	Config      RuntimeConfig
	Now         time.Time
	Confirm     bool
}

// completionStrategy is selected once at the process-completion edge.
// Classic keeps today's Complete + sequential availability bodies;
// draft-and-seal owns parallel in-Step availability + SaveDraft + Seal Step.
type completionStrategy interface {
	Name() string
	CompleteSubstep(ctx context.Context, cmd CompleteSubstepCmd) (*Process, error)
	ComputeAvailability(def WorkflowDef, process *Process) map[string]bool
	IsSequenceOK(def WorkflowDef, process *Process, substepID string) bool
	SaveDraft(ctx context.Context, cmd SaveDraftCmd) (*Process, error)
	SealStep(ctx context.Context, cmd SealStepCmd) (*Process, error)
}

func workflowCompletionStrategy(def WorkflowDef) string {
	switch strings.ToLower(strings.TrimSpace(def.CompletionStrategy)) {
	case "", CompletionStrategyClassic:
		return CompletionStrategyClassic
	case CompletionStrategyDraftAndSeal:
		return CompletionStrategyDraftAndSeal
	default:
		return strings.ToLower(strings.TrimSpace(def.CompletionStrategy))
	}
}

func normalizeCompletionStrategy(workflow *WorkflowDef) error {
	if workflow == nil {
		return nil
	}
	raw := strings.TrimSpace(workflow.CompletionStrategy)
	if raw == "" {
		workflow.CompletionStrategy = CompletionStrategyClassic
		return nil
	}
	normalized := strings.ToLower(raw)
	switch normalized {
	case CompletionStrategyClassic, CompletionStrategyDraftAndSeal:
		workflow.CompletionStrategy = normalized
		return nil
	default:
		return fmt.Errorf("unsupported completionStrategy %q (allowed: %s, %s)", raw, CompletionStrategyClassic, CompletionStrategyDraftAndSeal)
	}
}

func newCompletionStrategy(def WorkflowDef, svc *ProcessService) completionStrategy {
	switch workflowCompletionStrategy(def) {
	case CompletionStrategyDraftAndSeal:
		return draftAndSealCompletionStrategy{svc: svc}
	default:
		return classicCompletionStrategy{svc: svc}
	}
}

func (p *ProcessService) completionStrategy(def WorkflowDef) completionStrategy {
	return newCompletionStrategy(def, p)
}

// substepAvailability selects the Stream blueprint's completion strategy, then
// delegates. Classic Streams keep sequential unlock via computeAvailability.
func substepAvailability(def WorkflowDef, process *Process) map[string]bool {
	return newCompletionStrategy(def, nil).ComputeAvailability(def, process)
}

// substepSequenceOK selects the Stream blueprint's completion strategy, then
// delegates. Classic Streams keep sequential predecessor checks via isSequenceOK.
func substepSequenceOK(def WorkflowDef, process *Process, substepID string) bool {
	return newCompletionStrategy(def, nil).IsSequenceOK(def, process, substepID)
}
