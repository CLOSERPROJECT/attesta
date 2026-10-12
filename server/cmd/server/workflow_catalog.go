package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Callers may treat this as an empty catalog rather than a hard failure.
var errFormataWorkflowCatalogEmpty = errors.New("formata workflow catalog is empty")

type workflowCatalogLoad struct {
	catalog   map[string]RuntimeConfig
	modTimes  map[string]time.Time
	taxRev    int64
	cacheable bool
}

type workflowCatalogSources struct {
	streams  []FormataBuilderStream
	modTimes map[string]time.Time
}

func workflowCatalogModTime(stream FormataBuilderStream) time.Time {
	if !stream.UpdatedAt.IsZero() {
		return stream.UpdatedAt
	}
	if !stream.ID.IsZero() {
		return stream.ID.Timestamp()
	}
	return time.Time{}
}

func catalogTaxonomyRevision(ctx context.Context, store Store) (int64, error) {
	if store == nil {
		return 0, nil
	}
	return store.TaxonomyRevision(ctx)
}

func (s *Server) catalogTaxonomyRevision() (int64, error) {
	if s == nil {
		return 0, nil
	}
	return catalogTaxonomyRevision(context.Background(), s.store)
}

// Formata builder Streams only; empty catalog fails closed.
func loadWorkflowCatalog(ctx context.Context, store Store) (map[string]RuntimeConfig, error) {
	loaded, err := loadWorkflowCatalogResult(ctx, store)
	if err != nil {
		return nil, err
	}
	return loaded.catalog, nil
}

func loadWorkflowCatalogResult(ctx context.Context, store Store) (workflowCatalogLoad, error) {
	taxRev, taxErr := catalogTaxonomyRevision(ctx, store)
	if taxErr != nil {
		return workflowCatalogLoad{}, taxErr
	}
	sources, err := inspectWorkflowCatalogSources(ctx, store)
	if err != nil {
		return workflowCatalogLoad{}, err
	}
	catalog, cacheable, err := buildWorkflowCatalog(ctx, store, sources)
	if err != nil {
		return workflowCatalogLoad{}, err
	}
	return workflowCatalogLoad{
		catalog:   catalog,
		modTimes:  sources.modTimes,
		taxRev:    taxRev,
		cacheable: cacheable,
	}, nil
}

func (s *Server) workflowCatalog() (map[string]RuntimeConfig, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	ctx := context.Background()
	taxRev, taxErr := catalogTaxonomyRevision(ctx, s.store)
	if taxErr != nil {
		return nil, taxErr
	}
	sources, err := inspectWorkflowCatalogSources(ctx, s.store)
	if err != nil {
		return nil, err
	}
	if s.catalog != nil && sameCatalogModTimes(s.catalogModTime, sources.modTimes) && s.catalogTaxonomyRev == taxRev {
		return cloneWorkflowCatalog(s.catalog), nil
	}

	loaded, err := loadWorkflowCatalogResult(ctx, s.store)
	if err != nil {
		return nil, err
	}
	if loaded.cacheable {
		s.catalog = loaded.catalog
		s.catalogModTime = loaded.modTimes
		s.catalogTaxonomyRev = loaded.taxRev
	} else {
		s.catalog = nil
		s.catalogModTime = nil
	}
	return cloneWorkflowCatalog(loaded.catalog), nil
}

// Disk YAML is never a runtime fallback; empty Formata fails closed.
func inspectWorkflowCatalogSources(ctx context.Context, store Store) (workflowCatalogSources, error) {
	if store == nil {
		return workflowCatalogSources{}, errFormataWorkflowCatalogEmpty
	}
	streams, err := store.ListFormataBuilderStreams(ctx)
	if err != nil {
		return workflowCatalogSources{}, fmt.Errorf("list formata streams: %w", err)
	}
	if len(streams) == 0 {
		return workflowCatalogSources{}, errFormataWorkflowCatalogEmpty
	}
	modTimes := make(map[string]time.Time, len(streams))
	for _, stream := range streams {
		if stream.ID.IsZero() {
			return workflowCatalogSources{}, errors.New("formata stream id is empty")
		}
		key := stream.ID.Hex()
		modTimes[key] = workflowCatalogModTime(stream)
	}
	return workflowCatalogSources{streams: streams, modTimes: modTimes}, nil
}

func buildWorkflowCatalog(ctx context.Context, store Store, sources workflowCatalogSources) (map[string]RuntimeConfig, bool, error) {
	if len(sources.streams) == 0 {
		return nil, false, errFormataWorkflowCatalogEmpty
	}
	catalog := make(map[string]RuntimeConfig, len(sources.streams))
	cacheable := true
	for _, stream := range sources.streams {
		if stream.ID.IsZero() {
			return nil, false, errors.New("formata stream id is empty")
		}
		key := stream.ID.Hex()
		cfg, parseErr := parseRuntimeConfigData("stream "+key, []byte(stream.Stream))
		if parseErr != nil {
			return nil, false, parseErr
		}
		if resolveErr := resolveCatalogStreamCategorization(ctx, store, &cfg); resolveErr != nil {
			cacheable = false
		}
		catalog[key] = cfg
	}
	return catalog, cacheable, nil
}
