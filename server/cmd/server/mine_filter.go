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

func decorateTimelineViewerOwnership(timeline []TimelineStep, def WorkflowDef, actor Actor) []TimelineStep {
	if len(timeline) == 0 {
		return timeline
	}
	for stepIndex := range timeline {
		for subIndex := range timeline[stepIndex].Substeps {
			id := timeline[stepIndex].Substeps[subIndex].SubstepID
			timeline[stepIndex].Substeps[subIndex].ViewerOwned = substepMineForViewer(def, id, actor)
		}
	}
	return timeline
}
