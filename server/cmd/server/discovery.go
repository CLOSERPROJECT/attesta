package main

import (
	"context"
	"strings"
)

// StreamDiscoveryItem is a Stream blueprint the user's Organization can take
// part in, for the affiliated Operator home discovery list.
type StreamDiscoveryItem struct {
	WorkflowKey     string
	WorkflowName    string
	Description     string
	CategorySlug    string
	SubCategorySlug string
	Href            string
	Startable       bool
	StartAction     string
}

func streamIsStartable(user *AccountUser, cfg RuntimeConfig) bool {
	if user == nil || user.IsPlatformAdmin {
		return false
	}
	orgSlug := strings.TrimSpace(user.OrgSlug)
	if orgSlug == "" {
		return false
	}
	steps := sortedSteps(cfg.Workflow)
	if len(steps) == 0 {
		return false
	}
	first := steps[0]
	if strings.TrimSpace(first.OrganizationSlug) != orgSlug {
		return false
	}
	if userIsOrgAdmin(user) {
		return true
	}
	subs := sortedSubsteps(first)
	if len(subs) == 0 {
		return false
	}
	allowed := substepRoles(subs[0])
	return len(intersectRoles(allowed, user.RoleSlugs)) > 0
}

func (s *Server) listDiscoveryStreams(ctx context.Context, user *AccountUser) ([]StreamDiscoveryItem, error) {
	if s == nil || user == nil || user.IsPlatformAdmin {
		return nil, nil
	}
	orgSlug := strings.TrimSpace(user.OrgSlug)
	if orgSlug == "" {
		return nil, nil
	}
	catalog, err := s.workflowCatalog()
	if err != nil {
		// Home chrome should still render when the catalog is unloadable.
		return nil, nil
	}

	var startable, rest []StreamDiscoveryItem
	for _, key := range sortedWorkflowKeys(catalog) {
		cfg := catalog[key]
		if !streamInvolvesOrg(cfg, orgSlug) {
			continue
		}
		canStart := streamIsStartable(user, cfg)
		item := StreamDiscoveryItem{
			WorkflowKey:     key,
			WorkflowName:    strings.TrimSpace(cfg.Workflow.Name),
			Description:     strings.TrimSpace(cfg.Workflow.Description),
			CategorySlug:    strings.TrimSpace(cfg.Workflow.CategorySlug),
			SubCategorySlug: strings.TrimSpace(cfg.Workflow.SubCategorySlug),
			Href:            streamPath(key) + "/",
			Startable:       canStart,
		}
		if canStart {
			item.StartAction = streamPath(key) + "/instance/start"
			startable = append(startable, item)
			continue
		}
		rest = append(rest, item)
	}
	return append(startable, rest...), nil
}
