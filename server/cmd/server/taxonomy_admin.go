package main

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

// TaxonomyCategoryNode is a nested category in the platform taxonomy tree.
type TaxonomyCategoryNode struct {
	Slug          string                    `json:"slug"`
	Name          string                    `json:"name"`
	Icon          string                    `json:"icon"`
	IconURL       string                    `json:"iconURL"`
	SortOrder     int                       `json:"sortOrder"`
	CanDelete     bool                      `json:"canDelete"`
	CanMoveUp     bool                      `json:"canMoveUp"`
	CanMoveDown   bool                      `json:"canMoveDown"`
	DeleteReason  string                    `json:"deleteReason"`
	SubCategories []TaxonomySubCategoryNode `json:"subCategories"`
}

// TaxonomySubCategoryNode is a nested sub-category leaf under a category.
type TaxonomySubCategoryNode struct {
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	Icon         string `json:"icon"`
	IconURL      string `json:"iconURL"`
	SortOrder    int    `json:"sortOrder"`
	Description  string `json:"description"`
	CanDelete    bool   `json:"canDelete"`
	CanMoveUp    bool   `json:"canMoveUp"`
	CanMoveDown  bool   `json:"canMoveDown"`
	DeleteReason string `json:"deleteReason"`
}

// CategoriesEditorForm is inline create/edit state for the categories admin panel.
type CategoriesEditorForm struct {
	Open        bool
	Level       string // "group" | "leaf"
	Mode        string // "create" | "edit"
	ParentSlug  string
	Slug        string // edit only
	Name        string
	Icon        string
	Description string
	Error       string
}

// CategoriesEditorView is the server-rendered categories CRUD editor panel.
type CategoriesEditorView struct {
	Categories   []TaxonomyCategoryNode
	GroupCount   int
	LeafCount    int
	Form         CategoriesEditorForm
	IconKeys     []string
	Confirmation string
}

func taxonomyIconURL(icon string) string {
	key := strings.TrimSpace(icon)
	if err := validateTaxonomyIcon(key); err != nil {
		return ""
	}
	return fmt.Sprintf("/static/taxonomy/%s.svg", key)
}

func taxonomyIconKeys() []string {
	keys := make([]string, 0, len(taxonomyIconAllowlist))
	for key := range taxonomyIconAllowlist {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func enrichTaxonomyEditorTree(nodes []TaxonomyCategoryNode, referenced func(categorySlug, subSlug string) bool) []TaxonomyCategoryNode {
	if referenced == nil {
		referenced = func(string, string) bool { return false }
	}

	out := make([]TaxonomyCategoryNode, len(nodes))
	for i, category := range nodes {
		enriched := category
		enriched.CanDelete = len(category.SubCategories) == 0
		if !enriched.CanDelete {
			enriched.DeleteReason = "Has subcategories"
		}
		enriched.CanMoveUp = i > 0
		enriched.CanMoveDown = i < len(nodes)-1

		subs := make([]TaxonomySubCategoryNode, len(category.SubCategories))
		for j, sub := range category.SubCategories {
			enrichedSub := sub
			if referenced(category.Slug, sub.Slug) {
				enrichedSub.CanDelete = false
				enrichedSub.DeleteReason = "Referenced by a stream"
			} else {
				enrichedSub.CanDelete = true
			}
			enrichedSub.CanMoveUp = j > 0
			enrichedSub.CanMoveDown = j < len(category.SubCategories)-1
			subs[j] = enrichedSub
		}
		enriched.SubCategories = subs
		out[i] = enriched
	}
	return out
}

func parseCategoriesEditorQuery(q url.Values) CategoriesEditorForm {
	form := CategoriesEditorForm{}

	if newVal := strings.TrimSpace(q.Get("new")); newVal != "" {
		form.Open = true
		form.Mode = "create"
		switch newVal {
		case "group":
			form.Level = "group"
		case "sub":
			form.Level = "leaf"
			form.ParentSlug = strings.TrimSpace(q.Get("parent"))
		}
		return form
	}

	if editVal := strings.TrimSpace(q.Get("edit")); editVal != "" {
		form.Open = true
		form.Mode = "edit"
		switch editVal {
		case "group":
			form.Level = "group"
			form.Slug = strings.TrimSpace(q.Get("slug"))
		case "sub":
			form.Level = "leaf"
			form.ParentSlug = strings.TrimSpace(q.Get("parent"))
			form.Slug = strings.TrimSpace(q.Get("slug"))
		}
	}
	return form
}

func populateCategoriesEditorFormFromTree(form *CategoriesEditorForm, categories []TaxonomyCategoryNode) {
	if form == nil || !form.Open || form.Mode != "edit" {
		return
	}

	switch form.Level {
	case "group":
		for _, category := range categories {
			if category.Slug != form.Slug {
				continue
			}
			form.Name = category.Name
			form.Icon = category.Icon
			return
		}
	case "leaf":
		for _, category := range categories {
			if category.Slug != form.ParentSlug {
				continue
			}
			for _, sub := range category.SubCategories {
				if sub.Slug != form.Slug {
					continue
				}
				form.Name = sub.Name
				form.Icon = sub.Icon
				form.Description = sub.Description
				return
			}
		}
	}
}

func (s *Server) buildCategoriesEditorView(ctx context.Context, q url.Values, formErr, confirmation string, formState *CategoriesEditorForm) (CategoriesEditorView, error) {
	nodes, err := loadTaxonomyTree(ctx, s.store)
	if err != nil {
		return CategoriesEditorView{}, err
	}

	catalog, err := s.workflowCatalog()
	if err != nil {
		return CategoriesEditorView{}, err
	}
	referenced := func(categorySlug, subCategorySlug string) bool {
		return catalogReferencesSubCategoryPath(catalog, categorySlug, subCategorySlug)
	}

	categories := enrichTaxonomyEditorTree(nodes, referenced)
	var form CategoriesEditorForm
	if formState != nil {
		form = *formState
	} else {
		form = parseCategoriesEditorQuery(q)
		populateCategoriesEditorFormFromTree(&form, categories)
	}
	form.Error = strings.TrimSpace(formErr)

	groupCount := len(categories)
	leafCount := 0
	for _, category := range categories {
		leafCount += len(category.SubCategories)
	}

	return CategoriesEditorView{
		Categories:   categories,
		GroupCount:   groupCount,
		LeafCount:    leafCount,
		Form:         form,
		IconKeys:     taxonomyIconKeys(),
		Confirmation: strings.TrimSpace(confirmation),
	}, nil
}

func loadTaxonomyTree(ctx context.Context, store Store) ([]TaxonomyCategoryNode, error) {
	categories, err := store.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make([]TaxonomyCategoryNode, 0, len(categories))
	for _, category := range categories {
		subs, err := store.ListSubCategories(ctx, category.Slug)
		if err != nil {
			return nil, err
		}

		subNodes := make([]TaxonomySubCategoryNode, 0, len(subs))
		for _, sub := range subs {
			subNodes = append(subNodes, TaxonomySubCategoryNode{
				Name:        sub.Name,
				Slug:        sub.Slug,
				Icon:        sub.Icon,
				IconURL:     taxonomyIconURL(sub.Icon),
				SortOrder:   sub.SortOrder,
				Description: sub.Description,
			})
		}

		nodes = append(nodes, TaxonomyCategoryNode{
			Name:          category.Name,
			Slug:          category.Slug,
			Icon:          category.Icon,
			IconURL:       taxonomyIconURL(category.Icon),
			SortOrder:     category.SortOrder,
			SubCategories: subNodes,
		})
	}

	return nodes, nil
}
