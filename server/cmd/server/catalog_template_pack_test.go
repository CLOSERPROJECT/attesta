package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCatalogTemplatePackLoadsStreamsAndCategoriesPath(t *testing.T) {
	dir := t.TempDir()
	writeWorkflowConfig(t, filepath.Join(dir, "workflow.yaml"), "Main", "string")
	writeWorkflowConfig(t, filepath.Join(dir, "secondary.yaml"), "Secondary", "string")
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
		t.Fatalf("Streams = %d, want 2", len(pack.Streams))
	}
	if filepath.Base(pack.Streams[0].Path) != "secondary.yaml" {
		t.Fatalf("first stream path = %q, want secondary.yaml (sorted)", pack.Streams[0].Path)
	}
	if !strings.Contains(string(pack.Streams[0].Body), "Secondary") {
		t.Fatalf("secondary body missing name: %s", pack.Streams[0].Body)
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
