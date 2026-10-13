package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

// Explicit allow-list: each entry is one FormataBuilderStream with a stable
// ObjectID. Keep pilots out of openCatalogTemplatePack / isCatalogTemplateFile
// so the next Stream pilot cannot accidentally become N taxonomy clones.
type namedPilotStreamSpec struct {
	Basename string
	IDSeed   string
}

func namedPilotStreamSpecs() []namedPilotStreamSpec {
	return []namedPilotStreamSpec{{
		Basename: "feragame-pilot.yaml",
		IDSeed:   "named-pilot-stream:feragame-pilot",
	}}
}

func namedPilotStreamURLKey(idSeed string) string {
	return stableObjectID(strings.TrimSpace(idSeed)).Hex()
}

type namedPilotStreamsSeedResult struct {
	Streams int
	Keys    []string
}

func seedNamedPilotStreams(ctx context.Context, store Store, configDir string, now time.Time) (namedPilotStreamsSeedResult, error) {
	var zero namedPilotStreamsSeedResult
	if store == nil {
		return zero, fmt.Errorf("named pilot seed: store is required")
	}
	dir := strings.TrimSpace(configDir)
	if dir == "" {
		dir = "config"
	}
	creator := platformAdminStreamUserID()
	keys := make([]string, 0, len(namedPilotStreamSpecs()))
	for _, spec := range namedPilotStreamSpecs() {
		path := filepath.Join(dir, spec.Basename)
		body, err := os.ReadFile(path)
		if err != nil {
			return zero, fmt.Errorf("named pilot seed: read %s: %w", path, err)
		}
		if _, err := parseRuntimeConfigData(spec.Basename, body); err != nil {
			return zero, fmt.Errorf("named pilot seed: parse %s: %w", spec.Basename, err)
		}
		id := stableObjectID(spec.IDSeed)
		doc := FormataBuilderStream{
			ID:              id,
			Stream:          string(body),
			UpdatedAt:       now,
			CreatedByUserID: creator,
			UpdatedByUserID: creator,
		}
		existing, err := store.LoadFormataBuilderStreamByID(ctx, id)
		switch {
		case err == nil && existing != nil:
			doc.CreatedByUserID = existing.CreatedByUserID
			if strings.TrimSpace(doc.CreatedByUserID) == "" {
				doc.CreatedByUserID = creator
			}
			if _, updErr := store.UpdateFormataBuilderStream(ctx, doc); updErr != nil {
				return zero, fmt.Errorf("named pilot seed: update %s: %w", spec.Basename, updErr)
			}
		case errors.Is(err, mongo.ErrNoDocuments):
			if _, saveErr := store.SaveFormataBuilderStream(ctx, doc); saveErr != nil {
				return zero, fmt.Errorf("named pilot seed: save %s: %w", spec.Basename, saveErr)
			}
		default:
			return zero, fmt.Errorf("named pilot seed: load %s: %w", spec.Basename, err)
		}
		keys = append(keys, id.Hex())
	}
	return namedPilotStreamsSeedResult{Streams: len(keys), Keys: keys}, nil
}
