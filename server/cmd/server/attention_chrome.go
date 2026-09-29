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
	has, err := s.attentionService().HasAttention(context.Background(), identityUserForAffiliation(user))
	if err != nil {
		log.Printf("attention check failed for %s: %v", user.Email, err)
		return
	}
	base.HasAttention = has
}
