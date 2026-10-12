package main

import (
	"strings"
)

func isStandingOrgAdminRoleSlug(slug string) bool {
	slug = strings.TrimSpace(slug)
	return slug == "org-admin" || slug == "org_admin"
}

func actorStreamRoleSlugs(actor Actor) []string {
	owned := append([]string(nil), actor.RoleSlugs...)
	if len(owned) == 0 && strings.TrimSpace(actor.Role) != "" {
		owned = []string{strings.TrimSpace(actor.Role)}
	}
	filtered := make([]string, 0, len(owned))
	for _, slug := range owned {
		if isStandingOrgAdminRoleSlug(slug) {
			continue
		}
		filtered = append(filtered, slug)
	}
	return filtered
}

func parseMineFilterEnabled(raw string) bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "0", "false":
		return false
	default:
		return true
	}
}

func workflowSubstepByID(def WorkflowDef, substepID string) (WorkflowSub, bool) {
	substepID = strings.TrimSpace(substepID)
	for _, sub := range orderedSubsteps(def) {
		if sub.SubstepID == substepID {
			return sub, true
		}
	}
	return WorkflowSub{}, false
}

func substepMineForViewer(def WorkflowDef, substepID string, actor Actor) bool {
	sub, ok := workflowSubstepByID(def, substepID)
	if !ok {
		return false
	}
	stepOrg := strings.TrimSpace(substepOrganizationMap(def)[substepID])
	viewerOrg := strings.TrimSpace(actor.OrgSlug)
	if stepOrg != "" && viewerOrg != stepOrg {
		return false
	}
	return len(intersectRoles(substepRoles(sub), actorStreamRoleSlugs(actor))) > 0
}

func filterTimelineForMineFilter(timeline []TimelineStep, def WorkflowDef, actor Actor, carveOutSubstepID string, enabled bool) []TimelineStep {
	if !enabled || len(timeline) == 0 {
		return timeline
	}
	carveOutSubstepID = strings.TrimSpace(carveOutSubstepID)
	filtered := make([]TimelineStep, 0, len(timeline))
	for _, step := range timeline {
		substeps := make([]TimelineSubstep, 0, len(step.Substeps))
		for _, sub := range step.Substeps {
			if substepMineForViewer(def, sub.SubstepID, actor) {
				substeps = append(substeps, sub)
				continue
			}
			if carveOutSubstepID != "" && sub.SubstepID == carveOutSubstepID {
				substeps = append(substeps, sub)
			}
		}
		if len(substeps) == 0 {
			continue
		}
		step.Substeps = substeps
		filtered = append(filtered, step)
	}
	return filtered
}

func reconcileSelectedSubstepForMineFilter(selected, requestedSubstepID string, actions []SubstepBodyView, def WorkflowDef, actor Actor, processDone, mineFilterOn bool) string {
	if !mineFilterOn || processDone {
		return selected
	}
	if strings.TrimSpace(requestedSubstepID) != "" {
		return selected
	}
	for _, action := range actions {
		if action.Status == "available" && substepMineForViewer(def, action.SubstepID, actor) {
			return action.SubstepID
		}
	}
	for _, action := range actions {
		if substepMineForViewer(def, action.SubstepID, actor) {
			return action.SubstepID
		}
	}
	return ""
}
