package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Not the runtime Formata catalog.
type catalogTemplatePack struct {
	Dir            string
	CategoriesPath string
	Streams        []catalogTemplateStream
}

type catalogTemplateStream struct {
	Path string
	Body []byte
}

// Pack templates only — named pilots (feragame-pilot.yaml) are denylisted so
// seedCatalogStreams cannot multiply one pilot into N taxonomy clones.
func isCatalogTemplateFile(name string) bool {
	if !isWorkflowCatalogConfigFile(name) {
		return false
	}
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	switch base {
	case "feragame-pilot.yaml", "feragame-pilot.yml":
		return false
	}
	return true
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
		if !isCatalogTemplateFile(name) {
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
