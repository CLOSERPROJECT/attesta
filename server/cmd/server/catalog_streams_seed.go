package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
