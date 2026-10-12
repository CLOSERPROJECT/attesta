package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsCatalogTemplateFile(t *testing.T) {
	if !isCatalogTemplateFile("workflow.yaml") {
		t.Fatal("workflow.yaml should be a pack template")
	}
	if isCatalogTemplateFile("categories.yaml") {
		t.Fatal("categories.yaml should be excluded")
	}
	if isCatalogTemplateFile("feragame-pilot.yaml") {
		t.Fatal("feragame-pilot.yaml must not join the taxonomy pack")
	}
	if isCatalogTemplateFile("feragame-pilot.yml") {
		t.Fatal("feragame-pilot.yml must not join the taxonomy pack")
	}
	if isCatalogTemplateFile("readme.md") {
		t.Fatal("non-yaml should be excluded")
	}
}

func TestOpenCatalogTemplatePackLoadsStreamsAndCategoriesPath(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(dir, "workflow.yaml"), "Main", "string")
	writeWorkflowConfig(t, filepath.Join(dir, "secondary.yaml"), "Secondary", "string")
	writeWorkflowConfig(t, filepath.Join(dir, "feragame-pilot.yaml"), "Feragame pilot", "string")
	if err := os.WriteFile(filepath.Join(dir, "categories.yaml"), []byte("categories: []\n"), 0o644); err != nil {
		t.Fatalf("write categories.yaml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.txt"), []byte("ignore"), 0o644); err != nil {
		t.Fatalf("write note.txt: %v", err)
	}

	pack, err := openCatalogTemplatePack(dir)
	if err != nil {
		t.Fatalf("openCatalogTemplatePack: %v", err)
	}
	if pack.Dir != dir {
		t.Fatalf("Dir = %q, want %q", pack.Dir, dir)
	}
	if pack.CategoriesPath != filepath.Join(dir, "categories.yaml") {
		t.Fatalf("CategoriesPath = %q", pack.CategoriesPath)
	}
	if len(pack.Streams) != 2 {
		t.Fatalf("Streams = %d, want 2 (feragame-pilot excluded)", len(pack.Streams))
	}
	if filepath.Base(pack.Streams[0].Path) != "secondary.yaml" {
		t.Fatalf("first stream path = %q, want secondary.yaml (sorted)", pack.Streams[0].Path)
	}
	if !strings.Contains(string(pack.Streams[0].Body), "Secondary") {
		t.Fatalf("secondary body missing name: %s", pack.Streams[0].Body)
	}
	for _, s := range pack.Streams {
		if strings.Contains(filepath.Base(s.Path), "feragame") {
			t.Fatalf("pack included named pilot %q", s.Path)
		}
	}
}

func TestOpenCatalogTemplatePackDefaultDirMatchesCategoriesSeed(t *testing.T) {
	if got := resolveCategoriesSeedPath("", ""); got != filepath.Join("config", "categories.yaml") {
		t.Fatalf("resolveCategoriesSeedPath(\"\", \"\") = %q", got)
	}
}

func TestOpenCatalogTemplatePackErrors(t *testing.T) {
	t.Run("missing dir", func(t *testing.T) {
		_, err := openCatalogTemplatePack(filepath.Join(t.TempDir(), "missing"))
		if err == nil || !strings.Contains(err.Error(), "config dir not found") {
			t.Fatalf("err = %v", err)
		}
	})
	t.Run("empty configDir defaults missing", func(t *testing.T) {
		cwd := t.TempDir()
		t.Chdir(cwd)
		_, err := openCatalogTemplatePack("")
		if err == nil || !strings.Contains(err.Error(), "config dir not found") {
			t.Fatalf("err = %v, want default config missing", err)
		}
	})
	t.Run("empty pack", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "categories.yaml"), []byte("categories: []\n"), 0o644); err != nil {
			t.Fatalf("write categories.yaml: %v", err)
		}
		_, err := openCatalogTemplatePack(dir)
		if err == nil || !strings.Contains(err.Error(), "catalog template pack is empty") {
			t.Fatalf("err = %v", err)
		}
	})
}
