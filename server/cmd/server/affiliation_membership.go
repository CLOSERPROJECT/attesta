package main

import (
	"context"
	"errors"
	"strings"
)

// LeaveOrganization removes the session user from their organization when allowed.
// Sole org admins are blocked until another org admin exists.
func (a *Affiliation) LeaveOrganization(ctx context.Context, sessionSecret string, user IdentityUser) error {
	if !a.IsAffiliated(user) {
		return ErrAffiliationNotAffiliated
	}

	current, err := a.identity.GetCurrentUser(ctx, sessionSecret)
	if err != nil {
		return err
	}
	if strings.TrimSpace(current.ID) != strings.TrimSpace(user.ID) {
		return ErrIdentityUnauthorized
	}
	if !a.IsAffiliated(current) {
		return ErrAffiliationNotAffiliated
	}

	orgSlug := strings.TrimSpace(current.OrgSlug)
	if current.IsOrgAdmin {
		users, listErr := a.identity.ListOrganizationUsers(ctx, orgSlug)
		if listErr != nil {
			return listErr
		}
		adminCount := 0
		for _, orgUser := range users {
			if orgUser.IsOrgAdmin {
				adminCount++
			}
		}
		if adminCount < 2 {
			return ErrAffiliationSoleOrgAdmin
		}
	}

	membershipID := strings.TrimSpace(current.MembershipID)
	if membershipID == "" {
		return ErrIdentityNotFound
	}
	if err := a.identity.DeleteOrganizationMembership(ctx, sessionSecret, orgSlug, membershipID); err != nil {
		return err
	}

	return a.stripManagedIdentityLabels(ctx, current.ID)
}

// RemoveOrganizationMember deletes an organization membership as an org admin
// (session auth) and clears the target user's managed identity labels.
func (a *Affiliation) RemoveOrganizationMember(ctx context.Context, sessionSecret, orgSlug, membershipID, userID string) error {
	if err := a.identity.DeleteOrganizationMembership(ctx, sessionSecret, orgSlug, membershipID); err != nil {
		return err
	}
	if userID := strings.TrimSpace(userID); userID != "" {
		if err := a.stripManagedIdentityLabels(ctx, userID); err != nil && !errors.Is(err, ErrIdentityNotFound) {
			return err
		}
	}
	return nil
}
