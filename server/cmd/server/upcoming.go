package main

import (
	"context"
	"strings"
)

// StreamUpcomingItem is an active Stream instance that involves the user's
// Organization where the next actionable Substep is not theirs. Watchlist only —
// not an Attention item (does not light Attention dots).
type StreamUpcomingItem struct {
	ProcessID    string
	WorkflowKey  string
	WorkflowName string
	InstanceName string
	WaitingOn    string
	Href         string
}

func (s *Server) listUpcomingStreamInstances(ctx context.Context, user *AccountUser) ([]StreamUpcomingItem, error) {
	if s == nil || user == nil || user.IsPlatformAdmin {
		return nil, nil
	}
	orgSlug := strings.TrimSpace(user.OrgSlug)
	if orgSlug == "" || s.store == nil {
		return nil, nil
	}
	catalog, err := s.workflowCatalog()
	if err != nil {
		// Home chrome should still render Join / your-turn when the catalog is unloadable.
		return nil, nil
	}
	roleMeta := s.roleMetaIndex(ctx)
	items := make([]StreamUpcomingItem, 0)
	for _, key := range sortedWorkflowKeys(catalog) {
		cfg := catalog[key]
		if !streamInvolvesOrg(cfg, orgSlug) {
			continue
		}
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			return nil, listErr
		}
		actor := actorFromAccountUser(user, key)
		orgNames := organizationNameMap(cfg)
		substepOrgs := substepOrganizationMap(cfg.Workflow)
		for i := range processes {
			process := processes[i]
			process.Progress = normalizeProgressKeys(process.Progress)
			if deriveProcessStatus(cfg.Workflow, &process) != processStatusActive {
				continue
			}
			if _, yours := nextAuthorizedSubstepBody(cfg.Workflow, &process, key, actor, roleMeta, cfg.Roles); yours {
				continue
			}
			next, ok := nextAvailableSubstep(cfg.Workflow, &process)
			if !ok {
				continue
			}
			instanceName := strings.TrimSpace(process.Name)
			if instanceName == "" {
				instanceName = strings.TrimSpace(cfg.Workflow.Name)
			}
			items = append(items, StreamUpcomingItem{
				ProcessID:    process.ID.Hex(),
				WorkflowKey:  key,
				WorkflowName: strings.TrimSpace(cfg.Workflow.Name),
				InstanceName: instanceName,
				WaitingOn:    waitingOnCopy(orgSlug, substepOrgs[next.SubstepID], next, cfg, orgNames, roleMeta),
				Href:         streamInstanceSubstepPath(key, process.ID.Hex(), next.SubstepID),
			})
		}
	}
	return items, nil
}

func streamInvolvesOrg(cfg RuntimeConfig, orgSlug string) bool {
	org := strings.TrimSpace(orgSlug)
	if org == "" {
		return false
	}
	for _, o := range cfg.Organizations {
		if strings.TrimSpace(o.Slug) == org {
			return true
		}
	}
	return false
}

func waitingOnCopy(userOrg, stepOrg string, sub WorkflowSub, cfg RuntimeConfig, orgNames map[string]string, roleIndex map[roleMetaKey]RoleMeta) string {
	allowed := substepRoles(sub)
	primary := strings.TrimSpace(sub.Role)
	if primary == "" && len(allowed) > 0 {
		primary = allowed[0]
	}
	label := strings.TrimSpace(roleMetaForOrg(stepOrg, primary, roleIndex, cfg.Roles).Label)
	if label == "" || label == primary {
		if name := workflowRoleDisplayName(cfg.Roles, stepOrg, primary); name != "" {
			label = name
		}
	}
	if label == "" {
		label = primary
	}
	if label == "" {
		return "Waiting on another party"
	}
	stepOrg = strings.TrimSpace(stepOrg)
	userOrg = strings.TrimSpace(userOrg)
	if stepOrg != "" && stepOrg != userOrg {
		orgName := organizationDisplayName(stepOrg, orgNames)
		if orgName != "" {
			return "Waiting on " + orgName + " · " + label
		}
	}
	return "Waiting on " + label
}

func workflowRoleDisplayName(roles []WorkflowRole, stepOrg, roleSlug string) string {
	roleSlug = strings.TrimSpace(roleSlug)
	if roleSlug == "" {
		return ""
	}
	stepOrg = strings.TrimSpace(stepOrg)
	for _, role := range roles {
		if strings.TrimSpace(role.Slug) != roleSlug {
			continue
		}
		if stepOrg != "" && strings.TrimSpace(role.OrgSlug) != stepOrg {
			continue
		}
		if name := strings.TrimSpace(role.Name); name != "" {
			return name
		}
		return roleSlug
	}
	return ""
}
