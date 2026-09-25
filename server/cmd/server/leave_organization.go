package main

import (
	"errors"
	"net/http"
	"strings"
)

func (s *Server) handleLeaveOrganization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, session, ok := s.requireAuthenticatedPost(w, r)
	if !ok {
		return
	}
	sessionSecret := ""
	if session != nil {
		sessionSecret = strings.TrimSpace(session.Secret)
	}
	if sessionSecret == "" {
		var err error
		sessionSecret, err = sessionSecretFromRequest(r)
		if err != nil {
			logAndHTTPError(w, r, http.StatusUnauthorized, "unauthorized", err, "failed to read session secret for leave organization")
			return
		}
	}

	err := s.affiliationService().LeaveOrganization(r.Context(), sessionSecret, identityUserForAffiliation(user))
	switch {
	case err == nil:
		http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
	case errors.Is(err, ErrAffiliationSoleOrgAdmin):
		redirectHomeWithMessage(w, r, "error", reasonSoleOrgAdminLeave)
	case errors.Is(err, ErrAffiliationNotAffiliated):
		http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
	default:
		logRequestError(r, err, "failed to leave organization for user %s", identityUserForAffiliation(user).ID)
		redirectHomeWithMessage(w, r, "error", "failed to leave organization")
	}
}
