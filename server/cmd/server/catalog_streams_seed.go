package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var catalogStreamNameVariants = []string{"Pilot", "Standard", "Extended"}

type catalogStreamLeaf struct {
	CategorySlug    string
	SubCategorySlug string
	SubCategoryName string
}

type catalogStreamRNG interface {
	Intn(n int) int
}

func taxonomyLeavesFromTree(tree []TaxonomyCategoryNode) []catalogStreamLeaf {
	var out []catalogStreamLeaf
	for _, cat := range tree {
		for _, sub := range cat.SubCategories {
			out = append(out, catalogStreamLeaf{
				CategorySlug:    strings.TrimSpace(cat.Slug),
				SubCategorySlug: strings.TrimSpace(sub.Slug),
				SubCategoryName: strings.TrimSpace(sub.Name),
			})
		}
	}
	return out
}

func catalogStreamDisplayName(subName string, slotIndex int) string {
	variant := catalogStreamNameVariants[slotIndex%len(catalogStreamNameVariants)]
	return fmt.Sprintf("%s — %s", strings.TrimSpace(subName), variant)
}

var (
	reCatalogStreamCat = regexp.MustCompile(`(?m)^\s*categorySlug:.*(?:\n|$)`)
	reCatalogStreamSub = regexp.MustCompile(`(?m)^\s*subCategorySlug:.*(?:\n|$)`)
	reCatalogStreamWF  = regexp.MustCompile(`(?m)^workflow:\n`)
)

func applyCatalogStreamCategoryAndName(yamlText, categorySlug, subCategorySlug, name string) (string, error) {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	name = strings.TrimSpace(name)
	if cat == "" || sub == "" || name == "" {
		return "", fmt.Errorf("categorySlug, subCategorySlug, and name are required")
	}
	out := strings.TrimRight(yamlText, "\n") + "\n"
	out = reCatalogStreamCat.ReplaceAllString(out, "")
	out = reCatalogStreamSub.ReplaceAllString(out, "")
	if !reCatalogStreamWF.MatchString(out) {
		return "", fmt.Errorf("workflow: key not found")
	}
	out = reCatalogStreamWF.ReplaceAllString(out, fmt.Sprintf(
		"workflow:\n  categorySlug: %s\n  subCategorySlug: %s\n", cat, sub,
	))
	replaced := false
	out = regexp.MustCompile(`(?m)^workflow:\n(?:  .+\n)*?  name:\s*.+$`).ReplaceAllStringFunc(out, func(block string) string {
		if replaced {
			return block
		}
		replaced = true
		return regexp.MustCompile(`(?m)^(  name:\s*).+$`).ReplaceAllString(block, "${1}"+name)
	})
	if !replaced {
		return "", fmt.Errorf("workflow name: not found")
	}
	if _, err := parseRuntimeConfigData("catalog-stream-seed.yaml", []byte(out)); err != nil {
		return "", err
	}
	return out, nil
}

func loadCatalogStreamTemplateBodies(ctx context.Context, store Store, configDir string) ([]string, error) {
	if store != nil {
		streams, err := store.ListFormataBuilderStreams(ctx)
		if err != nil {
			return nil, err
		}
		if len(streams) > 0 {
			bodies := make([]string, 0, len(streams))
			for _, s := range streams {
				bodies = append(bodies, s.Stream)
			}
			return bodies, nil
		}
	}
	dir := strings.TrimSpace(configDir)
	if dir == "" {
		dir = "config"
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("config dir not found: %w", err)
	}
	var paths []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !isWorkflowCatalogConfigFile(entry.Name()) {
			continue
		}
		paths = append(paths, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, fmt.Errorf("catalog stream seed: no template workflows found")
	}
	bodies := make([]string, 0, len(paths))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read template %s: %w", path, readErr)
		}
		bodies = append(bodies, string(data))
	}
	return bodies, nil
}

type catalogStreamsSeedResult struct {
	Leaves  int
	Streams int
}

func seedCatalogStreams(ctx context.Context, server *Server, rng catalogStreamRNG) (catalogStreamsSeedResult, error) {
	var zero catalogStreamsSeedResult
	if server == nil || server.store == nil {
		return zero, fmt.Errorf("catalog stream seed: server store is required")
	}
	if rng == nil {
		return zero, fmt.Errorf("catalog stream seed: rng is required")
	}

	tree, err := loadTaxonomyTree(ctx, server.store)
	if err != nil {
		return zero, err
	}
	leaves := taxonomyLeavesFromTree(tree)
	if len(leaves) == 0 {
		return zero, fmt.Errorf("catalog stream seed: taxonomy is empty (run seed-categories first)")
	}

	templates, err := loadCatalogStreamTemplateBodies(ctx, server.store, server.configDir)
	if err != nil {
		return zero, err
	}

	catalog, err := server.workflowCatalog()
	if err != nil && err.Error() != "workflow config catalog is empty" {
		return zero, err
	}
	for key := range catalog {
		if delErr := server.store.DeleteWorkflowData(ctx, key); delErr != nil {
			return zero, fmt.Errorf("delete workflow data %s: %w", key, delErr)
		}
	}

	existing, err := server.store.ListFormataBuilderStreams(ctx)
	if err != nil {
		return zero, err
	}
	for _, doc := range existing {
		if delErr := server.store.DeleteFormataBuilderStream(ctx, doc.ID); delErr != nil {
			return zero, fmt.Errorf("delete formata stream %s: %w", doc.ID.Hex(), delErr)
		}
	}

	creator := platformAdminStreamUserID()
	now := time.Now().UTC()
	if server.now != nil {
		now = server.now().UTC()
	}
	inserted := 0
	for _, leaf := range leaves {
		n := 1 + rng.Intn(3)
		for i := 0; i < n; i++ {
			body := templates[rng.Intn(len(templates))]
			name := catalogStreamDisplayName(leaf.SubCategoryName, i)
			yamlOut, applyErr := applyCatalogStreamCategoryAndName(body, leaf.CategorySlug, leaf.SubCategorySlug, name)
			if applyErr != nil {
				return zero, fmt.Errorf("build %s/%s: %w", leaf.CategorySlug, leaf.SubCategorySlug, applyErr)
			}
			if _, saveErr := server.store.SaveFormataBuilderStream(ctx, FormataBuilderStream{
				Stream:          yamlOut,
				UpdatedAt:       now,
				CreatedByUserID: creator,
				UpdatedByUserID: creator,
			}); saveErr != nil {
				return zero, fmt.Errorf("save %s/%s: %w", leaf.CategorySlug, leaf.SubCategorySlug, saveErr)
			}
			inserted++
		}
	}
	return catalogStreamsSeedResult{Leaves: len(leaves), Streams: inserted}, nil
}
