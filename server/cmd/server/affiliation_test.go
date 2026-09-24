package main

import (
	"context"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAffiliationIsAffiliated(t *testing.T) {
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, time.Now)

	if aff.IsAffiliated(IdentityUser{}) {
		t.Fatal("empty OrgSlug should not be affiliated")
	}
	if aff.IsAffiliated(IdentityUser{OrgSlug: "   "}) {
		t.Fatal("whitespace OrgSlug should not be affiliated")
	}
	if !aff.IsAffiliated(IdentityUser{OrgSlug: "acme"}) {
		t.Fatal("non-empty OrgSlug should be affiliated")
	}
}

func TestAffiliationPendingIntentEmpty(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, time.Now)

	join, err := aff.PendingJoinRequestForUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("PendingJoinRequestForUser: %v", err)
	}
	if join != nil {
		t.Fatalf("expected nil join request, got %+v", join)
	}

	orgReq, err := aff.PendingOrganizationCreationRequestForUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("PendingOrganizationCreationRequestForUser: %v", err)
	}
	if orgReq != nil {
		t.Fatalf("expected nil org creation request, got %+v", orgReq)
	}

	pending, err := aff.HasPendingAffiliationIntent(ctx, "user-1")
	if err != nil {
		t.Fatalf("HasPendingAffiliationIntent: %v", err)
	}
	if pending {
		t.Fatal("expected no pending affiliation intent")
	}
}

func TestAffiliationJoinRequestRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, time.Now)

	saved, err := aff.SaveJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "user@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer", "editor"},
	})
	if err != nil {
		t.Fatalf("SaveJoinRequest: %v", err)
	}
	if saved.ID.IsZero() {
		t.Fatal("expected assigned ID")
	}
	if saved.Status != AffiliationStatusPending {
		t.Fatalf("status=%q, want pending", saved.Status)
	}
	if saved.CreatedAt.IsZero() || saved.UpdatedAt.IsZero() {
		t.Fatal("expected created/updated timestamps")
	}

	loaded, err := aff.LoadJoinRequestByID(ctx, saved.ID)
	if err != nil {
		t.Fatalf("LoadJoinRequestByID: %v", err)
	}
	if loaded == nil || loaded.ID != saved.ID {
		t.Fatalf("loaded=%+v", loaded)
	}
	if loaded.RequesterUserID != "user-1" || loaded.OrgSlug != "acme" {
		t.Fatalf("unexpected loaded fields: %+v", loaded)
	}
	if len(loaded.RoleSlugs) != 2 || loaded.RoleSlugs[0] != "viewer" {
		t.Fatalf("roleSlugs=%v", loaded.RoleSlugs)
	}

	pending, err := aff.PendingJoinRequestForUser(ctx, "user-1")
	if err != nil || pending == nil || pending.ID != saved.ID {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}
	hasPending, err := aff.HasPendingAffiliationIntent(ctx, "user-1")
	if err != nil || !hasPending {
		t.Fatalf("HasPendingAffiliationIntent=%v err=%v", hasPending, err)
	}

	updated := *loaded
	updated.Status = AffiliationStatusApproved
	updated.DecidedByUserID = "admin-1"
	updated.DecidedAt = time.Now().UTC()
	updated.UpdatedAt = time.Time{}
	got, err := store.UpdateJoinRequest(ctx, updated)
	if err != nil {
		t.Fatalf("UpdateJoinRequest: %v", err)
	}
	if got.Status != AffiliationStatusApproved || got.UpdatedAt.IsZero() {
		t.Fatalf("updated=%+v", got)
	}

	listed, err := store.ListJoinRequestsByOrg(ctx, "acme")
	if err != nil || len(listed) != 1 {
		t.Fatalf("listed=%v err=%v", listed, err)
	}
}

func TestAffiliationOrganizationCreationRequestRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, time.Now)

	saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-2",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("SaveOrganizationCreationRequest: %v", err)
	}
	if saved.ID.IsZero() || saved.Status != AffiliationStatusPending {
		t.Fatalf("saved=%+v", saved)
	}

	loaded, err := aff.LoadOrganizationCreationRequestByID(ctx, saved.ID)
	if err != nil || loaded == nil || loaded.ProposedSlug != "new-org" {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}

	pending, err := aff.PendingOrganizationCreationRequestForUser(ctx, "user-2")
	if err != nil || pending == nil || pending.ID != saved.ID {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}

	updated := *loaded
	updated.Status = AffiliationStatusRejected
	updated.RejectReason = "duplicate"
	updated.DecidedByUserID = "admin-1"
	updated.DecidedAt = time.Now().UTC()
	got, err := store.UpdateOrganizationCreationRequest(ctx, updated)
	if err != nil || got.Status != AffiliationStatusRejected {
		t.Fatalf("updated=%+v err=%v", got, err)
	}

	listed, err := store.ListOrganizationCreationRequests(ctx)
	if err != nil || len(listed) != 1 {
		t.Fatalf("listed=%v err=%v", listed, err)
	}
}

func TestRecordingMailerCapturesSend(t *testing.T) {
	mailer := &recordingMailer{}
	msg := MailMessage{
		Kind:    "affiliation.join.submitted",
		To:      []string{"admin@example.com"},
		Subject: "Join request",
		Body:    "Please review",
	}
	if err := mailer.Send(context.Background(), msg); err != nil {
		t.Fatalf("Send: %v", err)
	}
	got := mailer.Messages()
	if len(got) != 1 {
		t.Fatalf("messages=%d, want 1", len(got))
	}
	if got[0].Kind != msg.Kind || got[0].Subject != msg.Subject || got[0].Body != msg.Body {
		t.Fatalf("got=%+v", got[0])
	}
	if len(got[0].To) != 1 || got[0].To[0] != "admin@example.com" {
		t.Fatalf("to=%v", got[0].To)
	}
	got[0].To[0] = "mutated"
	if mailer.Messages()[0].To[0] != "admin@example.com" {
		t.Fatal("Messages() should return a defensive copy")
	}
}

func TestServerAffiliationServiceSmoke(t *testing.T) {
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	server := &Server{
		store:    store,
		identity: &fakeIdentityStore{},
		mailer:   mailer,
		now:      func() time.Time { return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC) },
	}

	aff := server.affiliationService()
	if aff == nil {
		t.Fatal("affiliationService returned nil")
	}
	if server.affiliationService() != aff {
		t.Fatal("affiliationService should cache instance")
	}
	if !aff.IsAffiliated(IdentityUser{OrgSlug: "acme"}) {
		t.Fatal("expected affiliated")
	}

	nilMailerServer := &Server{store: store, identity: &fakeIdentityStore{}}
	if nilMailerServer.affiliationService() == nil {
		t.Fatal("nil mailer should still construct Affiliation via noopMailer")
	}

	// Ensure Store round-trip works through the wired service.
	saved, err := aff.SaveJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "u",
		RequesterEmail:  "u@example.com",
		OrgSlug:         "acme",
	})
	if err != nil || saved.ID == (primitive.ObjectID{}) {
		t.Fatalf("SaveJoinRequest via server service: saved=%+v err=%v", saved, err)
	}
}
