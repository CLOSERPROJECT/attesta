package main

import (
	"net/http"
)

func wantsCategoriesPanelPartial(r *http.Request) bool {
	if !isHTMXRequest(r) {
		return false
	}
	return htmxTargetID(r) == "platform-admin-categories"
}

func (s *Server) handleAdminCategories(w http.ResponseWriter, r *http.Request) {
	admin, ok := s.requirePlatformAdmin(w, r)
	if !ok {
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.renderCategoriesEditor(w, r, admin, "", "")
}

func (s *Server) renderCategoriesEditor(w http.ResponseWriter, r *http.Request, admin *AccountUser, confirmation, formErr string) {
	editor, err := s.buildCategoriesEditorView(r.Context(), r.URL.Query(), formErr, confirmation)
	if err != nil {
		logAndHTTPError(w, r, http.StatusInternalServerError, "failed to load categories editor", err, "failed to load platform admin categories")
		return
	}
	view := PlatformAdminView{
		PageBase:         s.pageBaseForUser(admin, "platform_admin_body", "", ""),
		ActivePanel:      "categories",
		CategoriesEditor: editor,
		Breadcrumbs:      buildPlatformAdminBreadcrumbs("categories"),
		Confirmation:     confirmation,
	}
	view.Console = platformAdminConsole(view)

	if wantsCategoriesPanelPartial(r) {
		if err := s.tmpl.ExecuteTemplate(w, "platform_admin_categories_panel", view); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
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
