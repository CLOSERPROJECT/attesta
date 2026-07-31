package main

func buildMyHomeStreamGroups(categories []TaxonomyCategoryNode, cardsByKey map[string]ManagedPublicStreamCardView, catalog map[string]RuntimeConfig, accessibleKeys []string) []MyHomeStreamGroupView {
	remaining := make(map[string]struct{}, len(accessibleKeys))
	for _, key := range accessibleKeys {
		remaining[key] = struct{}{}
	}

	var groups []MyHomeStreamGroupView
	for _, category := range categories {
		for _, sub := range category.SubCategories {
			var streams []ManagedPublicStreamCardView
			for _, key := range accessibleKeys {
				if _, ok := remaining[key]; !ok {
					continue
				}
				cfg, ok := catalog[key]
				if !ok {
					continue
				}
				if cfg.Workflow.CategorySlug != category.Slug || cfg.Workflow.SubCategorySlug != sub.Slug {
					continue
				}
				card, ok := cardsByKey[key]
				if !ok {
					delete(remaining, key)
					continue
				}
				streams = append(streams, card)
				delete(remaining, key)
			}
			if len(streams) == 0 {
				continue
			}
			groups = append(groups, MyHomeStreamGroupView{
				CategoryName:           category.Name,
				CategoryIconURL:        category.IconURL,
				SubCategoryName:        sub.Name,
				SubCategoryIconURL:     sub.IconURL,
				SubCategoryDescription: sub.Description,
				Streams:                streams,
			})
		}
	}

	var uncategorized []ManagedPublicStreamCardView
	for _, key := range accessibleKeys {
		if _, ok := remaining[key]; !ok {
			continue
		}
		card, ok := cardsByKey[key]
		if !ok {
			continue
		}
		uncategorized = append(uncategorized, card)
	}
	if len(uncategorized) > 0 {
		groups = append(groups, MyHomeStreamGroupView{
			CategoryName:  "Uncategorized",
			Uncategorized: true,
			Streams:       uncategorized,
		})
	}
	return groups
}
