package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func TestIsMongoIndexNotFoundError(t *testing.T) {
	if isMongoIndexNotFoundError(nil) {
		t.Fatal("nil error should not match")
	}
	if !isMongoIndexNotFoundError(mongo.CommandError{Code: 26}) {
		t.Fatal("code 26 should match")
	}
	if !isMongoIndexNotFoundError(mongo.CommandError{Code: 27}) {
		t.Fatal("code 27 should match")
	}
	if !isMongoIndexNotFoundError(errors.New("Index not found for collection")) {
		t.Fatal("index not found message should match")
	}
	if !isMongoIndexNotFoundError(errors.New("ns not found")) {
		t.Fatal("ns not found message should match")
	}
	if isMongoIndexNotFoundError(errors.New("different error")) {
		t.Fatal("unexpected message should not match")
	}
}

func TestCanonifyOptionalSlugAndHashLookupToken(t *testing.T) {
	if got := canonifyOptionalSlug("   "); got != "" {
		t.Fatalf("canonifyOptionalSlug(blank) = %q, want empty", got)
	}
	if got := canonifyOptionalSlug("  ACME_Test Org  "); got != "acme-test-org" {
		t.Fatalf("canonifyOptionalSlug = %q, want acme-test-org", got)
	}

	token := "  invite-secret  "
	sum := sha256.Sum256([]byte("invite-secret"))
	want := hex.EncodeToString(sum[:])
	if got := hashLookupToken(token); got != want {
		t.Fatalf("hashLookupToken = %q, want %q", got, want)
	}
}

func TestMongoStoreGetUserByMongoID(t *testing.T) {
	t.Run("loads user", func(t *testing.T) {
		userID := primitive.NewObjectID()
		want := AccountUser{ID: userID, Email: "user@example.com"}
		collection := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{decodeFn: func(v interface{}) error {
					*(v.(*AccountUser)) = want
					return nil
				}}
			},
		}
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"users": collection}}
		store := &MongoStore{dbPort: db}

		got, err := store.GetUserByMongoID(t.Context(), userID)
		if err != nil {
			t.Fatalf("GetUserByMongoID error: %v", err)
		}
		if got.Email != want.Email || got.ID != want.ID {
			t.Fatalf("user = %#v, want %#v", got, want)
		}
		if len(collection.findOneFilters) != 1 || collection.findOneFilters[0] == nil {
			t.Fatalf("findOne filters = %#v", collection.findOneFilters)
		}
		filter, ok := collection.findOneFilters[0].(bson.M)
		if !ok || filter["_id"] != userID {
			t.Fatalf("filter = %#v, want _id=%s", collection.findOneFilters[0], userID.Hex())
		}
	})

	t.Run("propagates decode error", func(t *testing.T) {
		boom := errors.New("boom")
		collection := &fakeMongoCollection{
			findOneFn: func(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) mongoSingleResultPort {
				return fakeSingleResult{err: boom}
			},
		}
		db := &fakeMongoDatabase{collections: map[string]*fakeMongoCollection{"users": collection}}
		store := &MongoStore{dbPort: db}

		if _, err := store.GetUserByMongoID(t.Context(), primitive.NewObjectID()); !errors.Is(err, boom) {
			t.Fatalf("GetUserByMongoID error = %v, want %v", err, boom)
		}
	})
}
