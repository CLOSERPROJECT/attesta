package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LeaveOrganization removes the session user from their organization when allowed.
// Sole org admins are blocked until another org admin exists.
// On success, remaining Org admins are emailed (log-and-continue on mail failure).
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
	var orgUsers []IdentityUser
	users, listErr := a.identity.ListOrganizationUsers(ctx, orgSlug)
	if current.IsOrgAdmin {
		if listErr != nil {
			return listErr
		}
		orgUsers = users
		if decision := CanLeaveOrganization(true, CountOrgAdmins(users)); !decision.Allowed {
			return ErrAffiliationSoleOrgAdmin
		}
	} else if listErr == nil {
		orgUsers = users
	}

	membershipID := strings.TrimSpace(current.MembershipID)
	if membershipID == "" {
		return ErrIdentityNotFound
	}
	if err := a.identity.DeleteOrganizationMembership(ctx, sessionSecret, orgSlug, membershipID); err != nil {
		return err
	}
	if err := a.stripManagedIdentityLabels(ctx, current.ID); err != nil {
		return err
	}

	a.notifyMemberLeft(ctx, orgSlug, current, orgUsers)
	return nil
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

func (a *Affiliation) notifyMemberLeft(ctx context.Context, orgSlug string, leaver IdentityUser, orgUsers []IdentityUser) {
	to := organizationAdminEmails(orgUsers, leaver.ID)
	if len(to) == 0 {
		return
	}
	orgName := strings.TrimSpace(orgSlug)
	if org, err := a.identity.GetOrganizationBySlug(ctx, orgSlug); err == nil && org != nil {
		if name := strings.TrimSpace(org.Name); name != "" {
			orgName = name
		}
	}
	leaverEmail := strings.TrimSpace(leaver.Email)
	if leaverEmail == "" {
		leaverEmail = strings.TrimSpace(leaver.ID)
	}
	a.notify(ctx, MailMessage{
		Kind:    MailKindMemberLeft,
		To:      to,
		Subject: "Member left: " + orgName,
		Body: fmt.Sprintf(
			"%s left organization %q (slug %q).\n\nOpen: %s",
			leaverEmail, orgName, orgSlug, a.actionURL("/my/organization/members"),
		),
	})
}
