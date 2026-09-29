package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestAttentionHasAttentionOpenInvitation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	identity := &fakeIdentityStore{
		listUserMembershipsFunc: func(_ context.Context, userID string) ([]IdentityMembership, error) {
			if userID != "invitee-1" {
				return nil, nil
			}
			return []IdentityMembership{{
				ID:        "membership-1",
				TeamID:    "team-acme",
				OrgSlug:   "acme",
				OrgName:   "Acme",
				UserID:    "invitee-1",
				Confirmed: false,
				InvitedAt: now,
			}}, nil
		},
	}
	aff := NewAffiliation(identity, NewMemoryStore(), nil, func() time.Time { return now }, nil)
	attention := NewAttention(aff)

	has, err := attention.HasAttention(ctx, IdentityUser{ID: "invitee-1", Email: "invitee@example.com"})
	if err != nil {
		t.Fatalf("HasAttention: %v", err)
	}
	if !has {
		t.Fatal("open Invitation must be an Attention item")
	}
}

func TestAttentionHasAttentionWaitingAloneDoesNotCount(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	identity := &fakeIdentityStore{
		listUserMembershipsFunc: func(_ context.Context, _ string) ([]IdentityMembership, error) {
			return nil, nil
		},
	}

	t.Run("join request", func(t *testing.T) {
		store := NewMemoryStore()
		if _, err := store.InsertJoinRequest(ctx, JoinRequest{
			ID:              primitive.NewObjectID(),
			RequesterUserID: "waiter-1",
			RequesterEmail:  "waiter@example.com",
			OrgSlug:         "acme",
			RoleSlugs:       []string{"viewer"},
			Status:          AffiliationStatusPending,
			CreatedAt:       now,
			UpdatedAt:       now,
		}); err != nil {
			t.Fatalf("InsertJoinRequest: %v", err)
		}
		has, err := NewAttention(NewAffiliation(identity, store, nil, func() time.Time { return now }, nil)).HasAttention(ctx, IdentityUser{ID: "waiter-1"})
		if err != nil {
			t.Fatalf("HasAttention: %v", err)
		}
		if has {
			t.Fatal("Join Waiting must not light Attention")
		}
	})

	t.Run("organization creation request", func(t *testing.T) {
		store := NewMemoryStore()
		if _, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
			ID:              primitive.NewObjectID(),
			RequesterUserID: "waiter-2",
			RequesterEmail:  "founder@example.com",
			ProposedName:    "New Co",
			ProposedSlug:    "new-co",
			Status:          AffiliationStatusPending,
			CreatedAt:       now,
			UpdatedAt:       now,
		}); err != nil {
			t.Fatalf("InsertOrganizationCreationRequest: %v", err)
		}
		has, err := NewAttention(NewAffiliation(identity, store, nil, func() time.Time { return now }, nil)).HasAttention(ctx, IdentityUser{ID: "waiter-2"})
		if err != nil {
			t.Fatalf("HasAttention: %v", err)
		}
		if has {
			t.Fatal("Organization creation Waiting must not light Attention")
		}
	})
}

func TestAttentionHasAttentionSkipsAffiliatedWithoutInviteLookup(t *testing.T) {
	ctx := context.Background()
	called := false
	identity := &fakeIdentityStore{
		listUserMembershipsFunc: func(_ context.Context, _ string) ([]IdentityMembership, error) {
			called = true
			return []IdentityMembership{{
				ID: "m1", OrgSlug: "acme", UserID: "member-1", Confirmed: false,
			}}, nil
		},
	}
	has, err := NewAttention(NewAffiliation(identity, NewMemoryStore(), nil, time.Now, nil)).HasAttention(ctx, IdentityUser{
		ID: "member-1", OrgSlug: "acme",
	})
	if err != nil {
		t.Fatalf("HasAttention: %v", err)
	}
	if has {
		t.Fatal("affiliated Member must not get Invitation Attention")
	}
	if called {
		t.Fatal("affiliated path must not list memberships for Invitation Attention")
	}
}

func TestAttentionHasAttentionPlatformAdminPendingOrgCreation(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	if _, err := store.InsertOrganizationCreationRequest(ctx, OrganizationCreationRequest{
		ID:              primitive.NewObjectID(),
		RequesterUserID: "founder-1",
		RequesterEmail:  "founder@example.com",
		ProposedName:    "New Co",
		ProposedSlug:    "new-co",
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertOrganizationCreationRequest: %v", err)
	}
	attention := NewAttention(NewAffiliation(&fakeIdentityStore{}, store, nil, func() time.Time { return now }, nil))

	pa := IdentityUser{Email: "admin@example.com", IsPlatformAdmin: true}
	has, err := attention.HasAttention(ctx, pa)
	if err != nil {
		t.Fatalf("HasAttention: %v", err)
	}
	if !has {
		t.Fatal("platform admin with pending Organization creation must have Attention")
	}
	items, err := attention.PendingOrganizationCreationRequests(ctx, pa)
	if err != nil {
		t.Fatalf("PendingOrganizationCreationRequests: %v", err)
	}
	if len(items) != 1 || items[0].ProposedSlug != "new-co" {
		t.Fatalf("PendingOrganizationCreationRequests = %+v", items)
	}

	nonPA := IdentityUser{ID: "user-1", Email: "user@example.com"}
	has, err = attention.HasAttention(ctx, nonPA)
	if err != nil {
		t.Fatalf("HasAttention non-PA: %v", err)
	}
	if has {
		t.Fatal("non-platform-admin must not get Organization-creation Attention")
	}
	items, err = attention.PendingOrganizationCreationRequests(ctx, nonPA)
	if err != nil {
		t.Fatalf("PendingOrganizationCreationRequests non-PA: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("PendingOrganizationCreationRequests for non-PA = %+v", items)
	}
}

func TestAttentionHasAttentionOrgAdminPendingJoin(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	if _, err := store.InsertJoinRequest(ctx, JoinRequest{
		ID:              primitive.NewObjectID(),
		RequesterUserID: "joiner-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}
	aff := NewAffiliation(&fakeIdentityStore{}, store, nil, func() time.Time { return now }, nil)
	attention := NewAttention(aff)

	has, err := attention.HasAttention(ctx, IdentityUser{ID: "admin-1", OrgSlug: "acme", IsOrgAdmin: true})
	if err != nil {
		t.Fatalf("HasAttention: %v", err)
	}
	if !has {
		t.Fatal("Org admin with pending Join request must have Attention")
	}

	items, err := attention.PendingJoinRequests(ctx, IdentityUser{ID: "admin-1", OrgSlug: "acme", IsOrgAdmin: true})
	if err != nil {
		t.Fatalf("PendingJoinRequests: %v", err)
	}
	if len(items) != 1 || items[0].RequesterEmail != "joiner@example.com" {
		t.Fatalf("PendingJoinRequests = %+v", items)
	}
}

func TestAttentionHasAttentionMemberNeverSeesJoinQueue(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	store := NewMemoryStore()
	if _, err := store.InsertJoinRequest(ctx, JoinRequest{
		ID:              primitive.NewObjectID(),
		RequesterUserID: "joiner-1",
		RequesterEmail:  "joiner@example.com",
		OrgSlug:         "acme",
		RoleSlugs:       []string{"viewer"},
		Status:          AffiliationStatusPending,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		t.Fatalf("InsertJoinRequest: %v", err)
	}
	attention := NewAttention(NewAffiliation(&fakeIdentityStore{}, store, nil, func() time.Time { return now }, nil))

	has, err := attention.HasAttention(ctx, IdentityUser{ID: "member-1", OrgSlug: "acme", IsOrgAdmin: false})
	if err != nil {
		t.Fatalf("HasAttention: %v", err)
	}
	if has {
		t.Fatal("Member without Org admin standing must not see Join-request Attention")
	}
	items, err := attention.PendingJoinRequests(ctx, IdentityUser{ID: "member-1", OrgSlug: "acme"})
	if err != nil {
		t.Fatalf("PendingJoinRequests: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("PendingJoinRequests for Member = %+v", items)
	}
}

func TestAttentionHasAttentionEmpty(t *testing.T) {
	ctx := context.Background()
	aff := NewAffiliation(&fakeIdentityStore{
		listUserMembershipsFunc: func(_ context.Context, _ string) ([]IdentityMembership, error) {
			return nil, nil
		},
	}, NewMemoryStore(), nil, time.Now, nil)

	has, err := NewAttention(aff).HasAttention(ctx, IdentityUser{ID: "user-1"})
	if err != nil {
		t.Fatalf("HasAttention: %v", err)
	}
	if has {
		t.Fatal("no Attention items expected")
	}
}

func TestAttentionHasAttentionPropagatesInviteListError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("memberships unavailable")
	aff := NewAffiliation(&fakeIdentityStore{
		listUserMembershipsFunc: func(_ context.Context, _ string) ([]IdentityMembership, error) {
			return nil, wantErr
		},
	}, NewMemoryStore(), nil, time.Now, nil)

	has, err := NewAttention(aff).HasAttention(ctx, IdentityUser{ID: "user-1"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("err=%v has=%v", err, has)
	}
	if has {
		t.Fatal("error path must not report Attention")
	}
}

func TestAttentionHasAttentionNilSafe(t *testing.T) {
	var attention *Attention
	has, err := attention.HasAttention(context.Background(), IdentityUser{ID: "user-1"})
	if err != nil || has {
		t.Fatalf("nil Attention: has=%v err=%v", has, err)
	}
	has, err = NewAttention(nil).HasAttention(context.Background(), IdentityUser{ID: "user-1"})
	if err != nil || has {
		t.Fatalf("nil affiliation: has=%v err=%v", has, err)
	}
	has, err = NewAttention(NewAffiliation(nil, NewMemoryStore(), nil, time.Now, nil)).HasAttention(context.Background(), IdentityUser{})
	if err != nil || has {
		t.Fatalf("empty userID: has=%v err=%v", has, err)
	}
}
