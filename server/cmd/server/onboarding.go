package main

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const onboardingJoinSearchLimit = 12

const (
	onboardingPendingKindJoin        = "join"
	onboardingPendingKindOrgCreation = "org_creation"
)

type OnboardingHubView struct {
	PageBase
	JoinHref           string
	RequestOrgHref     string
	Pending            bool
	PendingKind        string
	PendingOrgName     string
	PendingOrgSlug     string
	PendingRoles       string
	PendingName        string
	PendingSlug        string
	PendingCreated     string
	WithdrawActionHref string
	FormError          string
}

type OnboardingRequestOrganizationView struct {
	PageBase
	BackLink  BackLinkView
	FormError string
	NameValue string
}

type OnboardingJoinOrgResult struct {
	Slug       string
	Name       string
	LogoURL    string
	DialogHref string
}

type OnboardingJoinView struct {
	PageBase
	BackLink         BackLinkView
	FormError        string
	SearchQuery      string
	Results          []OnboardingJoinOrgResult
	HasSearched      bool
	SelectedOrgSlug  string
	SelectedOrgName  string
	SelectedOrgRoles []Role
	CurrentPage      int
	TotalPages       int
	PageNumbers      []int
	HasPreviousPage  bool
	HasNextPage      bool
	PreviousPage     int
	NextPage         int
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

func (s *Server) redirectOnboardingIfPending(w http.ResponseWriter, r *http.Request, user *AccountUser) bool {
	hasPending, err := s.affiliationService().HasPendingAffiliationIntent(r.Context(), identityUserForAffiliation(user).ID)
	if err != nil {
		logRequestError(r, err, "failed to check pending affiliation for %s", user.Email)
		http.Error(w, "failed to load onboarding status", http.StatusInternalServerError)
		return true
	}
	if hasPending {
		http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
		return true
	}
	return false
}

func (s *Server) handleOnboardingHub(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUnaffiliatedOnboarding(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.renderOnboardingHub(w, r, user, "")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse onboarding hub form")
			return
		}
		if strings.TrimSpace(r.FormValue("intent")) != "withdraw" {
			http.Error(w, "unsupported intent", http.StatusBadRequest)
			return
		}
		identityUser := identityUserForAffiliation(user)
		aff := s.affiliationService()
		joinPending, err := aff.PendingJoinRequestForUser(r.Context(), identityUser.ID)
		if err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "failed to withdraw", err, "failed to load pending join for withdraw %s", user.Email)
			return
		}
		if joinPending != nil {
			if err := aff.WithdrawPendingJoinRequest(r.Context(), identityUser); err != nil {
				s.renderOnboardingHub(w, r, user, affiliationWithdrawFormError(err))
				return
			}
			http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
			return
		}
		orgPending, err := aff.PendingOrganizationCreationRequestForUser(r.Context(), identityUser.ID)
		if err != nil {
			logAndHTTPError(w, r, http.StatusInternalServerError, "failed to withdraw", err, "failed to load pending org creation for withdraw %s", user.Email)
			return
		}
		if orgPending != nil {
			if err := aff.WithdrawPendingOrganizationCreationRequest(r.Context(), identityUser); err != nil {
				s.renderOnboardingHub(w, r, user, affiliationWithdrawFormError(err))
				return
			}
			http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) renderOnboardingHub(w http.ResponseWriter, r *http.Request, user *AccountUser, formError string) {
	view := OnboardingHubView{
		PageBase:           s.pageBaseForUser(user, "onboarding_body", "", ""),
		JoinHref:           onboardingJoinPath(),
		RequestOrgHref:     onboardingRequestOrganizationPath(),
		WithdrawActionHref: onboardingPath(),
		FormError:          strings.TrimSpace(formError),
	}
	identityUser := identityUserForAffiliation(user)
	aff := s.affiliationService()

	joinPending, err := aff.PendingJoinRequestForUser(r.Context(), identityUser.ID)
	if err != nil {
		logRequestError(r, err, "failed to load pending join request for %s", user.Email)
		http.Error(w, "failed to load onboarding status", http.StatusInternalServerError)
		return
	}
	if joinPending != nil {
		view.Pending = true
		view.PendingKind = onboardingPendingKindJoin
		view.PendingOrgSlug = joinPending.OrgSlug
		view.PendingCreated = humanReadableTraceabilityTime(joinPending.CreatedAt)
		var orgRoles []IdentityRole
		if s.identity != nil {
			if org, orgErr := s.identity.GetOrganizationBySlug(r.Context(), joinPending.OrgSlug); orgErr == nil && org != nil {
				view.PendingOrgName = strings.TrimSpace(org.Name)
				orgRoles = org.Roles
			}
		}
		if view.PendingOrgName == "" {
			view.PendingOrgName = joinPending.OrgSlug
		}
		view.PendingRoles = roleLabelsForSlugs(orgRoles, joinPending.RoleSlugs)
		if err := s.tmpl.ExecuteTemplate(w, "onboarding.html", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	orgPending, err := aff.PendingOrganizationCreationRequestForUser(r.Context(), identityUser.ID)
	if err != nil {
		logRequestError(r, err, "failed to load pending organization creation request for %s", user.Email)
		http.Error(w, "failed to load onboarding status", http.StatusInternalServerError)
		return
	}
	if orgPending != nil {
		view.Pending = true
		view.PendingKind = onboardingPendingKindOrgCreation
		view.PendingName = orgPending.ProposedName
		view.PendingSlug = orgPending.ProposedSlug
		view.PendingCreated = humanReadableTraceabilityTime(orgPending.CreatedAt)
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
	if s.redirectOnboardingIfPending(w, r, user) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		searchQuery := strings.TrimSpace(r.URL.Query().Get("q"))
		page := parsePositiveInt(r.URL.Query().Get("page"), 1)
		if isHTMXRequest(r) && htmxTargetID(r) == "join-org-dialog-body" {
			s.renderOnboardingJoinDialog(w, r, user, "", strings.TrimSpace(r.URL.Query().Get("org")), searchQuery, page)
			return
		}
		s.renderOnboardingJoin(w, r, user, "", searchQuery, "", page)
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			logAndHTTPError(w, r, http.StatusBadRequest, "invalid form", err, "failed to parse join request form")
			return
		}
		orgSlug := strings.TrimSpace(r.FormValue("org_slug"))
		roles := requestedRoleSlugs(r.Form)
		searchQuery := strings.TrimSpace(r.FormValue("q"))
		page := parsePositiveInt(r.FormValue("page"), 1)
		_, err := s.affiliationService().SubmitJoinRequest(r.Context(), identityUserForAffiliation(user), orgSlug, roles)
		if err != nil {
			s.renderOnboardingJoin(w, r, user, affiliationJoinRequestFormError(err), searchQuery, orgSlug, page)
			return
		}
		http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleOnboardingRequestOrganization(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUnaffiliatedOnboarding(w, r)
	if !ok {
		return
	}
	if s.redirectOnboardingIfPending(w, r, user) {
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
		http.Redirect(w, r, onboardingPath(), http.StatusSeeOther)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) renderOnboardingJoin(w http.ResponseWriter, r *http.Request, user *AccountUser, formError, searchQuery, selectedOrgSlug string, requestedPage int) {
	view, ok := s.buildOnboardingJoinView(w, r, user, formError, searchQuery, selectedOrgSlug, requestedPage)
	if !ok {
		return
	}
	templateName := "onboarding_join.html"
	if isHTMXRequest(r) && htmxTargetID(r) == "onboarding-join-results" {
		templateName = "onboarding_join_results"
	}
	if err := s.tmpl.ExecuteTemplate(w, templateName, view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderOnboardingJoinDialog(w http.ResponseWriter, r *http.Request, user *AccountUser, formError, selectedOrgSlug, searchQuery string, page int) {
	view, ok := s.buildOnboardingJoinDialogView(w, r, user, formError, selectedOrgSlug, searchQuery, page)
	if !ok {
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "onboarding_join_dialog", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) buildOnboardingJoinDialogView(w http.ResponseWriter, r *http.Request, user *AccountUser, formError, selectedOrgSlug, searchQuery string, page int) (OnboardingJoinView, bool) {
	view := OnboardingJoinView{
		FormError:       strings.TrimSpace(formError),
		SearchQuery:     strings.TrimSpace(searchQuery),
		SelectedOrgSlug: strings.TrimSpace(selectedOrgSlug),
		CurrentPage:     page,
	}
	if view.CurrentPage < 1 {
		view.CurrentPage = 1
	}
	if view.SelectedOrgSlug == "" || s.identity == nil {
		if view.FormError == "" {
			view.FormError = "organization not found"
		}
		view.SelectedOrgSlug = ""
		return view, true
	}
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
		logRequestError(r, err, "failed to load organization %s for join onboarding dialog", view.SelectedOrgSlug)
		http.Error(w, "failed to load organization", http.StatusInternalServerError)
		return OnboardingJoinView{}, false
	}
	return view, true
}

func (s *Server) buildOnboardingJoinView(w http.ResponseWriter, r *http.Request, user *AccountUser, formError, searchQuery, selectedOrgSlug string, requestedPage int) (OnboardingJoinView, bool) {
	view := OnboardingJoinView{
		PageBase:        s.pageBaseForUser(user, "onboarding_join_body", "", ""),
		BackLink:        BackLinkView{Href: onboardingPath(), Label: "Get started"},
		FormError:       strings.TrimSpace(formError),
		SearchQuery:     strings.TrimSpace(searchQuery),
		SelectedOrgSlug: strings.TrimSpace(selectedOrgSlug),
		CurrentPage:     1,
		TotalPages:      1,
		PreviousPage:    1,
		NextPage:        1,
	}

	// Empty search browses the full catalog (paginated); non-empty q filters by name/slug.
	if s.identity != nil {
		view.HasSearched = true
		if requestedPage < 1 {
			requestedPage = 1
		}
		offset := (requestedPage - 1) * onboardingJoinSearchLimit
		orgPage, err := s.identity.ListOrganizationsPage(r.Context(), IdentityOrgListOptions{
			Search: view.SearchQuery,
			Limit:  onboardingJoinSearchLimit,
			Offset: offset,
		})
		if err != nil {
			logRequestError(r, err, "failed to search organizations for join onboarding")
			http.Error(w, "failed to search organizations", http.StatusInternalServerError)
			return OnboardingJoinView{}, false
		}

		currentPage := normalizeOnboardingJoinPage(requestedPage, orgPage.Total)
		if currentPage != requestedPage {
			offset = (currentPage - 1) * onboardingJoinSearchLimit
			orgPage, err = s.identity.ListOrganizationsPage(r.Context(), IdentityOrgListOptions{
				Search: view.SearchQuery,
				Limit:  onboardingJoinSearchLimit,
				Offset: offset,
			})
			if err != nil {
				logRequestError(r, err, "failed to search organizations for join onboarding")
				http.Error(w, "failed to search organizations", http.StatusInternalServerError)
				return OnboardingJoinView{}, false
			}
		}

		totalPages := 1
		if orgPage.Total > 0 {
			totalPages = (orgPage.Total + onboardingJoinSearchLimit - 1) / onboardingJoinSearchLimit
		}
		pageNumbers := make([]int, 0, totalPages)
		for page := 1; page <= totalPages; page++ {
			pageNumbers = append(pageNumbers, page)
		}

		view.CurrentPage = currentPage
		view.TotalPages = totalPages
		view.PageNumbers = pageNumbers
		view.HasPreviousPage = currentPage > 1
		view.HasNextPage = currentPage < totalPages
		view.PreviousPage = max(currentPage-1, 1)
		view.NextPage = min(currentPage+1, totalPages)
		view.Results = make([]OnboardingJoinOrgResult, 0, len(orgPage.Organizations))
		for _, org := range orgPage.Organizations {
			slug := strings.TrimSpace(org.Slug)
			result := OnboardingJoinOrgResult{
				Slug:       slug,
				Name:       strings.TrimSpace(org.Name),
				DialogHref: onboardingJoinDialogHref(view.SearchQuery, slug, currentPage),
			}
			if slug != "" && strings.TrimSpace(org.LogoFileID) != "" {
				result.LogoURL = "/organization/logo/" + url.PathEscape(slug)
			}
			view.Results = append(view.Results, result)
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
			return OnboardingJoinView{}, false
		}
	}

	return view, true
}

func normalizeOnboardingJoinPage(raw int, totalItems int) int {
	totalPages := 1
	if totalItems > 0 {
		totalPages = (totalItems + onboardingJoinSearchLimit - 1) / onboardingJoinSearchLimit
	}
	if raw < 1 {
		return 1
	}
	if raw > totalPages {
		return totalPages
	}
	return raw
}

func (s *Server) renderOnboardingRequestOrganization(w http.ResponseWriter, r *http.Request, user *AccountUser, formError, nameValue string) {
	view := OnboardingRequestOrganizationView{
		PageBase:  s.pageBaseForUser(user, "onboarding_request_organization_body", "", ""),
		BackLink:  BackLinkView{Href: onboardingPath(), Label: "Get started"},
		FormError: strings.TrimSpace(formError),
		NameValue: strings.TrimSpace(nameValue),
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

func onboardingJoinDialogHref(searchQuery, orgSlug string, page int) string {
	values := url.Values{}
	if slug := strings.TrimSpace(orgSlug); slug != "" {
		values.Set("org", slug)
	}
	if q := strings.TrimSpace(searchQuery); q != "" {
		values.Set("q", q)
	}
	if page > 1 {
		values.Set("page", strconv.Itoa(page))
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
		InvalidRoles:      "select one or more organization roles",
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

func affiliationWithdrawFormError(err error) string {
	return mapAffiliationFormError(err, affiliationFormErrorMessages{
		NotFound:   "no pending request to undo",
		NotPending: "request is not pending",
		Default:    "failed to undo request",
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

func roleLabelsForSlugs(roles []IdentityRole, slugs []string) string {
	bySlug := make(map[string]string, len(roles))
	for _, role := range roles {
		slug := strings.TrimSpace(role.Slug)
		if slug == "" {
			continue
		}
		name := strings.TrimSpace(role.Name)
		if name == "" {
			name = slug
		}
		bySlug[slug] = name
	}
	labels := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		slug = strings.TrimSpace(slug)
		if slug == "" {
			continue
		}
		if label, ok := bySlug[slug]; ok {
			labels = append(labels, label)
		} else {
			labels = append(labels, slug)
		}
	}
	return strings.Join(labels, ", ")
}

func isAppHomePath(path string) bool {
	path = strings.TrimSpace(path)
	return path == appHomePath || path == appHomePath+"/"
}
