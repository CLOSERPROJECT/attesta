package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type fakeLegacyLookupStore struct {
	*MemoryStore
	users map[primitive.ObjectID]AccountUser
}

func (s *fakeLegacyLookupStore) GetUserByMongoID(_ context.Context, userMongoID primitive.ObjectID) (*AccountUser, error) {
	user, ok := s.users[userMongoID]
	if !ok {
		return nil, errors.New("not found")
	}
	copy := user
	return &copy, nil
}

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
		ID:             primitive.NewObjectID(),
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
		t.Fatal("empty userMongoID should be unresolved")
	}
	if _, ok := server.lookupUserIdentityByActorID(context.Background(), "u-1", cache); ok {
		t.Fatal("missing store should be unresolved")
	}
	if _, exists := cache["u-1"]; !exists {
		t.Fatal("expected missing-store lookup to prime cache")
	}
	cache["u-2"] = userIdentityView{email: "cached@example.com", fallbackID: "u-2"}
	identity, ok := server.lookupUserIdentityByActorID(context.Background(), "u-2", cache)
	if !ok || identity.email != "cached@example.com" {
		t.Fatalf("cache hit mismatch: ok=%v identity=%#v", ok, identity)
	}
}

func TestLookupUserIdentityByActorIDMixedHistory(t *testing.T) {
	legacy := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "legacy@example.com",
		Status:    "active",
		CreatedAt: time.Now().UTC(),
	}
	store := &fakeLegacyLookupStore{
		MemoryStore: NewMemoryStore(),
		users:       map[primitive.ObjectID]AccountUser{legacy.ID: legacy},
	}
	server := &Server{
		store: store,
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				if userID != "user-1" {
					return IdentityUser{}, ErrIdentityNotFound
				}
				return IdentityUser{ID: "user-1", Email: "appwrite@example.com", Status: "active"}, nil
			},
		},
	}

	legacyIdentity, ok := server.lookupUserIdentityByActorID(context.Background(), legacy.ID.Hex(), map[string]userIdentityView{})
	if !ok || legacyIdentity.email != "legacy@example.com" || legacyIdentity.fallbackID != legacy.ID.Hex() {
		t.Fatalf("legacy identity = %#v ok=%v", legacyIdentity, ok)
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
	created := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "done@example.com",
		Status:    "active",
		RoleSlugs: []string{"dep1"},
		CreatedAt: time.Now().UTC(),
	}
	store := &fakeLegacyLookupStore{
		MemoryStore: NewMemoryStore(),
		users:       map[primitive.ObjectID]AccountUser{created.ID: created},
	}
	server := &Server{
		store: store,
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
	actions := []ActionView{
		{SubstepID: "1.1", DoneBy: created.ID.Hex()},
		{SubstepID: "1.15", DoneBy: "appwrite:user-1"},
		{SubstepID: "1.2", DoneBy: "legacy-user"},
	}
	timeline := []TimelineStep{
		{
			StepID: "1",
			Substeps: []TimelineSubstep{
				{SubstepID: "1.1", DoneBy: created.ID.Hex()},
				{SubstepID: "1.15", DoneBy: "appwrite:user-1"},
				{SubstepID: "1.2", DoneBy: "legacy-user"},
			},
		},
	}

	deniedActions := server.applyDoneByEmailToActions(context.Background(), def, Actor{OrgSlug: "org-z"}, append([]ActionView(nil), actions...))
	if deniedActions[0].DoneBy != created.ID.Hex() {
		t.Fatalf("denied action doneBy = %q, want mongoID %q", deniedActions[0].DoneBy, created.ID.Hex())
	}
	if deniedActions[1].DoneBy != "appwrite:user-1" {
		t.Fatalf("denied action doneBy = %q, want appwrite:user-1", deniedActions[1].DoneBy)
	}
	deniedTimeline := server.applyDoneByEmailToTimeline(context.Background(), def, Actor{OrgSlug: "org-z"}, cloneTimelineSteps(timeline))
	if deniedTimeline[0].Substeps[0].DoneBy != created.ID.Hex() {
		t.Fatalf("denied timeline doneBy = %q, want mongoID %q", deniedTimeline[0].Substeps[0].DoneBy, created.ID.Hex())
	}
	if deniedTimeline[0].Substeps[1].DoneBy != "appwrite:user-1" {
		t.Fatalf("denied timeline doneBy = %q, want appwrite:user-1", deniedTimeline[0].Substeps[1].DoneBy)
	}

	allowedActions := server.applyDoneByEmailToActions(context.Background(), def, Actor{OrgSlug: "org-a"}, append([]ActionView(nil), actions...))
	if allowedActions[0].DoneBy != "done@example.com" {
		t.Fatalf("allowed action doneBy = %q, want email", allowedActions[0].DoneBy)
	}
	if allowedActions[1].DoneBy != "appwrite@example.com" {
		t.Fatalf("allowed action doneBy = %q, want appwrite email", allowedActions[1].DoneBy)
	}
	if allowedActions[2].DoneBy != "legacy-user" {
		t.Fatalf("legacy action doneBy = %q, want unchanged userMongoID", allowedActions[2].DoneBy)
	}
	allowedTimeline := server.applyDoneByEmailToTimeline(context.Background(), def, Actor{OrgSlug: "org-a"}, cloneTimelineSteps(timeline))
	if allowedTimeline[0].Substeps[0].DoneBy != "done@example.com" {
		t.Fatalf("allowed timeline doneBy = %q, want email", allowedTimeline[0].Substeps[0].DoneBy)
	}
	if allowedTimeline[0].Substeps[1].DoneBy != "appwrite@example.com" {
		t.Fatalf("allowed timeline doneBy = %q, want appwrite email", allowedTimeline[0].Substeps[1].DoneBy)
	}
	if allowedTimeline[0].Substeps[2].DoneBy != "legacy-user" {
		t.Fatalf("legacy timeline doneBy = %q, want unchanged userMongoID", allowedTimeline[0].Substeps[2].DoneBy)
	}
}

func TestApplyDoneByMongoIDToDPPTraceability(t *testing.T) {
	created := AccountUser{
		ID:        primitive.NewObjectID(),
		Email:     "dpp@example.com",
		Status:    "active",
		RoleSlugs: []string{"dep1"},
		CreatedAt: time.Now().UTC(),
	}
	store := &fakeLegacyLookupStore{
		MemoryStore: NewMemoryStore(),
		users:       map[primitive.ObjectID]AccountUser{created.ID: created},
	}
	server := &Server{
		store: store,
		identity: &fakeIdentityStore{
			getUserByIDFunc: func(ctx context.Context, userID string) (IdentityUser, error) {
				if userID != "user-1" {
					return IdentityUser{}, ErrIdentityNotFound
				}
				return IdentityUser{ID: "user-1", Email: "dpp-appwrite@example.com", Status: "active"}, nil
			},
		},
	}

	traceability := []DPPTraceabilityStep{
		{
			StepID: "1",
			Substeps: []DPPTraceabilitySubstep{
				{SubstepID: "1.1", DoneBy: created.ID.Hex()},
				{SubstepID: "1.15", DoneBy: "appwrite:user-1"},
				{SubstepID: "1.2", DoneBy: "legacy-user"},
			},
		},
	}
	mapped := server.applyDoneByMongoIDToDPPTraceability(context.Background(), traceability)
	if mapped[0].Substeps[0].DoneBy != created.ID.Hex() {
		t.Fatalf("mapped dpp doneBy = %q, want mongoID %q", mapped[0].Substeps[0].DoneBy, created.ID.Hex())
	}
	if mapped[0].Substeps[1].DoneBy != "appwrite:user-1" {
		t.Fatalf("mapped dpp doneBy = %q, want appwrite fallback id", mapped[0].Substeps[1].DoneBy)
	}
	if mapped[0].Substeps[2].DoneBy != "legacy-user" {
		t.Fatalf("legacy dpp doneBy = %q, want unchanged userMongoID", mapped[0].Substeps[2].DoneBy)
	}
}

func cloneTimelineSteps(src []TimelineStep) []TimelineStep {
	out := append([]TimelineStep(nil), src...)
	for i := range out {
		out[i].Substeps = append([]TimelineSubstep(nil), out[i].Substeps...)
	}
	return out
}
