package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ensureEmptyConfigFoundation installs empty-only Mongo foundations for taxonomy
// and Formata builder Streams. Taxonomy uses bootstrapTaxonomy (categories.yaml
// when Categories are empty). Formata Streams are copied from disk catalog paths
// via inspectWorkflowCatalogSources when Mongo Formata is empty. Seed's demo
// overlay (wipe + leaf-multiplied Streams) is a separate path and must not be
// folded in here.
func ensureEmptyConfigFoundation(ctx context.Context, store Store, configDir string, now func() time.Time) error {
	if store == nil {
		return nil
	}
	if err := bootstrapTaxonomy(ctx, store, configDir); err != nil {
		return err
	}
	return ensureEmptyFormataBuilderStreams(ctx, store, configDir, now)
}

// ensureEmptyFormataBuilderStreams copies disk YAML into Formata builder Streams
// when Mongo Formata is empty. It reuses inspectWorkflowCatalogSources for the
// empty/disk walk; if inspect already returns Streams, Formata is treated as
// non-empty and this is a no-op.
func ensureEmptyFormataBuilderStreams(ctx context.Context, store Store, configDir string, now func() time.Time) error {
	if store == nil {
		return nil
	}
	sources, err := inspectWorkflowCatalogSources(ctx, store, configDir)
	if err != nil {
		return err
	}
	if len(sources.streams) > 0 {
		return nil
	}

	updatedAt := time.Now().UTC()
	if now != nil {
		updatedAt = now().UTC()
	}
	for _, path := range sources.paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read config %s: %w", path, readErr)
		}
		creatorID := platformAdminStreamUserID()
		if _, saveErr := store.SaveFormataBuilderStream(ctx, FormataBuilderStream{
			Stream:          string(data),
			UpdatedAt:       updatedAt,
			CreatedByUserID: creatorID,
			UpdatedByUserID: creatorID,
		}); saveErr != nil {
			return fmt.Errorf("seed formata stream %s: %w", filepath.Base(path), saveErr)
		}
	}
	return nil
}
