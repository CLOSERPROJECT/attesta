package main

import (
	"errors"
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

type OnboardingRequestOrganizationView struct {
	PageBase
	BackHref       string
	FormError      string
	NameValue      string
	Pending        bool
	PendingName    string
	PendingSlug    string
	PendingCreated string
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
	user, ok := s.requireUnaffiliatedOnboarding(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.renderOnboardingRequestOrganization(w, r, user, "", "")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse organization creation request form")
			return
		}
		name := strings.TrimSpace(r.FormValue("name"))
		_, err := s.affiliationService().SubmitOrganizationCreationRequest(r.Context(), identityUserForAffiliation(user), name)
		if err != nil {
			s.renderOnboardingRequestOrganization(w, r, user, affiliationOrganizationCreationFormError(err), name)
			return
		}
		http.Redirect(w, r, onboardingRequestOrganizationPath(), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) renderOnboardingRequestOrganization(w http.ResponseWriter, r *http.Request, user *AccountUser, formError, nameValue string) {
	view := OnboardingRequestOrganizationView{
		PageBase:  s.pageBaseForUser(user, "onboarding_request_organization_body", "", ""),
		BackHref:  onboardingPath(),
		FormError: strings.TrimSpace(formError),
		NameValue: strings.TrimSpace(nameValue),
	}
	if user != nil {
		pending, err := s.affiliationService().PendingOrganizationCreationRequestForUser(r.Context(), identityUserForAffiliation(user).ID)
		if err != nil {
			logRequestError(r, err, "failed to load pending organization creation request for %s", user.Email)
			http.Error(w, "failed to load organization creation request", http.StatusInternalServerError)
			return
		}
		if pending != nil {
			view.Pending = true
			view.PendingName = pending.ProposedName
			view.PendingSlug = pending.ProposedSlug
			view.PendingCreated = humanReadableTraceabilityTime(pending.CreatedAt)
			view.FormError = ""
			view.NameValue = ""
		}
	}
	if err := s.tmpl.ExecuteTemplate(w, "onboarding_request_organization.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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

func identityUserForAffiliation(user *AccountUser) IdentityUser {
	if user == nil {
		return IdentityUser{}
	}
	return IdentityUser{
		ID:      firstNonEmpty(strings.TrimSpace(user.IdentityUserID), strings.TrimSpace(user.Email)),
		Email:   strings.TrimSpace(user.Email),
		OrgSlug: strings.TrimSpace(user.OrgSlug),
	}
}

func affiliationOrganizationCreationFormError(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrAffiliationInvalidName):
		return "organization name is required"
	case errors.Is(err, ErrAffiliationAlreadyAffiliated):
		return "you already belong to an organization"
	case errors.Is(err, ErrAffiliationPendingExists):
		return "you already have a pending affiliation request"
	case errors.Is(err, ErrAffiliationOrganizationSlugExists):
		return "organization slug already exists"
	case errors.Is(err, ErrAffiliationNotFound):
		return "organization creation request not found"
	case errors.Is(err, ErrAffiliationNotPending):
		return "organization creation request is not pending"
	default:
		return "failed to process organization creation request"
	}
}
