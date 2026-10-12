package main

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
)

// Empty-only Mongo foundations for taxonomy and Formata builder Streams.
// Seed's demo overlay (wipe + leaf-multiplied Streams) is a separate path and
// must not be folded in here.
func ensureEmptyConfigFoundation(ctx context.Context, store Store, configDir string, now func() time.Time) error {
	if store == nil {
		return nil
	}
	if err := bootstrapTaxonomy(ctx, store, configDir); err != nil {
		return err
	}
	return ensureEmptyFormataBuilderStreams(ctx, store, configDir, now)
}

func ensureEmptyFormataBuilderStreams(ctx context.Context, store Store, configDir string, now func() time.Time) error {
	if store == nil {
		return nil
	}
	existing, err := store.ListFormataBuilderStreams(ctx)
	if err != nil {
		return fmt.Errorf("list formata streams: %w", err)
	}
	if len(existing) > 0 {
		return nil
	}

	pack, err := openCatalogTemplatePack(configDir)
	if err != nil {
		return err
	}

	updatedAt := time.Now().UTC()
	if now != nil {
		updatedAt = now().UTC()
	}
	for _, tmpl := range pack.Streams {
		creatorID := platformAdminStreamUserID()
		if _, saveErr := store.SaveFormataBuilderStream(ctx, FormataBuilderStream{
			Stream:          string(tmpl.Body),
			UpdatedAt:       updatedAt,
			CreatedByUserID: creatorID,
			UpdatedByUserID: creatorID,
		}); saveErr != nil {
			return fmt.Errorf("seed formata stream %s: %w", filepath.Base(tmpl.Path), saveErr)
		}
	}
	return nil
}
