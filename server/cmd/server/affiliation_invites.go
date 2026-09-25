package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// InvitationAccept is the unified accept command for email-secret and logged-in pending paths.
// Secret path: TeamID, MembershipID, UserID, Secret (non-empty).
// Logged-in path: User + MembershipID (Secret empty).
type InvitationAccept struct {
	TeamID       string
	MembershipID string
	UserID       string
	Secret       string
	User         IdentityUser
	// RedirectURL is used only to restore a pending invite if logged-in accept
	// deletes the membership then fails to grant.
	RedirectURL string
}

// InvitationAcceptResult is what HTTP handlers need after a successful accept.
type InvitationAcceptResult struct {
	Session       IdentitySession
	HasSession    bool
	NeedsPassword bool
}

// InviteUserCommand creates an outbound Invitation into an organization.
type InviteUserCommand struct {
	OrgSlug       string
	Email         string
	RedirectURL   string
	RoleSlugs     []string
	IsOrgAdmin    bool
	SessionSecret string // when empty, uses the admin API path
}

// InviteUserResult is the created (unconfirmed) membership.
type InviteUserResult struct {
	Membership IdentityMembership
}

// AcceptInvitation confirms an Invitation via email secret or logged-in pending membership.
func (a *Affiliation) AcceptInvitation(ctx context.Context, req InvitationAccept) (InvitationAcceptResult, error) {
	if strings.TrimSpace(req.Secret) != "" {
		return a.acceptInvitationWithSecret(ctx, req)
	}
	if strings.TrimSpace(req.User.ID) == "" || strings.TrimSpace(req.MembershipID) == "" {
		return InvitationAcceptResult{}, ErrAffiliationNotFound
	}
	if err := a.acceptPendingInvitation(ctx, req.User, req.MembershipID, req.RedirectURL); err != nil {
		return InvitationAcceptResult{}, err
	}
	return InvitationAcceptResult{}, nil
}

func (a *Affiliation) acceptInvitationWithSecret(ctx context.Context, req InvitationAccept) (InvitationAcceptResult, error) {
	teamID := strings.TrimSpace(req.TeamID)
	membershipID := strings.TrimSpace(req.MembershipID)
	userID := strings.TrimSpace(req.UserID)
	secret := strings.TrimSpace(req.Secret)
	if teamID == "" || membershipID == "" || userID == "" || secret == "" {
		return InvitationAcceptResult{}, ErrAffiliationNotFound
	}
	if err := a.ensureInviteAcceptCompatible(ctx, userID, teamID); err != nil {
		return InvitationAcceptResult{}, err
	}
	session, err := a.identity.AcceptInvite(ctx, teamID, membershipID, userID, secret)
	if err != nil {
		return InvitationAcceptResult{}, fmt.Errorf("%w: %w", ErrAffiliationInviteAcceptFailed, err)
	}
	result := InvitationAcceptResult{Session: session, HasSession: true}
	identityUser, err := a.identity.GetCurrentUser(ctx, session.Secret)
	if err != nil {
		return result, nil
	}
	result.NeedsPassword = !identityUser.PasswordSet
	_ = a.WithdrawPendingJoinRequest(ctx, identityUser)
	_ = a.WithdrawPendingOrganizationCreationRequest(ctx, identityUser)
	return result, nil
}

func (a *Affiliation) acceptPendingInvitation(ctx context.Context, user IdentityUser, membershipID, redirectURL string) error {
	if a.IsAffiliated(user) {
		return ErrAffiliationAlreadyAffiliated
	}
	invite, err := a.findPendingInvite(ctx, user.ID, membershipID)
	if err != nil {
		return err
	}
	if err := a.identity.DeleteOrganizationMembershipAsAdmin(ctx, invite.OrgSlug, invite.MembershipID); err != nil {
		return err
	}
	if err := a.grantOrganizationMembership(ctx, invite.OrgSlug, user.ID, invite.RoleSlugs, invite.IsOrgAdmin); err != nil {
		if email := strings.TrimSpace(user.Email); email != "" {
			_, _ = a.identity.InviteOrganizationUserAsAdmin(
				ctx,
				invite.OrgSlug,
				email,
				strings.TrimSpace(redirectURL),
				invite.RoleSlugs,
				invite.IsOrgAdmin,
			)
		}
		return err
	}
	_ = a.WithdrawPendingJoinRequest(ctx, user)
	_ = a.WithdrawPendingOrganizationCreationRequest(ctx, user)
	return nil
}

// AcceptPendingInvite confirms a pending invite without the email secret.
// Prefer AcceptInvitation; this remains as a thin wrapper for existing call sites.
func (a *Affiliation) AcceptPendingInvite(ctx context.Context, user IdentityUser, membershipID string) error {
	_, err := a.AcceptInvitation(ctx, InvitationAccept{
		User:         user,
		MembershipID: membershipID,
	})
	return err
}

// InviteUser sends an outbound Invitation after affiliation compatibility checks.
func (a *Affiliation) InviteUser(ctx context.Context, cmd InviteUserCommand) (InviteUserResult, error) {
	orgSlug := strings.TrimSpace(cmd.OrgSlug)
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	redirectURL := strings.TrimSpace(cmd.RedirectURL)
	if orgSlug == "" || email == "" {
		return InviteUserResult{}, ErrAffiliationNotFound
	}
	existingUser, err := a.identity.GetUserByEmail(ctx, email)
	switch {
	case err == nil:
		if err := a.ensureInviteOrgSlugCompatible(existingUser, orgSlug); err != nil {
			return InviteUserResult{}, err
		}
	case errors.Is(err, ErrIdentityNotFound):
		// new invitee
	default:
		return InviteUserResult{}, err
	}

	roleSlugs := append([]string(nil), cmd.RoleSlugs...)
	var membership IdentityMembership
	if secret := strings.TrimSpace(cmd.SessionSecret); secret != "" {
		membership, err = a.identity.InviteOrganizationUser(ctx, secret, orgSlug, email, redirectURL, roleSlugs, cmd.IsOrgAdmin)
	} else {
		membership, err = a.identity.InviteOrganizationUserAsAdmin(ctx, orgSlug, email, redirectURL, roleSlugs, cmd.IsOrgAdmin)
	}
	if err != nil {
		return InviteUserResult{}, err
	}
	return InviteUserResult{Membership: membership}, nil
}

// CancelPendingInvite deletes an unconfirmed invite membership as an org admin (session auth).
func (a *Affiliation) CancelPendingInvite(ctx context.Context, sessionSecret, orgSlug, membershipID string) error {
	sessionSecret = strings.TrimSpace(sessionSecret)
	orgSlug = strings.TrimSpace(orgSlug)
	membershipID = strings.TrimSpace(membershipID)
	if sessionSecret == "" || orgSlug == "" || membershipID == "" {
		return ErrAffiliationNotFound
	}
	memberships, err := a.identity.ListOrganizationMemberships(ctx, orgSlug)
	if err != nil {
		return err
	}
	var target *IdentityMembership
	for idx := range memberships {
		if strings.TrimSpace(memberships[idx].ID) != membershipID {
			continue
		}
		target = &memberships[idx]
		break
	}
	if target == nil || target.Confirmed {
		return ErrAffiliationNotFound
	}
	if isPlatformAdminMembership(*target) {
		return ErrAffiliationNotFound
	}
	if err := a.identity.DeleteOrganizationMembership(ctx, sessionSecret, orgSlug, target.ID); err != nil {
		return err
	}
	if userID := strings.TrimSpace(target.UserID); userID != "" {
		if err := a.stripManagedIdentityLabels(ctx, userID); err != nil && !errors.Is(err, ErrIdentityNotFound) {
			return err
		}
	}
	return nil
}
