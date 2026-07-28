package main

import (
	"context"
	"fmt"
	"strings"
)

// TaxonomyCategoryNode is a nested category in the platform taxonomy tree.
type TaxonomyCategoryNode struct {
	Slug          string
	Name          string
	Icon          string
	IconURL       string
	SortOrder     int
	SubCategories []TaxonomySubCategoryNode
}

// TaxonomySubCategoryNode is a nested sub-category leaf under a category.
type TaxonomySubCategoryNode struct {
	Slug        string
	Name        string
	Icon        string
	IconURL     string
	SortOrder   int
	Description string
}

func taxonomyIconURL(icon string) string {
	key := strings.TrimSpace(icon)
	if err := validateTaxonomyIcon(key); err != nil {
		return ""
	}
	return fmt.Sprintf("/static/taxonomy/%s.svg", key)
}

func loadTaxonomyTree(ctx context.Context, store Store) ([]TaxonomyCategoryNode, error) {
	categories, err := store.ListCategories(ctx)
	if err != nil {
		return nil, err
	}

	nodes := make([]TaxonomyCategoryNode, 0, len(categories))
	for _, category := range categories {
		subs, err := store.ListSubCategories(ctx, category.Slug)
		if err != nil {
			return nil, err
		}

		subNodes := make([]TaxonomySubCategoryNode, 0, len(subs))
		for _, sub := range subs {
			subNodes = append(subNodes, TaxonomySubCategoryNode{
				Name:        sub.Name,
				Slug:        sub.Slug,
				Icon:        sub.Icon,
				IconURL:     taxonomyIconURL(sub.Icon),
				SortOrder:   sub.SortOrder,
				Description: sub.Description,
			})
		}

		nodes = append(nodes, TaxonomyCategoryNode{
			Name:          category.Name,
			Slug:          category.Slug,
			Icon:          category.Icon,
			IconURL:       taxonomyIconURL(category.Icon),
			SortOrder:     category.SortOrder,
			SubCategories: subNodes,
		})
	}

	return nodes, nil
}
