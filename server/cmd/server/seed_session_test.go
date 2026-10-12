package main

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestOpenSeedSessionNilOpenStore(t *testing.T) {
	t.Parallel()
	_, err := openSeedSession(t.Context(), nil, "config", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "store opener is required") {
		t.Fatalf("err = %v, want store opener required", err)
	}
}

func TestOpenSeedSessionStoreOpenError(t *testing.T) {
	t.Parallel()
	want := errors.New("mongo down")
	_, err := openSeedSession(t.Context(), nil, "config", func(context.Context) (Store, func(), error) {
		return nil, nil, want
	}, nil)
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
}

func TestOpenSeedSessionSuccessWithoutIdentity(t *testing.T) {
	t.Parallel()
	store := NewMemoryStore()
	var cleaned bool
	sess, err := openSeedSession(t.Context(), []string{"--file", "/tmp/cats.yaml"}, "cfg-dir", func(context.Context) (Store, func(), error) {
		return store, func() { cleaned = true }, nil
	}, nil)
	if err != nil {
		t.Fatalf("openSeedSession: %v", err)
	}
	if sess.Store != store {
		t.Fatal("Store not set")
	}
	if sess.Identity != nil {
		t.Fatal("Identity should be unset when openIdentity is nil")
	}
	if sess.ConfigDir != "cfg-dir" {
		t.Fatalf("ConfigDir = %q", sess.ConfigDir)
	}
	if sess.CategoriesPath != "/tmp/cats.yaml" {
		t.Fatalf("CategoriesPath = %q", sess.CategoriesPath)
	}
	if sess.Cleanup == nil {
		t.Fatal("Cleanup should be set")
	}
	sess.Cleanup()
	if !cleaned {
		t.Fatal("Cleanup was not invoked")
	}
}

func TestOpenSeedSessionEmptyConfigDirUsesWorkflowConfigDir(t *testing.T) {
	store := NewMemoryStore()
	sess, err := openSeedSession(t.Context(), nil, "  ", func(context.Context) (Store, func(), error) {
		return store, func() {}, nil
	}, nil)
	if err != nil {
		t.Fatalf("openSeedSession: %v", err)
	}
	if sess.ConfigDir != "config" {
		t.Fatalf("ConfigDir = %q, want %q", sess.ConfigDir, "config")
	}
}

func TestOpenSeedSessionIdentityOpenErrorRunsCleanup(t *testing.T) {
	t.Parallel()
	var cleaned bool
	want := errors.New("appwrite down")
	_, err := openSeedSession(t.Context(), nil, "cfg", func(context.Context) (Store, func(), error) {
		return NewMemoryStore(), func() { cleaned = true }, nil
	}, func() (IdentityStore, error) {
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if !cleaned {
		t.Fatal("Cleanup should run when identity open fails")
	}
}

func TestOpenSeedSessionIdentitySuccess(t *testing.T) {
	t.Parallel()
	identity := &fakeIdentityStore{}
	sess, err := openSeedSession(t.Context(), nil, "cfg", func(context.Context) (Store, func(), error) {
		return NewMemoryStore(), func() {}, nil
	}, func() (IdentityStore, error) {
		return identity, nil
	})
	if err != nil {
		t.Fatalf("openSeedSession: %v", err)
	}
	if sess.Identity != identity {
		t.Fatal("Identity not set")
	}
}

func TestWithSeedSessionPropagatesOpenError(t *testing.T) {
	t.Parallel()
	called := false
	err := withSeedSession(t.Context(), nil, "cfg", nil, nil, func(seedSession) error {
		called = true
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "store opener is required") {
		t.Fatalf("err = %v", err)
	}
	if called {
		t.Fatal("fn must not run on open error")
	}
}

func TestWithSeedSessionRunsFnAndCleanup(t *testing.T) {
	t.Parallel()
	var cleaned bool
	var sawStore Store
	err := withSeedSession(t.Context(), nil, "cfg", func(context.Context) (Store, func(), error) {
		return NewMemoryStore(), func() { cleaned = true }, nil
	}, nil, func(sess seedSession) error {
		sawStore = sess.Store
		return nil
	})
	if err != nil {
		t.Fatalf("withSeedSession: %v", err)
	}
	if sawStore == nil {
		t.Fatal("fn did not receive store")
	}
	if !cleaned {
		t.Fatal("Cleanup should run after fn")
	}
}

func TestWithSeedSessionCleanupRunsOnFnError(t *testing.T) {
	t.Parallel()
	var cleaned bool
	want := errors.New("fn failed")
	err := withSeedSession(t.Context(), nil, "cfg", func(context.Context) (Store, func(), error) {
		return NewMemoryStore(), func() { cleaned = true }, nil
	}, nil, func(seedSession) error {
		return want
	})
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if !cleaned {
		t.Fatal("Cleanup should run when fn returns error")
	}
}
