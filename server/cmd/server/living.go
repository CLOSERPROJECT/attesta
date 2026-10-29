package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
)

func (s *Server) ensureLivingStores() {
	if s.livingDomains == nil {
		s.livingDomains = living.NewMemoryDomainStore()
	}
	if s.livingConfigs == nil {
		s.livingConfigs = living.NewMemoryConfigStore()
		if _, err := living.SeedFixtureConfigStore(s.livingConfigs); err != nil {
			log.Printf("living: seed truck-loop fixture: %v", err)
		}
	}
}

func (s *Server) livingDomainHandler() http.Handler {
	s.ensureLivingStores()
	return &living.Handler{
		Loader: &living.Loader{Domains: s.livingDomains, Configs: s.livingConfigs},
		Gate:   s.livingVerifiedGate,
	}
}

func (s *Server) livingVerifiedGate(w http.ResponseWriter, r *http.Request) (living.OperatorIdentity, bool) {
	user, _, ok := s.requireVerifiedPost(w, r)
	if !ok {
		return living.OperatorIdentity{}, false
	}
	org := ""
	roles := []string(nil)
	userID := ""
	if user != nil {
		org = strings.TrimSpace(user.OrgSlug)
		roles = append([]string(nil), user.RoleSlugs...)
		userID = strings.TrimSpace(user.IdentityUserID)
		if userID == "" {
			userID = strings.TrimSpace(user.Email)
		}
	}
	return living.OperatorIdentity{UserID: userID, Org: org, Roles: roles}, true
}
