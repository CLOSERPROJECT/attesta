package main

import "context"

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

// streamManagementFlags mirrors workflowOptions CanClone/CanEdit/CanDelete rules.
func (s *Server) streamManagementFlags(ctx context.Context, user *AccountUser, key string, stream FormataBuilderStream, hasProcesses, canEditSavedStreams bool) (canClone, canEdit, editRequiresPurge, canDelete bool) {
	if s.authorizer == nil || user == nil {
		return false, false, false, false
	}
	canClone = canEditSavedStreams
	if canEditSavedStreams {
		if allowed, err := s.canEditStream(ctx, user, key, formataStreamCreatorID(stream), hasProcesses); err == nil {
			canEdit = allowed
			editRequiresPurge = allowed && hasProcesses
		}
	}
	if allowed, err := s.authorizer.CanDeleteStream(ctx, user, key, formataStreamCreatorID(stream), hasProcesses); err == nil {
		canDelete = allowed
	}
	return canClone, canEdit, editRequiresPurge, canDelete
}

func (s *Server) buildMyHomeCatalog(ctx context.Context, user *AccountUser) ([]MyHomeStreamGroupView, error) {
	catalog, err := s.workflowCatalog()
	if err != nil {
		return nil, err
	}
	var categories []TaxonomyCategoryNode
	if s.store != nil {
		categories, err = loadTaxonomyTree(ctx, s.store)
		if err != nil {
			return nil, err
		}
	}
	logoURLs := organizationLogoURLMap(ctx, s.identity)

	canEditSavedStreams := false
	if user != nil {
		if allowed, err := s.canViewFormataBuilder(ctx, user); err == nil {
			canEditSavedStreams = allowed
		}
	}

	streamsByKey := map[string]FormataBuilderStream{}
	if s.store != nil {
		streams, listErr := s.store.ListFormataBuilderStreams(ctx)
		if listErr != nil {
			return nil, listErr
		}
		for _, stream := range streams {
			if stream.ID.IsZero() {
				continue
			}
			streamsByKey[stream.ID.Hex()] = stream
		}
	}

	var accessibleKeys []string
	for _, key := range sortedWorkflowKeys(catalog) {
		if userCanAccessStream(user, catalog[key]) {
			accessibleKeys = append(accessibleKeys, key)
		}
	}

	cardsByKey := make(map[string]ManagedPublicStreamCardView, len(accessibleKeys))
	for _, key := range accessibleKeys {
		cfg := catalog[key]
		card, buildErr := s.buildPublicStreamCardView(ctx, key, cfg, logoURLs)
		if buildErr != nil {
			return nil, buildErr
		}
		card.Href = streamPath(key) + "/"

		managed := ManagedPublicStreamCardView{
			Key:          key,
			Card:         card,
			EditAction:   organizationPath("formata-builder?stream=" + key),
			DeleteAction: streamPath(key) + "/delete",
		}
		if stream, ok := streamsByKey[key]; ok {
			hasProcesses := card.InstanceCount > 0
			managed.CanClone, managed.CanEdit, managed.EditRequiresPurge, managed.CanDelete = s.streamManagementFlags(ctx, user, key, stream, hasProcesses, canEditSavedStreams)
		}
		cardsByKey[key] = managed
	}

	return buildMyHomeStreamGroups(categories, cardsByKey, catalog, accessibleKeys), nil
}
