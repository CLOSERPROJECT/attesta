package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// Nil checker means taxonomy is unavailable (treat as uncategorized).
type subCategoryPathExists func(ctx context.Context, categorySlug, subCategorySlug string) (bool, error)

// Taxonomy seed files share the config directory but are not Stream blueprints.
func isWorkflowCatalogConfigFile(name string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	switch base {
	case "categories.yaml", "categories.yml":
		return false
	}
	return strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")
}

func (w WorkflowDef) IsCategorized() bool {
	return strings.TrimSpace(w.CategorySlug) != "" && strings.TrimSpace(w.SubCategorySlug) != ""
}

// Both set or both omitted — a Stream never attaches to a Category alone.
func validateWorkflowCategoryPair(workflow WorkflowDef) error {
	categorySlug := strings.TrimSpace(workflow.CategorySlug)
	subCategorySlug := strings.TrimSpace(workflow.SubCategorySlug)
	if (categorySlug == "") == (subCategorySlug == "") {
		return nil
	}
	return fmt.Errorf("categorySlug and subCategorySlug must both be set or both omitted")
}

// Unknown paths (or nil checker) become uncategorized without failing load.
// Lookup errors leave the declared pair unchanged and are returned so callers
// can avoid caching a transient resolution.
func applyEffectiveStreamCategorization(ctx context.Context, workflow *WorkflowDef, pathExists subCategoryPathExists) error {
	if workflow == nil {
		return nil
	}
	categorySlug := strings.TrimSpace(workflow.CategorySlug)
	subCategorySlug := strings.TrimSpace(workflow.SubCategorySlug)
	if categorySlug == "" && subCategorySlug == "" {
		workflow.CategorySlug = ""
		workflow.SubCategorySlug = ""
		return nil
	}

	if pathExists == nil {
		workflow.CategorySlug = ""
		workflow.SubCategorySlug = ""
		return nil
	}

	ok, err := pathExists(ctx, categorySlug, subCategorySlug)
	if err != nil {
		return err
	}
	if !ok {
		workflow.CategorySlug = ""
		workflow.SubCategorySlug = ""
		return nil
	}
	workflow.CategorySlug = categorySlug
	workflow.SubCategorySlug = subCategorySlug
	return nil
}

func storeSubCategoryPathExists(ctx context.Context, store Store, categorySlug, subCategorySlug string) (bool, error) {
	if store == nil {
		return false, nil
	}
	_, err := store.GetSubCategoryBySlug(ctx, categorySlug, subCategorySlug)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

func (s *Server) storeSubCategoryPathExists(ctx context.Context, categorySlug, subCategorySlug string) (bool, error) {
	if s == nil {
		return false, nil
	}
	return storeSubCategoryPathExists(ctx, s.store, categorySlug, subCategorySlug)
}

func resolveCatalogStreamCategorization(ctx context.Context, store Store, cfg *RuntimeConfig) error {
	if cfg == nil {
		return nil
	}
	var pathExists subCategoryPathExists
	if store != nil {
		pathExists = func(ctx context.Context, categorySlug, subCategorySlug string) (bool, error) {
			return storeSubCategoryPathExists(ctx, store, categorySlug, subCategorySlug)
		}
	}
	return applyEffectiveStreamCategorization(ctx, &cfg.Workflow, pathExists)
}

func (s *Server) resolveCatalogStreamCategorization(cfg *RuntimeConfig) error {
	var store Store
	if s != nil {
		store = s.store
	}
	return resolveCatalogStreamCategorization(context.Background(), store, cfg)
}
