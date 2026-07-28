package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// DeleteSubCategory refuses delete while any file-catalog Stream blueprint
// effectively references the Sub-category path, then deletes via the store.
func (s *Server) DeleteSubCategory(ctx context.Context, categorySlug, slug string) error {
	if s == nil || s.store == nil {
		return fmt.Errorf("store is required")
	}
	return deleteSubCategory(ctx, s.store, categorySlug, slug, s.fileCatalogReferencesSubCategory)
}

func (s *Server) fileCatalogReferencesSubCategory(ctx context.Context, categorySlug, subCategorySlug string) (bool, error) {
	catalog, err := s.loadFileWorkflowCatalog(ctx)
	if err != nil {
		return false, err
	}
	return catalogReferencesSubCategoryPath(catalog, categorySlug, subCategorySlug), nil
}

// loadFileWorkflowCatalog loads Stream blueprints from the config directory only
// (effective categorization applied). Formata DB streams are out of scope for
// this Restrict check — workflowCatalog() prefers Formata when present.
func (s *Server) loadFileWorkflowCatalog(_ context.Context) (map[string]RuntimeConfig, error) {
	dir := "config"
	if trimmed := strings.TrimSpace(s.configDir); trimmed != "" {
		dir = trimmed
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("config dir not found: %w", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isWorkflowCatalogConfigFile(name) {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)

	catalog := make(map[string]RuntimeConfig, len(paths))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read config %s: %w", path, readErr)
		}
		cfg, parseErr := parseRuntimeConfigData(filepath.Base(path), data)
		if parseErr != nil {
			return nil, parseErr
		}
		s.resolveCatalogStreamCategorization(&cfg)
		key := strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		if key == "" {
			return nil, fmt.Errorf("workflow key is empty for %s", filepath.Base(path))
		}
		if _, exists := catalog[key]; exists {
			return nil, fmt.Errorf("duplicate workflow key %q", key)
		}
		catalog[key] = cfg
	}
	return catalog, nil
}
