package main

import "strings"

func buildTimeline(def WorkflowDef, process *Process, workflowKey string, roleIndex map[roleMetaKey]RoleMeta, cfgRoles []WorkflowRole, orgNames map[string]string) []TimelineStep {
	steps := sortedSteps(def)
	substepOrgs := substepOrganizationMap(def)
	availableMap := computeAvailability(def, process)
	terminated := process != nil && process.Termination != nil
	terminationSubstepID := ""
	if terminated {
		terminationSubstepID = strings.TrimSpace(process.Termination.SubstepID)
	}
	pastTermination := false

	var timeline []TimelineStep
	for _, step := range steps {
		workflowSubsteps := sortedSubsteps(step)
		row := TimelineStep{
			OrgSlug: strings.TrimSpace(step.OrganizationSlug),
			Summary: buildStepSummary(step, workflowSubsteps, process, orgNames),
		}
		for _, sub := range workflowSubsteps {
			allowedRoles := substepRoles(sub)
			primaryRole := sub.Role
			if strings.TrimSpace(primaryRole) == "" && len(allowedRoles) > 0 {
				primaryRole = allowedRoles[0]
			}
			meta := roleMetaForOrg(substepOrgs[sub.SubstepID], primaryRole, roleIndex, cfgRoles)
			entry := TimelineSubstep{
				SubstepID: sub.SubstepID,
				Title:     sub.Title,
				Palette:   meta.Palette,
			}
			if process != nil {
				if progress, ok := process.Progress[sub.SubstepID]; ok && progress.State == "done" {
					entry.Status = "done"
					if progress.DoneBy != nil {
						entry.DoneBy = progress.DoneBy.ID
						entry.DoneRole = progress.DoneBy.Role
						selectedRole := strings.TrimSpace(progress.DoneBy.Role)
						if selectedRole != "" {
							selectedMeta := roleMetaForOrg(substepOrgs[sub.SubstepID], selectedRole, roleIndex, cfgRoles)
							entry.Palette = selectedMeta.Palette
						}
					}
					if progress.DoneAt != nil {
						entry.DoneAt = humanReadableTraceabilityTime(*progress.DoneAt)
					}
				} else if terminated && strings.TrimSpace(sub.SubstepID) == terminationSubstepID {
					entry.Status = processStatusTerminated
				} else if terminated && (pastTermination || terminationSubstepID == "") {
					entry.Status = "skipped"
				} else if availableMap[sub.SubstepID] {
					entry.Status = "available"
				} else {
					entry.Status = "locked"
				}
			} else {
				entry.Status = "locked"
			}
			entry.StatusLabel = processStatusLabel(entry.Status)
			row.Substeps = append(row.Substeps, entry)
			if terminated && strings.TrimSpace(sub.SubstepID) == terminationSubstepID {
				pastTermination = true
			}
		}
		timeline = append(timeline, row)
	}
	return timeline
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
			switch action.Status {
			case "done", "locked", processStatusTerminated, "skipped":
				timeline[stepIndex].Substeps[substepIndex].Status = action.Status
				timeline[stepIndex].Substeps[substepIndex].StatusLabel = processStatusLabel(action.Status)
			case "available":
				if action.Disabled {
					timeline[stepIndex].Substeps[substepIndex].Status = "active"
					timeline[stepIndex].Substeps[substepIndex].StatusLabel = processStatusLabel("active")
				} else {
					timeline[stepIndex].Substeps[substepIndex].Status = "available"
					timeline[stepIndex].Substeps[substepIndex].StatusLabel = processStatusLabel("available")
				}
			}
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
