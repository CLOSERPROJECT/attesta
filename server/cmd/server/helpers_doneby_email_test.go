package main

import (
	"context"
	"testing"
	"time"
)

func TestViewerCanSeeDoneByEmail(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{StepID: "1", OrganizationSlug: "org-a"},
			{StepID: "2", OrganizationSlug: "org-b"},
		},
	}
	if !viewerCanSeeDoneByEmail(def, Actor{OrgSlug: "org-a"}) {
		t.Fatal("expected org-a viewer to see done-by email")
	}
	if viewerCanSeeDoneByEmail(def, Actor{OrgSlug: "org-z"}) {
		t.Fatal("expected org-z viewer to be denied")
	}
	if viewerCanSeeDoneByEmail(def, Actor{}) {
		t.Fatal("expected empty-org viewer to be denied")
	}
}

func TestActorFromAccountUser(t *testing.T) {
	empty := actorFromAccountUser(nil, "wf-x")
	if empty.WorkflowKey != "wf-x" {
		t.Fatalf("workflow key = %q, want wf-x", empty.WorkflowKey)
	}
	if empty.UserID != "" || empty.OrgSlug != "" || len(empty.RoleSlugs) != 0 || empty.Role != "" {
		t.Fatalf("unexpected empty actor: %#v", empty)
	}

	user := &AccountUser{
		UserID:    "u-1",
		OrgSlug:   "org-a",
		RoleSlugs: []string{"dep2", "dep1"},
	}
	actor := actorFromAccountUser(user, "wf-y")
	if actor.WorkflowKey != "wf-y" || actor.UserID != "u-1" || actor.OrgSlug != "org-a" {
		t.Fatalf("unexpected actor identity: %#v", actor)
	}
	if actor.Role != "dep2" {
		t.Fatalf("actor role = %q, want dep2", actor.Role)
	}
	if len(actor.RoleSlugs) != 2 || actor.RoleSlugs[0] != "dep2" || actor.RoleSlugs[1] != "dep1" {
		t.Fatalf("unexpected actor role slugs: %#v", actor.RoleSlugs)
	}

	user.RoleSlugs[0] = "mutated"
	if actor.RoleSlugs[0] != "dep2" {
		t.Fatalf("actor role slugs should be copied, got %#v", actor.RoleSlugs)
	}
}

func TestLookupUserIdentityByUserIDBranches(t *testing.T) {
	server := &Server{}
	cache := map[string]userIdentityView{}
	if _, ok := server.lookupUserIdentityByUserID(context.Background(), "", cache); ok {
		t.Fatal("empty userID should be unresolved")
	}
	if _, ok := server.lookupUserIdentityByUserID(context.Background(), "u-1", cache); ok {
		t.Fatal("missing store should be unresolved")
	}
	if _, exists := cache["u-1"]; !exists {
		t.Fatal("expected missing-store lookup to prime cache")
	}
	cache["u-2"] = userIdentityView{email: "cached@example.com"}
	identity, ok := server.lookupUserIdentityByUserID(context.Background(), "u-2", cache)
	if !ok || identity.email != "cached@example.com" {
		t.Fatalf("cache hit mismatch: ok=%v identity=%#v", ok, identity)
	}
}

func TestApplyDoneByEmailVisibility(t *testing.T) {
	store := NewMemoryStore()
	server := &Server{store: store}
	created, err := store.CreateUser(context.Background(), AccountUser{
		UserID:    "u-done",
		Email:     "done@example.com",
		Status:    "active",
		RoleSlugs: []string{"dep1"},
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	def := WorkflowDef{
		Steps: []WorkflowStep{
			{StepID: "1", OrganizationSlug: "org-a"},
		},
	}
	actions := []ActionView{
		{SubstepID: "1.1", DoneBy: "u-done"},
		{SubstepID: "1.2", DoneBy: "legacy-user"},
	}
	timeline := []TimelineStep{
		{
			StepID: "1",
			Substeps: []TimelineSubstep{
				{SubstepID: "1.1", DoneBy: "u-done"},
				{SubstepID: "1.2", DoneBy: "legacy-user"},
			},
		},
	}

	deniedActions := server.applyDoneByEmailToActions(context.Background(), def, Actor{OrgSlug: "org-z"}, append([]ActionView(nil), actions...))
	if deniedActions[0].DoneBy != created.ID.Hex() {
		t.Fatalf("denied action doneBy = %q, want mongoID %q", deniedActions[0].DoneBy, created.ID.Hex())
	}
	deniedTimeline := server.applyDoneByEmailToTimeline(context.Background(), def, Actor{OrgSlug: "org-z"}, cloneTimelineSteps(timeline))
	if deniedTimeline[0].Substeps[0].DoneBy != created.ID.Hex() {
		t.Fatalf("denied timeline doneBy = %q, want mongoID %q", deniedTimeline[0].Substeps[0].DoneBy, created.ID.Hex())
	}

	allowedActions := server.applyDoneByEmailToActions(context.Background(), def, Actor{OrgSlug: "org-a"}, append([]ActionView(nil), actions...))
	if allowedActions[0].DoneBy != "done@example.com" {
		t.Fatalf("allowed action doneBy = %q, want email", allowedActions[0].DoneBy)
	}
	if allowedActions[1].DoneBy != "legacy-user" {
		t.Fatalf("legacy action doneBy = %q, want unchanged userID", allowedActions[1].DoneBy)
	}
	allowedTimeline := server.applyDoneByEmailToTimeline(context.Background(), def, Actor{OrgSlug: "org-a"}, cloneTimelineSteps(timeline))
	if allowedTimeline[0].Substeps[0].DoneBy != "done@example.com" {
		t.Fatalf("allowed timeline doneBy = %q, want email", allowedTimeline[0].Substeps[0].DoneBy)
	}
	if allowedTimeline[0].Substeps[1].DoneBy != "legacy-user" {
		t.Fatalf("legacy timeline doneBy = %q, want unchanged userID", allowedTimeline[0].Substeps[1].DoneBy)
	}
}

func cloneTimelineSteps(src []TimelineStep) []TimelineStep {
	out := append([]TimelineStep(nil), src...)
	for i := range out {
		out[i].Substeps = append([]TimelineSubstep(nil), out[i].Substeps...)
	}
	return out
}
