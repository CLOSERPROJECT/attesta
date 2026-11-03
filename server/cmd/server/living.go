package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
	"go.mongodb.org/mongo-driver/mongo"
)

func (s *Server) ensureLivingStores() {
	if s.livingDomains == nil {
		s.livingDomains = s.newLivingDomainStore()
	}
	if s.livingConfigs == nil {
		s.livingConfigs = s.newLivingConfigStore()
	}
	if s.livingLinks == nil {
		s.livingLinks = s.newLivingLinkStore()
	}
}

func (s *Server) livingMongoDB() *mongo.Database {
	if s == nil {
		return nil
	}
	ms, ok := s.store.(*MongoStore)
	if !ok || ms == nil {
		return nil
	}
	return ms.db
}

func (s *Server) newLivingDomainStore() living.DomainStore {
	if db := s.livingMongoDB(); db != nil {
		return living.NewMongoDomainStore(db)
	}
	return living.NewMemoryDomainStore()
}

func (s *Server) newLivingConfigStore() living.ConfigStore {
	if db := s.livingMongoDB(); db != nil {
		return living.NewMongoConfigStore(db)
	}
	return living.NewMemoryConfigStore()
}

func (s *Server) newLivingLinkStore() living.LinkStore {
	if db := s.livingMongoDB(); db != nil {
		return living.NewMongoLinkStore(db)
	}
	return living.NewMemoryLinkStore()
}

func (s *Server) livingProductAuthorizer() living.Authorizer {
	if s.livingAuthorizer != nil {
		return s.livingAuthorizer
	}
	now := s.now
	if now == nil {
		now = time.Now
	}
	url := strings.TrimSpace(os.Getenv("CERBOS_URL"))
	if url == "" {
		url = "http://localhost:3592"
	}
	return living.NewCerbosAuthorizer(url, http.DefaultClient, now)
}

func (s *Server) livingProductHttpPort() living.HttpPort {
	if s.livingHttpPort != nil {
		return s.livingHttpPort
	}
	return living.HttpPortFromEnv()
}

// livingProductArtifacts is the sidecar client used by the browser artifact proxy.
// Independent of HttpPort overrides so tests can plant FakeHttpPort without losing proxy config.
func (s *Server) livingProductArtifacts() *living.SidecarHttpPort {
	base := strings.TrimSpace(os.Getenv("LIVING_HTTP_URL"))
	if base == "" {
		return nil
	}
	return living.NewSidecarHttpPort(base, strings.TrimSpace(os.Getenv("LIVING_HTTP_TOKEN")), nil)
}

func (s *Server) livingHandler() http.Handler {
	s.ensureLivingStores()
	return &living.Handler{
		Loader: &living.Loader{
			Domains:   s.livingDomains,
			Configs:   s.livingConfigs,
			LinkStore: s.livingLinks,
		},
		Committer: &living.Committer{
			Domains:    s.livingDomains,
			Configs:    s.livingConfigs,
			Authorizer: s.livingProductAuthorizer(),
			HttpPort:   s.livingProductHttpPort(),
			LinkStore:  s.livingLinks,
		},
		Gate:      s.livingVerifiedGate,
		Artifacts: s.livingProductArtifacts(),
	}
}

func (s *Server) livingDomainHandler() http.Handler {
	return s.livingHandler()
}

func (s *Server) livingVerifiedGate(w http.ResponseWriter, r *http.Request) (living.OperatorIdentity, bool) {
	user, _, ok := s.requireVerifiedPost(w, r)
	if !ok {
		return living.OperatorIdentity{}, false
	}
	org := ""
	roles := []string(nil)
	userID := ""
	displayName := ""
	if user != nil {
		org = strings.TrimSpace(user.OrgSlug)
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
	return living.OperatorIdentity{UserID: userID, DisplayName: displayName, Org: org, Roles: roles}, true
}
