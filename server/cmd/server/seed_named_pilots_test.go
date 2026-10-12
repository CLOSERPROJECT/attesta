package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultDemoFixtureIncludesFeragame(t *testing.T) {
	fixture := defaultDemoFixture()
	var foundOrg bool
	for _, org := range fixture.Orgs {
		if demoOrgSlug(org.Name) != "feragame" {
			continue
		}
		foundOrg = true
		if len(org.Roles) != 1 || org.Roles[0].Slug != "chemist" {
			t.Fatalf("feragame roles = %+v, want chemist", org.Roles)
		}
	}
	if !foundOrg {
		t.Fatal("defaultDemoFixture missing Feragame org")
	}

	var foundUser bool
	for _, user := range fixture.Users {
		if user.Email != "admin@feragame.com" {
			continue
		}
		foundUser = true
		if user.Password != "admin@feragame.com" {
			t.Fatalf("password pattern = %q", user.Password)
		}
		if len(user.OrgAdminSlugs) != 1 || user.OrgAdminSlugs[0] != "feragame" {
			t.Fatalf("OrgAdminSlugs = %v", user.OrgAdminSlugs)
		}
		if roles := user.MembershipRoles["feragame"]; len(roles) != 1 || roles[0] != "chemist" {
			t.Fatalf("MembershipRoles = %v", user.MembershipRoles)
		}
	}
	if !foundUser {
		t.Fatal("defaultDemoFixture missing admin@feragame.com")
	}

	platform := fixture.Users[0]
	if !platform.PlatformAdmin {
		t.Fatal("expected platform admin first")
	}
	var hasFeragame bool
	for _, slug := range platform.OrgAdminSlugs {
		if slug == "feragame" {
			hasFeragame = true
			break
		}
	}
	if !hasFeragame {
		t.Fatalf("platform admin OrgAdminSlugs missing feragame: %v", platform.OrgAdminSlugs)
	}
}

func TestNamedPilotStreamURLKeyStable(t *testing.T) {
	want := "ef8d2f15c968fb8d879d56aa"
	if got := namedPilotStreamURLKey("named-pilot-stream:feragame-pilot"); got != want {
		t.Fatalf("URL key = %q, want %q", got, want)
	}
}

func TestSeedNamedPilotStreamsUpsertsStableID(t *testing.T) {
	dir := t.TempDir()
	body := `workflow:
  name: Feragame production day (pilot)
  completionStrategy: draft-and-seal
  categorySlug: recycling-and-recovery
  subCategorySlug: flat-screen-displays
  steps:
    - id: "1"
      title: Production day
      order: 1
      organization: feragame
      substeps:
        - id: "1.1"
          order: 1
          title: Batch loads
          inputKey: batches
          roles: ["chemist"]
          inputType: formata
          schema:
            type: object
organizations:
  - slug: feragame
    name: Feragame
roles:
  - orgSlug: feragame
    slug: chemist
    name: Chemist
`
	if err := os.WriteFile(filepath.Join(dir, "feragame-pilot.yaml"), []byte(body), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Must not enter the pack even when present beside templates.
	if err := os.WriteFile(filepath.Join(dir, "demo.yaml"), []byte(minimalCategorizedWorkflowYAML(
		"  categorySlug: supply-chain\n  subCategorySlug: procurement\n",
	)), 0o644); err != nil {
		t.Fatalf("write demo: %v", err)
	}
	pack, err := openCatalogTemplatePack(dir)
	if err != nil {
		t.Fatalf("openCatalogTemplatePack: %v", err)
	}
	if len(pack.Streams) != 1 || filepath.Base(pack.Streams[0].Path) != "demo.yaml" {
		t.Fatalf("pack streams = %+v, want only demo.yaml", pack.Streams)
	}

	store := NewMemoryStore()
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	res, err := seedNamedPilotStreams(t.Context(), store, dir, now)
	if err != nil {
		t.Fatalf("seedNamedPilotStreams: %v", err)
	}
	if res.Streams != 1 || len(res.Keys) != 1 {
		t.Fatalf("result = %+v", res)
	}
	wantKey := namedPilotStreamURLKey("named-pilot-stream:feragame-pilot")
	if res.Keys[0] != wantKey {
		t.Fatalf("key = %q, want %q", res.Keys[0], wantKey)
	}

	streams, err := store.ListFormataBuilderStreams(t.Context())
	if err != nil || len(streams) != 1 {
		t.Fatalf("listed=%v err=%v", streams, err)
	}
	if streams[0].ID.Hex() != wantKey {
		t.Fatalf("id = %s, want %s", streams[0].ID.Hex(), wantKey)
	}
	if !strings.Contains(streams[0].Stream, "completionStrategy: draft-and-seal") {
		t.Fatal("pilot body missing draft-and-seal")
	}
	if !strings.Contains(streams[0].Stream, "organization: feragame") {
		t.Fatal("pilot body missing feragame org")
	}

	updatedBody := strings.Replace(body, "Feragame production day (pilot)", "Feragame production day (pilot v2)", 1)
	if err := os.WriteFile(filepath.Join(dir, "feragame-pilot.yaml"), []byte(updatedBody), 0o644); err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	res2, err := seedNamedPilotStreams(t.Context(), store, dir, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("second seedNamedPilotStreams: %v", err)
	}
	if res2.Keys[0] != wantKey {
		t.Fatalf("second key = %q", res2.Keys[0])
	}
	streams2, _ := store.ListFormataBuilderStreams(t.Context())
	if len(streams2) != 1 {
		t.Fatalf("after upsert listed=%d", len(streams2))
	}
	if !strings.Contains(streams2[0].Stream, "pilot v2") {
		t.Fatal("upsert did not update stream body")
	}
}

func TestSeedNamedPilotStreamsRequiresStore(t *testing.T) {
	if _, err := seedNamedPilotStreams(t.Context(), nil, t.TempDir(), time.Now().UTC()); err == nil || !strings.Contains(err.Error(), "store is required") {
		t.Fatalf("err = %v", err)
	}
}
