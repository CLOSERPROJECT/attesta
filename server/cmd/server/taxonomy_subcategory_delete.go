package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrSubCategoryReferencedByStream is returned when normal Sub-category delete is
// refused because a catalog Stream blueprint still references that path.
var ErrSubCategoryReferencedByStream = errors.New("sub-category referenced by catalog stream")

// catalogSubCategoryReferenced reports whether any catalog Stream blueprint
// effectively references the Sub-category path. A nil checker skips the Restrict
// (unrestricted — seed ReplaceTaxonomy never uses this path).
type catalogSubCategoryReferenced func(ctx context.Context, categorySlug, subCategorySlug string) (bool, error)

// deleteSubCategory is the normal Sub-category delete API: refuse while a catalog
// Stream references (categorySlug, subCategorySlug), then delete from the store.
func deleteSubCategory(ctx context.Context, store Store, categorySlug, slug string, referenced catalogSubCategoryReferenced) error {
	if store == nil {
		return fmt.Errorf("store is required")
	}
	categorySlug = strings.TrimSpace(categorySlug)
	slug = strings.TrimSpace(slug)
	if referenced != nil {
		ok, err := referenced(ctx, categorySlug, slug)
		if err != nil {
			return err
		}
		if ok {
			return ErrSubCategoryReferencedByStream
		}
	}
	return store.DeleteSubCategory(ctx, categorySlug, slug)
}

// catalogReferencesSubCategoryPath reports whether any catalog Stream blueprint
// carries the given Sub-category path (both slugs set and matching).
func catalogReferencesSubCategoryPath(catalog map[string]RuntimeConfig, categorySlug, subCategorySlug string) bool {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	if cat == "" || sub == "" {
		return false
	}
	for _, cfg := range catalog {
		if strings.TrimSpace(cfg.Workflow.CategorySlug) == cat &&
			strings.TrimSpace(cfg.Workflow.SubCategorySlug) == sub {
			return true
		}
	}
	return false
}

// DeleteSubCategory refuses delete while any live-catalog Stream blueprint
// effectively references the Sub-category path, then deletes via the store.
func (s *Server) DeleteSubCategory(ctx context.Context, categorySlug, slug string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("store is required")
	}
	return deleteSubCategory(ctx, s.store, categorySlug, slug, s.liveCatalogReferencesSubCategory)
}

func (s *Server) liveCatalogReferencesSubCategory(ctx context.Context, categorySlug, subCategorySlug string) (bool, error) {
	catalog, err := s.workflowCatalog()
	if err != nil {
		return false, err
	}
	return catalogReferencesSubCategoryPath(catalog, categorySlug, subCategorySlug), nil
}
