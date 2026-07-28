package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestTaxonomySeedPathFromArgs(t *testing.T) {
	t.Setenv("CATEGORIES_SEED", "")
	t.Setenv("WORKFLOW_CONFIG", "config/workflow.yaml")
	t.Setenv("WORKFLOW_CONFIG_DIR", "")

	got := taxonomySeedPathFromArgs(nil)
	want := filepath.Join("config", "categories.yaml")
	if got != want {
		t.Fatalf("default path = %q, want %q", got, want)
	}

	got = taxonomySeedPathFromArgs([]string{"--file", "/tmp/custom.yaml"})
	if got != "/tmp/custom.yaml" {
		t.Fatalf("--file path = %q", got)
	}

	got = taxonomySeedPathFromArgs([]string{"--file=/tmp/eq.yaml"})
	if got != "/tmp/eq.yaml" {
		t.Fatalf("--file= path = %q", got)
	}

	t.Setenv("CATEGORIES_SEED", "/env/categories.yaml")
	got = taxonomySeedPathFromArgs(nil)
	if got != "/env/categories.yaml" {
		t.Fatalf("env path = %q", got)
	}
}

func TestSeedCategoriesWithStoreOpener(t *testing.T) {
	store := NewMemoryStore()
	path := writeTempTaxonomySeed(t, `
categories:
  - slug: supply-chain
    name: Supply Chain
    icon: batch-traceability
    sortOrder: 1
`)
	err := seedCategoriesWithStoreOpener(t.Context(), []string{"--file", path}, func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	})
	if err != nil {
		t.Fatalf("seedCategoriesWithStoreOpener: %v", err)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 1 || cats[0].Slug != "supply-chain" {
		t.Fatalf("cats = %#v", cats)
	}
}

func TestSeedCategoriesWithStoreOpenerErrors(t *testing.T) {
	if err := seedCategoriesWithStoreOpener(t.Context(), nil, nil); err == nil {
		t.Fatal("expected nil opener error")
	}
	openErr := errors.New("open failed")
	if err := seedCategoriesWithStoreOpener(t.Context(), nil, func(context.Context) (Store, func(), error) {
		return nil, nil, openErr
	}); !errors.Is(err, openErr) {
		t.Fatalf("err = %v, want openErr", err)
	}

	store := NewMemoryStore()
	if err := seedCategoriesWithStoreOpener(t.Context(), []string{"--file", filepath.Join(t.TempDir(), "missing.yaml")}, func(context.Context) (Store, func(), error) {
		return store, nil, nil
	}); err == nil {
		t.Fatal("expected seed file error")
	}
}

func TestRunSeedCategoriesCommandUsesOpener(t *testing.T) {
	// Bad URI should fail during mongo connect without needing a live server.
	t.Setenv("MONGODB_URI", "mongodb://127.0.0.1:1/?connectTimeoutMS=1&serverSelectionTimeoutMS=1")
	err := runSeedCategoriesCommand(t.Context(), []string{"--file", writeTempTaxonomySeed(t, "categories: []\n")})
	if err == nil {
		t.Fatal("expected mongo connect/ping failure")
	}
}

func TestOpenMongoTaxonomyStoreConnectError(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://127.0.0.1:1/?connectTimeoutMS=1&serverSelectionTimeoutMS=1")
	_, _, err := openMongoTaxonomyStore(t.Context())
	if err == nil {
		t.Fatal("expected openMongoTaxonomyStore error")
	}
}
