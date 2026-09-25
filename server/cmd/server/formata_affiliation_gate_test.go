package main

import (
	"context"
	"testing"
)

func TestCanViewFormataBuilderAffiliationGate(t *testing.T) {
	allowAll := fakeAuthorizer{
		accessDecide: func(_ *AccountUser, _, _ string, _ map[string]interface{}, _ string) (bool, error) {
			return true, nil
		},
	}
	server := &Server{authorizer: allowAll, store: NewMemoryStore()}

	t.Run("nil user denied", func(t *testing.T) {
		allowed, err := server.canViewFormataBuilder(context.Background(), nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("nil user should be denied")
		}
	})

	t.Run("unaffiliated denied even when cerbos allows", func(t *testing.T) {
		user := &AccountUser{
			Email:     "unaffiliated@example.com",
			RoleSlugs: []string{"org-admin"},
		}
		allowed, err := server.canViewFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("unaffiliated user should be denied before Cerbos")
		}
	})

	t.Run("affiliated org admin allowed via cerbos", func(t *testing.T) {
		user := &AccountUser{
			Email:     "org-admin@example.com",
			OrgSlug:   "acme",
			RoleSlugs: []string{"org-admin"},
		}
		allowed, err := server.canViewFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("affiliated org admin should be allowed via Cerbos")
		}
	})

	t.Run("platform admin without org allowed via cerbos", func(t *testing.T) {
		user := &AccountUser{
			Email:           "platform@example.com",
			IsPlatformAdmin: true,
		}
		allowed, err := server.canViewFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("platform admin without org should reach Cerbos and be allowed")
		}
	})

	t.Run("affiliated non org admin still cerbos denied", func(t *testing.T) {
		server := &Server{authorizer: fakeAuthorizer{}, store: NewMemoryStore()}
		user := &AccountUser{
			Email:     "member@example.com",
			OrgSlug:   "acme",
			RoleSlugs: []string{"viewer"},
		}
		allowed, err := server.canViewFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("affiliated non-org-admin should still be denied by Cerbos")
		}
	})
}

func TestCanSaveFormataBuilderAffiliationGate(t *testing.T) {
	allowAll := fakeAuthorizer{
		accessDecide: func(_ *AccountUser, _, _ string, _ map[string]interface{}, _ string) (bool, error) {
			return true, nil
		},
	}
	server := &Server{authorizer: allowAll, store: NewMemoryStore()}

	t.Run("unaffiliated denied even when cerbos allows", func(t *testing.T) {
		user := &AccountUser{
			Email:     "unaffiliated@example.com",
			RoleSlugs: []string{"org-admin"},
		}
		allowed, err := server.canSaveFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("unaffiliated user should be denied before Cerbos")
		}
	})

	t.Run("affiliated org admin allowed via cerbos", func(t *testing.T) {
		user := &AccountUser{
			Email:     "org-admin@example.com",
			OrgSlug:   "acme",
			RoleSlugs: []string{"org-admin"},
		}
		allowed, err := server.canSaveFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("affiliated org admin should be allowed via Cerbos")
		}
	})

	t.Run("platform admin without org allowed via cerbos", func(t *testing.T) {
		user := &AccountUser{
			Email:           "platform@example.com",
			IsPlatformAdmin: true,
		}
		allowed, err := server.canSaveFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !allowed {
			t.Fatal("platform admin without org should reach Cerbos and be allowed")
		}
	})

	t.Run("affiliated non org admin still cerbos denied", func(t *testing.T) {
		server := &Server{authorizer: fakeAuthorizer{}, store: NewMemoryStore()}
		user := &AccountUser{
			Email:     "member@example.com",
			OrgSlug:   "acme",
			RoleSlugs: []string{"viewer"},
		}
		allowed, err := server.canSaveFormataBuilder(context.Background(), user)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if allowed {
			t.Fatal("affiliated non-org-admin should still be denied by Cerbos")
		}
	})
}
