package main

import (
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func TestCatalogStreamDisplayName(t *testing.T) {
	if got := catalogStreamDisplayName("Procurement", 0); got != "Procurement — Pilot" {
		t.Fatalf("got %q", got)
	}
	if got := catalogStreamDisplayName("Procurement", 2); got != "Procurement — Extended" {
		t.Fatalf("got %q", got)
	}
}

func TestApplyCatalogStreamCategoryAndNameStripsTrailingDupe(t *testing.T) {
	// Formata-shaped: roles before workflow; orphan trailing subCategorySlug without final newline
	in := strings.TrimRight(`roles:
  - orgSlug: org1
    slug: dep1
    name: Department 1
workflow:
  name: Old Name
  categorySlug: supply-chain
  subCategorySlug: procurement
  steps:
    - id: "1"
      title: Step 1
      order: 1
      organization: org1
      substeps:
        - id: "1.1"
          title: Input
          order: 1
          roles: ["dep1"]
          inputKey: value
          inputType: formata
          schema:
            type: object
organizations:
  - slug: org1
    name: Organization 1
  subCategorySlug: supplier-onboarding`, "\n") // no trailing newline on last key

	out, err := applyCatalogStreamCategoryAndName(in, "recycling-and-recovery", "photovoltaic-panels", "Photovoltaic Panels — Pilot")
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if strings.Count(out, "categorySlug:") != 1 || strings.Count(out, "subCategorySlug:") != 1 {
		t.Fatalf("duplicate keys remain:\n%s", out)
	}
	if !strings.Contains(out, "categorySlug: recycling-and-recovery") || !strings.Contains(out, "subCategorySlug: photovoltaic-panels") {
		t.Fatalf("missing new pair:\n%s", out)
	}
	if !strings.Contains(out, "name: Photovoltaic Panels — Pilot") {
		t.Fatalf("workflow name not set:\n%s", out)
	}
	if strings.Contains(out, "name: Department 1") == false {
		t.Fatal("role name must be preserved")
	}
	if _, err := parseRuntimeConfigData("clone.yaml", []byte(out)); err != nil {
		t.Fatalf("parseRuntimeConfigData: %v", err)
	}
}

func TestTaxonomyLeavesFromTree(t *testing.T) {
	leaves := taxonomyLeavesFromTree([]TaxonomyCategoryNode{
		{Slug: "a", Name: "A", SubCategories: []TaxonomySubCategoryNode{
			{Slug: "a1", Name: "A1"},
			{Slug: "a2", Name: "A2"},
		}},
		{Slug: "b", Name: "B"}, // no subs
	})
	if len(leaves) != 2 {
		t.Fatalf("len=%d want 2", len(leaves))
	}
	if leaves[0].CategorySlug != "a" || leaves[0].SubCategorySlug != "a1" || leaves[0].SubCategoryName != "A1" {
		t.Fatalf("leaf0=%+v", leaves[0])
	}
}

func TestSeedCatalogStreamsFillsEveryLeafAndReplaces(t *testing.T) {
	store := NewMemoryStore()
	if err := store.ReplaceTaxonomy(t.Context(),
		[]Category{{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1},
			{Slug: "compliance-and-quality", Name: "Compliance", Icon: "quality-control", SortOrder: 2}},
		[]SubCategory{
			{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1},
			{CategorySlug: "compliance-and-quality", Slug: "inspection", Name: "Inspection", Icon: "inspection-workflow", SortOrder: 1},
		},
	); err != nil {
		t.Fatalf("ReplaceTaxonomy: %v", err)
	}

	tmpl := minimalCategorizedWorkflowYAML("")
	saved, err := store.SaveFormataBuilderStream(t.Context(), FormataBuilderStream{Stream: tmpl})
	if err != nil {
		t.Fatalf("SaveFormataBuilderStream: %v", err)
	}
	oldProcessID := primitive.NewObjectID()
	store.SeedProcess(Process{
		ID:          oldProcessID,
		WorkflowKey: saved.ID.Hex(),
		Status:      processStatusActive,
		CreatedAt:   time.Now().UTC(),
		Progress:    map[string]ProcessStep{},
	})

	server := &Server{store: store, configDir: t.TempDir()}
	rng := rand.New(rand.NewSource(1))
	res, err := seedCatalogStreams(t.Context(), server, rng)
	if err != nil {
		t.Fatalf("seedCatalogStreams: %v", err)
	}
	if res.Leaves != 2 {
		t.Fatalf("leaves=%d want 2", res.Leaves)
	}
	if res.Streams < 2 || res.Streams > 6 {
		t.Fatalf("streams=%d want 2..6", res.Streams)
	}

	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(streams) != res.Streams {
		t.Fatalf("listed=%d result=%d", len(streams), res.Streams)
	}
	for _, s := range streams {
		if s.ID == saved.ID {
			t.Fatal("old formata id survived wipe")
		}
	}
	if _, err := store.LoadProcessByID(t.Context(), oldProcessID); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("LoadProcessByID after DeleteWorkflowData: %v, want ErrNoDocuments", err)
	}
	recent, err := store.ListRecentProcessesByWorkflow(t.Context(), saved.ID.Hex(), 10)
	if err != nil {
		t.Fatalf("ListRecentProcessesByWorkflow: %v", err)
	}
	if len(recent) != 0 {
		t.Fatalf("expected no processes for old key, got %d", len(recent))
	}

	counts := map[string]int{}
	for _, s := range streams {
		cfg, err := parseRuntimeConfigData(s.ID.Hex(), []byte(s.Stream))
		if err != nil {
			t.Fatalf("parse %s: %v", s.ID.Hex(), err)
		}
		key := cfg.Workflow.CategorySlug + "/" + cfg.Workflow.SubCategorySlug
		counts[key]++
		if !strings.Contains(cfg.Workflow.Name, " — ") {
			t.Fatalf("name %q missing variant", cfg.Workflow.Name)
		}
	}
	for _, leaf := range []string{"supply-chain/procurement", "compliance-and-quality/inspection"} {
		n := counts[leaf]
		if n < 1 || n > 3 {
			t.Fatalf("%s count=%d", leaf, n)
		}
	}

	res2, err := seedCatalogStreams(t.Context(), server, rand.New(rand.NewSource(2)))
	if err != nil {
		t.Fatalf("second seed: %v", err)
	}
	streams2, _ := store.ListFormataBuilderStreams(t.Context())
	if len(streams2) != res2.Streams {
		t.Fatalf("after replace listed=%d", len(streams2))
	}
	ids := map[primitive.ObjectID]bool{}
	for _, s := range streams {
		ids[s.ID] = true
	}
	for _, s := range streams2 {
		if ids[s.ID] {
			t.Fatal("first-run id survived second seed")
		}
	}
}

func TestSeedCatalogStreamsUsesConfigDirWhenFormataEmpty(t *testing.T) {
	store := NewMemoryStore()
	_ = store.ReplaceTaxonomy(t.Context(),
		[]Category{{Slug: "supply-chain", Name: "Supply Chain", Icon: "batch-traceability", SortOrder: 1}},
		[]SubCategory{{CategorySlug: "supply-chain", Slug: "procurement", Name: "Procurement", Icon: "procurement-workflow", SortOrder: 1}},
	)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644); err != nil {
		t.Fatal(err)
	}
	server := &Server{store: store, configDir: dir}
	res, err := seedCatalogStreams(t.Context(), server, rand.New(rand.NewSource(3)))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if res.Streams < 1 {
		t.Fatal("expected at least one stream from file template")
	}
}

func TestSeedCatalogStreamsErrorsWithoutTaxonomy(t *testing.T) {
	store := NewMemoryStore()
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML("")), 0o644)
	server := &Server{store: store, configDir: dir}
	if _, err := seedCatalogStreams(t.Context(), server, rand.New(rand.NewSource(1))); err == nil {
		t.Fatal("expected error for empty taxonomy")
	}
}
