package main

import (
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

func TestOpenMongoStoreConnectError(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://127.0.0.1:1/?connectTimeoutMS=1&serverSelectionTimeoutMS=1")
	_, _, _, err := openMongoStore(t.Context())
	if err == nil {
		t.Fatal("expected openMongoStore error")
	}
}
