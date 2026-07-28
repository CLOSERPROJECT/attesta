package main

import (
	"context"
	"fmt"
	"strings"
)

type PlatformAdminCategoryRow struct {
	Name          string
	Slug          string
	Icon          string
	IconURL       string // "/static/taxonomy/"+Icon+".svg" if allowlisted
	SubCategories []PlatformAdminSubCategoryRow
}

type PlatformAdminSubCategoryRow struct {
	Name        string
	Slug        string
	Icon        string
	IconURL     string
	Description string
}

func taxonomyIconURL(icon string) string {
	key := strings.TrimSpace(icon)
	if err := validateTaxonomyIcon(key); err != nil {
		return ""
	}
	return fmt.Sprintf("/static/taxonomy/%s.svg", key)
}

func buildPlatformAdminTaxonomyTree(ctx context.Context, store Store) ([]PlatformAdminCategoryRow, error) {
	categories, err := store.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]PlatformAdminCategoryRow, 0, len(categories))
	for _, category := range categories {
		subs, err := store.ListSubCategories(ctx, category.Slug)
		if err != nil {
			return nil, err
		}

		subRows := make([]PlatformAdminSubCategoryRow, 0, len(subs))
		for _, sub := range subs {
			subRows = append(subRows, PlatformAdminSubCategoryRow{
				Name:        sub.Name,
				Slug:        sub.Slug,
				Icon:        sub.Icon,
				IconURL:     taxonomyIconURL(sub.Icon),
				Description: sub.Description,
			})
		}

		rows = append(rows, PlatformAdminCategoryRow{
			Name:          category.Name,
			Slug:          category.Slug,
			Icon:          category.Icon,
			IconURL:       taxonomyIconURL(category.Icon),
			SubCategories: subRows,
		})
	}

	return rows, nil
}
