// process_completion_classic.go
package main

import (
	"context"
	"fmt"
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
