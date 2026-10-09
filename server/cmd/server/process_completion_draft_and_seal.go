// process_completion_draft_and_seal.go
package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const progressStateDraft = "draft"

var (
	ErrDraftValidation   = errors.New("process: draft validation failed")
	ErrSealUnauthorized  = errors.New("process: seal requires Org admin standing for the Step Organization")
	ErrSealNotConfirmed  = errors.New("process: seal requires confirmation")
	ErrSealAlreadySealed = errors.New("process: step is already sealed")
)

// draftAndSealCompletionStrategy is the opt-in Feragame compromise path:
// parallel in-Step availability, draft saves (non-done, no notarization),
// CompleteSubstep rejected, and Seal Step as the terminal commit.
type draftAndSealCompletionStrategy struct {
	svc *ProcessService
}

func (draftAndSealCompletionStrategy) Name() string { return CompletionStrategyDraftAndSeal }

func (draftAndSealCompletionStrategy) CompleteSubstep(context.Context, CompleteSubstepCmd) (*Process, error) {
	return nil, fmt.Errorf("%w: CompleteSubstep is not available for draft-and-seal; use Seal Step", ErrCompletionUnsupported)
}

func (draftAndSealCompletionStrategy) ComputeAvailability(def WorkflowDef, process *Process) map[string]bool {
	available := map[string]bool{}
	closed := isProcessClosed(def, process)
	priorStepsDone := true
	for _, step := range sortedSteps(def) {
		stepAllDone := true
		for _, sub := range sortedSubsteps(step) {
			done := progressEntryDone(process, sub.SubstepID)
			if done {
				available[sub.SubstepID] = false
				continue
			}
			stepAllDone = false
			available[sub.SubstepID] = !closed && priorStepsDone
		}
		priorStepsDone = priorStepsDone && stepAllDone
	}
	return available
}

func (draftAndSealCompletionStrategy) IsSequenceOK(def WorkflowDef, process *Process, substepID string) bool {
	priorStepsDone := true
	for _, step := range sortedSteps(def) {
		for _, sub := range sortedSubsteps(step) {
			if sub.SubstepID == substepID {
				return priorStepsDone
			}
		}
		if !stepAllSubstepsDone(step, process) {
			priorStepsDone = false
		}
	}
	return false
}

func (s draftAndSealCompletionStrategy) SaveDraft(ctx context.Context, cmd SaveDraftCmd) (*Process, error) {
	if s.svc == nil {
		return nil, fmt.Errorf("missing process service")
	}
	if cmd.Process == nil {
		return nil, fmt.Errorf("missing process")
	}
	if isProcessClosed(cmd.Config.Workflow, cmd.Process) {
		return nil, fmt.Errorf("stream instance is closed")
	}
	if progressEntryDone(cmd.Process, cmd.SubstepID) {
		return nil, fmt.Errorf("substep is sealed and cannot be drafted")
	}
	if !s.IsSequenceOK(cmd.Config.Workflow, cmd.Process, cmd.SubstepID) {
		return nil, fmt.Errorf("substep is locked by a prior Step")
	}
	if err := validateDraftFormataPayload(cmd.Substep, cmd.Payload); err != nil {
		return nil, err
	}

	description := cmd.Substep.InputKey
	progressUpdate := ProcessStep{
		State:       progressStateDraft,
		Description: &description,
		Data:        cmd.Payload,
	}
	if err := s.svc.store.UpdateProcessProgress(ctx, cmd.Process.ID, cmd.WorkflowKey, cmd.SubstepID, progressUpdate); err != nil {
		return cmd.Process, fmt.Errorf("%w: %v", ErrProgressUpdate, err)
	}
	return s.svc.reloadProcess(ctx, cmd.Process.ID)
}

func (s draftAndSealCompletionStrategy) SealStep(ctx context.Context, cmd SealStepCmd) (*Process, error) {
	if s.svc == nil {
		return nil, fmt.Errorf("missing process service")
	}
	if cmd.Process == nil {
		return nil, fmt.Errorf("missing process")
	}
	if !cmd.Confirm {
		return nil, ErrSealNotConfirmed
	}

	step, err := findWorkflowStep(cmd.Config.Workflow, cmd.StepID)
	if err != nil {
		return nil, err
	}
	if !actorCanSealStep(cmd.Actor, step) {
		return nil, ErrSealUnauthorized
	}
	if stepAllSubstepsDone(step, cmd.Process) {
		return nil, ErrSealAlreadySealed
	}
	if isProcessClosed(cmd.Config.Workflow, cmd.Process) {
		return nil, fmt.Errorf("stream instance is closed")
	}

	now := cmd.Now
	if now.IsZero() {
		now = s.svc.serviceNow(time.Time{})
	}
	sealer := sealActor(cmd.Actor)

	for _, sub := range sortedSubsteps(step) {
		if progressEntryDone(cmd.Process, sub.SubstepID) {
			continue
		}
		payload := sealedSubstepPayload(cmd.Process, sub.SubstepID)
		description := sub.InputKey
		progressUpdate := ProcessStep{
			State:       "done",
			Description: &description,
			DoneAt:      &now,
			DoneBy:      &sealer,
			Data:        payload,
		}
		if err := s.svc.store.UpdateProcessProgress(ctx, cmd.Process.ID, cmd.WorkflowKey, sub.SubstepID, progressUpdate); err != nil {
			return cmd.Process, fmt.Errorf("%w: %v", ErrProgressUpdate, err)
		}
		notary := Notarization{
			ProcessID: cmd.Process.ID,
			SubstepID: sub.SubstepID,
			Payload:   payload,
			Actor:     sealer,
			CreatedAt: now,
			FakeNotary: FakeNotary{
				Method: "sha256",
				Digest: digestPayload(payload),
			},
		}
		if err := s.svc.store.InsertNotarization(ctx, notary); err != nil {
			return cmd.Process, fmt.Errorf("%w: %v", ErrNotarization, err)
		}
	}

	reloaded, err := s.svc.reloadProcess(ctx, cmd.Process.ID)
	if err != nil {
		return cmd.Process, err
	}
	if isProcessDone(cmd.Config.Workflow, reloaded) {
		return s.svc.finalizeProcessIfDone(ctx, cmd.Config, cmd.WorkflowKey, reloaded, now), nil
	}
	return reloaded, nil
}

func findWorkflowStep(def WorkflowDef, stepID string) (WorkflowStep, error) {
	want := strings.TrimSpace(stepID)
	if want == "" {
		return WorkflowStep{}, fmt.Errorf("missing step id")
	}
	for _, step := range def.Steps {
		if strings.TrimSpace(step.StepID) == want {
			return step, nil
		}
	}
	return WorkflowStep{}, fmt.Errorf("step %q not found", want)
}

// actorCanSealStep requires Org admin standing in the Step's Organization.
// Ordinary Substep roles alone cannot seal.
func actorCanSealStep(actor Actor, step WorkflowStep) bool {
	if !actorHasOrgAdminStanding(actor) {
		return false
	}
	stepOrg := strings.TrimSpace(step.OrganizationSlug)
	if stepOrg == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(actor.OrgSlug), stepOrg)
}

func actorHasOrgAdminStanding(actor Actor) bool {
	for _, slug := range actor.RoleSlugs {
		if isOrgAdminRoleSlug(slug) {
			return true
		}
	}
	return isOrgAdminRoleSlug(actor.Role)
}

func sealActor(actor Actor) Actor {
	out := actor
	if !isOrgAdminRoleSlug(out.Role) {
		out.Role = "org-admin"
	}
	if !containsRole(out.RoleSlugs, "org-admin") && !containsRole(out.RoleSlugs, "org_admin") {
		out.RoleSlugs = append(append([]string(nil), out.RoleSlugs...), "org-admin")
	}
	return out
}

func sealedSubstepPayload(process *Process, substepID string) map[string]interface{} {
	if process == nil {
		return map[string]interface{}{}
	}
	entry, ok := process.Progress[substepID]
	if !ok || entry.Data == nil {
		return map[string]interface{}{}
	}
	return entry.Data
}

func progressEntryDone(process *Process, substepID string) bool {
	if process == nil {
		return false
	}
	entry, ok := process.Progress[substepID]
	return ok && entry.State == "done"
}

func stepAllSubstepsDone(step WorkflowStep, process *Process) bool {
	subs := sortedSubsteps(step)
	if len(subs) == 0 {
		return true
	}
	for _, sub := range subs {
		if !progressEntryDone(process, sub.SubstepID) {
			return false
		}
	}
	return true
}

// validateDraftFormataPayload enforces row/field rules from the Substep Formata schema.
// Empty arrays are allowed; missing required fields on a row are rejected.
func validateDraftFormataPayload(sub WorkflowSub, payload map[string]interface{}) error {
	if payload == nil {
		return fmt.Errorf("%w: payload is required", ErrDraftValidation)
	}
	for key := range payload {
		if strings.Contains(key, "[]") || strings.Contains(key, "[") {
			return fmt.Errorf("%w: payload must be nested JSON (got form-encoded key %q)", ErrDraftValidation, key)
		}
	}
	schema := sub.Schema
	if len(schema) == 0 {
		return nil
	}
	if err := validateDraftRequiredFields(payload, schema["required"], ""); err != nil {
		return err
	}
	props, _ := schema["properties"].(map[string]interface{})
	if len(props) == 0 {
		return nil
	}
	for name, rawProp := range props {
		prop, ok := rawProp.(map[string]interface{})
		if !ok {
			continue
		}
		if schemaType(prop) != "array" {
			continue
		}
		rawVal, present := payload[name]
		if !present {
			continue
		}
		rows, ok := rawVal.([]interface{})
		if !ok {
			return fmt.Errorf("%w: %s must be an array", ErrDraftValidation, name)
		}
		itemSchema, _ := prop["items"].(map[string]interface{})
		for i, row := range rows {
			rowMap, ok := row.(map[string]interface{})
			if !ok {
				return fmt.Errorf("%w: %s[%d] must be an object", ErrDraftValidation, name, i)
			}
			if len(itemSchema) == 0 {
				continue
			}
			path := fmt.Sprintf("%s[%d]", name, i)
			if err := validateDraftRequiredFields(rowMap, itemSchema["required"], path); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDraftRequiredFields(obj map[string]interface{}, rawRequired interface{}, path string) error {
	required := stringList(rawRequired)
	for _, field := range required {
		val, ok := obj[field]
		if !ok || val == nil {
			if path == "" {
				return fmt.Errorf("%w: missing required field %q", ErrDraftValidation, field)
			}
			return fmt.Errorf("%w: %s missing required field %q", ErrDraftValidation, path, field)
		}
		if s, isString := val.(string); isString && strings.TrimSpace(s) == "" {
			if path == "" {
				return fmt.Errorf("%w: missing required field %q", ErrDraftValidation, field)
			}
			return fmt.Errorf("%w: %s missing required field %q", ErrDraftValidation, path, field)
		}
	}
	return nil
}

func schemaType(schema map[string]interface{}) string {
	if schema == nil {
		return ""
	}
	t, _ := schema["type"].(string)
	return strings.ToLower(strings.TrimSpace(t))
}

func stringList(raw interface{}) []string {
	switch typed := raw.(type) {
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if s, ok := item.(string); ok {
				if trimmed := strings.TrimSpace(s); trimmed != "" {
					out = append(out, trimmed)
				}
			}
		}
		return out
	default:
		return nil
	}
}
