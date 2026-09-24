package main

import (
	"net/http"
	"strings"
)

type OnboardingHubView struct {
	PageBase
	JoinHref       string
	RequestOrgHref string
	HomeHref       string
}

type OnboardingStubView struct {
	PageBase
	Title     string
	Message   string
	BackHref  string
	BackLabel string
}

func (s *Server) handleOnboardingRoutes(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/onboarding"), "/")
	switch {
	case rest == "":
		s.handleOnboardingHub(w, r)
	case rest == "join":
		s.handleOnboardingJoin(w, r)
	case rest == "request-organization":
		s.handleOnboardingRequestOrganization(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) requireUnaffiliatedOnboarding(w http.ResponseWriter, r *http.Request) (*AccountUser, bool) {
	user, _, ok := s.requireAuthenticatedPage(w, r)
	if !ok {
		return nil, false
	}
	if s.affiliationService().IsAffiliated(IdentityUser{OrgSlug: user.OrgSlug}) {
		http.Redirect(w, r, appHomePath, http.StatusSeeOther)
		return nil, false
	}
	return user, true
}

func (s *Server) handleOnboardingHub(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, ok := s.requireUnaffiliatedOnboarding(w, r)
	if !ok {
		return
	}
	view := OnboardingHubView{
		PageBase:       s.pageBaseForUser(user, "onboarding_body", "", ""),
		JoinHref:       onboardingJoinPath(),
		RequestOrgHref: onboardingRequestOrganizationPath(),
		HomeHref:       appHomePath,
	}
	if err := s.tmpl.ExecuteTemplate(w, "onboarding.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleOnboardingJoin(w http.ResponseWriter, r *http.Request) {
	s.renderOnboardingStub(w, r, "Join an organization", "Coming soon. Organization join requests will live here.")
}

func (s *Server) handleOnboardingRequestOrganization(w http.ResponseWriter, r *http.Request) {
	s.renderOnboardingStub(w, r, "Request a new organization", "Coming soon. Organization creation requests will live here.")
}

func (s *Server) renderOnboardingStub(w http.ResponseWriter, r *http.Request, title, message string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, ok := s.requireUnaffiliatedOnboarding(w, r)
	if !ok {
		return
	}
	view := OnboardingStubView{
		PageBase:  s.pageBaseForUser(user, "onboarding_stub_body", "", ""),
		Title:     title,
		Message:   message,
		BackHref:  onboardingPath(),
		BackLabel: "Back to onboarding",
	}
	if err := s.tmpl.ExecuteTemplate(w, "onboarding_stub.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
