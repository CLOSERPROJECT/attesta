package main

import (
	"context"
	"net/http"
	"strings"
)

// affiliationLeaveGate returns whether the user may Leave and a reason when blocked.
// Matches LeaveOrganization sole-Org-admin invariant; Members always may Leave.
func affiliationLeaveGate(ctx context.Context, identity IdentityStore, user *AccountUser) (canLeave bool, leaveReason string) {
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

func (s *Server) populateAccountSettings(base *PageBase, user *AccountUser) {
	if base == nil || user == nil {
		return
	}
	orgSlug := strings.TrimSpace(user.OrgSlug)
	if orgSlug == "" {
		return
	}
	base.ShowAccountSettings = true
	base.LeavePath = leaveOrganizationPath()
	base.CanLeave, base.LeaveReason = affiliationLeaveGate(context.Background(), s.identity, user)
	base.AccountOrgName = orgSlug
	if s.identity == nil {
		return
	}
	org, err := s.identity.GetOrganizationBySlug(context.Background(), orgSlug)
	if err != nil || org == nil {
		return
	}
	if name := strings.TrimSpace(org.Name); name != "" {
		base.AccountOrgName = name
	}
}

func (s *Server) handleOrganizationRoot(w http.ResponseWriter, r *http.Request) {
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
	if allowed, authErr := s.canAccessOrgAdminConsole(r.Context(), user); authErr != nil {
		logCapabilityCheckError(authErr, "cerbos check failed for organization root redirect")
	} else if allowed {
		http.Redirect(w, r, organizationPath("profile"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, appHomePath, http.StatusSeeOther)
}
