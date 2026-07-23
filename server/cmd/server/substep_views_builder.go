package main

import (
	"strings"
)

func nextAuthorizedSubstepBody(def WorkflowDef, process *Process, workflowKey string, actor Actor, roleIndex map[roleMetaKey]RoleMeta, cfgRoles []WorkflowRole) (SubstepBodyView, bool) {
	for _, action := range buildSubstepViews(def, process, workflowKey, actor, false, roleIndex, cfgRoles) {
		if action.Status == "available" && !action.Disabled {
			return action, true
		}
	}
	return SubstepBodyView{}, false
}

func buildSubstepViews(def WorkflowDef, process *Process, workflowKey string, actor Actor, onlyRole bool, roleIndex map[roleMetaKey]RoleMeta, cfgRoles []WorkflowRole) []SubstepBodyView {
	var actions []SubstepBodyView
	ordered := orderedSubsteps(def)
	availMap := computeAvailability(def, process)
	substepOrgs := substepOrganizationMap(def)
	terminated := process != nil && process.Termination != nil
	terminationSubstepID := ""
	terminationReason := ""
	if terminated {
		terminationSubstepID = strings.TrimSpace(process.Termination.SubstepID)
		terminationReason = strings.TrimSpace(process.Termination.Reason)
	}
	pastTermination := false
	for _, sub := range ordered {
		var override *SubstepOverride
		if process != nil {
			if item, ok := process.Overrides[sub.SubstepID]; ok {
				itemCopy := item
				override = &itemCopy
			}
		}
		effective := effectiveSubstep(sub, override)
		allowedRoles := substepRoles(sub)
		ownedRoles := append([]string(nil), actor.RoleSlugs...)
		if len(ownedRoles) == 0 && strings.TrimSpace(actor.Role) != "" {
			ownedRoles = []string{strings.TrimSpace(actor.Role)}
		}
		matchingRoleSlugs := intersectRoles(allowedRoles, ownedRoles)
		matchingRoles := make([]SubstepRoleOption, 0, len(matchingRoleSlugs))
		for _, role := range matchingRoleSlugs {
			meta := roleMetaForOrg(substepOrgs[sub.SubstepID], role, roleIndex, cfgRoles)
			label := strings.TrimSpace(meta.Label)
			if label == "" {
				label = role
			}
			matchingRoles = append(matchingRoles, SubstepRoleOption{
				Slug:  role,
				Label: label,
			})
		}
		primaryRole := sub.Role
		if primaryRole == "" && len(allowedRoles) > 0 {
			primaryRole = allowedRoles[0]
		}
		roleBadges := make([]SubstepRoleBadge, 0, len(allowedRoles))
		for _, role := range allowedRoles {
			meta := roleMetaForOrg(substepOrgs[sub.SubstepID], role, roleIndex, cfgRoles)
			roleBadges = append(roleBadges, SubstepRoleBadge{
				ID:      role,
				Label:   meta.Label,
				Palette: meta.Palette,
			})
		}
		if onlyRole && strings.TrimSpace(actor.Role) != "" && !containsRole(allowedRoles, actor.Role) {
			continue
		}
		meta := roleMetaForOrg(substepOrgs[sub.SubstepID], primaryRole, roleIndex, cfgRoles)
		role := primaryRole
		roleLabel := meta.Label
		palette := meta.Palette
		status := "locked"
		if process != nil {
			if step, ok := process.Progress[sub.SubstepID]; ok && step.State == "done" {
				status = "done"
			} else if terminated && strings.TrimSpace(sub.SubstepID) == terminationSubstepID {
				status = processStatusTerminated
			} else if terminated && (pastTermination || terminationSubstepID == "") {
				status = "skipped"
			} else if availMap[sub.SubstepID] {
				status = "available"
			}
		}
		stepOrgSlug := substepOrgs[sub.SubstepID]
		orgAuthorized := stepOrgSlug == "" || strings.TrimSpace(actor.OrgSlug) == stepOrgSlug
		disabled := status != "available" || len(matchingRoles) == 0 || !orgAuthorized
		reason := ""
		detailMessage := ""
		if status == "locked" {
			reason = "Locked by sequence"
		} else if status == "done" {
			reason = "Already completed"
		} else if status == processStatusTerminated {
			reason = "Stream ended early"
			detailMessage = terminationReason
			if detailMessage == "" {
				detailMessage = "No reason provided"
			}
		} else if status == "skipped" {
			reason = "Stream ended early"
			detailMessage = "Step not completed because the stream was ended before this."
		} else if !orgAuthorized {
			reason = "Not authorized for organization"
		} else if len(matchingRoles) == 0 {
			reason = "Not authorized"
		}
		formSchema := ""
		formUISchema := ""
		doneAt := ""
		doneAtISO := ""
		doneBy := ""
		doneRole := ""
		description := strings.TrimSpace(sub.InputKey)
		var values []SubstepKV
		var attachments []SubstepAttachmentView
		if status == "done" && process != nil {
			if progress, ok := process.Progress[sub.SubstepID]; ok {
				description = processStepDescription(progress, sub)
				if progress.DoneAt != nil {
					doneAt = humanReadableTraceabilityTime(*progress.DoneAt)
					doneAtISO = rfc3339UTC(*progress.DoneAt)
				}
				if progress.DoneBy != nil {
					doneBy = strings.TrimSpace(progress.DoneBy.ID)
					doneRole = strings.TrimSpace(progress.DoneBy.Role)
					if doneRole != "" {
						selectedMeta := roleMetaForOrg(substepOrgs[sub.SubstepID], doneRole, roleIndex, cfgRoles)
						role = doneRole
						roleBadges = []SubstepRoleBadge{
							{
								ID:      doneRole,
								Label:   selectedMeta.Label,
								Palette: selectedMeta.Palette,
							},
						}
						roleLabel = selectedMeta.Label
						palette = selectedMeta.Palette
					}
				}
				if value, ok := processStepDataValue(progress, sub); ok {
					values = flattenDisplayValues("", value)
				}
				attachments = buildSubstepAttachments(workflowKey, process, progress.Data)
			}
		}
		formSchema = marshalJSONCompact(effective.Schema)
		formUISchema = marshalJSONCompact(effective.UISchema)
		hasOverride := override != nil && strings.TrimSpace(override.SubstepID) != ""
		overrideReason := ""
		if hasOverride {
			overrideReason = strings.TrimSpace(override.Reason)
			if status == "done" {
				detail := "Completed with local form adaptation."
				if overrideReason != "" {
					detail += "\nReason: " + overrideReason
				}
				reason = detail
			}
		}
		canAdaptForm := status == "available" && !disabled && substepSupportsLocalOverride(sub)
		adaptURL := ""
		if canAdaptForm {
			adaptURL = streamInstancePath(workflowKey, processIDString(process)) + "/substep/" + sub.SubstepID + "/override"
		}
		actions = append(actions, withSubstepBodyMode(SubstepBodyView{
			WorkflowKey:    workflowKey,
			ProcessID:      processIDString(process),
			SubstepID:      sub.SubstepID,
			Title:          sub.Title,
			Role:           role,
			RoleBadges:     roleBadges,
			MatchingRoles:  matchingRoles,
			RoleLabel:      roleLabel,
			Palette:        palette,
			InputKey:       sub.InputKey,
			Description:    description,
			InputType:      sub.InputType,
			FormSchema:     formSchema,
			FormUISchema:   formUISchema,
			Status:         status,
			DoneAt:         doneAt,
			DoneAtISO:      doneAtISO,
			DoneBy:         doneBy,
			DoneRole:       doneRole,
			Values:         values,
			Attachments:    attachments,
			Disabled:       disabled,
			Reason:         reason,
			DetailMessage:  detailMessage,
			CanAdaptForm:   canAdaptForm,
			AdaptURL:       adaptURL,
			FormataArchURL: "",
			OverrideReason: overrideReason,
			HasOverride:    hasOverride,
		}))
		if terminated && strings.TrimSpace(sub.SubstepID) == terminationSubstepID {
			pastTermination = true
		}
	}
	return actions
}

func withSubstepBodyMode(v SubstepBodyView) SubstepBodyView {
	v.Mode = resolveSubstepBodyMode(v)
	return v
}

func resolveSelectedSubstepID(actions []SubstepBodyView, requested string, processDone bool) string {
	if processDone || len(actions) == 0 {
		return ""
	}
	requested = strings.TrimSpace(requested)
	if requested != "" {
		for _, action := range actions {
			if action.SubstepID == requested {
				return requested
			}
		}
	}
	for _, action := range actions {
		if action.Status == "available" {
			return action.SubstepID
		}
	}
	return actions[0].SubstepID
}

func selectedSubstepBody(actions []SubstepBodyView, selectedSubstepID string, processDone bool) (SubstepBodyView, bool) {
	if processDone {
		return SubstepBodyView{}, false
	}
	selectedSubstepID = strings.TrimSpace(selectedSubstepID)
	if selectedSubstepID == "" {
		if len(actions) == 0 {
			return SubstepBodyView{}, false
		}
		return actions[0], true
	}
	for _, action := range actions {
		if action.SubstepID == selectedSubstepID {
			return action, true
		}
	}
	return SubstepBodyView{}, false
}
