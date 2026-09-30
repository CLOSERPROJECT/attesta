package main

import (
	"context"
	"strings"
)

// StreamAttentionItem is an active Stream instance whose next actionable Substep
// is the current user's to complete (your-turn). Distinct from Upcoming / Waiting.
type StreamAttentionItem struct {
	ProcessID    string
	WorkflowKey  string
	WorkflowName string
	InstanceName string
	SubstepTitle string
	Href         string
}

// yourTurnStreamSource lists your-turn Stream Attention items for an affiliated user.
// Implementations must reuse existing actionable-Substep helpers (not invent eligibility).
type yourTurnStreamSource interface {
	ListYourTurn(ctx context.Context, user *AccountUser) ([]StreamAttentionItem, error)
}

// Attention aggregates Attention items for a user: things that need them to act.
// Waiting (own pending Join / Organization creation request) is not Attention.
type Attention struct {
	affiliation *Affiliation
	streams     yourTurnStreamSource
}

func NewAttention(affiliation *Affiliation) *Attention {
	return &Attention{affiliation: affiliation}
}

// WithYourTurnStreams attaches the Stream your-turn source used for affiliated Attention.
func (a *Attention) WithYourTurnStreams(source yourTurnStreamSource) *Attention {
	if a == nil {
		return nil
	}
	a.streams = source
	return a
}

// HasAttention reports whether the user has at least one Attention item.
// Sources: open Invitation (unaffiliated invitee); pending Join requests (Org admin
// for their Organization); your-turn Stream instances (affiliated Members / Org admins);
// pending Organization creation requests (platform admin).
// Waiting never contributes. Platform admins do not get stream Attention here.
func (a *Attention) HasAttention(ctx context.Context, user IdentityUser) (bool, error) {
	if a == nil || a.affiliation == nil {
		return false, nil
	}
	orgItems, err := a.PendingOrganizationCreationRequests(ctx, user)
	if err != nil {
		return false, err
	}
	if len(orgItems) > 0 {
		return true, nil
	}
	if user.IsPlatformAdmin {
		return false, nil
	}
	userID := strings.TrimSpace(user.ID)
	if userID == "" {
		return false, nil
	}
	if a.affiliation.IsAffiliated(user) {
		items, err := a.PendingJoinRequests(ctx, user)
		if err != nil {
			return false, err
		}
		if len(items) > 0 {
			return true, nil
		}
		streams, err := a.PendingStreamActions(ctx, user)
		if err != nil {
			return false, err
		}
		return len(streams) > 0, nil
	}
	invites, err := a.affiliation.ListPendingInvitesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(invites) > 0, nil
}

// PendingOrganizationCreationRequests returns Organization-creation Attention
// items for a platform admin. Empty when the user is not a platform admin or
// the queue is empty. Waiting never appears here.
func (a *Attention) PendingOrganizationCreationRequests(ctx context.Context, user IdentityUser) ([]OrganizationCreationRequest, error) {
	if a == nil || a.affiliation == nil {
		return nil, nil
	}
	if !user.IsPlatformAdmin {
		return nil, nil
	}
	return a.affiliation.ListPendingOrganizationCreationRequests(ctx)
}

// PendingJoinRequests returns Join-request Attention items for an Org admin's
// Organization. Empty when the user lacks Org admin standing, has no OrgSlug,
// or the queue is empty. Waiting never appears here.
func (a *Attention) PendingJoinRequests(ctx context.Context, user IdentityUser) ([]JoinRequest, error) {
	if a == nil || a.affiliation == nil {
		return nil, nil
	}
	if !user.IsOrgAdmin {
		return nil, nil
	}
	orgSlug := strings.TrimSpace(user.OrgSlug)
	if orgSlug == "" {
		return nil, nil
	}
	return a.affiliation.ListPendingJoinRequests(ctx, orgSlug)
}

// PendingStreamActions returns your-turn Stream Attention items for affiliated
// Members and Org admins. Empty for platform admins, unaffiliated users, or when
// no active instance has an actionable Substep for the user. Upcoming is excluded.
func (a *Attention) PendingStreamActions(ctx context.Context, user IdentityUser) ([]StreamAttentionItem, error) {
	if a == nil || a.streams == nil || a.affiliation == nil {
		return nil, nil
	}
	if user.IsPlatformAdmin || !a.affiliation.IsAffiliated(user) {
		return nil, nil
	}
	return a.streams.ListYourTurn(ctx, accountUserForStreamAttention(user))
}

func accountUserForStreamAttention(user IdentityUser) *AccountUser {
	roleSlugs := decodeIdentityRoleLabels(user.Labels)
	if user.IsOrgAdmin {
		roleSlugs = canonifyRoleSlugs(append(roleSlugs, "org-admin"))
	}
	return &AccountUser{
		IdentityUserID:  strings.TrimSpace(user.ID),
		Email:           strings.TrimSpace(user.Email),
		OrgSlug:         strings.TrimSpace(user.OrgSlug),
		RoleSlugs:       roleSlugs,
		IsPlatformAdmin: user.IsPlatformAdmin,
		Status:          firstNonEmpty(strings.TrimSpace(user.Status), "active"),
	}
}
