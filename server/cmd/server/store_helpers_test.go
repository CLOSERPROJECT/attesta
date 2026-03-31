package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
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
