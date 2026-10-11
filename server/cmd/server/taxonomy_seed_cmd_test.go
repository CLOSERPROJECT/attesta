package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestCategoriesSeedPathFromArgsResolved(t *testing.T) {
	t.Setenv("CATEGORIES_SEED", "")
	t.Setenv("WORKFLOW_CONFIG", "config/workflow.yaml")
	t.Setenv("WORKFLOW_CONFIG_DIR", "")

	got := resolveCategoriesSeedPath(categoriesSeedPathFromArgs(nil), workflowConfigDir())
	want := filepath.Join("config", "categories.yaml")
	if got != want {
		t.Fatalf("default path = %q, want %q", got, want)
	}

	got = resolveCategoriesSeedPath(categoriesSeedPathFromArgs([]string{"--file", "/tmp/custom.yaml"}), workflowConfigDir())
	if got != "/tmp/custom.yaml" {
		t.Fatalf("--file path = %q", got)
	}

	got = resolveCategoriesSeedPath(categoriesSeedPathFromArgs([]string{"--file=/tmp/eq.yaml"}), workflowConfigDir())
	if got != "/tmp/eq.yaml" {
		t.Fatalf("--file= path = %q", got)
	}

	t.Setenv("CATEGORIES_SEED", "/env/categories.yaml")
	got = resolveCategoriesSeedPath(categoriesSeedPathFromArgs(nil), workflowConfigDir())
	if got != "/env/categories.yaml" {
		t.Fatalf("env path = %q", got)
	}
}

func TestWithSeedSessionSeedsCategories(t *testing.T) {
	store := NewMemoryStore()
	path := writeTempTaxonomySeed(t, `
categories:
  - slug: supply-chain
    name: Supply Chain
    icon: batch-traceability
    sortOrder: 1
`)
	err := withSeedSession(t.Context(), []string{"--file", path}, "", func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, nil, func(sess seedSession) error {
		seedPath := resolveCategoriesSeedPath(sess.CategoriesPath, sess.ConfigDir)
		return seedTaxonomyFromFile(t.Context(), sess.Store, seedPath)
	})
	if err != nil {
		t.Fatalf("withSeedSession: %v", err)
	}
	cats, err := store.ListCategories(t.Context())
	if err != nil {
		t.Fatalf("ListCategories: %v", err)
	}
	if len(cats) != 1 || cats[0].Slug != "supply-chain" {
		t.Fatalf("cats = %#v", cats)
	}
}

func TestWithSeedSessionCategoriesErrors(t *testing.T) {
	if err := withSeedSession(t.Context(), nil, "", nil, nil, func(seedSession) error { return nil }); err == nil {
		t.Fatal("expected nil opener error")
	}
	openErr := errors.New("open failed")
	if err := withSeedSession(t.Context(), nil, "", func(context.Context) (Store, func(), error) {
		return nil, nil, openErr
	}, nil, func(seedSession) error { return nil }); !errors.Is(err, openErr) {
		t.Fatalf("err = %v, want openErr", err)
	}

	store := NewMemoryStore()
	if err := withSeedSession(t.Context(), []string{"--file", filepath.Join(t.TempDir(), "missing.yaml")}, "", func(context.Context) (Store, func(), error) {
		return store, nil, nil
	}, nil, func(sess seedSession) error {
		seedPath := resolveCategoriesSeedPath(sess.CategoriesPath, sess.ConfigDir)
		return seedTaxonomyFromFile(t.Context(), sess.Store, seedPath)
	}); err == nil {
		t.Fatal("expected seed file error")
	}
}

func TestOpenMongoTaxonomyStoreConnectError(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://127.0.0.1:1/?connectTimeoutMS=1&serverSelectionTimeoutMS=1")
	_, _, err := openMongoTaxonomyStore(t.Context())
	if err == nil {
		t.Fatal("expected openMongoTaxonomyStore error")
	}
}
