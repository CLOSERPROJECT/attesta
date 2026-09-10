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
	t.Run("catalog org role display churn with category add is presentation-only", func(t *testing.T) {
		// Formata App.build() rewrites organizations/roles from the live catalog on every
		// save (order + display names). Legacy uncategorized streams must add categories
		// before save is allowed; that should not be treated as a destructive rewrite.
		stored := formataLegacyUncategorizedYAML()
		saved := strings.Replace(stored, "  name: Legacy stream\n", "  categorySlug: materials\n  subCategorySlug: metals\n  name: Legacy stream\n", 1)
		saved = strings.Replace(saved, `organizations:
  - name: Org B
    slug: org-b
  - name: Org A
    slug: org-a
roles:
  - name: Role B
    orgSlug: org-b
    slug: role-b
  - name: Role A
    orgSlug: org-a
    slug: role-a
`, `organizations:
  - name: Catalog Org A
    slug: org-a
  - name: Catalog Org B
    slug: org-b
roles:
  - name: Catalog Role A
    orgSlug: org-a
    slug: role-a
  - name: Catalog Role B
    orgSlug: org-b
    slug: role-b
`, 1)
		ok, err := streamPresentationOnlyChange(stored, saved)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !ok {
			t.Fatal("expected presentation-only despite Formata build() org/role churn")
		}
	})
	t.Run("org slug membership change is not presentation-only", func(t *testing.T) {
		stored := formataLegacyUncategorizedYAML()
		saved := strings.Replace(stored, "  name: Legacy stream\n", "  categorySlug: materials\n  subCategorySlug: metals\n  name: Legacy stream\n", 1)
		saved = strings.Replace(saved, "slug: org-b", "slug: org-c", 1)
		saved = strings.Replace(saved, "organization: org-b", "organization: org-c", 1)
		saved = strings.Replace(saved, "orgSlug: org-b", "orgSlug: org-c", 1)
		ok, err := streamPresentationOnlyChange(stored, saved)
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if ok {
			t.Fatal("expected shape change when org slug membership changes")
		}
	})
}

func formataLegacyUncategorizedYAML() string {
	return `dpp:
  enabled: false
  gtin: ""
  lotDefault: ""
  lotInputKey: ""
  serialInputKey: ""
  serialStrategy: ""
  productName: ""
  productDescription: ""
  ownerName: ""
organizations:
  - name: Org B
    slug: org-b
  - name: Org A
    slug: org-a
roles:
  - name: Role B
    orgSlug: org-b
    slug: role-b
  - name: Role A
    orgSlug: org-a
    slug: role-a
workflow:
  name: Legacy stream
  description: demo stream without categories
  steps:
    - id: "1"
      title: Step One
      order: 1
      organization: org-a
      substeps:
        - id: "1.1"
          title: Input One
          order: 1
          roles: [role-a]
          inputKey: value
          inputType: formata
          schema:
            type: object
            properties:
              value: { type: string }
    - id: "2"
      title: Step Two
      order: 2
      organization: org-b
      substeps:
        - id: "2.1"
          title: Input Two
          order: 1
          roles: [role-b]
          inputKey: value
          inputType: formata
          schema:
            type: object
            properties:
              value: { type: string }
`
}
