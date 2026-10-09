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
	timeline = decorateTimelineDraftAndSealChrome(timeline, cfg.Workflow, process, workflowKey, actor)

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
	draftAndSealDraftingNotice = "Kind Substeps can be drafted in parallel. Saves do not freeze or notarize. Seal this Step to close the production day."
	draftAndSealSealedNotice   = "This Step is sealed. Kind Substeps are frozen evidence."
)

func decorateTimelineDraftAndSealChrome(timeline []TimelineStep, def WorkflowDef, process *Process, workflowKey string, actor Actor) []TimelineStep {
	if workflowCompletionStrategy(def) != CompletionStrategyDraftAndSeal {
		return timeline
	}
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
	return timeline
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
