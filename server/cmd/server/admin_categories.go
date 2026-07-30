package main

import (
	"errors"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
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
	switch r.Method {
	case http.MethodGet:
		s.renderCategoriesEditor(w, r, admin, "", "", nil)
	case http.MethodPost:
		if r.URL.Path != "/admin/categories" {
			http.NotFound(w, r)
			return
		}
		s.handleAdminCategoriesPost(w, r, admin)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleAdminCategoriesPost(w http.ResponseWriter, r *http.Request, admin *AccountUser) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	intent := strings.TrimSpace(r.Form.Get("intent"))
	switch intent {
	case "create":
		name := strings.TrimSpace(r.Form.Get("name"))
		icon := strings.TrimSpace(r.Form.Get("icon"))
		_, err := s.store.CreateCategory(r.Context(), Category{
			Slug: canonifySlug(name),
			Name: name,
			Icon: icon,
		})
		if err != nil {
			form := &CategoriesEditorForm{
				Open: true,
				Level: "group",
				Mode:  "create",
				Name:  name,
				Icon:  icon,
			}
			s.renderCategoriesEditor(w, r, admin, "", taxonomyGroupMutationError(err), form)
			return
		}
		s.renderCategoriesEditor(w, r, admin, "Category group created", "", nil)
	case "update":
		slug := strings.TrimSpace(r.Form.Get("slug"))
		name := strings.TrimSpace(r.Form.Get("name"))
		icon := strings.TrimSpace(r.Form.Get("icon"))
		_, err := s.store.UpdateCategory(r.Context(), slug, name, icon)
		if err != nil {
			form := &CategoriesEditorForm{
				Open: true,
				Level: "group",
				Mode:  "edit",
				Slug:  slug,
				Name:  name,
				Icon:  icon,
			}
			s.renderCategoriesEditor(w, r, admin, "", taxonomyGroupMutationError(err), form)
			return
		}
		s.renderCategoriesEditor(w, r, admin, "Category group updated", "", nil)
	case "delete":
		slug := strings.TrimSpace(r.Form.Get("slug"))
		if err := s.store.DeleteCategory(r.Context(), slug); err != nil {
			s.renderCategoriesEditor(w, r, admin, "", taxonomyGroupMutationError(err), nil)
			return
		}
		s.renderCategoriesEditor(w, r, admin, "Category group deleted", "", nil)
	case "reorder":
		slug := strings.TrimSpace(r.Form.Get("slug"))
		direction := strings.TrimSpace(r.Form.Get("direction"))
		if err := s.store.ReorderCategory(r.Context(), slug, direction); err != nil {
			s.renderCategoriesEditor(w, r, admin, "", taxonomyGroupMutationError(err), nil)
			return
		}
		s.renderCategoriesEditor(w, r, admin, "Category group reordered", "", nil)
	default:
		http.Error(w, "unknown intent", http.StatusBadRequest)
	}
}

func taxonomyGroupMutationError(err error) string {
	if err == nil {
		return ""
	}
	switch {
	case errors.Is(err, ErrTaxonomySlugExists):
		return "taxonomy slug already exists"
	case errors.Is(err, ErrInvalidTaxonomyIcon):
		return "invalid taxonomy icon"
	case errors.Is(err, ErrCategoryHasSubCategories):
		return "category has sub-categories"
	case errors.Is(err, ErrTaxonomyReorderBoundary):
		return "cannot move further"
	case errors.Is(err, mongo.ErrNoDocuments):
		return "category not found"
	default:
		return err.Error()
	}
}

func (s *Server) renderCategoriesEditor(w http.ResponseWriter, r *http.Request, admin *AccountUser, confirmation, formErr string, formState *CategoriesEditorForm) {
	editor, err := s.buildCategoriesEditorView(r.Context(), r.URL.Query(), formErr, confirmation, formState)
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
