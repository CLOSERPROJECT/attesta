package main

import (
	"context"
	"strings"
)

// Attention aggregates Attention items for a user: things that need them to act.
// Waiting (own pending Join / Organization creation request) is not Attention.
type Attention struct {
	affiliation *Affiliation
}

func NewAttention(affiliation *Affiliation) *Attention {
	return &Attention{affiliation: affiliation}
}

// HasAttention reports whether the user has at least one Attention item.
// Sources: open Invitation (unaffiliated invitee); pending Join requests (Org admin
// for their Organization). Waiting never contributes.
func (a *Attention) HasAttention(ctx context.Context, user IdentityUser) (bool, error) {
	if a == nil || a.affiliation == nil {
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
		return len(items) > 0, nil
	}
	invites, err := a.affiliation.ListPendingInvitesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(invites) > 0, nil
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
