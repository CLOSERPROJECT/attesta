package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func testMineFilterRuntimeConfig() RuntimeConfig {
	cfg := testRuntimeConfig()
	cfg.Workflow.Steps[0].OrganizationSlug = "org-a"
	cfg.Workflow.Steps[1].OrganizationSlug = "org-b"
	cfg.Workflow.Steps[2].OrganizationSlug = "org-c"
	return cfg
}

func TestParseMineFilterEnabled(t *testing.T) {
	if !parseMineFilterEnabled("") {
		t.Fatal("missing mine param should default to on")
	}
	if !parseMineFilterEnabled("1") {
		t.Fatal("mine=1 should be on")
	}
	if parseMineFilterEnabled("0") {
		t.Fatal("mine=0 should be off")
	}
	if parseMineFilterEnabled("false") {
		t.Fatal("mine=false should be off")
	}
}

func TestSubstepMineForViewerRequiresOrgAndRole(t *testing.T) {
	def := testMineFilterRuntimeConfig().Workflow

	if !substepMineForViewer(def, "1.1", Actor{OrgSlug: "org-a", RoleSlugs: []string{"dep1"}}) {
		t.Fatal("expected org-a dep1 to own substep 1.1")
	}
	if substepMineForViewer(def, "2.1", Actor{OrgSlug: "org-a", RoleSlugs: []string{"dep1"}}) {
		t.Fatal("org-a should not see org-b step substeps")
	}
	if substepMineForViewer(def, "1.1", Actor{OrgSlug: "org-a", RoleSlugs: []string{"dep2"}}) {
		t.Fatal("dep2 should not match dep1-gated substep")
	}
}

func TestSubstepMineForViewerIgnoresOrgAdminStanding(t *testing.T) {
	def := testMineFilterRuntimeConfig().Workflow
	actor := Actor{OrgSlug: "org-a", RoleSlugs: []string{"org-admin"}}
	if substepMineForViewer(def, "1.1", actor) {
		t.Fatal("org-admin standing alone must not grant mine substeps")
	}
	actor.RoleSlugs = []string{"org_admin", "dep1"}
	if !substepMineForViewer(def, "1.1", actor) {
		t.Fatal("dep1 should still match when paired with org_admin standing")
	}
}

func TestBuildStreamInstanceDetailViewMineFilterOnAndOff(t *testing.T) {
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"1.2": {State: "pending"},
			"2.1": {State: "pending"},
			"3.1": {State: "pending"},
		},
	}
	server := &Server{}
	cfg := testMineFilterRuntimeConfig()
	actor := Actor{OrgSlug: "org-a", RoleSlugs: []string{"dep1", "dep2"}, Role: "dep2"}

	on := server.buildStreamInstanceDetailView(t.Context(), cfg, "workflow", process, actor, "", "", streamInstanceDetailOptions{MineFilterOn: true, ShowMineFilter: true})
	if len(on.Timeline) != 1 {
		t.Fatalf("mine on: expected one step, got %d", len(on.Timeline))
	}
	if len(on.Timeline[0].Substeps) != 3 {
		t.Fatalf("mine on: expected three substeps in step 1, got %d", len(on.Timeline[0].Substeps))
	}

	off := server.buildStreamInstanceDetailView(t.Context(), cfg, "workflow", process, actor, "", "", streamInstanceDetailOptions{MineFilterOn: false, ShowMineFilter: true})
	if len(off.Timeline) != 3 {
		t.Fatalf("mine off: expected three steps, got %d", len(off.Timeline))
	}
}

func TestBuildStreamInstanceDetailViewMineFilterSubstepCarveOut(t *testing.T) {
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"3.1": {State: "pending"},
		},
	}
	server := &Server{}
	cfg := testMineFilterRuntimeConfig()
	actor := Actor{OrgSlug: "org-a", RoleSlugs: []string{"dep1"}}

	view := server.buildStreamInstanceDetailView(t.Context(), cfg, "workflow", process, actor, "3.1", "", streamInstanceDetailOptions{MineFilterOn: true, ShowMineFilter: true})
	if len(view.Timeline) != 2 {
		t.Fatalf("carve-out: expected step 1 and step 3, got %d steps", len(view.Timeline))
	}
	if view.SelectedSubstepID != "3.1" {
		t.Fatalf("SelectedSubstepID = %q, want 3.1", view.SelectedSubstepID)
	}
}

func TestBuildStreamInstanceDetailViewUsesAllRoleSlugsNotActiveRole(t *testing.T) {
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"2.1": {State: "pending"},
		},
	}
	server := &Server{}
	cfg := testMineFilterRuntimeConfig()
	actor := Actor{
		OrgSlug:   "org-a",
		Role:      "dep2",
		RoleSlugs: []string{"dep1", "dep2"},
	}

	view := server.buildStreamInstanceDetailView(t.Context(), cfg, "workflow", process, actor, "", "", streamInstanceDetailOptions{MineFilterOn: true, ShowMineFilter: true})
	if len(view.Timeline) != 1 {
		t.Fatalf("expected only org-a step with dep1 substeps, got %d steps", len(view.Timeline))
	}
	if view.SelectedBody == nil || view.SelectedBody.SubstepID != "1.1" {
		t.Fatalf("expected first available mine substep 1.1 via dep1 slug, got %#v", view.SelectedBody)
	}
}
