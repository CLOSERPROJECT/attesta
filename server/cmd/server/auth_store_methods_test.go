package main

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestMemoryStoreAuthPrimitives(t *testing.T) {
	store := NewMemoryStore()

	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Foods"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if org.Slug != "acme-foods" {
		t.Fatalf("organization slug = %q, want acme-foods", org.Slug)
	}

	role, err := store.CreateRole(t.Context(), Role{OrgSlug: org.Slug, Name: "Org Admin"})
	if err != nil {
		t.Fatalf("CreateRole error: %v", err)
	}
	if role.Slug != "org-admin" {
		t.Fatalf("role slug = %q, want org-admin", role.Slug)
	}

	user, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "u-1",
		OrgSlug:   org.Slug,
		Email:     "Admin@Acme.io",
		RoleSlugs: []string{"org-admin"},
		Status:    "active",
	})
	if err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if user.Email != "admin@acme.io" {
		t.Fatalf("normalized email = %q, want admin@acme.io", user.Email)
	}

	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "u-2",
		OrgSlug:   org.Slug,
		Email:     "admin@acme.io",
		RoleSlugs: []string{"org-admin"},
	}); err == nil {
		t.Fatal("expected duplicate email error")
	}

	if err := store.SetUserPasswordHash(t.Context(), "u-1", "hash-1"); err != nil {
		t.Fatalf("SetUserPasswordHash error: %v", err)
	}
	if err := store.SetUserRoles(t.Context(), "u-1", []string{"org-admin"}); err != nil {
		t.Fatalf("SetUserRoles error: %v", err)
	}
	if err := store.SetUserRoles(t.Context(), "u-1", []string{"missing-role"}); err == nil {
		t.Fatal("expected SetUserRoles missing role error")
	}
	if err := store.SetUserLastLogin(t.Context(), "u-1", time.Now().UTC()); err != nil {
		t.Fatalf("SetUserLastLogin error: %v", err)
	}

	invite, err := store.CreateInvite(t.Context(), Invite{
		OrgID:     org.ID,
		Email:     "new.user@acme.io",
		UserID:    "u-3",
		RoleSlugs: []string{"org-admin"},
		TokenHash: "invite-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateInvite error: %v", err)
	}
	if invite.TokenHash == "invite-token" {
		t.Fatal("expected invite token to be hashed")
	}
	loadedInvite, err := store.LoadInviteByTokenHash(t.Context(), "invite-token")
	if err != nil {
		t.Fatalf("LoadInviteByTokenHash error: %v", err)
	}
	if loadedInvite.ID != invite.ID {
		t.Fatalf("loaded invite id = %s, want %s", loadedInvite.ID.Hex(), invite.ID.Hex())
	}
	if err := store.MarkInviteUsed(t.Context(), "invite-token", time.Now().UTC()); err != nil {
		t.Fatalf("MarkInviteUsed error: %v", err)
	}
	if err := store.MarkInviteUsed(t.Context(), "invite-token", time.Now().UTC()); err == nil {
		t.Fatal("expected second MarkInviteUsed call to fail")
	}

	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "s-1",
		UserID:      "u-1",
		UserMongoID: user.ID,
		CreatedAt:   time.Now().UTC(),
		LastLoginAt: time.Now().UTC(),
		ExpiresAt:   time.Now().UTC().Add(30 * 24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreateSession error: %v", err)
	}
	gotSession, err := store.LoadSessionByID(t.Context(), "s-1")
	if err != nil {
		t.Fatalf("LoadSessionByID error: %v", err)
	}
	if gotSession.ID != session.ID {
		t.Fatalf("session id = %s, want %s", gotSession.ID.Hex(), session.ID.Hex())
	}
	if err := store.DeleteSession(t.Context(), "s-1"); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}
	if _, err := store.LoadSessionByID(t.Context(), "s-1"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("LoadSessionByID after delete error = %v, want %v", err, mongo.ErrNoDocuments)
	}

	reset, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:     "admin@acme.io",
		UserID:    "u-1",
		TokenHash: "reset-token",
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("CreatePasswordReset error: %v", err)
	}
	if reset.TokenHash == "reset-token" {
		t.Fatal("expected reset token to be hashed")
	}
	loadedReset, err := store.LoadPasswordResetByTokenHash(t.Context(), "reset-token")
	if err != nil {
		t.Fatalf("LoadPasswordResetByTokenHash error: %v", err)
	}
	if loadedReset.ID != reset.ID {
		t.Fatalf("loaded reset id = %s, want %s", loadedReset.ID.Hex(), reset.ID.Hex())
	}
	if err := store.MarkPasswordResetUsed(t.Context(), "reset-token", time.Now().UTC()); err != nil {
		t.Fatalf("MarkPasswordResetUsed error: %v", err)
	}
	if err := store.MarkPasswordResetUsed(t.Context(), "reset-token", time.Now().UTC()); err == nil {
		t.Fatal("expected second MarkPasswordResetUsed call to fail")
	}
}

func TestMongoStoreCreateOrganizationCanonifiesSlug(t *testing.T) {
	insertedID := primitive.NewObjectID()
	collection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			org, ok := document.(Organization)
			if !ok {
				t.Fatalf("inserted document type = %T, want Organization", document)
			}
			if org.Slug != "north-plant" {
				t.Fatalf("organization slug = %q, want north-plant", org.Slug)
			}
			return &mongo.InsertOneResult{InsertedID: insertedID}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{collectionOrganizations: collection}}
	store := &MongoStore{dbPort: db}

	org, err := store.CreateOrganization(t.Context(), Organization{Name: "North Plant"})
	if err != nil {
		t.Fatalf("CreateOrganization error: %v", err)
	}
	if org.ID != insertedID {
		t.Fatalf("organization id = %s, want %s", org.ID.Hex(), insertedID.Hex())
	}
}

func TestMongoStoreAuthFilterShapes(t *testing.T) {
	collection := &fakeMongoCollection{}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionInvites:       collection,
		collectionPasswordReset: collection,
		collectionUsers:         collection,
	}}
	store := &MongoStore{dbPort: db}

	if err := store.SetUserRoles(t.Context(), "u-1", []string{"Org Admin", "org_admin"}); err != nil {
		t.Fatalf("SetUserRoles error: %v", err)
	}
	wantRolesUpdate := bson.M{"$set": bson.M{"roleSlugs": []string{"org-admin"}}}
	if !reflect.DeepEqual(collection.updateOneUpdates[0], wantRolesUpdate) {
		t.Fatalf("roles update = %#v, want %#v", collection.updateOneUpdates[0], wantRolesUpdate)
	}

	if _, err := store.LoadInviteByTokenHash(t.Context(), "token-x"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("LoadInviteByTokenHash error = %v, want %v", err, mongo.ErrNoDocuments)
	}
	wantInviteFilter := bson.M{"tokenHash": hashLookupToken("token-x")}
	if !reflect.DeepEqual(collection.findOneFilters[0], wantInviteFilter) {
		t.Fatalf("invite filter = %#v, want %#v", collection.findOneFilters[0], wantInviteFilter)
	}

	if _, err := store.LoadPasswordResetByTokenHash(t.Context(), "reset-y"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("LoadPasswordResetByTokenHash error = %v, want %v", err, mongo.ErrNoDocuments)
	}
	wantResetFilter := bson.M{"tokenHash": hashLookupToken("reset-y")}
	if !reflect.DeepEqual(collection.findOneFilters[1], wantResetFilter) {
		t.Fatalf("reset filter = %#v, want %#v", collection.findOneFilters[1], wantResetFilter)
	}
}
