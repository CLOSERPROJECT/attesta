package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// On-disk install/seed pack (Stream YAML + categories path), not the runtime Formata catalog.
type catalogTemplatePack struct {
	Dir            string
	CategoriesPath string
	Streams        []catalogTemplateStream
}

type catalogTemplateStream struct {
	Path string
	Body []byte
}

func openCatalogTemplatePack(configDir string) (catalogTemplatePack, error) {
	dir := strings.TrimSpace(configDir)
	if dir == "" {
		dir = "config"
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return catalogTemplatePack{}, fmt.Errorf("config dir not found: %w", err)
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
		return catalogTemplatePack{}, fmt.Errorf("catalog template pack is empty")
	}

	streams := make([]catalogTemplateStream, 0, len(paths))
	for _, path := range paths {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return catalogTemplatePack{}, fmt.Errorf("read config %s: %w", path, readErr)
		}
		streams = append(streams, catalogTemplateStream{Path: path, Body: data})
	}

	return catalogTemplatePack{
		Dir:            dir,
		CategoriesPath: resolveCategoriesSeedPath("", dir),
		Streams:        streams,
	}, nil
}
