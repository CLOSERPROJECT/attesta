package main

import (
	"context"
	"testing"
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
	if empty.ID != "" || empty.OrgSlug != "" || len(empty.RoleSlugs) != 0 || empty.Role != "" {
		t.Fatalf("unexpected empty actor: %#v", empty)
	}

	user := &AccountUser{
		IdentityUserID: "user-1",
		OrgSlug:        "org-a",
		RoleSlugs:      []string{"dep2", "dep1"},
	}
	actor := actorFromAccountUser(user, "wf-y")
	if actor.WorkflowKey != "wf-y" || actor.ID != "appwrite:user-1" || actor.OrgSlug != "org-a" {
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

func TestLookupUserIdentityByActorIDBranches(t *testing.T) {
	server := &Server{}
	cache := map[string]userIdentityView{}
	if _, ok := server.lookupUserIdentityByActorID(context.Background(), "", cache); ok {
		t.Fatal("empty actor id should be unresolved")
	}
	if _, ok := server.lookupUserIdentityByActorID(context.Background(), "u-1", cache); ok {
		t.Fatal("non-appwrite actor id should be unresolved")
	}
	if _, exists := cache["u-1"]; !exists {
		t.Fatal("expected unresolved lookup to prime cache")
	}
	cache["u-2"] = userIdentityView{email: "cached@example.com", fallbackID: "u-2"}
	identity, ok := server.lookupUserIdentityByActorID(context.Background(), "u-2", cache)
	if !ok || identity.email != "cached@example.com" {
		t.Fatalf("cache hit mismatch: ok=%v identity=%#v", ok, identity)
	}
}

func TestLookupUserIdentityByActorIDAppwriteOnly(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				if userID != "user-1" {
					return IdentityUser{}, ErrIdentityNotFound
				}
				return IdentityUser{ID: "user-1", Email: "appwrite@example.com", Status: "active"}, nil
			},
		},
	}

	appwriteIdentity, ok := server.lookupUserIdentityByActorID(context.Background(), "appwrite:user-1", map[string]userIdentityView{})
	if !ok || appwriteIdentity.email != "appwrite@example.com" || appwriteIdentity.fallbackID != "appwrite:user-1" {
		t.Fatalf("appwrite identity = %#v ok=%v", appwriteIdentity, ok)
	}

	if _, ok := server.lookupUserIdentityByActorID(context.Background(), "appwrite:missing", map[string]userIdentityView{}); ok {
		t.Fatal("expected missing appwrite identity to fail closed")
	}
}

func TestApplyDoneByEmailVisibility(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				if userID != "user-1" {
					return IdentityUser{}, ErrIdentityNotFound
				}
				return IdentityUser{ID: "user-1", Email: "appwrite@example.com", Status: "active"}, nil
			},
		},
	}

	def := WorkflowDef{
		Steps: []WorkflowStep{
			{StepID: "1", OrganizationSlug: "org-a"},
		},
	}
	actions := []SubstepBodyView{
		{SubstepID: "1.1", DoneBy: "appwrite:user-1"},
		{SubstepID: "1.15", DoneBy: "appwrite:user-2"},
		{SubstepID: "1.2", DoneBy: "legacy-user"},
	}

	deniedActions := server.applyDoneByEmailToSubstepViews(context.Background(), def, Actor{OrgSlug: "org-z"}, append([]SubstepBodyView(nil), actions...))
	if deniedActions[0].DoneBy != "appwrite:user-1" {
		t.Fatalf("denied action doneBy = %q, want appwrite:user-1", deniedActions[0].DoneBy)
	}
	if deniedActions[1].DoneBy != "appwrite:user-2" {
		t.Fatalf("denied action doneBy = %q, want appwrite:user-2", deniedActions[1].DoneBy)
	}

	allowedActions := server.applyDoneByEmailToSubstepViews(context.Background(), def, Actor{OrgSlug: "org-a"}, append([]SubstepBodyView(nil), actions...))
	if allowedActions[0].DoneBy != "appwrite@example.com" {
		t.Fatalf("allowed action doneBy = %q, want appwrite@example.com", allowedActions[0].DoneBy)
	}
	if allowedActions[1].DoneBy != "appwrite:user-2" {
		t.Fatalf("allowed action doneBy = %q, want unresolved appwrite fallback", allowedActions[1].DoneBy)
	}
	if allowedActions[2].DoneBy != "legacy-user" {
		t.Fatalf("legacy action doneBy = %q, want unchanged legacy-user", allowedActions[2].DoneBy)
	}

	termination := &ProcessTerminationView{EndedBy: "appwrite:user-1"}
	deniedTermination := server.applyDoneByEmailToTermination(context.Background(), def, Actor{OrgSlug: "org-z"}, termination)
	if deniedTermination.EndedBy != "appwrite:user-1" {
		t.Fatalf("denied termination endedBy = %q, want appwrite:user-1", deniedTermination.EndedBy)
	}
	allowedTermination := server.applyDoneByEmailToTermination(context.Background(), def, Actor{OrgSlug: "org-a"}, termination)
	if allowedTermination.EndedBy != "appwrite@example.com" {
		t.Fatalf("allowed termination endedBy = %q, want appwrite@example.com", allowedTermination.EndedBy)
	}
	if termination.EndedBy != "appwrite:user-1" {
		t.Fatalf("original termination endedBy mutated to %q", termination.EndedBy)
	}
}

func TestApplyDoneByIdentityFallbackToDPPTraceability(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				if userID != "user-1" {
					return IdentityUser{}, ErrIdentityNotFound
				}
				return IdentityUser{ID: "user-1", Email: "dpp-appwrite@example.com", Status: "active"}, nil
			},
		},
	}

	traceability := []TimelineStep{
		{
			Substeps: []TimelineSubstep{
				{SubstepID: "1.1", DoneBy: "appwrite:user-1", Body: &SubstepBodyView{DoneBy: "appwrite:user-1"}},
				{SubstepID: "1.15", DoneBy: "appwrite:user-1", Body: &SubstepBodyView{DoneBy: "appwrite:user-1"}},
				{SubstepID: "1.2", DoneBy: "legacy-user", Body: &SubstepBodyView{DoneBy: "legacy-user"}},
			},
		},
	}
	mapped := server.applyDoneByIdentityFallbackToDPPTraceability(context.Background(), traceability)
	if mapped[0].Substeps[0].DoneBy != "appwrite:user-1" {
		t.Fatalf("mapped dpp doneBy = %q, want appwrite fallback id", mapped[0].Substeps[0].DoneBy)
	}
	if mapped[0].Substeps[1].DoneBy != "appwrite:user-1" {
		t.Fatalf("mapped dpp doneBy = %q, want appwrite fallback id", mapped[0].Substeps[1].DoneBy)
	}
	if mapped[0].Substeps[2].DoneBy != "legacy-user" {
		t.Fatalf("legacy dpp doneBy = %q, want unchanged legacy-user", mapped[0].Substeps[2].DoneBy)
	}
}

func TestLookupUserIdentityByActorIDUsesCache(t *testing.T) {
	calls := 0
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				calls++
				return IdentityUser{ID: userID, Email: "cached@example.com", Status: "active"}, nil
			},
		},
	}
	cache := map[string]userIdentityView{}

	first, ok := server.lookupUserIdentityByActorID(context.Background(), "appwrite:user-1", cache)
	if !ok || first.email != "cached@example.com" {
		t.Fatalf("first lookup = %#v (ok=%t)", first, ok)
	}
	second, ok := server.lookupUserIdentityByActorID(context.Background(), "appwrite:user-1", cache)
	if !ok || second.email != "cached@example.com" {
		t.Fatalf("cached lookup = %#v (ok=%t)", second, ok)
	}
	if calls != 1 {
		t.Fatalf("GetUserByID calls = %d, want 1", calls)
	}
}

func TestApplyDoneByEmailFallsBackToOpaqueActorIDWhenEmailUnavailable(t *testing.T) {
	server := &Server{
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Status: "active"}, nil
			},
		},
	}
	def := WorkflowDef{
		Steps: []WorkflowStep{{StepID: "1", OrganizationSlug: "org-a"}},
	}
	actions := []SubstepBodyView{{SubstepID: "1.1", DoneBy: "appwrite:user-no-email"}}

	mappedActions := server.applyDoneByEmailToSubstepViews(context.Background(), def, Actor{OrgSlug: "org-a"}, actions)
	if mappedActions[0].DoneBy != "appwrite:user-no-email" {
		t.Fatalf("mapped action doneBy = %q, want opaque appwrite actor id", mappedActions[0].DoneBy)
	}
}
