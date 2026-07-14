package main

import (
	"context"
	"strings"
)

func viewerCanSeeDoneByEmail(def WorkflowDef, viewer Actor) bool {
	orgSlug := strings.TrimSpace(viewer.OrgSlug)
	if orgSlug == "" {
		return false
	}
	for _, step := range def.Steps {
		if strings.TrimSpace(step.OrganizationSlug) == orgSlug {
			return true
		}
	}
	return false
}

type userIdentityView struct {
	email      string
	fallbackID string
}

func (s *Server) lookupUserIdentityByActorID(ctx context.Context, actorID string, cache map[string]userIdentityView) (userIdentityView, bool) {
	id := strings.TrimSpace(actorID)
	if id == "" {
		return userIdentityView{}, false
	}
	if identity, ok := cache[id]; ok {
		return identity, strings.TrimSpace(identity.email) != "" || strings.TrimSpace(identity.fallbackID) != ""
	}
	if appwriteUserID, ok := parseAppwriteActorID(id); ok {
		if s.identity == nil {
			cache[id] = userIdentityView{}
			return userIdentityView{}, false
		}
		user, err := s.identity.GetUserByID(ctx, appwriteUserID)
		if err != nil {
			cache[id] = userIdentityView{}
			return userIdentityView{}, false
		}
		identity := userIdentityView{
			email:      strings.TrimSpace(user.Email),
			fallbackID: appwriteActorID(firstNonEmpty(user.ID, appwriteUserID)),
		}
		cache[id] = identity
		return identity, identity.email != "" || identity.fallbackID != ""
	}
	cache[id] = userIdentityView{}
	return userIdentityView{}, false
}
