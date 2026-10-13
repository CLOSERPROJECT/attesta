// process_completion_http.go hosts completion/seal HTTP adapters and the
// Formata payload parse helpers they use. Route registration stays in main.go.
package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) handleCompleteSubstep(w http.ResponseWriter, r *http.Request, processID, substepID string) {
	user, _, ok := s.requireVerifiedPost(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, selected := s.selectedWorkflowOrRedirectHome(w, r)
	if !selected {
		return
	}
	actor := Actor{
		ID:          accountActorID(user),
		OrgSlug:     user.OrgSlug,
		RoleSlugs:   append([]string(nil), user.RoleSlugs...),
		WorkflowKey: workflowKey,
	}
	if len(user.RoleSlugs) > 0 {
		actor.Role = user.RoleSlugs[0]
	}
	if actor.WorkflowKey != "" && actor.WorkflowKey != workflowKey {
		s.renderActionErrorForRequest(w, r, http.StatusForbidden, "Not authorized for this action.", nil, actor)
		return
	}

	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			logRequestError(r, err, "failed to load process %s for substep %s completion", processID, substepID)
		}
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}

	substep, step, err := findSubstep(cfg.Workflow, substepID)
	if err != nil {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Substep not found.", process, actor)
		return
	}
	if len(actor.RoleSlugs) == 0 && strings.TrimSpace(actor.Role) != "" {
		actor.RoleSlugs = []string{strings.TrimSpace(actor.Role)}
	}
	r.Body = http.MaxBytesReader(w, r.Body, completionFormMaxBytes())
	if err := r.ParseForm(); err != nil {
		if isRequestTooLarge(err) {
			s.renderActionErrorForRequest(w, r, http.StatusRequestEntityTooLarge, "File too large.", process, actor)
			return
		}
		s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Invalid form.", process, actor)
		return
	}
	activeRole := strings.TrimSpace(r.FormValue("activeRole"))
	if activeRole == "" && len(actor.RoleSlugs) == 1 {
		activeRole = actor.RoleSlugs[0]
	}
	allowedRoles := substepRoles(substep)
	if !s.enforceAuth && activeRole == "" && len(allowedRoles) > 0 {
		activeRole = allowedRoles[0]
		actor.RoleSlugs = append([]string(nil), allowedRoles...)
	}
	if activeRole == "" || !containsRole(actor.RoleSlugs, activeRole) || !containsRole(allowedRoles, activeRole) {
		s.renderActionErrorForRequest(w, r, http.StatusForbidden, "Not authorized for this action.", process, actor)
		return
	}
	actor.Role = activeRole

	sequenceOK := substepSequenceOK(cfg.Workflow, process, substepID)
	if s.authorizer == nil {
		s.renderActionErrorForRequest(w, r, http.StatusBadGateway, "Cerbos check failed.", process, actor)
		return
	}
	allowed, err := s.authorizer.CanComplete(r.Context(), actor, processID, workflowKey, substep, step.Order, step.OrganizationSlug, sequenceOK)
	if err != nil {
		logRequestError(r, err, "cerbos check failed for process %s substep %s", processID, substepID)
		s.renderActionErrorForRequest(w, r, http.StatusBadGateway, "Cerbos check failed.", process, actor)
		return
	}
	if !sequenceOK {
		if progress, ok := process.Progress[substepID]; ok && progress.State == "done" && containsRole(allowedRoles, actor.Role) {
			nextReq := cloneRequestWithSelectedSubstep(r, "")
			if isProcessContentTargetRequest(r) {
				s.renderProcessContent(w, nextReq, process, actor, "")
				return
			}
			if isHTMXRequest(r) {
				s.renderProcessContent(w, nextReq, process, actor, "")
				return
			}
			s.renderDepartmentProcessPage(w, nextReq, process, actor, "")
			return
		}
		s.renderActionErrorForRequest(w, r, http.StatusConflict, "Step is locked: complete previous steps first.", process, actor)
		return
	}
	if !allowed {
		s.renderActionErrorForRequest(w, r, http.StatusForbidden, "Not authorized for this action.", process, actor)
		return
	}

	override := process.Overrides[strings.TrimSpace(substepID)]
	effective := effectiveSubstep(substep, &override)
	if strings.TrimSpace(override.SubstepID) == "" {
		effective = substep
	}
	now := s.nowUTC()
	payload, err := s.parseCompletionPayload(r, process.ID, effective, now)
	if err != nil {
		switch {
		case errors.Is(err, ErrAttachmentTooLarge):
			s.renderActionErrorForRequest(w, r, http.StatusRequestEntityTooLarge, "File too large.", process, actor)
		case errors.Is(err, errInvalidForm):
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Invalid form.", process, actor)
		default:
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, err.Error(), process, actor)
		}
		return
	}

	result, err := s.processService().SubmitSubstepForm(ctx, CompleteSubstepCmd{
		Process:     process,
		WorkflowKey: workflowKey,
		SubstepID:   substepID,
		Substep:     substep,
		Actor:       actor,
		Payload:     payload,
		Config:      cfg,
		Now:         now,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrDraftValidation):
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, err.Error(), process, actor)
		case errors.Is(err, ErrProgressUpdate):
			logRequestError(r, err, "failed to update process %s substep %s", process.ID.Hex(), substepID)
			s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to update process.", process, actor)
		case errors.Is(err, ErrNotarization):
			logRequestError(r, err, "failed to notarize process %s substep %s", process.ID.Hex(), substepID)
			s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to notarize payload.", process, actor)
		case errors.Is(err, ErrCompletionUnsupported):
			s.renderActionErrorForRequest(w, r, http.StatusConflict, "Complete is not available for this stream.", process, actor)
		default:
			logRequestError(r, err, "failed to complete process %s substep %s", process.ID.Hex(), substepID)
			s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to update process.", process, actor)
		}
		return
	}
	process = result.Process

	s.sse.Broadcast("process:"+workflowKey+":"+processID, "process-updated")
	for _, role := range s.roles(cfg) {
		s.sse.Broadcast("role:"+workflowKey+":"+role, "role-updated")
	}
	nextReq := cloneRequestWithSelectedSubstep(r, result.SelectedSubstepID)
	if isProcessContentTargetRequest(r) {
		s.renderProcessContentNotice(w, nextReq, process, actor, "", result.Confirmation)
		return
	}
	if isHTMXRequest(r) {
		s.renderProcessContentNotice(w, nextReq, process, actor, "", result.Confirmation)
		return
	}
	s.renderDepartmentProcessPageNotice(w, nextReq, process, actor, "", result.Confirmation)
}

func formConfirmTrue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func (s *Server) handleSealStep(w http.ResponseWriter, r *http.Request, processID, stepID string) {
	user, _, ok := s.requireVerifiedPost(w, r)
	if !ok {
		return
	}
	workflowKey, cfg, selected := s.selectedWorkflowOrRedirectHome(w, r)
	if !selected {
		return
	}
	actor := actorFromAccountUser(user, workflowKey)

	ctx := r.Context()
	process, err := s.loadProcess(ctx, processID)
	if err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			logRequestError(r, err, "failed to load process %s for step %s seal", processID, stepID)
		}
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}
	if !s.processBelongsToWorkflow(process, workflowKey) {
		s.renderActionErrorForRequest(w, r, http.StatusNotFound, "Process not found.", process, actor)
		return
	}

	_ = r.ParseForm()
	confirm := formConfirmTrue(r.FormValue("confirm"))
	now := s.nowUTC()
	process, err = s.processService().SealStep(ctx, SealStepCmd{
		Process:     process,
		WorkflowKey: workflowKey,
		StepID:      stepID,
		Actor:       actor,
		Config:      cfg,
		Now:         now,
		Confirm:     confirm,
	})
	if err != nil {
		switch {
		case errors.Is(err, ErrSealNotConfirmed):
			s.renderActionErrorForRequest(w, r, http.StatusBadRequest, "Seal requires confirmation.", process, actor)
		case errors.Is(err, ErrSealUnauthorized):
			s.renderActionErrorForRequest(w, r, http.StatusForbidden, "Not authorized to seal this Step.", process, actor)
		case errors.Is(err, ErrSealAlreadySealed):
			s.renderActionErrorForRequest(w, r, http.StatusConflict, "This Step is already sealed.", process, actor)
		case errors.Is(err, ErrCompletionUnsupported):
			s.renderActionErrorForRequest(w, r, http.StatusConflict, "Seal Step is not available for this stream.", process, actor)
		case errors.Is(err, ErrProgressUpdate):
			logRequestError(r, err, "failed to update process %s during seal of step %s", processID, stepID)
			s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to update process.", process, actor)
		case errors.Is(err, ErrNotarization):
			logRequestError(r, err, "failed to notarize process %s during seal of step %s", processID, stepID)
			s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to notarize payload.", process, actor)
		default:
			logRequestError(r, err, "failed to seal process %s step %s", processID, stepID)
			s.renderActionErrorForRequest(w, r, http.StatusInternalServerError, "Failed to seal Step.", process, actor)
		}
		return
	}

	s.sse.Broadcast("process:"+workflowKey+":"+processID, "process-updated")
	for _, role := range s.roles(cfg) {
		s.sse.Broadcast("role:"+workflowKey+":"+role, "role-updated")
	}
	nextReq := cloneRequestWithSelectedSubstep(r, "")
	if isProcessContentTargetRequest(r) {
		s.renderProcessContent(w, nextReq, process, actor, "")
		return
	}
	if isHTMXRequest(r) {
		s.renderProcessContent(w, nextReq, process, actor, "")
		return
	}
	http.Redirect(w, r, streamInstancePath(workflowKey, processID), http.StatusSeeOther)
}

var (
	errInvalidForm = errors.New("invalid form")
)

func (s *Server) parseCompletionPayload(r *http.Request, processID primitive.ObjectID, substep WorkflowSub, now time.Time) (map[string]interface{}, error) {
	return s.parseFormataPayload(r, processID, substep, now)
}

func (s *Server) parseFormataPayload(r *http.Request, processID primitive.ObjectID, substep WorkflowSub, now time.Time) (map[string]interface{}, error) {
	payload, err := parseFormataScalarPayload(r, substep)
	if err != nil {
		return nil, err
	}
	converted, err := s.persistFormataAttachments(r.Context(), processID, substep, payload, now, nil)
	if err != nil {
		return nil, err
	}
	convertedPayload, ok := converted.(map[string]interface{})
	if !ok {
		return nil, errInvalidForm
	}
	return convertedPayload, nil
}

func parseFormataScalarPayload(r *http.Request, substep WorkflowSub) (map[string]interface{}, error) {
	if err := r.ParseForm(); err != nil {
		return nil, errInvalidForm
	}
	rawValue := strings.TrimSpace(r.FormValue("value"))
	if rawValue == "" {
		fallback := formMapWithoutValue(r.PostForm)
		if len(fallback) == 0 {
			rawValue = "{}"
		} else {
			data, err := json.Marshal(fallback)
			if err != nil {
				return nil, errInvalidForm
			}
			rawValue = string(data)
		}
	}
	payload, err := normalizePayload(substep, rawValue)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func formMapWithoutValue(values url.Values) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	result := map[string]interface{}{}
	for key, rawValues := range values {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" || trimmedKey == "value" {
			continue
		}
		switch len(rawValues) {
		case 0:
			continue
		case 1:
			result[trimmedKey] = strings.TrimSpace(rawValues[0])
		default:
			items := make([]string, 0, len(rawValues))
			for _, item := range rawValues {
				items = append(items, strings.TrimSpace(item))
			}
			result[trimmedKey] = items
		}
	}
	return result
}
