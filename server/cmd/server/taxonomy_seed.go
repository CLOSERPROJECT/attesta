package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type taxonomySeedFile struct {
	Categories []taxonomySeedCategory `yaml:"categories"`
}

type taxonomySeedCategory struct {
	Slug          string                     `yaml:"slug"`
	Name          string                     `yaml:"name"`
	Icon          string                     `yaml:"icon"`
	SortOrder     int                        `yaml:"sortOrder"`
	SubCategories []taxonomySeedSubCategory  `yaml:"subCategories"`
}

type taxonomySeedSubCategory struct {
	Slug        string `yaml:"slug"`
	Name        string `yaml:"name"`
	Icon        string `yaml:"icon"`
	SortOrder   int    `yaml:"sortOrder"`
	Description string `yaml:"description,omitempty"`
}

func loadTaxonomySeedFile(path string) ([]Category, []SubCategory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return parseTaxonomySeedYAML(data)
}

func parseTaxonomySeedYAML(data []byte) ([]Category, []SubCategory, error) {
	var file taxonomySeedFile
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, nil, fmt.Errorf("parse taxonomy seed: %w", err)
	}

	categories := make([]Category, 0, len(file.Categories))
	subs := make([]SubCategory, 0)
	seenCategorySlugs := map[string]struct{}{}

	for _, raw := range file.Categories {
		slug := strings.TrimSpace(raw.Slug)
		if slug == "" {
			return nil, nil, fmt.Errorf("taxonomy seed: category slug is required")
		}
		if _, exists := seenCategorySlugs[slug]; exists {
			return nil, nil, fmt.Errorf("taxonomy seed: duplicate category slug %q", slug)
		}
		seenCategorySlugs[slug] = struct{}{}
		if err := validateTaxonomyIcon(raw.Icon); err != nil {
			return nil, nil, fmt.Errorf("taxonomy seed category %q: %w", slug, err)
		}
		categories = append(categories, Category{
			Slug:      slug,
			Name:      strings.TrimSpace(raw.Name),
			Icon:      strings.TrimSpace(raw.Icon),
			SortOrder: raw.SortOrder,
		})

		seenSubSlugs := map[string]struct{}{}
		for _, rawSub := range raw.SubCategories {
			subSlug := strings.TrimSpace(rawSub.Slug)
			if subSlug == "" {
				return nil, nil, fmt.Errorf("taxonomy seed category %q: sub-category slug is required", slug)
			}
			if _, exists := seenSubSlugs[subSlug]; exists {
				return nil, nil, fmt.Errorf("taxonomy seed category %q: duplicate sub-category slug %q", slug, subSlug)
			}
			seenSubSlugs[subSlug] = struct{}{}
			if err := validateTaxonomyIcon(rawSub.Icon); err != nil {
				return nil, nil, fmt.Errorf("taxonomy seed sub-category %q/%q: %w", slug, subSlug, err)
			}
			subs = append(subs, SubCategory{
				CategorySlug: slug,
				Slug:         subSlug,
				Name:         strings.TrimSpace(rawSub.Name),
				Icon:         strings.TrimSpace(rawSub.Icon),
				SortOrder:    rawSub.SortOrder,
				Description:  strings.TrimSpace(rawSub.Description),
			})
		}
	}

	return categories, subs, nil
}

func seedTaxonomyFromFile(ctx context.Context, store Store, path string) error {
	if store == nil {
		return fmt.Errorf("taxonomy seed: store is required")
	}
	categories, subs, err := loadTaxonomySeedFile(path)
	if err != nil {
		return err
	}
	if err := store.EnsureTaxonomyIndexes(ctx); err != nil {
		return fmt.Errorf("taxonomy seed indexes: %w", err)
	}
	if err := store.ReplaceTaxonomy(ctx, categories, subs); err != nil {
		return fmt.Errorf("taxonomy seed replace: %w", err)
	}
	return nil
}
