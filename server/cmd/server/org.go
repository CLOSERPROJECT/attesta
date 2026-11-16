package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/org"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) ensureOrgStores() {
	if s.orgDomains == nil {
		s.orgDomains = s.newOrgDomainStore()
	}
	if s.orgConfigs == nil {
		s.orgConfigs = s.newOrgConfigStore()
	}
	if s.orgLinks == nil {
		s.orgLinks = s.newOrgLinkStore()
	}
	if s.orgDPPs == nil {
		s.orgDPPs = s.newOrgDPPStore()
	}
}

func (s *Server) orgMongoDB() *mongo.Database {
	if s == nil {
		return nil
	}
	ms, ok := s.store.(*MongoStore)
	if !ok || ms == nil {
		return nil
	}
	return ms.db
}

func (s *Server) newOrgDomainStore() org.DomainStore {
	if db := s.orgMongoDB(); db != nil {
		return org.NewMongoDomainStore(db)
	}
	return org.NewMemoryDomainStore()
}

func (s *Server) newOrgConfigStore() org.ConfigStore {
	if db := s.orgMongoDB(); db != nil {
		return org.NewMongoConfigStore(db)
	}
	return org.NewMemoryConfigStore()
}

func (s *Server) newOrgLinkStore() org.LinkStore {
	if db := s.orgMongoDB(); db != nil {
		return org.NewMongoLinkStore(db)
	}
	return org.NewMemoryLinkStore()
}

func (s *Server) newOrgDPPStore() org.DPPStore {
	if db := s.orgMongoDB(); db != nil {
		return org.NewMongoDPPStore(db)
	}
	return org.NewMemoryDPPStore()
}

func (s *Server) orgProductAuthorizer() org.Authorizer {
	if s.orgAuthorizer != nil {
		return s.orgAuthorizer
	}
	now := s.now
	if now == nil {
		now = time.Now
	}
	url := strings.TrimSpace(os.Getenv("CERBOS_URL"))
	if url == "" {
		url = "http://localhost:3592"
	}
	return org.NewCerbosAuthorizer(url, http.DefaultClient, now)
}

func (s *Server) orgProductHttpPort() org.HttpPort {
	if s.orgHttpPort != nil {
		return s.orgHttpPort
	}
	return org.HttpPortFromEnv()
}

// orgProductArtifacts is the sidecar client used by the browser artifact proxy.
// Independent of HttpPort overrides so tests can plant FakeHttpPort without losing proxy config.
func (s *Server) orgProductArtifacts() *org.SidecarHttpPort {
	base := strings.TrimSpace(os.Getenv("ORG_HTTP_URL"))
	if base == "" {
		return nil
	}
	return org.NewSidecarHttpPort(base, strings.TrimSpace(os.Getenv("ORG_HTTP_TOKEN")), nil)
}

func (s *Server) orgAPI() http.Handler {
	s.ensureOrgStores()
	return &org.API{
		DomainLoader: &org.DomainLoader{
			Domains:   s.orgDomains,
			Configs:   s.orgConfigs,
			LinkStore: s.orgLinks,
		},
		TaskCommitter: &org.TaskCommitter{
			Domains:        s.orgDomains,
			Configs:        s.orgConfigs,
			Authorizer:     s.orgProductAuthorizer(),
			HttpPort:       s.orgProductHttpPort(),
			SubmitBindings: org.SubmitBindingsFromEnv(),
			LinkStore:      s.orgLinks,
			DPPs:      s.orgDPPs,
		},
		Gate:      s.orgVerifiedGate,
		Artifacts: s.orgProductArtifacts(),
	}
}

func (s *Server) orgDomainHandler() http.Handler {
	return s.orgAPI()
}

func (s *Server) orgVerifiedGate(w http.ResponseWriter, r *http.Request) (org.OperatorIdentity, bool) {
	user, _, ok := s.requireVerifiedPost(w, r)
	if !ok {
		return org.OperatorIdentity{}, false
	}
	orgSlug := ""
	roles := []string(nil)
	userID := ""
	displayName := ""
	if user != nil {
		orgSlug = strings.TrimSpace(user.OrgSlug)
		roles = append([]string(nil), user.RoleSlugs...)
		userID = strings.TrimSpace(user.IdentityUserID)
		displayName = strings.TrimSpace(user.Email)
		if userID == "" {
			userID = displayName
		}
		if displayName == "" {
			displayName = userID
		}
	}
	return org.OperatorIdentity{UserID: userID, DisplayName: displayName, Org: orgSlug, Roles: roles}, true
}
