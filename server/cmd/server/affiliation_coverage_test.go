package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// affiliationStoreHook wraps MemoryStore so tests can force error / non-pending paths.
type affiliationStoreHook struct {
	*MemoryStore
	findPendingJoinErr      error
	findPendingJoinOverride *JoinRequest
	findPendingOrgErr       error
	findPendingOrgOverride  *OrganizationCreationRequest
	deleteJoinErr           error
	deleteOrgErr            error
	insertJoinErr           error
	insertOrgErr            error
	updateJoinErr           error
	updateOrgErr            error
}

func (h *affiliationStoreHook) FindPendingJoinRequestByUser(ctx context.Context, userID string) (*JoinRequest, error) {
	if h.findPendingJoinErr != nil {
		return nil, h.findPendingJoinErr
	}
	if h.findPendingJoinOverride != nil {
		cloned := *h.findPendingJoinOverride
		return &cloned, nil
	}
	return h.MemoryStore.FindPendingJoinRequestByUser(ctx, userID)
}

func (h *affiliationStoreHook) FindPendingOrganizationCreationRequestByUser(ctx context.Context, userID string) (*OrganizationCreationRequest, error) {
	if h.findPendingOrgErr != nil {
		return nil, h.findPendingOrgErr
	}
	if h.findPendingOrgOverride != nil {
		cloned := *h.findPendingOrgOverride
		return &cloned, nil
	}
	return h.MemoryStore.FindPendingOrganizationCreationRequestByUser(ctx, userID)
}

func (h *affiliationStoreHook) DeleteJoinRequest(ctx context.Context, id primitive.ObjectID) error {
	if h.deleteJoinErr != nil {
		return h.deleteJoinErr
	}
	return h.MemoryStore.DeleteJoinRequest(ctx, id)
}

func (h *affiliationStoreHook) DeleteOrganizationCreationRequest(ctx context.Context, id primitive.ObjectID) error {
	if h.deleteOrgErr != nil {
		return h.deleteOrgErr
	}
	return h.MemoryStore.DeleteOrganizationCreationRequest(ctx, id)
}

func (h *affiliationStoreHook) InsertJoinRequest(ctx context.Context, req JoinRequest) (JoinRequest, error) {
	if h.insertJoinErr != nil {
		return JoinRequest{}, h.insertJoinErr
	}
	return h.MemoryStore.InsertJoinRequest(ctx, req)
}

func (h *affiliationStoreHook) InsertOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	if h.insertOrgErr != nil {
		return OrganizationCreationRequest{}, h.insertOrgErr
	}
	return h.MemoryStore.InsertOrganizationCreationRequest(ctx, req)
}

func (h *affiliationStoreHook) UpdateJoinRequest(ctx context.Context, req JoinRequest) (JoinRequest, error) {
	if h.updateJoinErr != nil {
		return JoinRequest{}, h.updateJoinErr
	}
	return h.MemoryStore.UpdateJoinRequest(ctx, req)
}

func (h *affiliationStoreHook) UpdateOrganizationCreationRequest(ctx context.Context, req OrganizationCreationRequest) (OrganizationCreationRequest, error) {
	if h.updateOrgErr != nil {
		return OrganizationCreationRequest{}, h.updateOrgErr
	}
	return h.MemoryStore.UpdateOrganizationCreationRequest(ctx, req)
}

func TestNewAffiliationDefaults(t *testing.T) {
	aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), nil, nil, []string{"  ", " admin@example.com ", ""})
	if aff.mailer == nil {
		t.Fatal("expected noop mailer")
	}
	if aff.now == nil {
		t.Fatal("expected default now")
	}
	if len(aff.platformAdminNotifyEmails) != 1 || aff.platformAdminNotifyEmails[0] != "admin@example.com" {
		t.Fatalf("emails=%v", aff.platformAdminNotifyEmails)
	}
	if got := normalizePlatformAdminNotifyEmails(nil); got != nil {
		t.Fatalf("nil emails=%v", got)
	}
	if got := normalizePlatformAdminNotifyEmails([]string{" ", ""}); got != nil {
		t.Fatalf("blank emails=%v", got)
	}
}

func TestWithdrawPendingNotPendingAndDeleteErrors(t *testing.T) {
	ctx := context.Background()
	user := IdentityUser{ID: "user-1", Email: "u@example.com"}

	t.Run("join empty user", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingJoinRequest(ctx, IdentityUser{}); !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("join find error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinErr: errors.New("boom")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingJoinRequest(ctx, user); err == nil || err.Error() != "boom" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("join not pending", func(t *testing.T) {
		req := &JoinRequest{ID: primitive.NewObjectID(), RequesterUserID: "user-1", Status: AffiliationStatusApproved}
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinOverride: req}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingJoinRequest(ctx, user); !errors.Is(err, ErrAffiliationNotPending) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("join delete no documents", func(t *testing.T) {
		req := &JoinRequest{ID: primitive.NewObjectID(), RequesterUserID: "user-1", Status: AffiliationStatusPending}
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinOverride: req, deleteJoinErr: mongo.ErrNoDocuments}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingJoinRequest(ctx, user); !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("join delete other error", func(t *testing.T) {
		req := &JoinRequest{ID: primitive.NewObjectID(), RequesterUserID: "user-1", Status: AffiliationStatusPending}
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinOverride: req, deleteJoinErr: errors.New("delete failed")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingJoinRequest(ctx, user); err == nil || err.Error() != "delete failed" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org empty user", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingOrganizationCreationRequest(ctx, IdentityUser{}); !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org find error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingOrgErr: errors.New("org boom")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingOrganizationCreationRequest(ctx, user); err == nil || err.Error() != "org boom" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org nil pending", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingOrganizationCreationRequest(ctx, user); !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org not pending", func(t *testing.T) {
		req := &OrganizationCreationRequest{ID: primitive.NewObjectID(), RequesterUserID: "user-1", Status: AffiliationStatusRejected}
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingOrgOverride: req}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingOrganizationCreationRequest(ctx, user); !errors.Is(err, ErrAffiliationNotPending) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org delete no documents", func(t *testing.T) {
		req := &OrganizationCreationRequest{ID: primitive.NewObjectID(), RequesterUserID: "user-1", Status: AffiliationStatusPending}
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingOrgOverride: req, deleteOrgErr: mongo.ErrNoDocuments}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingOrganizationCreationRequest(ctx, user); !errors.Is(err, ErrAffiliationNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org delete other error", func(t *testing.T) {
		req := &OrganizationCreationRequest{ID: primitive.NewObjectID(), RequesterUserID: "user-1", Status: AffiliationStatusPending}
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingOrgOverride: req, deleteOrgErr: errors.New("org delete failed")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		if err := aff.WithdrawPendingOrganizationCreationRequest(ctx, user); err == nil || err.Error() != "org delete failed" {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestHasPendingAffiliationIntentErrors(t *testing.T) {
	ctx := context.Background()

	t.Run("join error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinErr: errors.New("join err")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		ok, err := aff.HasPendingAffiliationIntent(ctx, "user-1")
		if ok || err == nil || err.Error() != "join err" {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})

	t.Run("org error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingOrgErr: errors.New("org err")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		ok, err := aff.HasPendingAffiliationIntent(ctx, "user-1")
		if ok || err == nil || err.Error() != "org err" {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
	})
}

func TestLeaveOrganizationErrorBranches(t *testing.T) {
	ctx := context.Background()

	t.Run("get current user error", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{}, errors.New("session gone")
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "u", OrgSlug: "acme"})
		if err == nil || err.Error() != "session gone" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("current unaffiliated", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{ID: "u"}, nil
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "u", OrgSlug: "acme"})
		if !errors.Is(err, ErrAffiliationNotAffiliated) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("list org users error", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{ID: "admin", OrgSlug: "acme", IsOrgAdmin: true, MembershipID: "m1"}, nil
			},
			listOrganizationUsersFunc: func(_ context.Context, _ string) ([]IdentityUser, error) {
				return nil, errors.New("list failed")
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "admin", OrgSlug: "acme"})
		if err == nil || err.Error() != "list failed" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("missing membership id", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{ID: "u", OrgSlug: "acme"}, nil
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "u", OrgSlug: "acme"})
		if !errors.Is(err, ErrIdentityNotFound) {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("delete membership error", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getCurrentUserFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{ID: "u", OrgSlug: "acme", MembershipID: "m1"}, nil
			},
			deleteOrganizationMembershipFunc: func(_ context.Context, _, _, _ string) error {
				return errors.New("delete membership failed")
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		err := aff.LeaveOrganization(ctx, "session", IdentityUser{ID: "u", OrgSlug: "acme"})
		if err == nil || err.Error() != "delete membership failed" {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestStripManagedIdentityLabelsEdges(t *testing.T) {
	ctx := context.Background()

	t.Run("empty user id", func(t *testing.T) {
		aff := NewAffiliation(&fakeIdentityStore{}, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		if err := aff.stripManagedIdentityLabels(ctx, "  "); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("user not found", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{}, ErrIdentityNotFound
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		if err := aff.stripManagedIdentityLabels(ctx, "missing"); err != nil {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("get user error", func(t *testing.T) {
		identity := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{}, errors.New("lookup failed")
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		if err := aff.stripManagedIdentityLabels(ctx, "u"); err == nil || err.Error() != "lookup failed" {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestNotifyJoinSubmittedEdges(t *testing.T) {
	ctx := context.Background()
	req := JoinRequest{RequesterEmail: "joiner@example.com", RoleSlugs: []string{"viewer"}}
	org := IdentityOrg{Slug: "acme", Name: "Acme"}

	t.Run("list users error", func(t *testing.T) {
		identity := &fakeIdentityStore{
			listOrganizationUsersFunc: func(_ context.Context, _ string) ([]IdentityUser, error) {
				return nil, errors.New("list failed")
			},
		}
		aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
		aff.notifyJoinSubmitted(ctx, req, org)
	})

	t.Run("no admins empty emails and duplicates", func(t *testing.T) {
		identity := &fakeIdentityStore{
			listOrganizationUsersFunc: func(_ context.Context, _ string) ([]IdentityUser, error) {
				return []IdentityUser{
					{IsOrgAdmin: false, Email: "member@example.com"},
					{IsOrgAdmin: true, Email: "  "},
					{IsOrgAdmin: true, Email: "Owner@example.com"},
					{IsOrgAdmin: true, Email: "owner@example.com"},
				}, nil
			},
		}
		mailer := &recordingMailer{}
		aff := NewAffiliation(identity, NewMemoryStore(), mailer, fixedNow, nil)
		aff.notifyJoinSubmitted(ctx, req, org)
		msgs := mailer.Messages()
		if len(msgs) != 1 || len(msgs[0].To) != 1 || msgs[0].To[0] != "Owner@example.com" {
			t.Fatalf("messages=%+v", msgs)
		}
	})

	t.Run("no recipients", func(t *testing.T) {
		identity := &fakeIdentityStore{
			listOrganizationUsersFunc: func(_ context.Context, _ string) ([]IdentityUser, error) {
				return []IdentityUser{{IsOrgAdmin: true, Email: ""}}, nil
			},
		}
		mailer := &recordingMailer{}
		aff := NewAffiliation(identity, NewMemoryStore(), mailer, fixedNow, nil)
		aff.notifyJoinSubmitted(ctx, req, org)
		if len(mailer.Messages()) != 0 {
			t.Fatalf("expected no mail, got %+v", mailer.Messages())
		}
	})
}

func TestAffiliationActionURLAndNotifyNilMailer(t *testing.T) {
	aff := &Affiliation{}
	if got := aff.actionURL("/my"); got == "" {
		t.Fatal("expected absolute-ish URL from nil mailer path")
	}
	aff.notify(context.Background(), MailMessage{Kind: "x", To: []string{"a@b.c"}, Subject: "s", Body: "b"})
}

func TestMapPendingAffiliationLoadDefaultError(t *testing.T) {
	err := mapPendingAffiliationLoad(errors.New("weird"), false, AffiliationStatusPending)
	if err == nil || err.Error() != "weird" {
		t.Fatalf("err=%v", err)
	}
}

func TestLoadOrganizationBySlugAndEnsureSlugAvailableErrors(t *testing.T) {
	ctx := context.Background()
	identity := &fakeIdentityStore{
		getOrganizationBySlugFunc: func(_ context.Context, slug string) (*IdentityOrg, error) {
			if slug == "boom" {
				return nil, errors.New("identity down")
			}
			if slug == "nil-org" {
				return nil, nil
			}
			return &IdentityOrg{Slug: slug, Name: slug}, nil
		},
	}
	aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)

	if _, err := aff.loadOrganizationBySlug(ctx, "boom"); err == nil || err.Error() != "identity down" {
		t.Fatalf("load err=%v", err)
	}
	if _, err := aff.loadOrganizationBySlug(ctx, "nil-org"); !errors.Is(err, ErrAffiliationNotFound) {
		t.Fatalf("nil org err=%v", err)
	}
	if err := aff.ensureOrganizationSlugAvailable(ctx, "boom"); err == nil || err.Error() != "identity down" {
		t.Fatalf("ensure err=%v", err)
	}
	if err := aff.ensureOrganizationSlugAvailable(ctx, "nil-org"); err != nil {
		t.Fatalf("nil existing should be available: %v", err)
	}
}

func TestEnsureDeciderOrgMatchesEmpty(t *testing.T) {
	if err := ensureDeciderOrgMatches(IdentityUser{}, "acme"); err != nil {
		t.Fatalf("err=%v", err)
	}
	if err := ensureDeciderOrgMatches(IdentityUser{OrgSlug: "acme"}, "other"); !errors.Is(err, ErrAffiliationNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestMemoryStoreAffiliationEdgeCases(t *testing.T) {
	ctx := context.Background()

	t.Run("ensure maps on zero store", func(t *testing.T) {
		store := &MemoryStore{}
		if _, err := store.InsertJoinRequest(ctx, JoinRequest{RequesterUserID: "u", OrgSlug: "acme"}); err != nil {
			t.Fatalf("InsertJoinRequest: %v", err)
		}
		store2 := &MemoryStore{}
		if _, err := store2.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			RequesterUserID: "u",
			ProposedName:    "N",
			ProposedSlug:    "n",
		}); err != nil {
			t.Fatalf("InsertOrganizationCreationRequest: %v", err)
		}
	})

	t.Run("update delete zero and missing ids", func(t *testing.T) {
		store := NewMemoryStore()
		if _, err := store.UpdateJoinRequest(ctx, JoinRequest{}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("zero update join err=%v", err)
		}
		if _, err := store.UpdateJoinRequest(ctx, JoinRequest{ID: primitive.NewObjectID()}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("missing update join err=%v", err)
		}
		if err := store.DeleteJoinRequest(ctx, primitive.NilObjectID); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("zero delete join err=%v", err)
		}
		if err := store.DeleteJoinRequest(ctx, primitive.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("missing delete join err=%v", err)
		}
		if _, err := store.UpdateOrganizationCreationRequest(ctx, OrganizationCreationRequest{}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("zero update org err=%v", err)
		}
		if _, err := store.UpdateOrganizationCreationRequest(ctx, OrganizationCreationRequest{ID: primitive.NewObjectID()}); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("missing update org err=%v", err)
		}
		if err := store.DeleteOrganizationCreationRequest(ctx, primitive.NilObjectID); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("zero delete org err=%v", err)
		}
		if err := store.DeleteOrganizationCreationRequest(ctx, primitive.NewObjectID()); !errors.Is(err, mongo.ErrNoDocuments) {
			t.Fatalf("missing delete org err=%v", err)
		}
	})

	t.Run("find pending empty user id", func(t *testing.T) {
		store := NewMemoryStore()
		join, err := store.FindPendingJoinRequestByUser(ctx, "  ")
		if err != nil || join != nil {
			t.Fatalf("join=%v err=%v", join, err)
		}
		org, err := store.FindPendingOrganizationCreationRequestByUser(ctx, "")
		if err != nil || org != nil {
			t.Fatalf("org=%v err=%v", org, err)
		}
	})

	t.Run("list pending sort ties", func(t *testing.T) {
		store := NewMemoryStore()
		now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
		idLow := primitive.NewObjectID()
		idHigh := primitive.NewObjectID()
		if idHigh.Hex() < idLow.Hex() {
			idLow, idHigh = idHigh, idLow
		}
		for _, req := range []JoinRequest{
			{ID: idLow, RequesterUserID: "a", OrgSlug: "acme", Status: AffiliationStatusPending, CreatedAt: now, UpdatedAt: now},
			{ID: idHigh, RequesterUserID: "b", OrgSlug: "acme", Status: AffiliationStatusPending, CreatedAt: now, UpdatedAt: now},
			{ID: primitive.NewObjectID(), RequesterUserID: "c", OrgSlug: "other", Status: AffiliationStatusPending, CreatedAt: now.Add(time.Hour), UpdatedAt: now},
			{ID: primitive.NewObjectID(), RequesterUserID: "d", OrgSlug: "acme", Status: AffiliationStatusApproved, CreatedAt: now, UpdatedAt: now},
		} {
			store.joinRequests[req.ID] = req
		}
		items, err := store.ListPendingJoinRequestsByOrg(ctx, "acme")
		if err != nil || len(items) != 2 {
			t.Fatalf("items=%v err=%v", items, err)
		}
		if items[0].ID != idHigh || items[1].ID != idLow {
			t.Fatalf("tie-break order=%s,%s want %s,%s", items[0].ID.Hex(), items[1].ID.Hex(), idHigh.Hex(), idLow.Hex())
		}

		orgLow := primitive.NewObjectID()
		orgHigh := primitive.NewObjectID()
		if orgHigh.Hex() < orgLow.Hex() {
			orgLow, orgHigh = orgHigh, orgLow
		}
		for _, req := range []OrganizationCreationRequest{
			{ID: orgLow, RequesterUserID: "a", ProposedSlug: "a", Status: AffiliationStatusPending, CreatedAt: now, UpdatedAt: now},
			{ID: orgHigh, RequesterUserID: "b", ProposedSlug: "b", Status: AffiliationStatusPending, CreatedAt: now, UpdatedAt: now},
			{ID: primitive.NewObjectID(), RequesterUserID: "c", ProposedSlug: "c", Status: AffiliationStatusRejected, CreatedAt: now, UpdatedAt: now},
		} {
			store.organizationCreationRequests[req.ID] = req
		}
		orgItems, err := store.ListPendingOrganizationCreationRequests(ctx)
		if err != nil || len(orgItems) != 2 {
			t.Fatalf("orgItems=%v err=%v", orgItems, err)
		}
		if orgItems[0].ID != orgHigh || orgItems[1].ID != orgLow {
			t.Fatalf("org tie-break order=%s,%s", orgItems[0].ID.Hex(), orgItems[1].ID.Hex())
		}
	})

	t.Run("update sets updatedAt when zero", func(t *testing.T) {
		store := NewMemoryStore()
		saved, err := store.InsertJoinRequest(ctx, JoinRequest{RequesterUserID: "u", OrgSlug: "acme"})
		if err != nil {
			t.Fatalf("insert: %v", err)
		}
		saved.UpdatedAt = time.Time{}
		saved.Status = AffiliationStatusApproved
		updated, err := store.UpdateJoinRequest(ctx, saved)
		if err != nil || updated.UpdatedAt.IsZero() {
			t.Fatalf("updated=%+v err=%v", updated, err)
		}
		orgSaved, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			RequesterUserID: "u",
			ProposedName:    "N",
			ProposedSlug:    "n",
		})
		if err != nil {
			t.Fatalf("insert org: %v", err)
		}
		orgSaved.UpdatedAt = time.Time{}
		orgSaved.Status = AffiliationStatusRejected
		orgUpdated, err := store.UpdateOrganizationCreationRequest(ctx, orgSaved)
		if err != nil || orgUpdated.UpdatedAt.IsZero() {
			t.Fatalf("orgUpdated=%+v err=%v", orgUpdated, err)
		}
	})
}

func TestStampManagedMembershipLabelsGetUserError(t *testing.T) {
	ctx := context.Background()
	identity := &fakeIdentityStore{
		getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
			return IdentityUser{}, errors.New("labels lookup failed")
		},
	}
	aff := NewAffiliation(identity, NewMemoryStore(), &recordingMailer{}, fixedNow, nil)
	if err := aff.stampManagedMembershipLabels(ctx, "user-1", []string{"viewer"}, false); err == nil || err.Error() != "labels lookup failed" {
		t.Fatalf("err=%v", err)
	}
}

func TestSubmitJoinAndOrgCreationStoreErrors(t *testing.T) {
	ctx := context.Background()
	identity := joinRequestTestIdentity(nil)

	t.Run("join has pending error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinErr: errors.New("pending check failed")}
		aff := NewAffiliation(identity, hook, &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", []string{"viewer"})
		if err == nil || err.Error() != "pending check failed" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("join insert error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), insertJoinErr: errors.New("insert join failed")}
		aff := NewAffiliation(identity, hook, &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "acme", []string{"viewer"})
		if err == nil || err.Error() != "insert join failed" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org create pending error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), findPendingJoinErr: errors.New("pending org check")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "New Org")
		if err == nil || err.Error() != "pending org check" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("org create insert error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: NewMemoryStore(), insertOrgErr: errors.New("insert org failed")}
		aff := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		_, err := aff.SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "u", Email: "u@example.com"}, "New Org")
		if err == nil || err.Error() != "insert org failed" {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestApproveRejectStoreAndIdentityErrors(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	identity := joinRequestTestIdentity(map[string]IdentityUser{
		"joiner": {ID: "joiner", Email: "j@example.com"},
	})
	aff := NewAffiliation(identity, store, &recordingMailer{}, fixedNow, nil)
	saved, err := aff.SubmitJoinRequest(ctx, IdentityUser{ID: "joiner", Email: "j@example.com"}, "acme", []string{"viewer"})
	if err != nil {
		t.Fatalf("seed join: %v", err)
	}

	t.Run("approve get requester error", func(t *testing.T) {
		id := &fakeIdentityStore{
			getOrganizationBySlugFunc: identity.getOrganizationBySlugFunc,
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{}, errors.New("requester missing")
			},
		}
		a := NewAffiliation(id, store, &recordingMailer{}, fixedNow, nil)
		_, err := a.ApproveJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin", OrgSlug: "acme"})
		if err == nil || err.Error() != "requester missing" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("approve load org error", func(t *testing.T) {
		id := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, userID string) (IdentityUser, error) {
				return IdentityUser{ID: userID, Email: "j@example.com"}, nil
			},
			getOrganizationBySlugFunc: func(_ context.Context, _ string) (*IdentityOrg, error) {
				return nil, errors.New("org load failed")
			},
		}
		a := NewAffiliation(id, store, &recordingMailer{}, fixedNow, nil)
		_, err := a.ApproveJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin", OrgSlug: "acme"})
		if err == nil || err.Error() != "org load failed" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("approve update error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: store, updateJoinErr: errors.New("update join failed")}
		a := NewAffiliation(identity, hook, &recordingMailer{}, fixedNow, nil)
		_, err := a.ApproveJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin", OrgSlug: "acme"})
		if err == nil || err.Error() != "update join failed" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("reject join update error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: store, updateJoinErr: errors.New("reject update failed")}
		a := NewAffiliation(identity, hook, &recordingMailer{}, fixedNow, nil)
		_, err := a.RejectJoinRequest(ctx, saved.ID, IdentityUser{ID: "admin", OrgSlug: "acme"}, "nope")
		if err == nil || err.Error() != "reject update failed" {
			t.Fatalf("err=%v", err)
		}
	})

	orgSaved, err := NewAffiliation(&fakeIdentityStore{}, store, &recordingMailer{}, fixedNow, nil).
		SubmitOrganizationCreationRequest(ctx, IdentityUser{ID: "founder", Email: "f@example.com"}, "Brand New")
	if err != nil {
		t.Fatalf("seed org: %v", err)
	}

	t.Run("approve org get requester error", func(t *testing.T) {
		id := &fakeIdentityStore{
			getUserByIDFunc: func(_ context.Context, _ string) (IdentityUser, error) {
				return IdentityUser{}, errors.New("founder missing")
			},
		}
		a := NewAffiliation(id, store, &recordingMailer{}, fixedNow, nil)
		_, _, err := a.ApproveOrganizationCreationRequest(ctx, orgSaved.ID, IdentityUser{ID: "admin"})
		if err == nil || err.Error() != "founder missing" {
			t.Fatalf("err=%v", err)
		}
	})

	t.Run("reject org update error", func(t *testing.T) {
		hook := &affiliationStoreHook{MemoryStore: store, updateOrgErr: errors.New("reject org update failed")}
		a := NewAffiliation(&fakeIdentityStore{}, hook, &recordingMailer{}, fixedNow, nil)
		_, err := a.RejectOrganizationCreationRequest(ctx, orgSaved.ID, IdentityUser{ID: "admin"}, "no")
		if err == nil || err.Error() != "reject org update failed" {
			t.Fatalf("err=%v", err)
		}
	})
}
