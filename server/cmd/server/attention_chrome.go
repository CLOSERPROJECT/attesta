package main

import (
	"context"
	"log"
	"strings"
)

func (s *Server) attentionService() *Attention {
	return NewAttention(s.affiliationService()).WithYourTurnStreams(s)
}

func (s *Server) populateAttention(base *PageBase, user *AccountUser) {
	if base == nil || user == nil || s == nil {
		return
	}
	identity := identityUserForAffiliation(user)
	attention := s.attentionService()

	// Platform-admin Organization-creation Attention does not require an Appwrite user id.
	// PA sessions stay org-creation-only — no stream Attention.
	if user.IsPlatformAdmin {
		items, err := attention.PendingOrganizationCreationRequests(context.Background(), identity)
		if err != nil {
			log.Printf("org-creation attention check failed for %s: %v", user.Email, err)
			return
		}
		base.HasOrgCreationAttention = len(items) > 0
		base.HasAttention = base.HasOrgCreationAttention
		return
	}

	// Invitation Attention lookups need an Appwrite user id; email fallback is not a membership key.
	if strings.TrimSpace(user.IdentityUserID) == "" {
		return
	}
	if s.affiliationService().IsAffiliated(identity) {
		joins, err := attention.PendingJoinRequests(context.Background(), identity)
		if err != nil {
			log.Printf("join attention check failed for %s: %v", user.Email, err)
			return
		}
		base.HasJoinRequestAttention = len(joins) > 0
		streams, streamErr := attention.PendingStreamActions(context.Background(), identity)
		if streamErr != nil {
			log.Printf("stream attention check failed for %s: %v", user.Email, streamErr)
			// Join Attention still applies when the stream catalog is unavailable.
			base.HasAttention = base.HasJoinRequestAttention
			return
		}
		base.HasAttention = base.HasJoinRequestAttention || len(streams) > 0
		return
	}
	has, err := attention.HasAttention(context.Background(), identity)
	if err != nil {
		log.Printf("attention check failed for %s: %v", user.Email, err)
		return
	}
	base.HasAttention = has
}
