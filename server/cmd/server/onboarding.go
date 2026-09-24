package main

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

const onboardingJoinSearchLimit = 12

type OnboardingHubView struct {
	PageBase
	JoinHref       string
	RequestOrgHref string
	HomeHref       string
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

type OnboardingJoinOrgResult struct {
	Slug       string
	Name       string
	SelectHref string
}

type OnboardingJoinView struct {
	PageBase
	BackHref         string
	FormError        string
	SearchQuery      string
	Results          []OnboardingJoinOrgResult
	HasSearched      bool
	SelectedOrgSlug  string
	SelectedOrgName  string
	SelectedOrgRoles []Role
	Pending          bool
	PendingOrgSlug   string
	PendingOrgName   string
	PendingRoles     string
	PendingCreated   string
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
	if s.affiliationService().IsAffiliated(identityUserForAffiliation(user)) {
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
	user, ok := s.requireUnaffiliatedOnboarding(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.renderOnboardingJoin(w, r, user, "", strings.TrimSpace(r.URL.Query().Get("q")), strings.TrimSpace(r.URL.Query().Get("org")))
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse join request form")
			return
		}
		orgSlug := strings.TrimSpace(r.FormValue("org_slug"))
		roles := requestedRoleSlugs(r.Form)
		searchQuery := strings.TrimSpace(r.FormValue("q"))
		_, err := s.affiliationService().SubmitJoinRequest(r.Context(), identityUserForAffiliation(user), orgSlug, roles)
		if err != nil {
			s.renderOnboardingJoin(w, r, user, affiliationJoinRequestFormError(err), searchQuery, orgSlug)
			return
		}
		http.Redirect(w, r, onboardingJoinPath(), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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

func (s *Server) renderOnboardingJoin(w http.ResponseWriter, r *http.Request, user *AccountUser, formError, searchQuery, selectedOrgSlug string) {
	view := OnboardingJoinView{
		PageBase:        s.pageBaseForUser(user, "onboarding_join_body", "", ""),
		BackHref:        onboardingPath(),
		FormError:       strings.TrimSpace(formError),
		SearchQuery:     strings.TrimSpace(searchQuery),
		SelectedOrgSlug: strings.TrimSpace(selectedOrgSlug),
	}
	if user != nil {
		pending, err := s.affiliationService().PendingJoinRequestForUser(r.Context(), identityUserForAffiliation(user).ID)
		if err != nil {
			logRequestError(r, err, "failed to load pending join request for %s", user.Email)
			http.Error(w, "failed to load join request", http.StatusInternalServerError)
			return
		}
		if pending != nil {
			view.Pending = true
			view.PendingOrgSlug = pending.OrgSlug
			view.PendingRoles = strings.Join(pending.RoleSlugs, ", ")
			view.PendingCreated = humanReadableTraceabilityTime(pending.CreatedAt)
			view.FormError = ""
			view.SearchQuery = ""
			view.SelectedOrgSlug = ""
			if s.identity != nil {
				if org, orgErr := s.identity.GetOrganizationBySlug(r.Context(), pending.OrgSlug); orgErr == nil && org != nil {
					view.PendingOrgName = org.Name
				}
			}
			if view.PendingOrgName == "" {
				view.PendingOrgName = pending.OrgSlug
			}
			if err := s.tmpl.ExecuteTemplate(w, "onboarding_join.html", view); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	if view.SearchQuery != "" && s.identity != nil {
		view.HasSearched = true
		page, err := s.identity.ListOrganizationsPage(r.Context(), IdentityOrgListOptions{
			Search: view.SearchQuery,
			Limit:  onboardingJoinSearchLimit,
			Offset: 0,
		})
		if err != nil {
			logRequestError(r, err, "failed to search organizations for join onboarding")
			http.Error(w, "failed to search organizations", http.StatusInternalServerError)
			return
		}
		view.Results = make([]OnboardingJoinOrgResult, 0, len(page.Organizations))
		for _, org := range page.Organizations {
			slug := strings.TrimSpace(org.Slug)
			view.Results = append(view.Results, OnboardingJoinOrgResult{
				Slug:       slug,
				Name:       strings.TrimSpace(org.Name),
				SelectHref: onboardingJoinSelectHref(view.SearchQuery, slug),
			})
		}
	}

	if view.SelectedOrgSlug != "" && s.identity != nil {
		org, err := s.identity.GetOrganizationBySlug(r.Context(), view.SelectedOrgSlug)
		switch {
		case err == nil && org != nil:
			view.SelectedOrgSlug = strings.TrimSpace(org.Slug)
			view.SelectedOrgName = strings.TrimSpace(org.Name)
			view.SelectedOrgRoles = s.affiliationService().RequestableJoinRoles(*org)
		case errors.Is(err, ErrIdentityNotFound), err == nil && org == nil:
			if view.FormError == "" {
				view.FormError = "organization not found"
			}
			view.SelectedOrgSlug = ""
		default:
			logRequestError(r, err, "failed to load organization %s for join onboarding", view.SelectedOrgSlug)
			http.Error(w, "failed to load organization", http.StatusInternalServerError)
			return
		}
	}

	if err := s.tmpl.ExecuteTemplate(w, "onboarding_join.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func onboardingJoinSelectHref(searchQuery, orgSlug string) string {
	values := url.Values{}
	if q := strings.TrimSpace(searchQuery); q != "" {
		values.Set("q", q)
	}
	if slug := strings.TrimSpace(orgSlug); slug != "" {
		values.Set("org", slug)
	}
	href := onboardingJoinPath()
	if encoded := values.Encode(); encoded != "" {
		href += "?" + encoded
	}
	return href
}

type affiliationFormErrorMessages struct {
	AlreadyAffiliated      string
	PendingExists          string
	NotFound               string
	NotPending             string
	InvalidRoles           string
	InvalidName            string
	OrganizationSlugExists string
	Default                string
}

func mapAffiliationFormError(err error, messages affiliationFormErrorMessages) string {
	switch {
	case err == nil:
		return ""
	case messages.AlreadyAffiliated != "" && errors.Is(err, ErrAffiliationAlreadyAffiliated):
		return messages.AlreadyAffiliated
	case messages.PendingExists != "" && errors.Is(err, ErrAffiliationPendingExists):
		return messages.PendingExists
	case messages.NotFound != "" && errors.Is(err, ErrAffiliationNotFound):
		return messages.NotFound
	case messages.NotPending != "" && errors.Is(err, ErrAffiliationNotPending):
		return messages.NotPending
	case messages.InvalidRoles != "" && errors.Is(err, ErrAffiliationInvalidRoles):
		return messages.InvalidRoles
	case messages.InvalidName != "" && errors.Is(err, ErrAffiliationInvalidName):
		return messages.InvalidName
	case messages.OrganizationSlugExists != "" && errors.Is(err, ErrAffiliationOrganizationSlugExists):
		return messages.OrganizationSlugExists
	default:
		return messages.Default
	}
}

func affiliationJoinRequestFormError(err error) string {
	return mapAffiliationFormError(err, affiliationFormErrorMessages{
		AlreadyAffiliated: "you already belong to an organization",
		PendingExists:     "you already have a pending affiliation request",
		NotFound:          "organization not found",
		NotPending:        "join request is not pending",
		InvalidRoles:      "select one or more roles from the organization catalog",
		Default:           "failed to process join request",
	})
}

func affiliationOrganizationCreationFormError(err error) string {
	return mapAffiliationFormError(err, affiliationFormErrorMessages{
		AlreadyAffiliated:      "you already belong to an organization",
		PendingExists:          "you already have a pending affiliation request",
		NotFound:               "organization creation request not found",
		NotPending:             "organization creation request is not pending",
		InvalidName:            "organization name is required",
		OrganizationSlugExists: "organization slug already exists",
		Default:                "failed to process organization creation request",
	})
}

func affiliationJoinDecideFormError(err error) string {
	return mapAffiliationFormError(err, affiliationFormErrorMessages{
		AlreadyAffiliated: "requester already belongs to an organization",
		NotFound:          "join request not found",
		NotPending:        "join request is not pending",
		InvalidRoles:      "requested roles are no longer valid",
		Default:           "failed to process join request",
	})
}
