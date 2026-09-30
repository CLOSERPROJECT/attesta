package main

import (
	"context"
	"strings"
)

// ListYourTurn implements yourTurnStreamSource for Server by reusing
// nextAuthorizedSubstepBody (same eligibility as stream-dashboard "available").
func (s *Server) ListYourTurn(ctx context.Context, user *AccountUser) ([]StreamAttentionItem, error) {
	return s.listYourTurnStreamAttention(ctx, user)
}

func (s *Server) listYourTurnStreamAttention(ctx context.Context, user *AccountUser) ([]StreamAttentionItem, error) {
	if s == nil || user == nil || user.IsPlatformAdmin {
		return nil, nil
	}
	if strings.TrimSpace(user.OrgSlug) == "" || s.store == nil {
		return nil, nil
	}
	catalog, err := s.workflowCatalog()
	if err != nil {
		// Chrome runs on pages without a workflow configDir; treat unloadable
		// catalogs as no your-turn items rather than poisoning Join Attention.
		return nil, nil
	}
	roleMeta := s.roleMetaIndex(ctx)
	items := make([]StreamAttentionItem, 0)
	for _, key := range sortedWorkflowKeys(catalog) {
		cfg := catalog[key]
		if !userCanAccessStream(user, cfg) {
			continue
		}
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			return nil, listErr
		}
		actor := actorFromAccountUser(user, key)
		for i := range processes {
			process := processes[i]
			process.Progress = normalizeProgressKeys(process.Progress)
			if deriveProcessStatus(cfg.Workflow, &process) != processStatusActive {
				continue
			}
			action, ok := nextAuthorizedSubstepBody(cfg.Workflow, &process, key, actor, roleMeta, cfg.Roles)
			if !ok {
				continue
			}
			instanceName := strings.TrimSpace(process.Name)
			if instanceName == "" {
				instanceName = strings.TrimSpace(cfg.Workflow.Name)
			}
			items = append(items, StreamAttentionItem{
				ProcessID:    process.ID.Hex(),
				WorkflowKey:  key,
				WorkflowName: strings.TrimSpace(cfg.Workflow.Name),
				InstanceName: instanceName,
				SubstepTitle: strings.TrimSpace(action.Title),
				Href:         streamInstancePath(key, process.ID.Hex()),
			})
		}
	}
	return items, nil
}
