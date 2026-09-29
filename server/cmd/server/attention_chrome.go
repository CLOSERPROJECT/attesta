package main

import (
	"context"
	"log"
	"strings"
)

func (s *Server) attentionService() *Attention {
	return NewAttention(s.affiliationService())
}

func (s *Server) populateAttention(base *PageBase, user *AccountUser) {
	if base == nil || user == nil || s == nil {
		return
	}
	// Invitation Attention lookups need an Appwrite user id; email fallback is not a membership key.
	if strings.TrimSpace(user.IdentityUserID) == "" {
		return
	}
	identity := identityUserForAffiliation(user)
	attention := s.attentionService()
	if s.affiliationService().IsAffiliated(identity) {
		joins, err := attention.PendingJoinRequests(context.Background(), identity)
		if err != nil {
			log.Printf("join attention check failed for %s: %v", user.Email, err)
			return
		}
		base.HasJoinRequestAttention = len(joins) > 0
		// Affiliated Attention sources today: Join requests only (stream Attention lands later).
		base.HasAttention = base.HasJoinRequestAttention
		return
	}
	has, err := attention.HasAttention(context.Background(), identity)
	if err != nil {
		log.Printf("attention check failed for %s: %v", user.Email, err)
		return
	}
	base.HasAttention = has
}
