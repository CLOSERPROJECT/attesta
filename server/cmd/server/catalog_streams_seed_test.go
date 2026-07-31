package main

import (
	"strings"
	"testing"
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
