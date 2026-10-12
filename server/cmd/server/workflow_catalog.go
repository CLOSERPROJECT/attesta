package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// workflowCatalogLoad is the full catalog load result, including cache metadata
// used by Server.workflowCatalog. Seed callers use loadWorkflowCatalog instead.
type workflowCatalogLoad struct {
	catalog   map[string]RuntimeConfig
	modTimes  map[string]time.Time
	taxRev    int64
	cacheable bool
}

type workflowCatalogSources struct {
	streams  []FormataBuilderStream
	paths    []string
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

// loadWorkflowCatalog loads workflow runtime configs from Formata builder streams
// when present, otherwise from configDir YAML files. It applies Store-backed
// stream categorization. Seed calls this directly.
func loadWorkflowCatalog(ctx context.Context, store Store, configDir string) (map[string]RuntimeConfig, error) {
	loaded, err := loadWorkflowCatalogResult(ctx, store, configDir)
	if err != nil {
		return nil, err
	}
	return loaded.catalog, nil
}

func loadWorkflowCatalogResult(ctx context.Context, store Store, configDir string) (workflowCatalogLoad, error) {
	taxRev, taxErr := catalogTaxonomyRevision(ctx, store)
	if taxErr != nil {
		return workflowCatalogLoad{}, taxErr
	}
	sources, err := inspectWorkflowCatalogSources(ctx, store, configDir)
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

// workflowCatalog is a mutex/cache adapter over loadWorkflowCatalogResult.
func (s *Server) workflowCatalog() (map[string]RuntimeConfig, error) {
	s.configMu.Lock()
	defer s.configMu.Unlock()

	ctx := context.Background()
	taxRev, taxErr := catalogTaxonomyRevision(ctx, s.store)
	if taxErr != nil {
		return nil, taxErr
	}
	sources, err := inspectWorkflowCatalogSources(ctx, s.store, s.configDir)
	if err != nil {
		return nil, err
	}
	if s.catalog != nil && sameCatalogModTimes(s.catalogModTime, sources.modTimes) && s.catalogTaxonomyRev == taxRev {
		return cloneWorkflowCatalog(s.catalog), nil
	}

	loaded, err := loadWorkflowCatalogResult(ctx, s.store, s.configDir)
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

func inspectWorkflowCatalogSources(ctx context.Context, store Store, configDir string) (workflowCatalogSources, error) {
	if store != nil {
		streams, err := store.ListFormataBuilderStreams(ctx)
		if err != nil {
			return workflowCatalogSources{}, fmt.Errorf("list formata streams: %w", err)
		}
		if len(streams) > 0 {
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
	}

	dir := strings.TrimSpace(configDir)
	if dir == "" {
		dir = "config"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return workflowCatalogSources{}, fmt.Errorf("config dir not found: %w", err)
	}

	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !isWorkflowCatalogConfigFile(name) {
			continue
		}
		paths = append(paths, filepath.Join(dir, name))
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return workflowCatalogSources{}, errors.New("workflow config catalog is empty")
	}

	modTimes := make(map[string]time.Time, len(paths))
	for _, path := range paths {
		info, statErr := os.Stat(path)
		if statErr != nil {
			return workflowCatalogSources{}, fmt.Errorf("config stat failed for %s: %w", path, statErr)
		}
		modTimes[path] = info.ModTime()
	}
	return workflowCatalogSources{paths: paths, modTimes: modTimes}, nil
}

func buildWorkflowCatalog(ctx context.Context, store Store, sources workflowCatalogSources) (map[string]RuntimeConfig, bool, error) {
	if len(sources.streams) > 0 {
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

	catalog := make(map[string]RuntimeConfig, len(sources.paths))
	cacheable := true
	for _, path := range sources.paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, false, fmt.Errorf("read config %s: %w", path, readErr)
		}
		cfg, parseErr := parseRuntimeConfigData(filepath.Base(path), data)
		if parseErr != nil {
			return nil, false, parseErr
		}
		if resolveErr := resolveCatalogStreamCategorization(ctx, store, &cfg); resolveErr != nil {
			cacheable = false
		}
		key := strings.TrimSpace(strings.TrimSuffix(filepath.Base(path), filepath.Ext(path)))
		if key == "" {
			return nil, false, fmt.Errorf("workflow key is empty for %s", filepath.Base(path))
		}
		if _, exists := catalog[key]; exists {
			return nil, false, fmt.Errorf("duplicate workflow key %q", key)
		}
		catalog[key] = cfg
	}
	return catalog, cacheable, nil
}
