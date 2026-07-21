package main

import "strings"

func resolveTimelineSubstepStatus(substepID string, process *Process, availableMap map[string]bool, terminated bool, terminationSubstepID string, pastTermination bool) string {
	if process == nil {
		return "locked"
	}
	if progress, ok := process.Progress[substepID]; ok && progress.State == "done" {
		return "done"
	}
	if terminated && strings.TrimSpace(substepID) == terminationSubstepID {
		return processStatusTerminated
	}
	if terminated && (pastTermination || terminationSubstepID == "") {
		return "skipped"
	}
	if availableMap[substepID] {
		return "available"
	}
	return "locked"
}

func advanceTimelinePastTermination(substepID string, terminated bool, terminationSubstepID string, pastTermination bool) bool {
	return pastTermination || (terminated && strings.TrimSpace(substepID) == terminationSubstepID)
}

type timelineWalkState struct {
	substepOrgs          map[string]string
	availableMap         map[string]bool
	terminated           bool
	terminationSubstepID string
	terminationReason    string
	pastTermination      bool
}

func newTimelineWalkState(def WorkflowDef, process *Process) timelineWalkState {
	substepOrgs := substepOrganizationMap(def)
	availableMap := computeAvailability(def, process)
	terminated := process != nil && process.Termination != nil
	terminationSubstepID := ""
	terminationReason := ""
	if terminated {
		terminationSubstepID = strings.TrimSpace(process.Termination.SubstepID)
		terminationReason = strings.TrimSpace(process.Termination.Reason)
	}
	return timelineWalkState{
		substepOrgs:          substepOrgs,
		availableMap:         availableMap,
		terminated:           terminated,
		terminationSubstepID: terminationSubstepID,
		terminationReason:    terminationReason,
	}
}

type timelineSubstepBuildContext struct {
	state       *timelineWalkState
	step        WorkflowStep
	sub         WorkflowSub
	status      string
	process     *Process
	workflowKey string
	roleIndex   map[roleMetaKey]RoleMeta
	cfgRoles    []WorkflowRole
}

type timelineStepsOptions struct {
	emptyIfNilProcess bool
	decorateStep      func(*TimelineStep)
	buildSubstep      func(timelineSubstepBuildContext) TimelineSubstep
}

func buildTimelineSteps(def WorkflowDef, process *Process, orgNames map[string]string, workflowKey string, roleIndex map[roleMetaKey]RoleMeta, cfgRoles []WorkflowRole, opts timelineStepsOptions) []TimelineStep {
	if opts.emptyIfNilProcess && process == nil {
		return nil
	}
	state := newTimelineWalkState(def, process)
	var timeline []TimelineStep
	for _, step := range sortedSteps(def) {
		workflowSubsteps := sortedSubsteps(step)
		row := TimelineStep{
			OrgSlug: strings.TrimSpace(step.OrganizationSlug),
			Summary: buildStepSummary(step, workflowSubsteps, process, orgNames),
		}
		if opts.decorateStep != nil {
			opts.decorateStep(&row)
		}
		for _, sub := range workflowSubsteps {
			status := resolveTimelineSubstepStatus(sub.SubstepID, process, state.availableMap, state.terminated, state.terminationSubstepID, state.pastTermination)
			entry := opts.buildSubstep(timelineSubstepBuildContext{
				state:       &state,
				step:        step,
				sub:         sub,
				status:      status,
				process:     process,
				workflowKey: workflowKey,
				roleIndex:   roleIndex,
				cfgRoles:    cfgRoles,
			})
			row.Substeps = append(row.Substeps, entry)
			state.pastTermination = advanceTimelinePastTermination(sub.SubstepID, state.terminated, state.terminationSubstepID, state.pastTermination)
		}
		timeline = append(timeline, row)
	}
	return timeline
}

func buildTimeline(def WorkflowDef, process *Process, workflowKey string, roleIndex map[roleMetaKey]RoleMeta, cfgRoles []WorkflowRole, orgNames map[string]string) []TimelineStep {
	return buildTimelineSteps(def, process, orgNames, workflowKey, roleIndex, cfgRoles, timelineStepsOptions{
		buildSubstep: buildTimelineSubstep,
	})
}

func buildTimelineSubstepShellBody(ctx timelineSubstepBuildContext, palette, doneBy, doneAtHuman, doneAtISO string) SubstepBodyView {
	status := ctx.status
	disabled := false
	if status == "available" {
		// Shell-only stub; authorization applied when full body replaces this.
		disabled = false
	}
	return SubstepBodyView{
		SubstepID: ctx.sub.SubstepID,
		Title:     ctx.sub.Title,
		Status:    status,
		Palette:   palette,
		DoneBy:    doneBy,
		DoneAt:    doneAtHuman,
		DoneAtISO: doneAtISO,
		Disabled:  disabled,
	}
}

func buildTimelineSubstep(ctx timelineSubstepBuildContext) TimelineSubstep {
	sub := ctx.sub
	allowedRoles := substepRoles(sub)
	primaryRole := sub.Role
	if strings.TrimSpace(primaryRole) == "" && len(allowedRoles) > 0 {
		primaryRole = allowedRoles[0]
	}
	meta := roleMetaForOrg(ctx.state.substepOrgs[sub.SubstepID], primaryRole, ctx.roleIndex, ctx.cfgRoles)
	entry := TimelineSubstep{
		SubstepID: sub.SubstepID,
		Title:     sub.Title,
		Palette:   meta.Palette,
		Status:    ctx.status,
	}
	if entry.Status == "done" && ctx.process != nil {
		progress := ctx.process.Progress[sub.SubstepID]
		if progress.DoneBy != nil {
			entry.DoneBy = progress.DoneBy.ID
			entry.DoneRole = progress.DoneBy.Role
			selectedRole := strings.TrimSpace(progress.DoneBy.Role)
			if selectedRole != "" {
				selectedMeta := roleMetaForOrg(ctx.state.substepOrgs[sub.SubstepID], selectedRole, ctx.roleIndex, ctx.cfgRoles)
				entry.Palette = selectedMeta.Palette
			}
		}
		if progress.DoneAt != nil {
			entry.DoneAt = humanReadableTraceabilityTime(*progress.DoneAt)
			entry.DoneAtISO = rfc3339UTC(*progress.DoneAt)
		}
	}
	entry.StatusLabel = processStatusLabel(entry.Status)
	doneBy := entry.DoneBy
	doneAt := entry.DoneAt
	doneAtISO := entry.DoneAtISO
	shellBody := buildTimelineSubstepShellBody(ctx, entry.Palette, doneBy, doneAt, doneAtISO)
	entry.Body = &shellBody
	return entry
}

func decorateTimelineSelection(timeline []TimelineStep, selectedSubstepID string) []TimelineStep {
	selectedSubstepID = strings.TrimSpace(selectedSubstepID)
	for stepIndex := range timeline {
		expanded := false
		for substepIndex := range timeline[stepIndex].Substeps {
			selected := selectedSubstepID != "" && timeline[stepIndex].Substeps[substepIndex].SubstepID == selectedSubstepID
			timeline[stepIndex].Substeps[substepIndex].Selected = selected
			if selected {
				expanded = true
			}
		}
		timeline[stepIndex].Expanded = expanded
	}
	return timeline
}

func decorateTimelineSubstepBodies(timeline []TimelineStep, actions []SubstepBodyView) []TimelineStep {
	if len(timeline) == 0 || len(actions) == 0 {
		return timeline
	}
	actionsBySubstep := make(map[string]SubstepBodyView, len(actions))
	for _, action := range actions {
		actionsBySubstep[strings.TrimSpace(action.SubstepID)] = action
	}
	for stepIndex := range timeline {
		for substepIndex := range timeline[stepIndex].Substeps {
			substepID := strings.TrimSpace(timeline[stepIndex].Substeps[substepIndex].SubstepID)
			action, ok := actionsBySubstep[substepID]
			if !ok {
				continue
			}
			actionCopy := action
			timeline[stepIndex].Substeps[substepIndex].Body = &actionCopy
		}
	}
	return timeline
}

func decorateTimelineOrganizationLogos(timeline []TimelineStep, logoURLs map[string]string) []TimelineStep {
	if len(timeline) == 0 || len(logoURLs) == 0 {
		return timeline
	}
	for stepIndex := range timeline {
		timeline[stepIndex].Summary.OrgLogoURL = strings.TrimSpace(logoURLs[strings.TrimSpace(timeline[stepIndex].OrgSlug)])
	}
	return timeline
}
