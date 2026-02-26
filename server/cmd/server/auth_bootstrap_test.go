package main

import (
	"errors"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

func TestBootstrapPlatformAdminRequiresEnvWhenClosedSignup(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "false")
	t.Setenv("ADMIN_EMAIL", "")
	t.Setenv("ADMIN_PASSWORD", "")

	err := bootstrapPlatformAdmin(t.Context(), NewMemoryStore(), time.Now)
	if err == nil {
		t.Fatal("expected error when closed signup misses admin credentials")
	}
}

func TestBootstrapPlatformAdminCreatesAndIsIdempotent(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "false")
	t.Setenv("ADMIN_EMAIL", "owner@example.com")
	t.Setenv("ADMIN_PASSWORD", "super-secure-password")

	store := NewMemoryStore()
	fixedNow := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)
	now := func() time.Time { return fixedNow }

	if err := bootstrapPlatformAdmin(t.Context(), store, now); err != nil {
		t.Fatalf("bootstrapPlatformAdmin first run error: %v", err)
	}
	user, err := store.GetUserByEmail(t.Context(), "owner@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail error: %v", err)
	}
	if !user.IsPlatformAdmin {
		t.Fatal("expected platform admin flag")
	}
	if user.OrgID != nil || user.OrgSlug != "" {
		t.Fatalf("platform admin org fields should be empty, got orgID=%v orgSlug=%q", user.OrgID, user.OrgSlug)
	}
	if user.CreatedAt != fixedNow {
		t.Fatalf("createdAt = %s, want %s", user.CreatedAt, fixedNow)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("super-secure-password")); err != nil {
		t.Fatalf("password hash check failed: %v", err)
	}

	firstID := user.ID
	firstHash := user.PasswordHash
	if err := bootstrapPlatformAdmin(t.Context(), store, now); err != nil {
		t.Fatalf("bootstrapPlatformAdmin second run error: %v", err)
	}
	userAfter, err := store.GetUserByEmail(t.Context(), "owner@example.com")
	if err != nil {
		t.Fatalf("GetUserByEmail after second run error: %v", err)
	}
	if userAfter.ID != firstID {
		t.Fatalf("id after second run = %s, want %s", userAfter.ID.Hex(), firstID.Hex())
	}
	if userAfter.PasswordHash != firstHash {
		t.Fatal("expected second run not to overwrite existing password hash")
	}
}

func TestBootstrapPlatformAdminSkipsWithoutAdminCredentialsWhenOpenSignup(t *testing.T) {
	t.Setenv("ANYONE_CAN_CREATE_ACCOUNT", "true")
	t.Setenv("ADMIN_EMAIL", "")
	t.Setenv("ADMIN_PASSWORD", "")

	store := NewMemoryStore()
	if err := bootstrapPlatformAdmin(t.Context(), store, time.Now); err != nil {
		t.Fatalf("bootstrapPlatformAdmin error: %v", err)
	}

	if _, err := store.GetUserByEmail(t.Context(), "owner@example.com"); !errors.Is(err, mongo.ErrNoDocuments) {
		t.Fatalf("unexpected user lookup result: %v", err)
	}
}
