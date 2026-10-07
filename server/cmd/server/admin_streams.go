package main

import "net/http"

const (
	platformStreamsCatalogTargetID = "platform-admin-stream-catalog"
	platformStreamsResultsTargetID = "platform-admin-stream-results"
)

func (s *Server) handleAdminStreams(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requirePlatformAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.renderStreamsCatalog(w, r, admin)
}

func (s *Server) renderStreamsCatalog(w http.ResponseWriter, r *http.Request, admin *AccountUser) {
	categories, groups, err := s.buildMyHomeCatalogData(r.Context(), admin)
	if err != nil {
		logAndHTTPError(w, r, http.StatusInternalServerError, "failed to load platform stream catalog", err, "failed to load platform admin streams")
		return
	}
	categorySlug, subCategorySlug := streamTaxonomySelectionFromRequest(r, platformStreamsCatalogTargetID, platformStreamsResultsTargetID)
	streamCatalog := buildStreamTaxonomyCatalogView(streamTaxonomyCatalogSpec{
		ID:                      platformStreamsCatalogTargetID,
		FormID:                  "platform-admin-stream-filter",
		Action:                  "/admin/streams",
		ResultsID:               platformStreamsResultsTargetID,
		IncludeAllCategories:    true,
		AlwaysShowUncategorized: true,
		EmptyHint:               "Create a stream to populate the platform catalog.",
	}, categories, streamTaxonomyCandidatesFromGroups(groups), categorySlug, subCategorySlug)
	showCreateStream, authErr := s.canViewFormataBuilder(r.Context(), admin)
	if authErr != nil {
		logRequestError(r, authErr, "cerbos check failed for formata builder card")
	}
	view := PlatformAdminView{
		PageBase:         s.pageBaseForUser(admin, "platform_admin_body", "", ""),
		ActivePanel:      "streams",
		Groups:           groups,
		StreamCatalog:    streamCatalog,
		ShowCreateStream: showCreateStream && authErr == nil,
		Breadcrumbs:      buildPlatformAdminBreadcrumbs("streams"),
	}
	view.Console = platformAdminConsole(view)

	if s.writeStreamTaxonomyHTMXPartial(w, r, "platform_admin_stream_catalog", view.StreamCatalog) {
		return
	}

	if wantsAdminConsolePartial(r) {
		if err := s.tmpl.ExecuteTemplate(w, "admin_console", view.Console); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "platform_admin.html", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
