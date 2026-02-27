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

func TestMemoryStoreOrgAdminManagementMethods(t *testing.T) {
	store := NewMemoryStore()
	orgA, err := store.CreateOrganization(t.Context(), Organization{Name: "Org A"})
	if err != nil {
		t.Fatalf("CreateOrganization orgA error: %v", err)
	}
	orgB, err := store.CreateOrganization(t.Context(), Organization{Name: "Org B"})
	if err != nil {
		t.Fatalf("CreateOrganization orgB error: %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{OrgSlug: orgA.Slug, Name: "Reviewer"}); err != nil {
		t.Fatalf("CreateRole orgA error: %v", err)
	}
	if _, err := store.CreateRole(t.Context(), Role{OrgSlug: orgB.Slug, Name: "Reviewer"}); err != nil {
		t.Fatalf("CreateRole orgB error: %v", err)
	}

	orgAID := orgA.ID
	orgBID := orgB.ID
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "u-b",
		OrgID:        &orgAID,
		OrgSlug:      orgA.Slug,
		Email:        "b@orga.example",
		PasswordHash: "hash-b",
		RoleSlugs:    []string{"reviewer"},
		Status:       "active",
	}); err != nil {
		t.Fatalf("CreateUser u-b error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:       "u-a",
		OrgID:        &orgAID,
		OrgSlug:      orgA.Slug,
		Email:        "a@orga.example",
		PasswordHash: "hash-a",
		RoleSlugs:    []string{"reviewer"},
		Status:       "active",
	}); err != nil {
		t.Fatalf("CreateUser u-a error: %v", err)
	}
	if _, err := store.CreateUser(t.Context(), AccountUser{
		UserID:    "u-z",
		OrgID:     &orgBID,
		OrgSlug:   orgB.Slug,
		Email:     "z@orgb.example",
		RoleSlugs: []string{"reviewer"},
		Status:    "active",
	}); err != nil {
		t.Fatalf("CreateUser u-z error: %v", err)
	}

	users, err := store.ListUsersByOrgID(t.Context(), orgA.ID)
	if err != nil {
		t.Fatalf("ListUsersByOrgID error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("user count = %d, want %d", len(users), 2)
	}
	if users[0].Email != "a@orga.example" || users[1].Email != "b@orga.example" {
		t.Fatalf("users not sorted by email: %#v", users)
	}

	now := time.Now().UTC()
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:           orgA.ID,
		Email:           "one@orga.example",
		UserID:          "u-one",
		RoleSlugs:       []string{"reviewer"},
		TokenHash:       "token-one",
		ExpiresAt:       now.Add(24 * time.Hour),
		CreatedAt:       now.Add(-1 * time.Hour),
		CreatedByUserID: "admin-1",
	}); err != nil {
		t.Fatalf("CreateInvite one error: %v", err)
	}
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:           orgA.ID,
		Email:           "two@orga.example",
		UserID:          "u-two",
		RoleSlugs:       []string{"reviewer"},
		TokenHash:       "token-two",
		ExpiresAt:       now.Add(24 * time.Hour),
		CreatedAt:       now,
		CreatedByUserID: "admin-1",
	}); err != nil {
		t.Fatalf("CreateInvite two error: %v", err)
	}
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:           orgA.ID,
		Email:           "three@orga.example",
		UserID:          "u-three",
		RoleSlugs:       []string{"reviewer"},
		TokenHash:       "token-three",
		ExpiresAt:       now.Add(24 * time.Hour),
		CreatedAt:       now.Add(1 * time.Hour),
		CreatedByUserID: "admin-2",
	}); err != nil {
		t.Fatalf("CreateInvite three error: %v", err)
	}
	if _, err := store.CreateInvite(t.Context(), Invite{
		OrgID:           orgB.ID,
		Email:           "four@orgb.example",
		UserID:          "u-four",
		RoleSlugs:       []string{"reviewer"},
		TokenHash:       "token-four",
		ExpiresAt:       now.Add(24 * time.Hour),
		CreatedAt:       now.Add(2 * time.Hour),
		CreatedByUserID: "admin-1",
	}); err != nil {
		t.Fatalf("CreateInvite four error: %v", err)
	}

	invites, err := store.ListInvitesByCreator(t.Context(), "admin-1", orgA.ID)
	if err != nil {
		t.Fatalf("ListInvitesByCreator error: %v", err)
	}
	if len(invites) != 2 {
		t.Fatalf("invite count = %d, want %d", len(invites), 2)
	}
	if invites[0].Email != "two@orga.example" || invites[1].Email != "one@orga.example" {
		t.Fatalf("invites not sorted by createdAt desc: %#v", invites)
	}

	if err := store.DisableUser(t.Context(), "u-a"); err != nil {
		t.Fatalf("DisableUser error: %v", err)
	}
	disabled, err := store.GetUserByUserID(t.Context(), "u-a")
	if err != nil {
		t.Fatalf("GetUserByUserID disabled error: %v", err)
	}
	if disabled.Status != "deleted" {
		t.Fatalf("disabled user status = %q, want deleted", disabled.Status)
	}
	if disabled.PasswordHash != "" {
		t.Fatalf("disabled user password hash = %q, want empty", disabled.PasswordHash)
	}
	if len(disabled.RoleSlugs) != 0 {
		t.Fatalf("disabled user roles = %#v, want empty", disabled.RoleSlugs)
	}
	if err := store.DisableUser(t.Context(), "missing"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("DisableUser missing err = %v, want %v", err, mongo.ErrNoDocuments)
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

func TestMongoStoreOrgAdminManagementFilterShapes(t *testing.T) {
	userCollection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{AccountUser{UserID: "u-1"}}}, nil
		},
	}
	inviteCollection := &fakeMongoCollection{
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{Invite{Email: "invitee@example.com"}}}, nil
		},
	}
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionUsers:   userCollection,
		collectionInvites: inviteCollection,
	}}
	store := &MongoStore{dbPort: db}

	orgID := primitive.NewObjectID()
	users, err := store.ListUsersByOrgID(t.Context(), orgID)
	if err != nil {
		t.Fatalf("ListUsersByOrgID error: %v", err)
	}
	if len(users) != 1 || users[0].UserID != "u-1" {
		t.Fatalf("ListUsersByOrgID result = %#v", users)
	}

	invites, err := store.ListInvitesByCreator(t.Context(), " admin-1 ", orgID)
	if err != nil {
		t.Fatalf("ListInvitesByCreator error: %v", err)
	}
	if len(invites) != 1 || invites[0].Email != "invitee@example.com" {
		t.Fatalf("ListInvitesByCreator result = %#v", invites)
	}

	if err := store.DisableUser(t.Context(), " u-1 "); err != nil {
		t.Fatalf("DisableUser error: %v", err)
	}

	wantUsersFilter := bson.M{"orgId": orgID}
	if !reflect.DeepEqual(userCollection.findFilters[0], wantUsersFilter) {
		t.Fatalf("user find filter = %#v, want %#v", userCollection.findFilters[0], wantUsersFilter)
	}
	wantUsersSort := bson.D{{Key: "email", Value: 1}}
	if !reflect.DeepEqual(userCollection.findOptionsCalls[0][0].Sort, wantUsersSort) {
		t.Fatalf("user sort = %#v, want %#v", userCollection.findOptionsCalls[0][0].Sort, wantUsersSort)
	}

	wantInvitesFilter := bson.M{"createdByUserId": "admin-1", "orgId": orgID}
	if !reflect.DeepEqual(inviteCollection.findFilters[0], wantInvitesFilter) {
		t.Fatalf("invite find filter = %#v, want %#v", inviteCollection.findFilters[0], wantInvitesFilter)
	}
	wantInvitesSort := bson.D{{Key: "createdAt", Value: -1}}
	if !reflect.DeepEqual(inviteCollection.findOptionsCalls[0][0].Sort, wantInvitesSort) {
		t.Fatalf("invite sort = %#v, want %#v", inviteCollection.findOptionsCalls[0][0].Sort, wantInvitesSort)
	}

	wantDisableFilter := bson.M{"userId": "u-1"}
	if !reflect.DeepEqual(userCollection.updateOneFilters[0], wantDisableFilter) {
		t.Fatalf("disable filter = %#v, want %#v", userCollection.updateOneFilters[0], wantDisableFilter)
	}
	wantDisableUpdate := bson.M{"$set": bson.M{
		"status":       "deleted",
		"passwordHash": "",
		"roleSlugs":    []string{},
	}}
	if !reflect.DeepEqual(userCollection.updateOneUpdates[0], wantDisableUpdate) {
		t.Fatalf("disable update = %#v, want %#v", userCollection.updateOneUpdates[0], wantDisableUpdate)
	}
}

func TestMongoStoreAuthMethodsCovered(t *testing.T) {
	now := time.Date(2026, 2, 26, 18, 0, 0, 0, time.UTC)
	orgID := primitive.NewObjectID()
	roleID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	inviteID := primitive.NewObjectID()
	sessionID := primitive.NewObjectID()
	resetID := primitive.NewObjectID()

	orgCollection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: orgID}, nil
		},
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Organization)) = Organization{ID: orgID, Name: "Acme Org", Slug: "acme-org"}
				return nil
			}}
		},
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{
				Organization{ID: orgID, Name: "Acme Org", Slug: "acme-org"},
			}}, nil
		},
	}
	roleCollection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: roleID}, nil
		},
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Role)) = Role{ID: roleID, OrgSlug: "acme-org", Slug: "org-admin", Name: "Org Admin"}
				return nil
			}}
		},
		findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
			return &fakeAnyCursor{items: []interface{}{
				Role{ID: roleID, OrgSlug: "acme-org", Slug: "org-admin", Name: "Org Admin"},
			}}, nil
		},
	}
	userCollection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: userID}, nil
		},
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*AccountUser)) = AccountUser{ID: userID, UserID: "u-1", Email: "admin@acme.org"}
				return nil
			}}
		},
	}
	inviteCollection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: inviteID}, nil
		},
	}
	sessionCollection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: sessionID}, nil
		},
		findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
			return fakeSingleResult{decodeFn: func(v interface{}) error {
				*(v.(*Session)) = Session{ID: sessionID, SessionID: "session-1", UserID: "u-1"}
				return nil
			}}
		},
	}
	resetCollection := &fakeMongoCollection{
		insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
			return &mongo.InsertOneResult{InsertedID: resetID}, nil
		},
	}

	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionOrganizations: orgCollection,
		collectionRoles:         roleCollection,
		collectionUsers:         userCollection,
		collectionInvites:       inviteCollection,
		collectionSessions:      sessionCollection,
		collectionPasswordReset: resetCollection,
	}}
	store := &MongoStore{dbPort: db}

	org, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme Org"})
	if err != nil || org.ID != orgID {
		t.Fatalf("CreateOrganization result error=%v org=%+v", err, org)
	}
	if _, err := store.GetOrganizationBySlug(t.Context(), "acme-org"); err != nil {
		t.Fatalf("GetOrganizationBySlug error: %v", err)
	}
	if _, err := store.ListOrganizations(t.Context()); err != nil {
		t.Fatalf("ListOrganizations error: %v", err)
	}

	role, err := store.CreateRole(t.Context(), Role{OrgID: orgID, OrgSlug: "acme-org", Name: "Org Admin", CreatedAt: now})
	if err != nil || role.ID != roleID {
		t.Fatalf("CreateRole result error=%v role=%+v", err, role)
	}
	if _, err := store.GetRoleBySlug(t.Context(), "acme-org", "org-admin"); err != nil {
		t.Fatalf("GetRoleBySlug error: %v", err)
	}
	if _, err := store.ListRolesByOrg(t.Context(), "acme-org"); err != nil {
		t.Fatalf("ListRolesByOrg error: %v", err)
	}

	user, err := store.CreateUser(t.Context(), AccountUser{UserID: "u-1", Email: "Admin@Acme.Org", CreatedAt: now})
	if err != nil || user.ID != userID {
		t.Fatalf("CreateUser result error=%v user=%+v", err, user)
	}
	if _, err := store.GetUserByEmail(t.Context(), "admin@acme.org"); err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if _, err := store.GetUserByUserID(t.Context(), "u-1"); err != nil {
		t.Fatalf("GetUserByUserID error: %v", err)
	}
	if err := store.SetUserPasswordHash(t.Context(), "u-1", "hash-1"); err != nil {
		t.Fatalf("SetUserPasswordHash error: %v", err)
	}
	if err := store.SetUserLastLogin(t.Context(), "u-1", now); err != nil {
		t.Fatalf("SetUserLastLogin error: %v", err)
	}

	invite, err := store.CreateInvite(t.Context(), Invite{
		OrgID:     orgID,
		Email:     "invitee@acme.org",
		UserID:    "u-2",
		RoleSlugs: []string{"org-admin"},
		TokenHash: "invite-token",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})
	if err != nil || invite.ID != inviteID {
		t.Fatalf("CreateInvite result error=%v invite=%+v", err, invite)
	}
	if err := store.MarkInviteUsed(t.Context(), "invite-token", now); err != nil {
		t.Fatalf("MarkInviteUsed error: %v", err)
	}

	session, err := store.CreateSession(t.Context(), Session{
		SessionID:   "session-1",
		UserID:      "u-1",
		UserMongoID: userID,
		CreatedAt:   now,
		LastLoginAt: now,
		ExpiresAt:   now.Add(30 * 24 * time.Hour),
	})
	if err != nil || session.ID != sessionID {
		t.Fatalf("CreateSession result error=%v session=%+v", err, session)
	}
	if _, err := store.LoadSessionByID(t.Context(), "session-1"); err != nil {
		t.Fatalf("LoadSessionByID error: %v", err)
	}
	if err := store.DeleteSession(t.Context(), "session-1"); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}

	reset, err := store.CreatePasswordReset(t.Context(), PasswordReset{
		Email:     "admin@acme.org",
		UserID:    "u-1",
		TokenHash: "reset-token",
		ExpiresAt: now.Add(24 * time.Hour),
		CreatedAt: now,
	})
	if err != nil || reset.ID != resetID {
		t.Fatalf("CreatePasswordReset result error=%v reset=%+v", err, reset)
	}
	if err := store.MarkPasswordResetUsed(t.Context(), "reset-token", now); err != nil {
		t.Fatalf("MarkPasswordResetUsed error: %v", err)
	}
}

func TestMongoStoreAuthMethodsErrors(t *testing.T) {
	t.Run("create organization insert error", func(t *testing.T) {
		orgCollection := &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, errors.New("insert organization failed")
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionOrganizations: orgCollection,
		}}}
		if _, err := store.CreateOrganization(t.Context(), Organization{Name: "Acme"}); err == nil {
			t.Fatal("expected CreateOrganization error")
		}
	})

	t.Run("list and get organizations errors", func(t *testing.T) {
		orgCollection := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: errors.New("find one org failed")}
			},
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, errors.New("find orgs failed")
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionOrganizations: orgCollection,
		}}}
		if _, err := store.GetOrganizationBySlug(t.Context(), "acme"); err == nil {
			t.Fatal("expected GetOrganizationBySlug error")
		}
		if _, err := store.ListOrganizations(t.Context()); err == nil {
			t.Fatal("expected ListOrganizations error")
		}
	})

	t.Run("role methods errors", func(t *testing.T) {
		roleCollection := &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, errors.New("insert role failed")
			},
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: errors.New("find role failed")}
			},
			findFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (mongoCursorPort, error) {
				return nil, errors.New("find roles failed")
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionRoles: roleCollection,
		}}}
		if _, err := store.CreateRole(t.Context(), Role{OrgSlug: "acme", Name: "Role"}); err == nil {
			t.Fatal("expected CreateRole error")
		}
		if _, err := store.GetRoleBySlug(t.Context(), "acme", "role"); err == nil {
			t.Fatal("expected GetRoleBySlug error")
		}
		if _, err := store.ListRolesByOrg(t.Context(), "acme"); err == nil {
			t.Fatal("expected ListRolesByOrg error")
		}
	})

	t.Run("user methods errors", func(t *testing.T) {
		userCollection := &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, errors.New("insert user failed")
			},
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: errors.New("find user failed")}
			},
			updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, errors.New("update user failed")
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionUsers: userCollection,
		}}}
		if _, err := store.CreateUser(t.Context(), AccountUser{UserID: "u1", Email: "u1@example.com"}); err == nil {
			t.Fatal("expected CreateUser error")
		}
		if _, err := store.GetUserByEmail(t.Context(), "u1@example.com"); err == nil {
			t.Fatal("expected GetUserByEmail error")
		}
		if _, err := store.GetUserByUserID(t.Context(), "u1"); err == nil {
			t.Fatal("expected GetUserByUserID error")
		}
		if err := store.SetUserPasswordHash(t.Context(), "u1", "hash"); err == nil {
			t.Fatal("expected SetUserPasswordHash error")
		}
	})

	t.Run("invite session reset errors", func(t *testing.T) {
		inviteCollection := &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, errors.New("insert invite failed")
			},
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: errors.New("find invite failed")}
			},
			updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, errors.New("update invite failed")
			},
		}
		sessionCollection := &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, errors.New("insert session failed")
			},
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: errors.New("find session failed")}
			},
			updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, errors.New("delete session failed")
			},
		}
		resetCollection := &fakeMongoCollection{
			insertOneFn: func(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
				return nil, errors.New("insert reset failed")
			},
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: errors.New("find reset failed")}
			},
			updateOneFn: func(ctx context.Context, filter interface{}, update interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
				return nil, errors.New("mark reset failed")
			},
		}
		store := &MongoStore{dbPort: &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionInvites:       inviteCollection,
			collectionSessions:      sessionCollection,
			collectionPasswordReset: resetCollection,
		}}}
		if _, err := store.CreateInvite(t.Context(), Invite{Email: "x@example.com", TokenHash: "token"}); err == nil {
			t.Fatal("expected CreateInvite error")
		}
		if _, err := store.LoadInviteByTokenHash(t.Context(), "token"); err == nil {
			t.Fatal("expected LoadInviteByTokenHash error")
		}
		if err := store.MarkInviteUsed(t.Context(), "token", time.Now().UTC()); err == nil {
			t.Fatal("expected MarkInviteUsed error")
		}
		if _, err := store.CreateSession(t.Context(), Session{SessionID: "s1"}); err == nil {
			t.Fatal("expected CreateSession error")
		}
		if _, err := store.LoadSessionByID(t.Context(), "s1"); err == nil {
			t.Fatal("expected LoadSessionByID error")
		}
		if err := store.DeleteSession(t.Context(), "s1"); err == nil {
			t.Fatal("expected DeleteSession error")
		}
		if _, err := store.CreatePasswordReset(t.Context(), PasswordReset{Email: "x@example.com", TokenHash: "reset"}); err == nil {
			t.Fatal("expected CreatePasswordReset error")
		}
		if _, err := store.LoadPasswordResetByTokenHash(t.Context(), "reset"); err == nil {
			t.Fatal("expected LoadPasswordResetByTokenHash error")
		}
		if err := store.MarkPasswordResetUsed(t.Context(), "reset", time.Now().UTC()); err == nil {
			t.Fatal("expected MarkPasswordResetUsed error")
		}
	})
}
