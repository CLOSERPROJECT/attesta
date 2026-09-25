package main

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// OrganizationHomeView is the affiliated-user organization home page.
type OrganizationHomeView struct {
	PageBase
	OrganizationName    string
	OrganizationSlug    string
	OrganizationLogoURL string
	Roles               []OrgAdminRoleOption
	LeavePath           string
	CanLeave            bool
	LeaveReason         string
	ShowManageOrgLink   bool
	ManageOrgHref       string
	Error               string
}

func (s *Server) handleOrganizationHome(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return
	}
	if !s.affiliationService().IsAffiliated(identityUserForAffiliation(user)) {
		http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
		return
	}

	orgSlug := strings.TrimSpace(user.OrgSlug)
	org, err := s.identity.GetOrganizationBySlug(r.Context(), orgSlug)
	if err != nil || org == nil {
		if err != nil && !errors.Is(err, ErrIdentityNotFound) {
			logRequestError(r, err, "failed to load organization home for %s", orgSlug)
			http.Error(w, "failed to load organization", http.StatusInternalServerError)
			return
		}
		http.Error(w, "organization not found", http.StatusNotFound)
		return
	}

	roles := memberRolesForOrganizationHome(user, *org)
	logoURL := ""
	if strings.TrimSpace(org.LogoFileID) != "" {
		logoURL = "/organization/logo/" + url.PathEscape(strings.TrimSpace(org.Slug))
	}

	showManage := false
	if allowed, authErr := s.canAccessOrgAdminConsole(r.Context(), user); authErr != nil {
		logCapabilityCheckError(authErr, "cerbos check failed for organization home manage link")
	} else {
		showManage = allowed
	}

	canLeave, leaveReason := organizationHomeLeaveGate(r.Context(), s.identity, user)

	view := OrganizationHomeView{
		PageBase:            s.pageBaseForUser(user, "organization_home_body", "", ""),
		OrganizationName:    strings.TrimSpace(org.Name),
		OrganizationSlug:    strings.TrimSpace(org.Slug),
		OrganizationLogoURL: logoURL,
		Roles:               roles,
		LeavePath:           leaveOrganizationPath(),
		CanLeave:            canLeave,
		LeaveReason:         leaveReason,
		ShowManageOrgLink:   showManage,
		ManageOrgHref:       organizationPath("profile"),
		Error:               homePickerMessage(r, "error"),
	}
	if err := s.tmpl.ExecuteTemplate(w, "organization_home.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func memberRolesForOrganizationHome(user *AccountUser, org IdentityOrg) []OrgAdminRoleOption {
	if user == nil {
		return nil
	}
	catalog := rolesFromIdentityOrg(org)
	bySlug := make(map[string]Role, len(catalog))
	for _, role := range catalog {
		bySlug[canonifySlug(role.Slug)] = role
	}

	out := make([]OrgAdminRoleOption, 0, len(user.RoleSlugs))
	seen := make(map[string]struct{}, len(user.RoleSlugs))
	for _, slug := range canonifyRoleSlugs(user.RoleSlugs) {
		key := canonifySlug(slug)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if role, ok := bySlug[key]; ok {
			out = append(out, OrgAdminRoleOption{
				Slug:    role.Slug,
				Name:    role.Name,
				Palette: role.Palette,
			})
			continue
		}
		name := slug
		if containsRole([]string{slug}, "org-admin") || containsRole([]string{slug}, "org_admin") {
			name = "Org Admin"
		}
		out = append(out, OrgAdminRoleOption{
			Slug: slug,
			Name: name,
		})
	}
	return out
}

func organizationHomeLeaveGate(ctx context.Context, identity IdentityStore, user *AccountUser) (canLeave bool, leaveReason string) {
	if user == nil {
		return false, ""
	}
	if !userIsOrgAdmin(user) {
		return true, ""
	}
	if identity == nil {
		return true, ""
	}
	users, err := identity.ListOrganizationUsers(ctx, strings.TrimSpace(user.OrgSlug))
	if err != nil {
		return true, ""
	}
	adminCount := 0
	for _, orgUser := range users {
		if orgUser.IsOrgAdmin {
			adminCount++
		}
	}
	if adminCount < 2 {
		return false, "You're the only Org admin. Add another before leaving."
	}
	return true, ""
}
