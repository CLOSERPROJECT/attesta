package main

import (
	"context"
	"strings"
)

func (s *Server) buildStreamInstanceDetailView(ctx context.Context, cfg RuntimeConfig, workflowKey string, process *Process, actor Actor, selectedSubstepID, message string, onlyRole bool) StreamInstanceDetailView {
	roleMeta := s.roleMetaIndex(ctx)
	actions := buildSubstepViews(cfg.Workflow, process, workflowKey, actor, onlyRole, roleMeta, cfg.Roles)
	processDone := process != nil && isProcessClosed(cfg.Workflow, process)
	selected := resolveSelectedSubstepID(actions, selectedSubstepID, processDone)
	timeline := decorateTimelineSelection(buildTimeline(cfg.Workflow, process, workflowKey, roleMeta, cfg.Roles, organizationNameMap(cfg)), selected)
	timeline = decorateTimelineOrganizationLogos(timeline, organizationLogoURLMap(ctx, s.identity))
	actions = s.applyDoneByEmailToSubstepViews(ctx, cfg.Workflow, actor, actions)
	timeline = decorateTimelineSubstepBodies(timeline, actions)
	timeline, actions = applyDraftAndSealChrome(timeline, actions, cfg.Workflow, process, actor, workflowKey)

	view := StreamInstanceDetailView{
		WorkflowKey:       workflowKey,
		WorkflowPath:      streamPath(workflowKey),
		ProcessID:         processIDString(process),
		CurrentUser:       actor,
		SelectedSubstepID: selected,
		ProcessDone:       processDone,
		Error:             message,
		Timeline:          timeline,
	}
	if process != nil && !processDone {
		if action, ok := nextAuthorizedSubstepBody(cfg.Workflow, process, workflowKey, actor, roleMeta, cfg.Roles); ok {
			view.CanTerminate = true
			view.TerminateAction = streamInstancePath(workflowKey, process.ID.Hex()) + "/terminate"
			view.TerminateSubstep = action.SubstepID
			view.TerminateRoles = append([]SubstepRoleOption(nil), action.MatchingRoles...)
		}
	}
	if action, ok := selectedSubstepBody(actions, selected, processDone); ok {
		view.SelectedBody = &action
	}
	if process != nil && process.Termination != nil {
		view.Termination = s.buildStreamTerminationDetailsView(ctx, cfg.Workflow, actor, process.Termination)
	}

	if processDone {
		view.Attachments = buildProcessDownloadAttachments(workflowKey, process, collectProcessAttachments(cfg.Workflow, process))
		if process != nil && process.DPP != nil {
			view.DPPURL = digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
			view.DPPGS1 = gs1ElementString(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
		}
	}
	if view.SelectedBody != nil {
		action := *view.SelectedBody
		view.SelectedBody = &action
	}
	return view
}

func (s *Server) applyDoneByEmailToSubstepViews(ctx context.Context, def WorkflowDef, viewer Actor, actions []SubstepBodyView) []SubstepBodyView {
	if len(actions) == 0 {
		return actions
	}
	canSeeEmail := viewerCanSeeDoneByEmail(def, viewer)
	cache := map[string]userIdentityView{}
	for idx := range actions {
		identity, ok := s.lookupUserIdentityByActorID(ctx, actions[idx].DoneBy, cache)
		if !ok {
			continue
		}
		if canSeeEmail {
			if strings.TrimSpace(identity.email) != "" {
				actions[idx].DoneBy = identity.email
			} else if strings.TrimSpace(identity.fallbackID) != "" {
				actions[idx].DoneBy = identity.fallbackID
			}
			continue
		}
		if strings.TrimSpace(identity.fallbackID) != "" {
			actions[idx].DoneBy = identity.fallbackID
		}
	}
	return actions
}

func (s *Server) applyDoneByEmailToTermination(ctx context.Context, def WorkflowDef, viewer Actor, termination *ProcessTerminationView) *ProcessTerminationView {
	if termination == nil {
		return nil
	}
	identity, ok := s.lookupUserIdentityByActorID(ctx, termination.EndedBy, map[string]userIdentityView{})
	if !ok {
		return termination
	}
	mapped := *termination
	if viewerCanSeeDoneByEmail(def, viewer) && strings.TrimSpace(identity.email) != "" {
		mapped.EndedBy = identity.email
		return &mapped
	}
	if strings.TrimSpace(identity.fallbackID) != "" {
		mapped.EndedBy = identity.fallbackID
	}
	return &mapped
}

const (
	draftAndSealSubstepHint    = "Save a draft. This does not freeze or notarize. Seal the Step when the day is closed."
	draftAndSealSealedReason   = "Sealed. This Substep is frozen evidence."
	draftAndSealLockedReason   = "Locked by a prior Step"
	draftAndSealDraftingLabel  = "drafting"
	draftAndSealSealedLabel    = "sealed"
	draftAndSealDraftingNotice = "Kind Substeps can be drafted in parallel. Saves do not freeze or notarize. Seal this Step to close the production day."
	draftAndSealSealedNotice   = "This Step is sealed. Kind Substeps are frozen evidence."
)

// applyDraftAndSealChrome owns draft-and-seal view chrome for Stream instance detail:
// Substep body flags (SaveDraft/Sealed/Hint/StatusLabel/FormValue/reason copy) and
// timeline Seal controls. buildSubstepViews stays strategy-blind for this chrome.
func applyDraftAndSealChrome(timeline []TimelineStep, actions []SubstepBodyView, def WorkflowDef, process *Process, actor Actor, workflowKey string) ([]TimelineStep, []SubstepBodyView) {
	if workflowCompletionStrategy(def) != CompletionStrategyDraftAndSeal {
		return timeline, actions
	}
	for i := range actions {
		actions[i] = applyDraftAndSealBodyChrome(actions[i], process)
	}
	timeline = decorateTimelineSubstepBodies(timeline, actions)
	closed := isProcessClosed(def, process)
	for i := range timeline {
		step, err := findWorkflowStep(def, timeline[i].Summary.StepID)
		if err != nil {
			continue
		}
		timeline[i].DraftAndSeal = true
		if stepAllSubstepsDone(step, process) {
			timeline[i].Sealed = true
			timeline[i].SealNotice = draftAndSealSealedNotice
			continue
		}
		timeline[i].SealNotice = draftAndSealDraftingNotice
		if process != nil && !closed && actorCanSealStep(actor, step) {
			timeline[i].CanSeal = true
			timeline[i].SealAction = streamInstancePath(workflowKey, processIDString(process)) + "/step/" + step.StepID + "/seal"
		}
	}
	return timeline, actions
}

func applyDraftAndSealBodyChrome(action SubstepBodyView, process *Process) SubstepBodyView {
	switch action.Status {
	case "locked":
		if action.Reason == "Locked by sequence" {
			action.Reason = draftAndSealLockedReason
		}
	case "done":
		action.Sealed = true
		action.StatusLabel = draftAndSealSealedLabel
		if action.Reason == "Already completed" {
			action.Reason = draftAndSealSealedReason
		}
	case "available":
		action.StatusLabel = draftAndSealDraftingLabel
		if !action.Disabled {
			action.SaveDraft = true
			action.Hint = draftAndSealSubstepHint
		}
		if process != nil {
			if progress, ok := process.Progress[action.SubstepID]; ok && progress.Data != nil {
				action.FormValue = marshalJSONCompact(progress.Data)
			}
		}
	}
	return action
}

func terminationEndedByDisplay(endedBy string) string {
	trimmed := strings.TrimSpace(endedBy)
	if userID, ok := parseAppwriteActorID(trimmed); ok {
		return userID
	}
	return trimmed
}

func (s *Server) applyDoneByIdentityFallbackToTermination(ctx context.Context, termination *ProcessTerminationView) *ProcessTerminationView {
	if termination == nil {
		return nil
	}
	identity, ok := s.lookupUserIdentityByActorID(ctx, termination.EndedBy, map[string]userIdentityView{})
	if !ok {
		return termination
	}
	mapped := *termination
	if strings.TrimSpace(identity.fallbackID) != "" {
		mapped.EndedBy = identity.fallbackID
	}
	return &mapped
}

func (s *Server) buildStreamTerminationDetailsView(ctx context.Context, def WorkflowDef, viewer Actor, termination *ProcessTermination) *StreamTerminationDetailsView {
	if termination == nil {
		return nil
	}
	view := processTerminationView(termination)
	if strings.TrimSpace(viewer.OrgSlug) == "" {
		view = s.applyDoneByIdentityFallbackToTermination(ctx, view)
	} else {
		view = s.applyDoneByEmailToTermination(ctx, def, viewer, view)
	}
	if view == nil {
		return nil
	}
	return &StreamTerminationDetailsView{
		EndedAt:      view.EndedAt,
		EndedAtHuman: view.EndedAtHuman,
		EndedBy:      terminationEndedByDisplay(view.EndedBy),
		SubstepID:    view.SubstepID,
		Reason:       view.Reason,
	}
}
