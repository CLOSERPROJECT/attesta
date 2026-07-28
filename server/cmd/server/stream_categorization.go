package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

// subCategoryPathExists reports whether a Sub-category path exists in taxonomy.
// A nil checker means taxonomy is unavailable (treat as uncategorized).
type subCategoryPathExists func(ctx context.Context, categorySlug, subCategorySlug string) (bool, error)

// isWorkflowCatalogConfigFile reports whether a config-dir entry should be loaded
// as a Stream blueprint. Taxonomy seed files share the directory but are not workflows.
func isWorkflowCatalogConfigFile(name string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	switch base {
	case "categories.yaml", "categories.yml":
		return false
	}
	return strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml")
}

// IsCategorized reports whether the Stream blueprint currently references a
// Sub-category path (both slugs set after effective categorization).
func (w WorkflowDef) IsCategorized() bool {
	return strings.TrimSpace(w.CategorySlug) != "" && strings.TrimSpace(w.SubCategorySlug) != ""
}

// validateWorkflowCategoryPair requires categorySlug and subCategorySlug together
// (both set or both omitted). A Stream never attaches to a Category alone.
func validateWorkflowCategoryPair(workflow WorkflowDef) error {
	categorySlug := strings.TrimSpace(workflow.CategorySlug)
	subCategorySlug := strings.TrimSpace(workflow.SubCategorySlug)
	if (categorySlug == "") == (subCategorySlug == "") {
		return nil
	}
	return fmt.Errorf("categorySlug and subCategorySlug must both be set or both omitted")
}

// applyEffectiveStreamCategorization keeps a declared Sub-category path only when
// it exists in taxonomy. Unknown paths (or a nil checker) become uncategorized
// without failing blueprint load. Lookup errors leave the declared pair unchanged
// and are returned so callers can avoid caching a transient resolution.
// Pair completeness is validated separately by validateWorkflowCategoryPair.
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

func (s *Server) storeSubCategoryPathExists(ctx context.Context, categorySlug, subCategorySlug string) (bool, error) {
	if s == nil || s.store == nil {
		return false, nil
	}
	_, err := s.store.GetSubCategoryBySlug(ctx, categorySlug, subCategorySlug)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, mongo.ErrNoDocuments) {
		return false, nil
	}
	return false, err
}

func (s *Server) resolveCatalogStreamCategorization(cfg *RuntimeConfig) error {
	if cfg == nil {
		return nil
	}
	var pathExists subCategoryPathExists
	if s != nil {
		pathExists = s.storeSubCategoryPathExists
	}
	return applyEffectiveStreamCategorization(context.Background(), &cfg.Workflow, pathExists)
}
