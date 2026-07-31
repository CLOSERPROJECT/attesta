package main

import "strings"

func userCanAccessStream(user *AccountUser, cfg RuntimeConfig) bool {
	if user == nil {
		return false
	}
	if user.IsPlatformAdmin {
		return true
	}
	org := strings.TrimSpace(user.OrgSlug)
	if org == "" {
		return false
	}
	participates := false
	for _, o := range cfg.Organizations {
		if strings.TrimSpace(o.Slug) == org {
			participates = true
			break
		}
	}
	if !participates {
		return false
	}
	if userIsOrgAdmin(user) {
		return true
	}
	for _, role := range cfg.Roles {
		if strings.TrimSpace(role.OrgSlug) != org {
			continue
		}
		if containsRole(user.RoleSlugs, strings.TrimSpace(role.Slug)) {
			return true
		}
	}
	return false
}
