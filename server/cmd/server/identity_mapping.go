package main

import "strings"

const (
	identityOrgAdminLabel          = "attesta:org-admin"
	identityRoleLabelPrefix        = "attesta:role:"
	identityInviteRolePrefix       = "attesta-role:"
	identityMembershipOwnerRole    = "owner"
	identityMembershipMemberRole   = "member"
	identityTeamPrefsSchemaVersion = 1
)

type identityInviteRoles struct {
	IsOrgAdmin      bool
	MembershipRoles []string
	BusinessRoles   []string
}

func encodeIdentityRoleLabel(slug string) string {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ""
	}
	return identityRoleLabelPrefix + slug
}

func decodeIdentityRoleLabels(labels []string) []string {
	roleSlugs := make([]string, 0, len(labels))
	for _, label := range labels {
		label = strings.TrimSpace(label)
		if !strings.HasPrefix(label, identityRoleLabelPrefix) {
			continue
		}
		slug := strings.TrimSpace(strings.TrimPrefix(label, identityRoleLabelPrefix))
		if slug == "" {
			continue
		}
		roleSlugs = append(roleSlugs, slug)
	}
	return uniqueIdentityStrings(roleSlugs)
}

func encodeIdentityOrgPrefs(org IdentityOrg) appwriteTeamPrefs {
	return appwriteTeamPrefs{
		SchemaVersion: identityTeamPrefsSchemaVersion,
		Slug:          strings.TrimSpace(org.Slug),
		LogoFileID:    strings.TrimSpace(org.LogoFileID),
		Roles:         append([]IdentityRole(nil), org.Roles...),
	}
}

func decodeIdentityOrgFromTeam(id, name string, prefs appwriteTeamPrefs) IdentityOrg {
	slug := strings.TrimSpace(prefs.Slug)
	if slug == "" {
		slug = strings.TrimSpace(id)
	}
	return IdentityOrg{
		ID:         strings.TrimSpace(id),
		Slug:       slug,
		Name:       strings.TrimSpace(name),
		LogoFileID: strings.TrimSpace(prefs.LogoFileID),
		Roles:      append([]IdentityRole(nil), prefs.Roles...),
	}
}

func encodeInviteMembershipRoles(roleSlugs []string, isOrgAdmin bool) []string {
	roles := []string{identityMembershipMemberRole}
	if isOrgAdmin {
		roles[0] = identityMembershipOwnerRole
	}
	for _, slug := range uniqueIdentityStrings(roleSlugs) {
		if slug == "" {
			continue
		}
		roles = append(roles, identityInviteRolePrefix+slug)
	}
	return roles
}

func decodeInviteMembershipRoles(roles []string) identityInviteRoles {
	decoded := identityInviteRoles{
		MembershipRoles: []string{identityMembershipMemberRole},
	}
	for _, role := range roles {
		role = strings.TrimSpace(role)
		switch {
		case strings.EqualFold(role, identityMembershipOwnerRole):
			decoded.IsOrgAdmin = true
			decoded.MembershipRoles = []string{identityMembershipOwnerRole}
		case strings.EqualFold(role, identityMembershipMemberRole):
		case strings.HasPrefix(role, identityInviteRolePrefix):
			slug := strings.TrimSpace(strings.TrimPrefix(role, identityInviteRolePrefix))
			if slug != "" {
				decoded.BusinessRoles = append(decoded.BusinessRoles, slug)
			}
		}
	}
	decoded.BusinessRoles = uniqueIdentityStrings(decoded.BusinessRoles)
	return decoded
}

func uniqueIdentityStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
