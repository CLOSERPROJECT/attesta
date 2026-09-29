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
// v1 sources: open Invitation for an unaffiliated invitee. Affiliated users
// return false until later slices add Join-request / org-creation / stream sources.
// Waiting never contributes.
func (a *Attention) HasAttention(ctx context.Context, user IdentityUser) (bool, error) {
	if a == nil || a.affiliation == nil {
		return false, nil
	}
	userID := strings.TrimSpace(user.ID)
	if userID == "" {
		return false, nil
	}
	if a.affiliation.IsAffiliated(user) {
		return false, nil
	}
	invites, err := a.affiliation.ListPendingInvitesForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	return len(invites) > 0, nil
}
