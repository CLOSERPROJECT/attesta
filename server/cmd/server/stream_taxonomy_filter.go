package main

import (
	"net/http"
	"strings"
)

const (
	streamTaxonomyUncategorized  = "uncategorized"
	homeDiscoveryCatalogTargetID = "home-stream-discovery"
	homeDiscoveryResultsTargetID = "home-stream-discovery-results"
)

type streamTaxonomyCandidate struct {
	Card            ManagedPublicStreamCardView
	CategorySlug    string
	SubCategorySlug string
}

type streamTaxonomyCatalogSpec struct {
	ID                      string
	FormID                  string
	Action                  string
	ResultsID               string
	IncludeAllCategories    bool
	AlwaysShowUncategorized bool
	EmptyHint               string
}

func streamTaxonomyTarget(r *http.Request, catalogTargetID, resultsTargetID string) string {
	if r == nil || !isHTMXRequest(r) {
		return ""
	}
	target := strings.TrimSpace(r.Header.Get("HX-Target"))
	if target == catalogTargetID || target == resultsTargetID {
		return target
	}
	return ""
}

func streamTaxonomySelectionFromRequest(r *http.Request, catalogTargetID, resultsTargetID string) (string, string) {
	if streamTaxonomyTarget(r, catalogTargetID, resultsTargetID) == "" {
		return "", ""
	}
	return strings.TrimSpace(r.URL.Query().Get("category")), strings.TrimSpace(r.URL.Query().Get("subCategory"))
}

// writeStreamTaxonomyHTMXPartial writes the catalog or results fragment when
// HX-Target matches. Returns true when a partial response was written.
func (s *Server) writeStreamTaxonomyHTMXPartial(w http.ResponseWriter, r *http.Request, catalogTemplate string, catalog StreamTaxonomyCatalogView) bool {
	switch streamTaxonomyTarget(r, catalog.ID, catalog.Results.ID) {
	case catalog.ID:
		if err := s.tmpl.ExecuteTemplate(w, catalogTemplate, catalog); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true
	case catalog.Results.ID:
		if err := s.tmpl.ExecuteTemplate(w, "stream_taxonomy_results", catalog.Results); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return true
	}
	return false
}

func streamTaxonomyCategory(categories []TaxonomyCategoryNode, slug string) (TaxonomyCategoryNode, bool) {
	trimmed := strings.TrimSpace(slug)
	for _, category := range categories {
		if category.Slug == trimmed {
			return category, true
		}
	}
	return TaxonomyCategoryNode{}, false
}

func streamTaxonomyCandidateIsUncategorized(candidate streamTaxonomyCandidate, categories []TaxonomyCategoryNode) bool {
	return !taxonomyHasPath(categories, candidate.CategorySlug, candidate.SubCategorySlug)
}

func streamTaxonomyCategories(categories []TaxonomyCategoryNode, candidates []streamTaxonomyCandidate, includeAll bool) []TaxonomyCategoryNode {
	if includeAll {
		return categories
	}
	used := make(map[string]struct{})
	for _, candidate := range candidates {
		if taxonomyHasPath(categories, candidate.CategorySlug, candidate.SubCategorySlug) {
			used[strings.TrimSpace(candidate.CategorySlug)] = struct{}{}
		}
	}
	filtered := make([]TaxonomyCategoryNode, 0, len(used))
	for _, category := range categories {
		if _, ok := used[category.Slug]; ok {
			filtered = append(filtered, category)
		}
	}
	return filtered
}

func buildStreamTaxonomyCatalogView(spec streamTaxonomyCatalogSpec, categories []TaxonomyCategoryNode, candidates []streamTaxonomyCandidate, requestedCategory, requestedSubCategory string) StreamTaxonomyCatalogView {
	filterCategories := streamTaxonomyCategories(categories, candidates, spec.IncludeAllCategories)
	showUncategorized := spec.AlwaysShowUncategorized
	if !showUncategorized {
		for _, candidate := range candidates {
			if streamTaxonomyCandidateIsUncategorized(candidate, categories) {
				showUncategorized = true
				break
			}
		}
	}

	categorySlug := ""
	subCategorySlug := ""
	switch strings.TrimSpace(requestedCategory) {
	case "":
	case streamTaxonomyUncategorized:
		if showUncategorized {
			categorySlug = streamTaxonomyUncategorized
		}
	default:
		if category, ok := streamTaxonomyCategory(filterCategories, requestedCategory); ok {
			categorySlug = category.Slug
			for _, subCategory := range category.SubCategories {
				if subCategory.Slug == strings.TrimSpace(requestedSubCategory) {
					subCategorySlug = subCategory.Slug
					break
				}
			}
		}
	}

	categoryOptions := make([]StreamTaxonomyOptionView, 0, len(filterCategories))
	var subCategoryOptions []StreamTaxonomyOptionView
	for _, category := range filterCategories {
		categoryOptions = append(categoryOptions, StreamTaxonomyOptionView{
			Value:    category.Slug,
			Label:    category.Name,
			IconURL:  category.IconURL,
			Selected: category.Slug == categorySlug,
		})
		if category.Slug != categorySlug {
			continue
		}
		subCategoryOptions = make([]StreamTaxonomyOptionView, 0, len(category.SubCategories))
		for _, subCategory := range category.SubCategories {
			subCategoryOptions = append(subCategoryOptions, StreamTaxonomyOptionView{
				Value:    subCategory.Slug,
				Label:    subCategory.Name,
				IconURL:  subCategory.IconURL,
				Selected: subCategory.Slug == subCategorySlug,
			})
		}
	}

	streams := make([]ManagedPublicStreamCardView, 0, len(candidates))
	for _, candidate := range candidates {
		switch categorySlug {
		case "":
			streams = append(streams, candidate.Card)
		case streamTaxonomyUncategorized:
			if streamTaxonomyCandidateIsUncategorized(candidate, categories) {
				streams = append(streams, candidate.Card)
			}
		default:
			if candidate.CategorySlug != categorySlug {
				continue
			}
			if subCategorySlug != "" && candidate.SubCategorySlug != subCategorySlug {
				continue
			}
			streams = append(streams, candidate.Card)
		}
	}

	return StreamTaxonomyCatalogView{
		ID: spec.ID,
		Filter: StreamTaxonomyFilterView{
			FormID:              spec.FormID,
			Action:              spec.Action,
			CatalogTargetID:     spec.ID,
			ResultsTargetID:     spec.ResultsID,
			CategoryOptions:     categoryOptions,
			SubCategoryOptions:  subCategoryOptions,
			ShowUncategorized:   showUncategorized,
			AllCategoriesActive: categorySlug == "",
			UncategorizedActive: categorySlug == streamTaxonomyUncategorized,
			SubCategoryDisabled: categorySlug == "" || categorySlug == streamTaxonomyUncategorized,
		},
		Results: StreamTaxonomyResultsView{
			ID:        spec.ResultsID,
			Streams:   streams,
			EmptyHint: spec.EmptyHint,
		},
	}
}

func streamTaxonomyCandidatesFromGroups(groups []MyHomeStreamGroupView) []streamTaxonomyCandidate {
	var candidates []streamTaxonomyCandidate
	for _, group := range groups {
		for _, card := range group.Streams {
			candidate := streamTaxonomyCandidate{Card: card}
			if !group.Uncategorized {
				candidate.CategorySlug = group.CategorySlug
				candidate.SubCategorySlug = group.SubCategorySlug
			}
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func streamTaxonomyCandidatesFromDiscovery(items []StreamDiscoveryItem, cards []ManagedPublicStreamCardView) []streamTaxonomyCandidate {
	taxonomyByKey := make(map[string]StreamDiscoveryItem, len(items))
	for _, item := range items {
		taxonomyByKey[item.WorkflowKey] = item
	}
	candidates := make([]streamTaxonomyCandidate, 0, len(cards))
	for _, card := range cards {
		item, ok := taxonomyByKey[card.Key]
		if !ok {
			continue
		}
		candidates = append(candidates, streamTaxonomyCandidate{
			Card:            card,
			CategorySlug:    item.CategorySlug,
			SubCategorySlug: item.SubCategorySlug,
		})
	}
	return candidates
}
