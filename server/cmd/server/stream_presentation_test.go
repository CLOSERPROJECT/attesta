package main

import (
	"strings"
	"testing"
)

func TestStreamPresentationOnlyChange(t *testing.T) {
	base := workflowStreamYAML("Base stream")

	t.Run("name only", func(t *testing.T) {
		ok, err := streamPresentationOnlyChange(base, workflowStreamYAML("Renamed stream"))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !ok {
			t.Fatal("expected presentation-only")
		}
	})

	t.Run("description only", func(t *testing.T) {
		updated := strings.Replace(base, `description: "demo"`, `description: "new blurb"`, 1)
		ok, err := streamPresentationOnlyChange(base, updated)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !ok {
			t.Fatal("expected presentation-only")
		}
	})

	t.Run("categorization only", func(t *testing.T) {
		updated := strings.Replace(base, "  name:", "  categorySlug: supply-chain\n  subCategorySlug: procurement\n  name:", 1)
		ok, err := streamPresentationOnlyChange(base, updated)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !ok {
			t.Fatal("expected presentation-only")
		}
	})

	t.Run("name description and category", func(t *testing.T) {
		updated := workflowStreamYAML("All presentation")
		updated = strings.Replace(updated, "  name:", "  categorySlug: supply-chain\n  subCategorySlug: procurement\n  name:", 1)
		updated = strings.Replace(updated, `description: "demo"`, `description: "other"`, 1)
		ok, err := streamPresentationOnlyChange(base, updated)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !ok {
			t.Fatal("expected presentation-only")
		}
	})

	t.Run("step title change is not presentation-only", func(t *testing.T) {
		updated := strings.Replace(base, `title: "Step 1"`, `title: "Step changed"`, 1)
		ok, err := streamPresentationOnlyChange(base, updated)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if ok {
			t.Fatal("expected shape change")
		}
	})

	t.Run("invalid yaml fails closed", func(t *testing.T) {
		ok, err := streamPresentationOnlyChange(base, "not: valid: workflow")
		if err == nil {
			t.Fatal("expected error")
		}
		if ok {
			t.Fatal("expected ok=false")
		}
	})

	t.Run("empty fails closed", func(t *testing.T) {
		ok, err := streamPresentationOnlyChange(base, "  ")
		if err == nil {
			t.Fatal("expected error")
		}
		if ok {
			t.Fatal("expected ok=false")
		}
	})
}

func TestStreamPresentationOnlyChangeOrgAndRoles(t *testing.T) {
	base := workflowStreamYAML("Base stream")
	t.Run("organization change is not presentation-only", func(t *testing.T) {
		updated := strings.Replace(base, `organization: "org1"`, `organization: "org2"`, 1)
		updated = strings.Replace(updated, `slug: "org1"`, `slug: "org2"`, 1)
		ok, err := streamPresentationOnlyChange(base, updated)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if ok {
			t.Fatal("expected shape change for organization")
		}
	})
	t.Run("substep roles change is not presentation-only", func(t *testing.T) {
		updated := strings.Replace(base, `roles: ["dep1"]`, `roles: ["dep1", "dep2"]`, 1)
		ok, err := streamPresentationOnlyChange(base, updated)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if ok {
			t.Fatal("expected shape change for roles")
		}
	})
}
