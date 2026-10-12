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

func TestBuildStreamInstanceDetailViewStampsViewerOwned(t *testing.T) {
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

	view := server.buildStreamInstanceDetailView(t.Context(), cfg, "workflow", process, actor, "", "", streamInstanceDetailOptions{ShowMineFilter: true})
	if len(view.Timeline) != 3 {
		t.Fatalf("timeline must stay full for client filter, got %d steps", len(view.Timeline))
	}
	owned := map[string]bool{}
	for _, step := range view.Timeline {
		for _, sub := range step.Substeps {
			owned[sub.SubstepID] = sub.ViewerOwned
		}
	}
	if !owned["1.1"] || !owned["1.2"] || !owned["1.3"] {
		t.Fatalf("expected org-a dep1/dep2 substeps owned, got %#v", owned)
	}
	if owned["2.1"] || owned["3.1"] {
		t.Fatalf("expected other-org substeps not owned, got %#v", owned)
	}
}

func TestBuildStreamInstanceDetailViewHidesMineFilterWhenProcessDone(t *testing.T) {
	cfg := testMineFilterRuntimeConfig()
	process := &Process{
		ID:     primitive.NewObjectID(),
		Status: processStatusDone,
		Progress: map[string]ProcessStep{
			"1.1": {State: "done"},
			"1.2": {State: "done"},
			"2.1": {State: "done"},
			"3.1": {State: "done"},
		},
	}
	server := &Server{}
	actor := Actor{OrgSlug: "org-a", RoleSlugs: []string{"dep1"}}

	view := server.buildStreamInstanceDetailView(t.Context(), cfg, "workflow", process, actor, "", "", streamInstanceDetailOptions{ShowMineFilter: true})
	if !view.ProcessDone {
		t.Fatal("expected ProcessDone")
	}
	if view.ShowMineFilter {
		t.Fatal("expected ShowMineFilter false on completed stream")
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

	view := server.buildStreamInstanceDetailView(t.Context(), cfg, "workflow", process, actor, "", "", streamInstanceDetailOptions{ShowMineFilter: true})
	var found11 bool
	for _, step := range view.Timeline {
		for _, sub := range step.Substeps {
			if sub.SubstepID == "1.1" {
				found11 = true
				if !sub.ViewerOwned {
					t.Fatal("1.1 should be viewer-owned via dep1 slug, not only active Role")
				}
			}
			if sub.SubstepID == "2.1" && sub.ViewerOwned {
				t.Fatal("2.1 is other org — must not be viewer-owned")
			}
		}
	}
	if !found11 {
		t.Fatal("expected substep 1.1 in full timeline")
	}
}
