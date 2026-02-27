package main

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestCanonifySlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim and lowercase", input: "  Quality Control  ", want: "quality-control"},
		{name: "replace separators", input: "Org_Admin--West", want: "org-admin-west"},
		{name: "empty fallback", input: "   ", want: "item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canonifySlug(tt.input); got != tt.want {
				t.Fatalf("canonifySlug(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestHashLookupToken(t *testing.T) {
	token := "token-123"
	got := hashLookupToken(token)
	if got == "" {
		t.Fatal("expected hashLookupToken to return non-empty hash")
	}
	if got != hashLookupToken(token) {
		t.Fatal("expected deterministic token hash")
	}
	if got == hashLookupToken("another") {
		t.Fatal("expected different tokens to produce different hashes")
	}
}

func TestMongoStoreEnsureAuthIndexes(t *testing.T) {
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{}}
	store := &MongoStore{dbPort: db}

	if err := store.EnsureAuthIndexes(t.Context()); err != nil {
		t.Fatalf("EnsureAuthIndexes returned error: %v", err)
	}

	assertIndexes := func(collection string, want []mongo.IndexModel) {
		t.Helper()
		c := db.collections[collection]
		if c == nil {
			t.Fatalf("missing collection %q", collection)
		}
		if len(c.createIndexesModels) != 1 {
			t.Fatalf("%s createIndexes calls = %d, want 1", collection, len(c.createIndexesModels))
		}
		got := c.createIndexesModels[0]
		if len(got) != len(want) {
			t.Fatalf("%s indexes len = %d, want %d", collection, len(got), len(want))
		}
		for i := range want {
			if !reflect.DeepEqual(got[i].Keys, want[i].Keys) {
				t.Fatalf("%s index[%d] keys = %#v, want %#v", collection, i, got[i].Keys, want[i].Keys)
			}
			gotUnique := indexOptionBool(got[i].Options, "unique")
			wantUnique := indexOptionBool(want[i].Options, "unique")
			if gotUnique != wantUnique {
				t.Fatalf("%s index[%d] unique = %v, want %v", collection, i, gotUnique, wantUnique)
			}
			gotTTL := indexOptionInt(got[i].Options, "expireAfterSeconds")
			wantTTL := indexOptionInt(want[i].Options, "expireAfterSeconds")
			if gotTTL != wantTTL {
				t.Fatalf("%s index[%d] ttl = %d, want %d", collection, i, gotTTL, wantTTL)
			}
		}
	}

	assertIndexes(collectionOrganizations, []mongo.IndexModel{
		{Keys: bson.D{{Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	assertIndexes(collectionRoles, []mongo.IndexModel{
		{Keys: bson.D{{Key: "orgId", Value: 1}, {Key: "slug", Value: 1}}, Options: options.Index().SetUnique(true)},
	})
	assertIndexes(collectionUsers, []mongo.IndexModel{
		{Keys: bson.D{{Key: "email", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "userId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "orgId", Value: 1}}},
	})
	assertIndexes(collectionInvites, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "email", Value: 1}}},
	})
	assertIndexes(collectionSessions, []mongo.IndexModel{
		{Keys: bson.D{{Key: "sessionId", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
		{Keys: bson.D{{Key: "userId", Value: 1}}},
	})
	assertIndexes(collectionPasswordReset, []mongo.IndexModel{
		{Keys: bson.D{{Key: "tokenHash", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "expiresAt", Value: 1}}, Options: options.Index().SetExpireAfterSeconds(0)},
	})
}

func TestMongoStoreEnsureAuthIndexesError(t *testing.T) {
	indexErr := errors.New("index failed")
	db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
		collectionOrganizations: {
			createIndexesFn: func(_ context.Context, _ []mongo.IndexModel) error {
				return indexErr
			},
		},
	}}
	store := &MongoStore{dbPort: db}

	if err := store.EnsureAuthIndexes(t.Context()); !errors.Is(err, indexErr) {
		t.Fatalf("EnsureAuthIndexes error = %v, want %v", err, indexErr)
	}
}

func TestEnsureStoreIndexes(t *testing.T) {
	t.Run("noop for non-mongo store", func(t *testing.T) {
		if err := ensureStoreIndexes(t.Context(), NewMemoryStore()); err != nil {
			t.Fatalf("ensureStoreIndexes returned error: %v", err)
		}
	})

	t.Run("runs mongo auth indexes", func(t *testing.T) {
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{}}
		store := &MongoStore{dbPort: db}
		if err := ensureStoreIndexes(t.Context(), store); err != nil {
			t.Fatalf("ensureStoreIndexes returned error: %v", err)
		}
		if len(db.collections[collectionOrganizations].createIndexesModels) != 1 {
			t.Fatalf("expected organizations indexes creation call, got %#v", db.collections[collectionOrganizations].createIndexesModels)
		}
		if len(db.collections[collectionRoles].createIndexesModels) != 1 {
			t.Fatalf("expected roles indexes creation call, got %#v", db.collections[collectionRoles].createIndexesModels)
		}
	})

	t.Run("returns mongo index error", func(t *testing.T) {
		indexErr := errors.New("indexes failed")
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{
			collectionOrganizations: {
				createIndexesFn: func(_ context.Context, _ []mongo.IndexModel) error {
					return indexErr
				},
			},
		}}
		store := &MongoStore{dbPort: db}
		if err := ensureStoreIndexes(t.Context(), store); !errors.Is(err, indexErr) {
			t.Fatalf("ensureStoreIndexes error = %v, want %v", err, indexErr)
		}
	})
}

func indexOptionBool(opts *options.IndexOptions, name string) bool {
	if opts == nil {
		return false
	}
	switch name {
	case "unique":
		if opts.Unique == nil {
			return false
		}
		return *opts.Unique
	default:
		return false
	}
}

func indexOptionInt(opts *options.IndexOptions, name string) int32 {
	if opts == nil {
		return -1
	}
	switch name {
	case "expireAfterSeconds":
		if opts.ExpireAfterSeconds == nil {
			return -1
		}
		return *opts.ExpireAfterSeconds
	default:
		return -1
	}
}
