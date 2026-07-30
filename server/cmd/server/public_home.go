package main

import (
	"context"
	"net/http"
	"net/url"
	"strings"
)

const publicHomeStreamCardLimit = 6
const publicHomeStreamOrgAvatarLimit = 4

func taxonomyHasPath(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) bool {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	if cat == "" || sub == "" {
		return false
	}
	for _, category := range categories {
		if category.Slug != cat {
			continue
		}
		for _, leaf := range category.SubCategories {
			if leaf.Slug == sub {
				return true
			}
		}
		return false
	}
	return false
}

func resolvePublicHomeSelection(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) (string, string) {
	if taxonomyHasPath(categories, categorySlug, subCategorySlug) {
		return strings.TrimSpace(categorySlug), strings.TrimSpace(subCategorySlug)
	}
	if len(categories) == 0 || len(categories[0].SubCategories) == 0 {
		return "", ""
	}
	return categories[0].Slug, categories[0].SubCategories[0].Slug
}

func buildPublicHomeStreamResultsView(categories []TaxonomyCategoryNode, cat, sub string, streams []PublicStreamCardView, createHref string) PublicHomeStreamResultsView {
	view := PublicHomeStreamResultsView{
		Streams:          streams,
		CreateStreamHref: createHref,
	}
	for _, category := range categories {
		if category.Slug != cat {
			continue
		}
		view.CategoryName = category.Name
		view.CategoryIconURL = category.IconURL
		for _, leaf := range category.SubCategories {
			if leaf.Slug != sub {
				continue
			}
			view.SubCategoryName = leaf.Name
			view.SubCategoryIconURL = leaf.IconURL
			view.SubCategoryDescription = leaf.Description
			break
		}
		break
	}
	return view
}

func buildPublicHomeCategories(categories []TaxonomyCategoryNode, selectedCat, selectedSub string) []PublicHomeCategoryView {
	out := make([]PublicHomeCategoryView, 0, len(categories))
	for _, cat := range categories {
		subs := make([]PublicHomeSubCategoryView, 0, len(cat.SubCategories))
		for _, sub := range cat.SubCategories {
			query := url.Values{
				"category":    {cat.Slug},
				"subCategory": {sub.Slug},
			}
			encoded := query.Encode()
			subs = append(subs, PublicHomeSubCategoryView{
				Slug:       sub.Slug,
				Name:       sub.Name,
				Active:     cat.Slug == selectedCat && sub.Slug == selectedSub,
				PartialURL: "/streams/public?" + encoded,
				PushURL:    "/?" + encoded,
			})
		}
		out = append(out, PublicHomeCategoryView{
			Slug:          cat.Slug,
			Name:          cat.Name,
			IconURL:       cat.IconURL,
			Expanded:      cat.Slug == selectedCat,
			SubCategories: subs,
		})
	}
	return out
}

func publicHomeCreateStreamHref(signedIn bool) string {
	target := organizationPath("formata-builder")
	if signedIn {
		return target
	}
	return "/login?next=" + url.QueryEscape(target)
}

func (s *Server) publicStreamCardsForPath(ctx context.Context, categorySlug, subCategorySlug string) ([]PublicStreamCardView, error) {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	if cat == "" || sub == "" {
		return nil, nil
	}
	catalog, err := s.workflowCatalog()
	if err != nil {
		if err.Error() == "workflow config catalog is empty" {
			return nil, nil
		}
		return nil, err
	}
	keys := sortedWorkflowKeys(catalog)
	logoURLs := organizationLogoURLMap(ctx, s.identity)
	cards := make([]PublicStreamCardView, 0, publicHomeStreamCardLimit)
	for _, key := range keys {
		cfg := catalog[key]
		if !cfg.Workflow.IsCategorized() {
			continue
		}
		if cfg.Workflow.CategorySlug != cat || cfg.Workflow.SubCategorySlug != sub {
			continue
		}
		card, buildErr := s.buildPublicStreamCardView(ctx, key, cfg, logoURLs)
		if buildErr != nil {
			return nil, buildErr
		}
		cards = append(cards, card)
		if len(cards) >= publicHomeStreamCardLimit {
			break
		}
	}
	return cards, nil
}

func (s *Server) handlePublicStreamsPartial(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cat := strings.TrimSpace(r.URL.Query().Get("category"))
	sub := strings.TrimSpace(r.URL.Query().Get("subCategory"))
	categories, err := loadTaxonomyTree(r.Context(), s.store)
	if err != nil {
		http.Error(w, "failed to load categories", http.StatusInternalServerError)
		return
	}
	if !taxonomyHasPath(categories, cat, sub) {
		http.NotFound(w, r)
		return
	}
	streams, err := s.publicStreamCardsForPath(r.Context(), cat, sub)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	signedIn := false
	if _, _, err := s.currentUser(r); err == nil {
		signedIn = true
	}
	view := buildPublicHomeStreamResultsView(categories, cat, sub, streams, publicHomeCreateStreamHref(signedIn))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.ExecuteTemplate(w, "public_home_stream_results", view); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) buildPublicStreamCardView(ctx context.Context, key string, cfg RuntimeConfig, logoURLs map[string]string) (PublicStreamCardView, error) {
	steps := sortedSteps(cfg.Workflow)
	stepViews := make([]PublicStreamCardStepView, 0, len(steps))
	for _, step := range steps {
		stepViews = append(stepViews, PublicStreamCardStepView{
			Title:        step.Title,
			SubstepCount: len(step.Substep),
		})
	}
	orgs, overflow := publicStreamCardOrganizations(cfg.Organizations, logoURLs)
	instanceCount := 0
	activeCount := 0
	allCompleted := false
	if s.store != nil {
		processes, listErr := s.store.ListRecentProcessesByWorkflow(ctx, key, 0)
		if listErr != nil {
			return PublicStreamCardView{}, listErr
		}
		instanceCount = len(processes)
		if instanceCount > 0 {
			for i := range processes {
				processes[i].Progress = normalizeProgressKeys(processes[i].Progress)
				if deriveProcessStatus(cfg.Workflow, &processes[i]) == processStatusActive {
					activeCount++
				}
			}
			allCompleted = activeCount == 0
		}
	}
	return PublicStreamCardView{
		Name:                  cfg.Workflow.Name,
		Description:           strings.TrimSpace(cfg.Workflow.Description),
		Steps:                 stepViews,
		PassportEnabled:       cfg.DPP.Enabled,
		InstanceCount:         instanceCount,
		ActiveCount:           activeCount,
		AllCompleted:          allCompleted,
		Organizations:         orgs,
		OrganizationsOverflow: overflow,
	}, nil
}

func publicStreamCardOrganizations(orgs []WorkflowOrganization, logoURLs map[string]string) ([]PublicStreamCardOrgView, int) {
	out := make([]PublicStreamCardOrgView, 0, len(orgs))
	for _, org := range orgs {
		name := strings.TrimSpace(org.Name)
		if name == "" {
			name = strings.TrimSpace(org.Slug)
		}
		if name == "" {
			continue
		}
		slug := strings.TrimSpace(org.Slug)
		view := PublicStreamCardOrgView{
			Name:     name,
			Initials: organizationInitials(name),
		}
		if logoURLs != nil {
			view.LogoURL = strings.TrimSpace(logoURLs[slug])
		}
		out = append(out, view)
	}
	if len(out) > publicHomeStreamOrgAvatarLimit {
		return out[:publicHomeStreamOrgAvatarLimit], len(out) - publicHomeStreamOrgAvatarLimit
	}
	return out, 0
}
