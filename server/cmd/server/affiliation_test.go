package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAffiliationIsAffiliated(t *testing.T) {
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, time.Now, nil)

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

func TestAffiliationEnsureInviteOrgSlugCompatible(t *testing.T) {
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, time.Now, nil)

	t.Run("unaffiliated ok", func(t *testing.T) {
		if err := aff.EnsureInviteOrgSlugCompatible(IdentityUser{ID: "u"}, "acme"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("whitespace org slug unaffiliated ok", func(t *testing.T) {
		if err := aff.EnsureInviteOrgSlugCompatible(IdentityUser{ID: "u", OrgSlug: "  "}, "acme"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("same org ok", func(t *testing.T) {
		if err := aff.EnsureInviteOrgSlugCompatible(IdentityUser{ID: "u", OrgSlug: "Acme"}, "acme"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("cross org rejected", func(t *testing.T) {
		err := aff.EnsureInviteOrgSlugCompatible(IdentityUser{ID: "u", OrgSlug: "acme"}, "other")
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestAffiliationEnsureInviteAcceptCompatible(t *testing.T) {
	ctx := context.Background()

	t.Run("user not found ok", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{}, ErrIdentityNotFound
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, time.Now, nil)
		if err := aff.EnsureInviteAcceptCompatible(ctx, "user-1", "team-acme"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("unaffiliated ok", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "new@example.com"}, nil
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, time.Now, nil)
		if err := aff.EnsureInviteAcceptCompatible(ctx, "user-1", "team-acme"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("same org ok", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "m@example.com", OrgSlug: "acme"}, nil
			},
			getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
				if slug != "acme" {
					return nil, ErrIdentityNotFound
				}
				return &IdentityOrg{ID: "team-acme", Slug: "acme"}, nil
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, time.Now, nil)
		if err := aff.EnsureInviteAcceptCompatible(ctx, "user-1", "team-acme"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("cross org rejected", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "m@example.com", OrgSlug: "acme"}, nil
			},
			getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
				return &IdentityOrg{ID: "team-acme", Slug: "acme"}, nil
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, time.Now, nil)
		err := aff.EnsureInviteAcceptCompatible(ctx, "user-1", "team-other")
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org missing rejected", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "m@example.com", OrgSlug: "ghost"}, nil
			},
			getOrganizationBySlugFunc: func(_ context.Context, _ string) (*IdentityOrg, error) {
				return nil, ErrIdentityNotFound
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, time.Now, nil)
		err := aff.EnsureInviteAcceptCompatible(ctx, "user-1", "team-acme")
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("get user failure propagates", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{}, errors.New("identity down")
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, time.Now, nil)
		err := aff.EnsureInviteAcceptCompatible(ctx, "user-1", "team-acme")
		if err == nil || err.Error() != "identity down" {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestAffiliationPendingIntentEmpty(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, time.Now, nil)

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

func TestWithdrawPendingJoinAndOrgCreationRequests(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	identity := &fakeIdentityStore{
		getOrganizationBySlugFunc: func(ctx context.Context, slug string) (*IdentityOrg, error) {
			if slug == "acme" {
				return &IdentityOrg{
					Slug:  "acme",
					Name:  "Acme",
					Roles: []IdentityRole{{Slug: "operator", Name: "Operator"}},
				}, nil
			}
			return nil, ErrIdentityNotFound
		},
		listOrganizationUsersFunc: func(ctx context.Context, orgSlug string) ([]IdentityUser, error) {
			return []IdentityUser{{ID: "admin-1", Email: "a@example.com", IsOrgAdmin: true}}, nil
		},
	}
	aff := NewAffiliation(identity, store, &recordingMailer{}, time.Now, []string{"platform@example.com"})
	user := IdentityUser{ID: "user-1", Email: "u@example.com"}

	if _, err := aff.SubmitJoinRequest(ctx, user, "acme", []string{"operator"}); err != nil {
		t.Fatalf("SubmitJoinRequest: %v", err)
	}
	if err := aff.WithdrawPendingJoinRequest(ctx, user); err != nil {
		t.Fatalf("WithdrawPendingJoinRequest: %v", err)
	}
	pending, err := aff.HasPendingAffiliationIntent(ctx, user.ID)
	if err != nil || pending {
		t.Fatalf("pending after join withdraw=%v err=%v", pending, err)
	}
	if err := aff.WithdrawPendingJoinRequest(ctx, user); !errors.Is(err, ErrAffiliationNotFound) {
		t.Fatalf("second withdraw err=%v, want NotFound", err)
	}

	if _, err := aff.SubmitOrganizationCreationRequest(ctx, user, "New Co"); err != nil {
		t.Fatalf("SubmitOrganizationCreationRequest: %v", err)
	}
	if err := aff.WithdrawPendingOrganizationCreationRequest(ctx, user); err != nil {
		t.Fatalf("WithdrawPendingOrganizationCreationRequest: %v", err)
	}
	pending, err = aff.HasPendingAffiliationIntent(ctx, user.ID)
	if err != nil || pending {
		t.Fatalf("pending after org withdraw=%v err=%v", pending, err)
	}
}

func TestAffiliationJoinRequestRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, time.Now, nil)

	saved, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "user@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer", "editor"},
	})
	if err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
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

	loaded, err := store.LoadJoinRequestByID(ctx, saved.ID)
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

	pendingListed, err := store.ListPendingJoinRequestsByOrg(ctx, "acme")
	if err != nil || len(pendingListed) != 1 || pendingListed[0].ID != saved.ID {
		t.Fatalf("pending listed=%v err=%v", pendingListed, err)
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

	pendingAfter, err := aff.PendingJoinRequestForUser(ctx, "user-1")
	if err != nil || pendingAfter != nil {
		t.Fatalf("pending after approve=%+v err=%v", pendingAfter, err)
	}
	pendingListedAfter, err := store.ListPendingJoinRequestsByOrg(ctx, "acme")
	if err != nil || len(pendingListedAfter) != 0 {
		t.Fatalf("pending listed after approve=%v err=%v", pendingListedAfter, err)
	}
}

func TestAffiliationOrganizationCreationRequestRoundTrip(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, time.Now, nil)

	saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-2",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("InsertOrganizationCreationRequest: %v", err)
	}
	if saved.ID.IsZero() || saved.Status != AffiliationStatusPending {
		t.Fatalf("saved=%+v", saved)
	}

	loaded, err := store.LoadOrganizationCreationRequestByID(ctx, saved.ID)
	if err != nil || loaded == nil || loaded.ProposedSlug != "new-org" {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}

	pending, err := aff.PendingOrganizationCreationRequestForUser(ctx, "user-2")
	if err != nil || pending == nil || pending.ID != saved.ID {
		t.Fatalf("pending=%+v err=%v", pending, err)
	}

	pendingListed, err := store.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(pendingListed) != 1 || pendingListed[0].ID != saved.ID {
		t.Fatalf("pending listed=%v err=%v", pendingListed, err)
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

	pendingAfter, err := aff.PendingOrganizationCreationRequestForUser(ctx, "user-2")
	if err != nil || pendingAfter != nil {
		t.Fatalf("pending after reject=%+v err=%v", pendingAfter, err)
	}
	pendingListedAfter, err := store.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(pendingListedAfter) != 0 {
		t.Fatalf("pending listed after reject=%v err=%v", pendingListedAfter, err)
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
	saved, err := store.InsertJoinRequest(context.Background(), JoinRequest{
		RequesterUserID: "u",
		RequesterEmail:  "u@example.com",
		OrgSlug:         "acme",
	})
	if err != nil || saved.ID == (primitive.ObjectID{}) {
		t.Fatalf("InsertJoinRequest via store: saved=%+v err=%v", saved, err)
	}
}

func TestSubmitOrganizationCreationRequestHappyPath(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{publicBaseURL: "https://attesta.example"}
	aff := NewAffiliation(&fakeIdentityStore{}, store, mailer, fixedNow, []string{"platform@example.com"})

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
	if !strings.Contains(msgs[0].Body, "Open: https://attesta.example/admin/organizations") {
		t.Fatalf("body missing action link: %q", msgs[0].Body)
	}

	pending, err := aff.ListPendingOrganizationCreationRequests(ctx)
	if err != nil || len(pending) != 1 || pending[0].ID != saved.ID {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
}

func TestSubmitOrganizationCreationRequestInvariants(t *testing.T) {
	ctx := context.Background()

	t.Run("empty name", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u"}, "  ")
		if !errors.Is(err, ErrAffiliationInvalidName) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("already affiliated", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", OrgSlug: "acme"}, "New Org")
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("pending create exists", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow, nil)
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
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow, nil)
		if _, err := store.InsertJoinRequest(ctx, JoinRequest{
			RequesterUserID: "u",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
		}); err != nil {
			t.Fatalf("InsertJoinRequest: %v", err)
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
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "Taken Org")
		if !errors.Is(err, ErrAffiliationOrganizationSlugExists) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestSubmitOrganizationCreationRequestMailFailureDoesNotFailCommand(t *testing.T) {
	ctx := context.Background()

	mailer := &recordingMailer{err: errors.New("smtp down")}
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), mailer, fixedNow, []string{"platform@example.com"})
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
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)

	saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, _, err := aff.ApproveOrganizationCreationRequest(ctx, primitive.NewObjectID(), decidedBy)
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("not pending", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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
		aff := NewAffiliation(identity, store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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
		loaded, err := store.LoadOrganizationCreationRequestByID(ctx, saved.ID)
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
		aff := NewAffiliation(identity, store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)
	saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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

	store := NewMemoryStore()
	mailer := &recordingMailer{}
	aff := NewAffiliation(&fakeIdentityStore{}, store, mailer, fixedNow, []string{"platform@example.com"})
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
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.RejectOrganizationCreationRequest(ctx, primitive.NewObjectID(), decidedBy, "")
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("not pending", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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
	aff := NewAffiliation(&fakeIdentityStore{}, store, mailer, fixedNow, nil)
	saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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
	aff := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow, nil)

	pendingReq, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "a@example.com",
		ProposedName:    "Pending Org",
		ProposedSlug:    "pending-org",
	})
	if err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	rejected, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
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

func TestAffiliationRequestableJoinRoles(t *testing.T) {
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
	org := IdentityOrg{
		Slug: "acme",
		Name: "Acme",
		Roles: []IdentityRole{
			{Slug: "viewer", Name: "Viewer"},
			{Slug: "org-admin", Name: "Org Admin"},
			{Slug: "org_admin", Name: "Org Admin Underscore"},
			{Slug: "editor", Name: "Editor"},
		},
	}
	got := aff.RequestableJoinRoles(org)
	if len(got) != 2 {
		t.Fatalf("RequestableJoinRoles len=%d, want 2; got=%+v", len(got), got)
	}
	if got[0].Slug != "viewer" || got[1].Slug != "editor" {
		t.Fatalf("RequestableJoinRoles=%+v, want viewer then editor", got)
	}
}

func TestSubmitJoinRequestHappyPath(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	identity := joinRequestTestIdentity(nil)
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)

	user := IdentityUser{ID: "user-1", Email: "joiner@example.com"}
	saved, err := aff.SubmitJoinRequest(ctx, user, "acme", []string{" viewer ", "editor", "viewer"})
	if err != nil {
		t.Fatalf("SubmitJoinRequest: %v", err)
	}
	if saved.ID.IsZero() || saved.Status != AffiliationStatusPending {
		t.Fatalf("saved=%+v", saved)
	}
	if saved.OrgSlug != "acme" || saved.RequesterUserID != "user-1" || saved.RequesterEmail != "joiner@example.com" {
		t.Fatalf("saved fields=%+v", saved)
	}
	if len(saved.RoleSlugs) != 2 || saved.RoleSlugs[0] != "viewer" || saved.RoleSlugs[1] != "editor" {
		t.Fatalf("roleSlugs=%v", saved.RoleSlugs)
	}

	msgs := mailer.Messages()
	if len(msgs) != 1 || msgs[0].Kind != MailKindJoinSubmitted {
		t.Fatalf("messages=%+v", msgs)
	}
	if len(msgs[0].To) != 1 || msgs[0].To[0] != "owner@example.com" {
		t.Fatalf("to=%v", msgs[0].To)
	}

	pending, err := aff.ListPendingJoinRequests(ctx, "acme")
	if err != nil || len(pending) != 1 || pending[0].ID != saved.ID {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
}

func TestSubmitJoinRequestInvariants(t *testing.T) {
	ctx := context.Background()

	t.Run("already affiliated", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", OrgSlug: "other"}, "acme", []string{"viewer"})
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("pending join exists", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(joinRequestTestIdentity(nil), store, &recordingMailer{}, fixedNow, nil)
		user := IdentityUser{ID: "u", Email: "u@example.com"}
		if _, err := aff.SubmitJoinRequest(ctx, user, "acme", []string{"viewer"}); err != nil {
			t.Fatalf("first: %v", err)
		}
		_, err := aff.SubmitJoinRequest(ctx, user, "acme", []string{"editor"})
		if !errors.Is(err, ErrAffiliationPendingExists) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("pending create exists", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(joinRequestTestIdentity(nil), store, &recordingMailer{}, fixedNow, nil)
		if _, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			RequesterUserID: "u",
			RequesterEmail:  "u@example.com",
			ProposedName:    "New Org",
			ProposedSlug:    "new-org",
		}); err != nil {
			t.Fatalf("seed create: %v", err)
		}
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", []string{"viewer"})
		if !errors.Is(err, ErrAffiliationPendingExists) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org missing", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "missing", []string{"viewer"})
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("empty org slug", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u"}, "  ", []string{"viewer"})
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("zero roles", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", nil)
		if !errors.Is(err, ErrAffiliationInvalidRoles) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("unknown role", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", []string{"ghost"})
		if !errors.Is(err, ErrAffiliationInvalidRoles) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org-admin rejected", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", []string{"org-admin"})
		if !errors.Is(err, ErrAffiliationInvalidRoles) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org_admin rejected", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", []string{"org_admin"})
		if !errors.Is(err, ErrAffiliationInvalidRoles) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestSubmitJoinRequestMailFailureDoesNotFailCommand(t *testing.T) {
	ctx := context.Background()
	mailer := &recordingMailer{err: errors.New("smtp down")}
	aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), mailer, fixedNow, nil)
	saved, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", []string{"viewer"})
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

func TestApproveJoinRequestHappyPath(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}

	var addedSlug, addedUserID string
	var addedRoles []string
	var addedAsAdmin bool
	var updatedLabels []string
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "joiner@example.com", Labels: []string{"customKeep", encodeIdentityRoleLabel("stale")}},
	}
	identity := joinRequestTestIdentity(users)
	identity.addOrganizationUserByIDAsAdminFunc = func(_ context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		addedSlug, addedUserID, addedAsAdmin = orgSlug, userID, isOrgAdmin
		addedRoles = append([]string(nil), roleSlugs...)
		return IdentityMembership{ID: "mem-1", UserID: userID, RoleSlugs: roleSlugs, IsOrgAdmin: isOrgAdmin, Confirmed: true}, nil
	}
	identity.updateUserLabelsFunc = func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
		updatedLabels = append([]string(nil), labels...)
		user := users[userID]
		user.Labels = append([]string(nil), labels...)
		users[userID] = user
		return user, nil
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)

	saved, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer", "editor"},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	decidedBy := IdentityUser{ID: "admin-1", Email: "owner@example.com", OrgSlug: "acme"}
	updated, err := aff.ApproveJoinRequest(ctx, saved.ID, decidedBy)
	if err != nil {
		t.Fatalf("ApproveJoinRequest: %v", err)
	}
	if updated.Status != AffiliationStatusApproved || updated.DecidedByUserID != "admin-1" {
		t.Fatalf("updated=%+v", updated)
	}
	if updated.DecidedAt.IsZero() {
		t.Fatal("expected DecidedAt")
	}
	if addedSlug != "acme" || addedUserID != "user-1" || addedAsAdmin {
		t.Fatalf("add membership slug=%q user=%q admin=%v", addedSlug, addedUserID, addedAsAdmin)
	}
	if len(addedRoles) != 2 || addedRoles[0] != "viewer" || addedRoles[1] != "editor" {
		t.Fatalf("addedRoles=%v", addedRoles)
	}
	if len(updatedLabels) != 3 || updatedLabels[0] != "customKeep" {
		t.Fatalf("labels=%v", updatedLabels)
	}
	if updatedLabels[1] != encodeIdentityRoleLabel("viewer") || updatedLabels[2] != encodeIdentityRoleLabel("editor") {
		t.Fatalf("role labels=%v", updatedLabels)
	}
	for _, label := range updatedLabels {
		if label == identityOrgAdminLabel {
			t.Fatal("must not stamp org-admin label")
		}
		if label == encodeIdentityRoleLabel("stale") {
			t.Fatal("managed stale role label should be replaced")
		}
	}

	msgs := mailer.Messages()
	if len(msgs) != 1 || msgs[0].Kind != MailKindJoinApproved {
		t.Fatalf("messages=%+v", msgs)
	}
	if len(msgs[0].To) != 1 || msgs[0].To[0] != "joiner@example.com" {
		t.Fatalf("to=%v", msgs[0].To)
	}

	pending, err := aff.ListPendingJoinRequests(ctx, "acme")
	if err != nil || len(pending) != 0 {
		t.Fatalf("pending after approve=%v err=%v", pending, err)
	}
}

func TestApproveJoinRequestInvariants(t *testing.T) {
	ctx := context.Background()
	decidedBy := IdentityUser{ID: "admin-1", OrgSlug: "acme"}

	t.Run("not found", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.ApproveJoinRequest(ctx, primitive.NewObjectID(), decidedBy)
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("not pending", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(joinRequestTestIdentity(nil), store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertJoinRequest(ctx, JoinRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		saved.Status = AffiliationStatusRejected
		if _, err := store.UpdateJoinRequest(ctx, saved); err != nil {
			t.Fatalf("update: %v", err)
		}
		_, err = aff.ApproveJoinRequest(ctx, saved.ID, decidedBy)
		if !errors.Is(err, ErrAffiliationNotPending) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("decider org mismatch", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(joinRequestTestIdentity(nil), store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertJoinRequest(ctx, JoinRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, err = aff.ApproveJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin-1", OrgSlug: "other"})
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
		loaded, err := store.LoadJoinRequestByID(ctx, saved.ID)
		if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
			t.Fatalf("loaded=%+v err=%v", loaded, err)
		}
	})

	t.Run("requester now affiliated leaves pending", func(t *testing.T) {
		store := NewMemoryStore()
		users := map[string]IdentityUser{
			"user-1": {ID: "user-1", Email: "u@example.com", OrgSlug: "other"},
		}
		aff := NewAffiliation(joinRequestTestIdentity(users), store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertJoinRequest(ctx, JoinRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, err = aff.ApproveJoinRequest(ctx, saved.ID, decidedBy)
		if !errors.Is(err, ErrAffiliationAlreadyAffiliated) {
			t.Fatalf("err=%v", err)
		}
		loaded, err := store.LoadJoinRequestByID(ctx, saved.ID)
		if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
			t.Fatalf("loaded=%+v err=%v", loaded, err)
		}
	})

	t.Run("roles revalidated on approve", func(t *testing.T) {
		store := NewMemoryStore()
		users := map[string]IdentityUser{
			"user-1": {ID: "user-1", Email: "u@example.com"},
		}
		identity := joinRequestTestIdentity(users)
		identity.getOrganizationBySlugFunc = func(_ context.Context, slug string) (*IdentityOrg, error) {
			if slug != "acme" {
				return nil, ErrIdentityNotFound
			}
			// Role catalog no longer includes "viewer".
			return &IdentityOrg{
				ID:   "team-1",
				Slug: "acme",
				Name: "Acme",
				Roles: []IdentityRole{
					{Slug: "editor", Name: "Editor"},
				},
			}, nil
		}
		aff := NewAffiliation(identity, store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertJoinRequest(ctx, JoinRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, err = aff.ApproveJoinRequest(ctx, saved.ID, decidedBy)
		if !errors.Is(err, ErrAffiliationInvalidRoles) {
			t.Fatalf("err=%v", err)
		}
		loaded, err := store.LoadJoinRequestByID(ctx, saved.ID)
		if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
			t.Fatalf("loaded=%+v err=%v", loaded, err)
		}
	})
}

func TestApproveJoinRequestMailFailureDoesNotFailCommand(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{err: errors.New("smtp down")}
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "joiner@example.com"},
	}
	identity := joinRequestTestIdentity(users)
	identity.addOrganizationUserByIDAsAdminFunc = func(_ context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		return IdentityMembership{ID: "mem-1", UserID: userID, RoleSlugs: roleSlugs, IsOrgAdmin: isOrgAdmin}, nil
	}
	identity.updateUserLabelsFunc = func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
		user := users[userID]
		user.Labels = append([]string(nil), labels...)
		users[userID] = user
		return user, nil
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)
	saved, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	updated, err := aff.ApproveJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin-1", OrgSlug: "acme"})
	if err != nil {
		t.Fatalf("approve should succeed despite mail failure: %v", err)
	}
	if updated.Status != AffiliationStatusApproved {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestRejectJoinRequestAndResubmit(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	aff := NewAffiliation(joinRequestTestIdentity(nil), store, mailer, fixedNow, nil)
	user := IdentityUser{ID: "user-1", Email: "joiner@example.com"}

	saved, err := aff.SubmitJoinRequest(ctx, user, "acme", []string{"viewer"})
	if err != nil {
		t.Fatalf("submit: %v", err)
	}

	rejected, err := aff.RejectJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin-1", OrgSlug: "acme"}, "not a fit")
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if rejected.Status != AffiliationStatusRejected || rejected.RejectReason != "not a fit" {
		t.Fatalf("rejected=%+v", rejected)
	}
	if rejected.DecidedByUserID != "admin-1" || rejected.DecidedAt.IsZero() {
		t.Fatalf("decision fields=%+v", rejected)
	}

	msgs := mailer.Messages()
	if len(msgs) != 2 {
		t.Fatalf("messages=%d, want submit+reject", len(msgs))
	}
	if msgs[1].Kind != MailKindJoinRejected || msgs[1].To[0] != "joiner@example.com" {
		t.Fatalf("reject mail=%+v", msgs[1])
	}

	pending, err := aff.HasPendingAffiliationIntent(ctx, user.ID)
	if err != nil || pending {
		t.Fatalf("pending after reject=%v err=%v", pending, err)
	}

	resubmitted, err := aff.SubmitJoinRequest(ctx, user, "acme", []string{"editor"})
	if err != nil {
		t.Fatalf("resubmit: %v", err)
	}
	if resubmitted.Status != AffiliationStatusPending || len(resubmitted.RoleSlugs) != 1 || resubmitted.RoleSlugs[0] != "editor" {
		t.Fatalf("resubmitted=%+v", resubmitted)
	}
}

func TestRejectJoinRequestInvariants(t *testing.T) {
	ctx := context.Background()
	decidedBy := IdentityUser{ID: "admin-1", OrgSlug: "acme"}

	t.Run("not found", func(t *testing.T) {
		aff := NewAffiliation(joinRequestTestIdentity(nil), NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		_, err := aff.RejectJoinRequest(ctx, primitive.NewObjectID(), decidedBy, "")
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("not pending", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(joinRequestTestIdentity(nil), store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertJoinRequest(ctx, JoinRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		saved.Status = AffiliationStatusApproved
		if _, err := store.UpdateJoinRequest(ctx, saved); err != nil {
			t.Fatalf("update: %v", err)
		}
		_, err = aff.RejectJoinRequest(ctx, saved.ID, decidedBy, "late")
		if !errors.Is(err, ErrAffiliationNotPending) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("decider org mismatch", func(t *testing.T) {
		store := NewMemoryStore()
		aff := NewAffiliation(joinRequestTestIdentity(nil), store, &recordingMailer{}, fixedNow, nil)
		saved, err := store.InsertJoinRequest(ctx, JoinRequest{
			RequesterUserID: "user-1",
			RequesterEmail:  "u@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
		})
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, err = aff.RejectJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin-1", OrgSlug: "other"}, "nope")
		if !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestRejectJoinRequestMailFailureDoesNotFailCommand(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{err: errors.New("smtp down")}
	aff := NewAffiliation(joinRequestTestIdentity(nil), store, mailer, fixedNow, nil)
	saved, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	updated, err := aff.RejectJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin-1", OrgSlug: "acme"}, "nope")
	if err != nil {
		t.Fatalf("reject should succeed despite mail failure: %v", err)
	}
	if updated.Status != AffiliationStatusRejected {
		t.Fatalf("updated=%+v", updated)
	}
}

func TestListPendingJoinRequestsFiltersNonPending(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	aff := NewAffiliation(joinRequestTestIdentity(nil), store, &recordingMailer{}, fixedNow, nil)

	pendingReq, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "a@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
	})
	if err != nil {
		t.Fatalf("seed pending: %v", err)
	}
	rejected, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-2",
		RequesterEmail:  "b@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"editor"},
	})
	if err != nil {
		t.Fatalf("seed rejected: %v", err)
	}
	rejected.Status = AffiliationStatusRejected
	if _, err := store.UpdateJoinRequest(ctx, rejected); err != nil {
		t.Fatalf("mark rejected: %v", err)
	}
	if _, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-3",
		RequesterEmail:  "c@example.com",
		OrgSlug:         "other",
		RoleSlugs:       []string{"viewer"},
	}); err != nil {
		t.Fatalf("seed other org: %v", err)
	}

	listed, err := aff.ListPendingJoinRequests(ctx, "acme")
	if err != nil || len(listed) != 1 || listed[0].ID != pendingReq.ID {
		t.Fatalf("listed=%v err=%v", listed, err)
	}
}

func joinRequestTestIdentity(users map[string]IdentityUser) *fakeIdentityStore {
	if users == nil {
		users = map[string]IdentityUser{}
	}
	return &fakeIdentityStore{
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			user, ok := users[userID]
			if !ok {
				return IdentityUser{}, ErrIdentityNotFound
			}
			return user, nil
		},
		getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
			if slug != "acme" {
				return nil, ErrIdentityNotFound
			}
			return &IdentityOrg{
				ID:   "team-1",
				Slug: "acme",
				Name: "Acme",
				Roles: []IdentityRole{
					{Slug: "viewer", Name: "Viewer"},
					{Slug: "editor", Name: "Editor"},
					{Slug: "org-admin", Name: "Org Admin"},
				},
			}, nil
		},
		listOrganizationUsersFunc: func(_ context.Context, orgSlug string) ([]IdentityUser, error) {
			if orgSlug != "acme" {
				return nil, nil
			}
			return []IdentityUser{
				{ID: "admin-1", Email: "owner@example.com", OrgSlug: "acme", IsOrgAdmin: true},
				{ID: "member-1", Email: "member@example.com", OrgSlug: "acme", IsOrgAdmin: false},
			}, nil
		},
	}
}

func fixedNow() time.Time {
	return time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
}

func TestAffiliationLeaveOrganizationNotAffiliated(t *testing.T) {
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
	err := aff.LeaveOrganization(context.Background(), "session", IdentityUser{ID: "user-1"})
	if !errors.Is(err, ErrAffiliationNotAffiliated) {
		t.Fatalf("LeaveOrganization error = %v, want %v", err, ErrAffiliationNotAffiliated)
	}
}

func TestAffiliationLeaveOrganizationMemberAllowed(t *testing.T) {
	ctx := context.Background()
	mailer := &recordingMailer{}
	var deleted struct {
		sessionSecret string
		orgSlug       string
		membershipID  string
	}
	var updatedLabels []string
	var updatedUserID string
	usersByID := map[string]IdentityUser{
		"member-1": {
			ID:           "member-1",
			Email:        "member@example.com",
			OrgSlug:      "acme",
			MembershipID: "mem-member",
			Labels:       []string{"custom:keep", encodeIdentityRoleLabel("viewer"), identityOrgAdminLabel},
			IsOrgAdmin:   false,
		},
	}
	identity := &fakeIdentityStore{
		getCurrentUserFunc: func(_ context.Context, sessionSecret string) (IdentityUser, error) {
			if sessionSecret != "session-member" {
				return IdentityUser{}, ErrIdentityUnauthorized
			}
			return usersByID["member-1"], nil
		},
		listOrganizationUsersFunc: func(_ context.Context, orgSlug string) ([]IdentityUser, error) {
			t.Fatalf("ListOrganizationUsers should not run for non-admin leave, org=%q", orgSlug)
			return nil, nil
		},
		deleteOrganizationMembershipFunc: func(_ context.Context, sessionSecret, orgSlug, membershipID string) error {
			deleted.sessionSecret = sessionSecret
			deleted.orgSlug = orgSlug
			deleted.membershipID = membershipID
			u := usersByID["member-1"]
			u.OrgSlug = ""
			u.MembershipID = ""
			usersByID["member-1"] = u
			return nil
		},
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			user, ok := usersByID[userID]
			if !ok {
				return IdentityUser{}, ErrIdentityNotFound
			}
			return user, nil
		},
		updateUserLabelsFunc: func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
			updatedUserID = userID
			updatedLabels = append([]string(nil), labels...)
			user := usersByID[userID]
			user.Labels = append([]string(nil), labels...)
			usersByID[userID] = user
			return user, nil
		},
	}
	aff := NewAffiliation(identity, NewMemoryStore(), mailer, fixedNow, nil)

	err := aff.LeaveOrganization(ctx, "session-member", IdentityUser{
		ID:      "member-1",
		OrgSlug: "acme",
	})
	if err != nil {
		t.Fatalf("LeaveOrganization: %v", err)
	}
	if deleted.sessionSecret != "session-member" || deleted.orgSlug != "acme" || deleted.membershipID != "mem-member" {
		t.Fatalf("delete params = %#v", deleted)
	}
	if updatedUserID != "member-1" {
		t.Fatalf("updated user id = %q", updatedUserID)
	}
	if len(updatedLabels) != 1 || updatedLabels[0] != "custom:keep" {
		t.Fatalf("updated labels = %#v, want [custom:keep]", updatedLabels)
	}
	if msgs := mailer.Messages(); len(msgs) != 0 {
		t.Fatalf("expected no mail on leave, got %#v", msgs)
	}
}

func TestAffiliationLeaveOrganizationOrgAdminWithPeerAllowed(t *testing.T) {
	ctx := context.Background()
	mailer := &recordingMailer{}
	var deletedMembershipID string
	usersByID := map[string]IdentityUser{
		"admin-1": {
			ID:           "admin-1",
			Email:        "owner@example.com",
			OrgSlug:      "acme",
			MembershipID: "mem-admin-1",
			Labels:       []string{identityOrgAdminLabel, "custom:keep"},
			IsOrgAdmin:   true,
		},
	}
	identity := &fakeIdentityStore{
		getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
			return usersByID["admin-1"], nil
		},
		listOrganizationUsersFunc: func(_ context.Context, orgSlug string) ([]IdentityUser, error) {
			if orgSlug != "acme" {
				return nil, nil
			}
			return []IdentityUser{
				{ID: "admin-1", OrgSlug: "acme", IsOrgAdmin: true},
				{ID: "admin-2", OrgSlug: "acme", IsOrgAdmin: true},
			}, nil
		},
		deleteOrganizationMembershipFunc: func(_ context.Context, _, _, membershipID string) error {
			deletedMembershipID = membershipID
			return nil
		},
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			return usersByID[userID], nil
		},
		updateUserLabelsFunc: func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
			user := usersByID[userID]
			user.Labels = append([]string(nil), labels...)
			usersByID[userID] = user
			return user, nil
		},
	}
	aff := NewAffiliation(identity, NewMemoryStore(), mailer, fixedNow, nil)

	if err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "admin-1", OrgSlug: "acme"}); err != nil {
		t.Fatalf("LeaveOrganization: %v", err)
	}
	if deletedMembershipID != "mem-admin-1" {
		t.Fatalf("membership id = %q", deletedMembershipID)
	}
	if got := usersByID["admin-1"].Labels; len(got) != 1 || got[0] != "custom:keep" {
		t.Fatalf("labels after leave = %#v", got)
	}
	if msgs := mailer.Messages(); len(msgs) != 0 {
		t.Fatalf("expected no mail on leave, got %#v", msgs)
	}
}

func TestAffiliationLeaveOrganizationSoleOrgAdminDenied(t *testing.T) {
	ctx := context.Background()
	var deleteCalled bool
	var labelsCalled bool
	identity := &fakeIdentityStore{
		getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
			return IdentityUser{
				ID:           "admin-1",
				OrgSlug:      "acme",
				MembershipID: "mem-admin-1",
				IsOrgAdmin:   true,
				Labels:       []string{identityOrgAdminLabel},
			}, nil
		},
		listOrganizationUsersFunc: func(_ context.Context, _ string) ([]IdentityUser, error) {
			return []IdentityUser{
				{ID: "admin-1", OrgSlug: "acme", IsOrgAdmin: true},
				{ID: "member-1", OrgSlug: "acme", IsOrgAdmin: false},
			}, nil
		},
		deleteOrganizationMembershipFunc: func(_ context.Context, _, _, _ string) error {
			deleteCalled = true
			return nil
		},
		updateUserLabelsFunc: func(_ context.Context, _ string, labels []string) (IdentityUser, error) {
			labelsCalled = true
			return IdentityUser{Labels: labels}, nil
		},
	}
	aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)

	err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "admin-1", OrgSlug: "acme"})
	if !errors.Is(err, ErrAffiliationSoleOrgAdmin) {
		t.Fatalf("LeaveOrganization error = %v, want %v", err, ErrAffiliationSoleOrgAdmin)
	}
	if deleteCalled || labelsCalled {
		t.Fatalf("sole admin leave must not mutate: delete=%v labels=%v", deleteCalled, labelsCalled)
	}
}

func TestAffiliationLeaveOrganizationSessionUserMismatch(t *testing.T) {
	identity := &fakeIdentityStore{
		getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
			return IdentityUser{ID: "other-user", OrgSlug: "acme", MembershipID: "mem-1"}, nil
		},
	}
	aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
	err := aff.LeaveOrganization(context.Background(), "session", IdentityUser{ID: "user-1", OrgSlug: "acme"})
	if !errors.Is(err, ErrIdentityUnauthorized) {
		t.Fatalf("LeaveOrganization error = %v, want %v", err, ErrIdentityUnauthorized)
	}
}

func TestAffiliationLeaveOrganizationThenJoinAllowed(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	left := false
	identity := joinRequestTestIdentity(map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "member@example.com"},
	})
	identity.getCurrentUserFunc = func(_ context.Context, _ string) (IdentityUser, error) {
		if left {
			return IdentityUser{ID: "user-1", Email: "member@example.com"}, nil
		}
		return IdentityUser{
			ID:           "user-1",
			Email:        "member@example.com",
			OrgSlug:      "acme",
			MembershipID: "mem-1",
			IsOrgAdmin:   false,
			Labels:       []string{encodeIdentityRoleLabel("viewer")},
		}, nil
	}
	identity.deleteOrganizationMembershipFunc = func(_ context.Context, _, _, _ string) error {
		left = true
		return nil
	}
	identity.updateUserLabelsFunc = func(_ context.Context, userID string, labels []string) (IdentityUser, error) {
		return IdentityUser{ID: userID, Labels: labels}, nil
	}
	identity.getUserByIDFunc = func(_ context.Context, userID string) (IdentityUser, error) {
		if left {
			return IdentityUser{ID: userID, Email: "member@example.com", Labels: []string{"custom:keep"}}, nil
		}
		return IdentityUser{
			ID:      userID,
			Email:   "member@example.com",
			OrgSlug: "acme",
			Labels:  []string{"custom:keep", encodeIdentityRoleLabel("viewer")},
		}, nil
	}
	aff := NewAffiliation(identity, store, &recordingMailer{}, fixedNow, nil)

	if err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "user-1", OrgSlug: "acme"}); err != nil {
		t.Fatalf("LeaveOrganization: %v", err)
	}
	saved, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "user-1", Email: "member@example.com"}, "acme", []string{"viewer"})
	if err != nil {
		t.Fatalf("SubmitJoinRequest after leave: %v", err)
	}
	if saved.Status != AffiliationStatusPending {
		t.Fatalf("join status = %q", saved.Status)
	}
}

func TestApproveJoinRequestCompensatesWhenIdentityFails(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "joiner@example.com", Labels: []string{"customKeep"}},
	}
	identity := joinRequestTestIdentity(users)
	var savedID primitive.ObjectID
	var calls []string
	identity.addOrganizationUserByIDAsAdminFunc = func(_ context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		loaded, err := store.LoadJoinRequestByID(ctx, savedID)
		if err != nil || loaded == nil || loaded.Status != AffiliationStatusApproved {
			t.Fatalf("expected approved status before identity add: loaded=%+v err=%v", loaded, err)
		}
		calls = append(calls, "add")
		return IdentityMembership{}, errors.New("add membership failed")
	}
	identity.updateUserLabelsFunc = func(_ context.Context, _ string, _ []string) (IdentityUser, error) {
		calls = append(calls, "labels")
		return IdentityUser{}, nil
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)

	saved, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	savedID = saved.ID

	_, err = aff.ApproveJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin-1", OrgSlug: "acme"})
	if err == nil || err.Error() != "add membership failed" {
		t.Fatalf("err=%v", err)
	}
	if len(calls) != 1 || calls[0] != "add" {
		t.Fatalf("calls=%v", calls)
	}
	loaded, err := store.LoadJoinRequestByID(ctx, saved.ID)
	if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if loaded.DecidedByUserID != "" || !loaded.DecidedAt.IsZero() || loaded.RejectReason != "" {
		t.Fatalf("decided fields not cleared: %+v", loaded)
	}
	if len(mailer.Messages()) != 0 {
		t.Fatalf("mail must not send after identity failure: %+v", mailer.Messages())
	}
}

func TestApproveJoinRequestCompensatesWhenStampLabelsFails(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "joiner@example.com", Labels: []string{"customKeep"}},
	}
	identity := joinRequestTestIdentity(users)
	var calls []string
	var deletedOrgSlug, deletedMembershipID string
	identity.addOrganizationUserByIDAsAdminFunc = func(_ context.Context, orgSlug, userID string, roleSlugs []string, isOrgAdmin bool) (IdentityMembership, error) {
		calls = append(calls, "add")
		return IdentityMembership{ID: "mem-1", UserID: userID, RoleSlugs: roleSlugs}, nil
	}
	identity.updateUserLabelsFunc = func(_ context.Context, _ string, _ []string) (IdentityUser, error) {
		calls = append(calls, "labels")
		return IdentityUser{}, errors.New("stamp labels failed")
	}
	identity.deleteOrganizationMembershipAsAdminFunc = func(_ context.Context, orgSlug, membershipID string) error {
		calls = append(calls, "delete")
		deletedOrgSlug = orgSlug
		deletedMembershipID = membershipID
		return nil
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)

	saved, err := store.InsertJoinRequest(ctx, JoinRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, err = aff.ApproveJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin-1", OrgSlug: "acme"})
	if err == nil || err.Error() != "stamp labels failed" {
		t.Fatalf("err=%v", err)
	}
	if len(calls) != 3 || calls[0] != "add" || calls[1] != "labels" || calls[2] != "delete" {
		t.Fatalf("calls=%v", calls)
	}
	if deletedOrgSlug != "acme" || deletedMembershipID != "mem-1" {
		t.Fatalf("delete args org=%q membership=%q", deletedOrgSlug, deletedMembershipID)
	}
	loaded, err := store.LoadJoinRequestByID(ctx, saved.ID)
	if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if loaded.DecidedByUserID != "" || !loaded.DecidedAt.IsZero() || loaded.RejectReason != "" {
		t.Fatalf("decided fields not cleared: %+v", loaded)
	}
	if len(mailer.Messages()) != 0 {
		t.Fatalf("mail must not send after identity failure: %+v", mailer.Messages())
	}
}

func TestApproveOrganizationCreationRequestCompensatesWhenCreateOrgFails(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "founder@example.com"},
	}
	var savedID primitive.ObjectID
	var calls []string
	identity := &fakeIdentityStore{
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			return users[userID], nil
		},
		getOrganizationBySlugFunc: func(_ context.Context, _ string) (*IdentityOrg, error) {
			return nil, ErrIdentityNotFound
		},
		createOrganizationAsAdminFunc: func(_ context.Context, name string) (IdentityOrg, error) {
			loaded, err := store.LoadOrganizationCreationRequestByID(ctx, savedID)
			if err != nil || loaded == nil || loaded.Status != AffiliationStatusApproved {
				t.Fatalf("expected approved before create: loaded=%+v err=%v", loaded, err)
			}
			calls = append(calls, "create")
			return IdentityOrg{}, errors.New("create org failed")
		},
		addOrganizationUserByIDAsAdminFunc: func(_ context.Context, _, _ string, _ []string, _ bool) (IdentityMembership, error) {
			calls = append(calls, "add")
			return IdentityMembership{}, nil
		},
		deleteOrganizationAsAdminFunc: func(_ context.Context, _ string) error {
			calls = append(calls, "delete")
			return nil
		},
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)
	saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	savedID = saved.ID

	_, _, err = aff.ApproveOrganizationCreationRequest(ctx, saved.ID, IdentityUser{ID: "admin-1"})
	if err == nil || err.Error() != "create org failed" {
		t.Fatalf("err=%v", err)
	}
	if len(calls) != 1 || calls[0] != "create" {
		t.Fatalf("calls=%v", calls)
	}
	loaded, err := store.LoadOrganizationCreationRequestByID(ctx, saved.ID)
	if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if len(mailer.Messages()) != 0 {
		t.Fatalf("mail must not send: %+v", mailer.Messages())
	}
}

func TestApproveOrganizationCreationRequestCompensatesWhenAddMembershipFails(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	mailer := &recordingMailer{}
	users := map[string]IdentityUser{
		"user-1": {ID: "user-1", Email: "founder@example.com"},
	}
	var calls []string
	var deletedSlug string
	identity := &fakeIdentityStore{
		getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
			return users[userID], nil
		},
		getOrganizationBySlugFunc: func(_ context.Context, _ string) (*IdentityOrg, error) {
			return nil, ErrIdentityNotFound
		},
		createOrganizationAsAdminFunc: func(_ context.Context, name string) (IdentityOrg, error) {
			calls = append(calls, "create")
			return IdentityOrg{ID: "team-1", Slug: "new-org", Name: name}, nil
		},
		addOrganizationUserByIDAsAdminFunc: func(_ context.Context, orgSlug, userID string, _ []string, isOrgAdmin bool) (IdentityMembership, error) {
			calls = append(calls, "add")
			return IdentityMembership{}, errors.New("add membership failed")
		},
		deleteOrganizationAsAdminFunc: func(_ context.Context, orgSlug string) error {
			calls = append(calls, "delete")
			deletedSlug = orgSlug
			return nil
		},
		updateUserLabelsFunc: func(_ context.Context, _ string, _ []string) (IdentityUser, error) {
			calls = append(calls, "labels")
			return IdentityUser{}, nil
		},
	}
	aff := NewAffiliation(identity, store, mailer, fixedNow, nil)
	saved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		RequesterUserID: "user-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Org",
		ProposedSlug:    "new-org",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	_, _, err = aff.ApproveOrganizationCreationRequest(ctx, saved.ID, IdentityUser{ID: "admin-1"})
	if err == nil || err.Error() != "add membership failed" {
		t.Fatalf("err=%v", err)
	}
	if len(calls) != 3 || calls[0] != "create" || calls[1] != "add" || calls[2] != "delete" {
		t.Fatalf("calls=%v", calls)
	}
	if deletedSlug != "new-org" {
		t.Fatalf("deletedSlug=%q", deletedSlug)
	}
	loaded, err := store.LoadOrganizationCreationRequestByID(ctx, saved.ID)
	if err != nil || loaded == nil || loaded.Status != AffiliationStatusPending {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if loaded.DecidedByUserID != "" || !loaded.DecidedAt.IsZero() {
		t.Fatalf("decided fields not cleared: %+v", loaded)
	}
	if len(mailer.Messages()) != 0 {
		t.Fatalf("mail must not send: %+v", mailer.Messages())
	}
}
