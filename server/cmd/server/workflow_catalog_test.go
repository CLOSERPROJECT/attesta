package main

import (
	"context"
	"testing"
)

func TestCatalogTaxonomyRevision(t *testing.T) {
	if revision, err := catalogTaxonomyRevision(context.Background(), nil); err != nil || revision != 0 {
		t.Fatalf("nil store = (%d, %v)", revision, err)
	}

	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(context.Background(), []Category{{Slug: "c", Icon: "weee"}}, nil); err != nil {
		t.Fatal(err)
	}
	want := 1
	if revision, err := catalogTaxonomyRevision(context.Background(), store); err != nil || revision != int64(want) {
		t.Fatalf("store revision = (%d, %v), want %d", revision, err, want)
	}

	if revision, err := (&Server{}).catalogTaxonomyRevision(); err != nil || revision != 0 {
		t.Fatalf("server without store = (%d, %v)", revision, err)
	}
	if revision, err := (&Server{store: store}).catalogTaxonomyRevision(); err != nil || revision != int64(want) {
		t.Fatalf("server revision = (%d, %v), want %d", revision, err, want)
	}
	var server *Server
	if revision, err := server.catalogTaxonomyRevision(); err != nil || revision != 0 {
		t.Fatalf("nil server = (%d, %v)", revision, err)
	}
}
