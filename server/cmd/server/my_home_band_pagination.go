package main

import (
	"net/http"
	"strconv"
	"strings"
)

const myHomeBandPageSize = 5

const (
	myHomeYourTurnPath     = "/my/home/your-turn"
	myHomeUpcomingPath     = "/my/home/upcoming"
	myHomeJoinRequestsPath = "/my/home/join-requests"
	myHomeOrgCreationPath  = "/my/home/org-creation"
	myHomeYourTurnID       = "my-home-your-turn"
	myHomeUpcomingID       = "my-home-upcoming"
	myHomeJoinRequestsID   = "my-home-join-requests"
	myHomeOrgCreationID    = "my-home-org-creation"
)

// MyHomeYourTurnBandView is the HTMX/page partial for the Your turn band.
type MyHomeYourTurnBandView struct {
	PendingStreamActions []StreamAttentionItem
	Pagination           PaginationView
	TotalPages           int
}

// MyHomeUpcomingBandView is the HTMX/page partial for the Upcoming band.
type MyHomeUpcomingBandView struct {
	UpcomingStreams []StreamUpcomingItem
	Pagination      PaginationView
	TotalPages      int
}

// MyHomeJoinRequestsBandView is the HTMX/page partial for Join-request Attention.
type MyHomeJoinRequestsBandView struct {
	PendingJoinRequests []OrgAdminJoinRequestRow
	Pagination          PaginationView
	TotalPages          int
}

// MyHomeOrgCreationBandView is the HTMX/page partial for org-creation Attention.
type MyHomeOrgCreationBandView struct {
	PendingOrgCreationRequests []PlatformAdminOrgCreationRequestRow
	Pagination                 PaginationView
	TotalPages                 int
}

func normalizeMyHomeBandPage(raw int, totalItems int) int {
	totalPages := 1
	if totalItems > 0 {
		totalPages = (totalItems + myHomeBandPageSize - 1) / myHomeBandPageSize
	}
	if raw < 1 {
		return 1
	}
	if raw > totalPages {
		return totalPages
	}
	return raw
}

func myHomeBandHref(basePath string, page int) string {
	if page < 1 {
		page = 1
	}
	return basePath + "?page=" + strconv.Itoa(page)
}

func paginateMyHomeBand[T any](items []T, page int) (paged []T, currentPage, totalPages int) {
	total := len(items)
	currentPage = normalizeMyHomeBandPage(page, total)
	totalPages = 1
	if total > 0 {
		totalPages = (total + myHomeBandPageSize - 1) / myHomeBandPageSize
	}
	if total == 0 {
		return items, currentPage, totalPages
	}
	start := (currentPage - 1) * myHomeBandPageSize
	end := min(start+myHomeBandPageSize, total)
	return items[start:end], currentPage, totalPages
}

func buildMyHomeBandPagination(basePath, sectionID, ariaLabel string, currentPage, totalPages int) PaginationView {
	if totalPages < 1 {
		totalPages = 1
	}
	links := make([]PaginationLink, 0, totalPages)
	for page := 1; page <= totalPages; page++ {
		links = append(links, PaginationLink{
			Page:      page,
			URL:       myHomeBandHref(basePath, page),
			IsCurrent: page == currentPage,
		})
	}
	previousPage := max(currentPage-1, 1)
	nextPage := min(currentPage+1, totalPages)
	target := "#" + sectionID
	return PaginationView{
		AriaLabel:       ariaLabel,
		Inline:          true,
		Links:           links,
		HasPreviousPage: currentPage > 1,
		HasNextPage:     currentPage < totalPages,
		PreviousURL:     myHomeBandHref(basePath, previousPage),
		NextURL:         myHomeBandHref(basePath, nextPage),
		HxTarget:        target,
		HxSelect:        target,
		PushURL:         false,
	}
}

func buildMyHomeYourTurnBand(items []StreamAttentionItem, page int) MyHomeYourTurnBandView {
	paged, currentPage, totalPages := paginateMyHomeBand(items, page)
	return MyHomeYourTurnBandView{
		PendingStreamActions: paged,
		TotalPages:           totalPages,
		Pagination: buildMyHomeBandPagination(
			myHomeYourTurnPath,
			myHomeYourTurnID,
			"Your turn pagination",
			currentPage,
			totalPages,
		),
	}
}

func buildMyHomeUpcomingBand(items []StreamUpcomingItem, page int) MyHomeUpcomingBandView {
	paged, currentPage, totalPages := paginateMyHomeBand(items, page)
	return MyHomeUpcomingBandView{
		UpcomingStreams: paged,
		TotalPages:      totalPages,
		Pagination: buildMyHomeBandPagination(
			myHomeUpcomingPath,
			myHomeUpcomingID,
			"Upcoming pagination",
			currentPage,
			totalPages,
		),
	}
}

func buildMyHomeJoinRequestsBand(items []OrgAdminJoinRequestRow, page int) MyHomeJoinRequestsBandView {
	paged, currentPage, totalPages := paginateMyHomeBand(items, page)
	return MyHomeJoinRequestsBandView{
		PendingJoinRequests: paged,
		TotalPages:          totalPages,
		Pagination: buildMyHomeBandPagination(
			myHomeJoinRequestsPath,
			myHomeJoinRequestsID,
			"Join requests pagination",
			currentPage,
			totalPages,
		),
	}
}

func buildMyHomeOrgCreationBand(items []PlatformAdminOrgCreationRequestRow, page int) MyHomeOrgCreationBandView {
	paged, currentPage, totalPages := paginateMyHomeBand(items, page)
	return MyHomeOrgCreationBandView{
		PendingOrgCreationRequests: paged,
		TotalPages:                 totalPages,
		Pagination: buildMyHomeBandPagination(
			myHomeOrgCreationPath,
			myHomeOrgCreationID,
			"Organization requests pagination",
			currentPage,
			totalPages,
		),
	}
}

func (s *Server) handleMyHomeYourTurn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isHTMXRequest(r) {
		http.Redirect(w, r, appHomePath, http.StatusFound)
		return
	}
	user, _, ok := s.requireVerifiedPage(w, r)
	if !ok {
		return
	}
	items, err := s.streamAttentionRows(r.Context(), user)
	if err != nil {
		logRequestError(r, err, "load stream attention for your-turn band")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	view := buildMyHomeYourTurnBand(items, page)
	if err := s.tmpl.ExecuteTemplate(w, "my_home_your_turn", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleMyHomeUpcoming(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isHTMXRequest(r) {
		http.Redirect(w, r, appHomePath, http.StatusFound)
		return
	}
	user, _, ok := s.requireVerifiedPage(w, r)
	if !ok {
		return
	}
	items, err := s.upcomingStreamRows(r.Context(), user)
	if err != nil {
		logRequestError(r, err, "load upcoming streams for upcoming band")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	view := buildMyHomeUpcomingBand(items, page)
	if err := s.tmpl.ExecuteTemplate(w, "my_home_upcoming", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleMyHomeJoinRequests(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isHTMXRequest(r) {
		http.Redirect(w, r, appHomePath, http.StatusFound)
		return
	}
	user, _, ok := s.requireVerifiedPage(w, r)
	if !ok {
		return
	}
	items, err := s.joinAttentionRows(r.Context(), user)
	if err != nil {
		logRequestError(r, err, "load join attention for join-requests band")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	view := buildMyHomeJoinRequestsBand(items, page)
	if err := s.tmpl.ExecuteTemplate(w, "my_home_join_requests", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleMyHomeOrgCreation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !isHTMXRequest(r) {
		http.Redirect(w, r, appHomePath, http.StatusFound)
		return
	}
	user, _, ok := s.requireVerifiedPage(w, r)
	if !ok {
		return
	}
	if user == nil || !user.IsPlatformAdmin {
		http.NotFound(w, r)
		return
	}
	items := platformAdminOrgCreationRequestRows(r.Context(), s)
	page := parsePositiveInt(r.URL.Query().Get("page"), 1)
	view := buildMyHomeOrgCreationBand(items, page)
	if err := s.tmpl.ExecuteTemplate(w, "my_home_org_creation", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// myHomeBandRest matches home band fragment paths (optional trailing slash).
func myHomeBandRest(rest string) (band string, ok bool) {
	rest = strings.Trim(rest, "/")
	switch rest {
	case "home/your-turn":
		return "your-turn", true
	case "home/upcoming":
		return "upcoming", true
	case "home/join-requests":
		return "join-requests", true
	case "home/org-creation":
		return "org-creation", true
	default:
		return "", false
	}
}
