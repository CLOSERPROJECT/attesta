package main

import (
	"context"
	"strings"
)

type roleMetaKey struct {
	OrgSlug  string
	RoleSlug string
}

func (s *Server) roleMetaIndex(ctx context.Context) map[roleMetaKey]RoleMeta {
	if s == nil || s.identity == nil {
		return map[roleMetaKey]RoleMeta{}
	}
	orgs, err := s.identity.ListOrganizations(ctx)
	if err != nil {
		return map[roleMetaKey]RoleMeta{}
	}
	index := make(map[roleMetaKey]RoleMeta)
	for _, org := range orgs {
		orgSlug := strings.TrimSpace(org.Slug)
		if orgSlug == "" {
			continue
		}
		for _, role := range org.Roles {
			roleSlug := strings.TrimSpace(role.Slug)
			if roleSlug == "" {
				continue
			}
			label := strings.TrimSpace(role.Name)
			if label == "" {
				label = roleSlug
			}
			index[roleMetaKey{OrgSlug: orgSlug, RoleSlug: roleSlug}] = RoleMeta{
				ID:      roleSlug,
				Label:   label,
				Palette: rolePaletteKeyFromStyle(role.Color, role.Border, role.Name),
			}
		}
	}
	return index
}

func resolveRoleOrgSlug(stepOrgSlug, roleSlug string, cfgRoles []WorkflowRole) string {
	stepOrgSlug = strings.TrimSpace(stepOrgSlug)
	if stepOrgSlug != "" {
		return stepOrgSlug
	}
	roleSlug = strings.TrimSpace(roleSlug)
	for _, role := range cfgRoles {
		if strings.TrimSpace(role.Slug) == roleSlug {
			if org := strings.TrimSpace(role.OrgSlug); org != "" {
				return org
			}
		}
	}
	return ""
}

func roleMetaForOrg(stepOrgSlug, roleSlug string, index map[roleMetaKey]RoleMeta, cfgRoles []WorkflowRole) RoleMeta {
	roleSlug = strings.TrimSpace(roleSlug)
	if roleSlug == "" {
		return RoleMeta{Palette: "fallback"}
	}
	orgSlug := resolveRoleOrgSlug(stepOrgSlug, roleSlug, cfgRoles)
	if orgSlug != "" {
		if meta, ok := index[roleMetaKey{OrgSlug: orgSlug, RoleSlug: roleSlug}]; ok {
			return meta
		}
	}
	for key, meta := range index {
		if key.RoleSlug == roleSlug {
			return meta
		}
	}
	return RoleMeta{
		ID:      roleSlug,
		Label:   roleSlug,
		Palette: "fallback",
	}
}
