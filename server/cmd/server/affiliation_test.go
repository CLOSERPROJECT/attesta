package main

import (
	"context"
	"errors"
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

func TestSubmitOrganizationCreationRequestHappyPath(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	aff := NewAffiliation(&fakeIdentityStore{}, store, mailer, fixedNow)

	t.Setenv("ADMIN_EMAIL", "platform@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	user := IdentityUser{ID: "user-1", Email: "founder@example.com"}
	saved, err := aff.SubmitOrganizationCreationRequest(ctx, user, "  New Org  ")
	if err != nil {
		t.Fatalf("SubmitOrganizationCreationRequest: %v", err)
	}
	if saved.ID.IsZero() || saved.Status != AffiliationStatusPending {
		t.Fatalf("saved=%+v", saved)
	}
	if saved.ProposedName != "New Org" || saved.ProposedSlug != "new-org" {
		t.Fatalf("name/slug=%q/%q", saved.ProposedName, saved.ProposedSlug)
	}
	if saved.RequesterUserID != "user-1" || saved.RequesterEmail != "founder@example.com" {
		t.Fatalf("requester fields=%+v", saved)
	}

	msgs := mailer.Messages()
	if len(msgs) != 1 {
		t.Fatalf("messages=%d, want 1", len(msgs))
	}
	if msgs[0].Kind != MailKindOrgCreationSubmitted {
		t.Fatalf("kind=%q", msgs[0].Kind)
	}
	if len(msgs[0].To) != 1 || msgs[0].To[0] != "platform@example.com" {
		t.Fatalf("to=%v", msgs[0].To)
	}

	pending, err := aff.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(pending) != 1 || pending[0].ID != saved.ID {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
}

func TestSubmitOrganizationCreationRequestInvariants(t *testing.T) {
	ctx := context.Background()
	t.Setenv("ADMIN_EMAIL", "platform@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	t.Run("empty name", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u"}, "  ")
		if !errors.Is(err, ErrAffiliationInvalidName) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("already affiliated", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", OrgSlug: "acme"}, "New Org")
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("pending create exists", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow)
		if _, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "First"); err != nil {
			t.Fatalf("first submit: %v", err)
		}
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "Second")
		if !errors.Is(err, ErrAffiliationPendingExists) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("pending join exists", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow)
		if _, err := aff.SaveJoinRequest(ctx, JoinRequest{
			RequesterUserID: "u",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
		}); err != nil {
			t.Fatalf("SaveJoinRequest: %v", err)
		}
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "New Org")
		if !errors.Is(err, ErrAffiliationPendingExists) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("slug already exists", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
				if slug == "taken-org" {
					return &IdentityOrg{ID: "org-1", Slug: "taken-org", Name: "Taken Org"}, nil
				}
				return nil, ErrIdentityNotFound
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "Taken Org")
		if !errors.Is(err, ErrAffiliationOrganizationSlugExists) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestSubmitOrganizationCreationRequestMailFailureDoesNotFailCommand(t *testing.T) {
	ctx := context.Background()
	t.Setenv("ADMIN_EMAIL", "platform@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	mailer := &recordingMailer{err: errors.New("smtp down")}
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), mailer, fixedNow)
	saved, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "New Org")
	if err != nil {
		t.Fatalf("submit should succeed despite mail failure: %v", err)
	}
	if saved.Status != AffiliationStatusPending {
		t.Fatalf("saved=%+v", saved)
	}
	if len(mailer.Messages()) != 1 {
		t.Fatalf("expected mail attempt recorded")
	}
}

func TestApproveOrganizationCreationRequestHappyPath(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}

	var createdName string
	var addedSlug, addedUserID string
	var addedAsAdmin bool
	var updatedLabels []string
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "founder@example.com", Labels: []string{"customKeep"}},
	}
	identity := &fakeIdentityStore{
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			user, ok := users[userID]
			if !ok {
				return IdentityUser{}, ErrIdentityNotFound
			}
			return user, nil
		},
		getOrganizationBySlugFunc: func(_ context.Context, _ string) (*IdentityOrg, error) {
			return nil, ErrIdentityNotFound
		},
		createOrganizationAsAdminFunc: func(_ context.Context, name string) (IdentityOrg, error) {
			createdName = name
			return IdentityOrg{ID: "team-1", Slug: "new-org", Name: name}, nil
		},
		addOrganizationUserByIDAsAdminFunc: func(_ context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
			addedSlug, addedUserID, addedAsAdmin = orgSlug, userID, isOrgAdmin
			if len(roleSlugs) != 0 {
				t.Fatalf("roleSlugs=%v, want nil/empty", roleSlugs)
			}
			return IdentityMembership{ID: "mem-1", UserID: userID, IsOrgAdmin: isOrgAdmin}, nil
		},
		updateUserLabelsFunc: func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
			updatedLabels = append([]string(nil), labels...)
			user := users[userID]
			user.Labels = append([]string(nil), labels...)
			users[userID] = user
			return user, nil
		},
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow)

	saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("seed request: %v", err)
	}

	decidedBy := IdentityUser{ID: "admin-1", Email: "platform@example.com"}
	updated, org, err := aff.ApproveOrganizationCreationRequest(ctx, saved.ID, decidedBy)
	if err != nil {
		t.Fatalf("ApproveOrganizationCreationRequest: %v", err)
	}
	if updated.Status != AffiliationStatusApproved || updated.DecidedByUserID != "admin-1" {
		t.Fatalf("updated=%+v", updated)
	}
	if updated.DecidedAt.IsZero() {
		t.Fatal("expected DecidedAt")
	}
	if org.Slug != "new-org" || createdName != "New Org" {
		t.Fatalf("org=%+v createdName=%q", org, createdName)
	}
	if addedSlug != "new-org" || addedUserID != "user-1" || !addedAsAdmin {
		t.Fatalf("add membership slug=%q user=%q admin=%v", addedSlug, addedUserID, addedAsAdmin)
	}
	if len(updatedLabels) != 2 || updatedLabels[0] != "customKeep" || updatedLabels[1] != identityOrgAdminLabel {
		t.Fatalf("labels=%v", updatedLabels)
	}

	msgs := mailer.Messages()
	if len(msgs) != 1 || msgs[0].Kind != MailKindOrgCreationApproved {
		t.Fatalf("messages=%+v", msgs)
	}
	if len(msgs[0].To) != 1 || msgs[0].To[0] != "founder@example.com" {
		t.Fatalf("to=%v", msgs[0].To)
	}

	pending, err := aff.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending after approve=%v err=%v", pending, err)
	}
}

func TestApproveOrganizationCreationRequestInvariants(t *testing.T) {
	ctx := context.Background()
	decidedBy := IdentityUser{ID: "admin-1"}

	t.Run("not found", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow)
		_, _, err := aff.ApproveOrganizationCreationRequest(ctx, primitive.NewObjectID(), decidedBy)
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("not pending", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow)
		saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			ProposedName:    "New Org",
			ProposedSlug:    "new-org",
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		saved.Status = AffiliationStatusRejected
		if _, err := store.UpdateOrganizationCreationRequest(ctx, saved); err != nil {
			t.Fatalf("update: %v", err)
		}
		_, _, err = aff.ApproveOrganizationCreationRequest(ctx, saved.ID, decidedBy)
		if !errors.Is(err, ErrAffiliationNotPending) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("requester now affiliated leaves pending", func(t *testing.T) {
		store := NewMemoryStore()
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "u@example.com", OrgSlug: "other"}, nil
			},
		}
		aff := NewAffiliation(identity, store, &recordingMailer{}, fixedNow)
		saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			ProposedName:    "New Org",
			ProposedSlug:    "new-org",
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, _, err = aff.ApproveOrganizationCreationRequest(ctx, saved.ID, decidedBy)
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
		loaded, err := aff.LoadOrganizationCreationRequestByID(ctx, saved.ID)
		if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
			t.Fatalf("loaded=%+v err=%v", loaded, err)
		}
	})

	t.Run("slug collision", func(t *testing.T) {
		store := NewMemoryStore()
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{ID: "user-1", Email: "u@example.com"}, nil
			},
			getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
				if slug == "new-org" {
					return &IdentityOrg{ID: "existing", Slug: "new-org", Name: "Existing"}, nil
				}
				return nil, ErrIdentityNotFound
			},
		}
		aff := NewAffiliation(identity, store, &recordingMailer{}, fixedNow)
		saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			ProposedName:    "New Org",
			ProposedSlug:    "new-org",
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, _, err = aff.ApproveOrganizationCreationRequest(ctx, saved.ID, decidedBy)
		if !errors.Is(err, ErrAffiliationOrganizationSlugExists) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestApproveOrganizationCreationRequestMailFailureDoesNotFailCommand(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{err: errors.New("smtp down")}
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "founder@example.com"},
	}
	identity := &fakeIdentityStore{
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			return users[userID], nil
		},
		getOrganizationBySlugFunc: func(_ context.Context, _ string) (*IdentityOrg, error) {
			return nil, ErrIdentityNotFound
		},
		createOrganizationAsAdminFunc: func(_ context.Context, name string) (IdentityOrg, error) {
			return IdentityOrg{ID: "team-1", Slug: "new-org", Name: name}, nil
		},
		addOrganizationUserByIDAsAdminFunc: func(_ context.Context, orgSlug, userID string, _ []string, isOrgAdmin bool) (IdentityMembership, error) {
			return IdentityMembership{ID: "mem-1", UserID: userID, IsOrgAdmin: isOrgAdmin}, nil
		},
		updateUserLabelsFunc: func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
			user := users[userID]
			user.Labels = append([]string(nil), labels...)
			users[userID] = user
			return user, nil
		},
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow)
	saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	updated, _, err := aff.ApproveOrganizationCreationRequest(ctx, saved.ID, IdentityUser{ID: "admin-1"})
	if err != nil {
		t.Fatalf("approve should succeed despite mail failure: %v", err)
	}
	if updated.Status != AffiliationStatusApproved {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestRejectOrganizationCreationRequestAndResubmit(t *testing.T) {
	ctx := context.Background()
	t.Setenv("ADMIN_EMAIL", "platform@example.com")
	t.Setenv("ADMIN_PASSWORD", "secret")

	store := NewMemoryStore()
	mailer := &recordingMailer{}
	aff := NewAffiliation(&fakeIdentityStore{}, store, mailer, fixedNow)
	user := IdentityUser{ID: "user-1", Email: "founder@example.com"}

	saved, err := aff.SubmitOrganizationCreationRequest(ctx, user, "New Org")
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	rejected, err := aff.RejectOrganizationCreationRequest(ctx, saved.ID, IdentityUser{ID: "admin-1"}, "duplicate brand")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != AffiliationStatusRejected || rejected.RejectReason != "duplicate brand" {
		t.Fatalf("rejected=%+v", rejected)
	}
	if rejected.DecidedByUserID != "admin-1" || rejected.DecidedAt.IsZero() {
		t.Fatalf("decision fields=%+v", rejected)
	}

	msgs := mailer.Messages()
	if len(msgs) != 2 {
		t.Fatalf("messages=%d, want submit+reject", len(msgs))
	}
	if msgs[1].Kind != MailKindOrgCreationRejected || msgs[1].To[0] != "founder@example.com" {
		t.Fatalf("reject mail=%+v", msgs[1])
	}

	pending, err := aff.HasPendingAffiliationIntent(ctx, user.ID)
	if err != nil || pending {
		t.Fatalf("pending after reject=%v err=%v", pending, err)
	}

	resubmitted, err := aff.SubmitOrganizationCreationRequest(ctx, user, "Fresh Org")
	if err != nil {
		t.Fatalf("resubmit: %v", err)
	}
	if resubmitted.Status != AffiliationStatusPending || resubmitted.ProposedSlug != "fresh-org" {
		t.Fatalf("resubmitted=%+v", resubmitted)
	}
}

func TestRejectOrganizationCreationRequestInvariants(t *testing.T) {
	ctx := context.Background()
	decidedBy := IdentityUser{ID: "admin-1"}

	t.Run("not found", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow)
		_, err := aff.RejectOrganizationCreationRequest(ctx, primitive.NewObjectID(), decidedBy, "")
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("not pending", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow)
		saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			ProposedName:    "New Org",
			ProposedSlug:    "new-org",
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		saved.Status = AffiliationStatusApproved
		if _, err := store.UpdateOrganizationCreationRequest(ctx, saved); err != nil {
			t.Fatalf("update: %v", err)
		}
		_, err = aff.RejectOrganizationCreationRequest(ctx, saved.ID, decidedBy, "late")
		if !errors.Is(err, ErrAffiliationNotPending) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestRejectOrganizationCreationRequestMailFailureDoesNotFailCommand(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{err: errors.New("smtp down")}
	aff := NewAffiliation(&fakeIdentityStore{}, store, mailer, fixedNow)
	saved, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	updated, err := aff.RejectOrganizationCreationRequest(ctx, saved.ID, IdentityUser{ID: "admin-1"}, "nope")
	if err != nil {
		t.Fatalf("reject should succeed despite mail failure: %v", err)
	}
	if updated.Status != AffiliationStatusRejected {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestListPendingOrganizationCreationRequestsFiltersNonPending(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow)

	pendingReq, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "a@example.com",
		ProposedName:    "Pending Org",
		ProposedSlug:    "pending-org",
	})
	if err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	rejected, err := aff.SaveOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-2",
		RequesterEmail:  "b@example.com",
		ProposedName:    "Rejected Org",
		ProposedSlug:    "rejected-org",
	})
	if err != nil {
		t.Fatalf("seed rejected: %v", err)
	}
	rejected.Status = AffiliationStatusRejected
	if _, err := store.UpdateOrganizationCreationRequest(ctx, rejected); err != nil {
		t.Fatalf("mark rejected: %v", err)
	}

	listed, err := aff.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != pendingReq.ID {
		t.Fatalf("listed=%v err=%v", listed, err)
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
}
