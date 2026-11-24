package org

import (
	"context"
	"errors"
	"testing"
)

// assertDomainStoreRevisionContract runs against a store whose scope is empty:
// t.Name() keeps the scope unique per test so Mongo cases start from a fresh doc.
func assertDomainStoreRevisionContract(t *testing.T, s DomainStore) {
	t.Helper()
	ctx := t.Context()
	scope := Scope{Org: t.Name(), Stream: "main"}
	other := Scope{Org: t.Name(), Stream: "other"}

	revision, err := s.Revision(ctx, scope)
	if err != nil || revision != 0 {
		t.Fatalf("missing revision = %d err=%v, want 0", revision, err)
	}

	if err := s.Put(ctx, scope, EmptyDomain()); err != nil {
		t.Fatal(err)
	}
	revision, err = s.Revision(ctx, scope)
	if err != nil || revision != 1 {
		t.Fatalf("after Put revision = %d err=%v, want 1", revision, err)
	}

	got, revision, err := s.Update(ctx, scope, func(d Domain) (Domain, error) {
		d.Submissions = append(d.Submissions, Submission{JobID: "j", TaskID: "t"})
		return d, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if revision != 2 || len(got.Submissions) != 1 {
		t.Fatalf("Update = revision %d domain %#v, want revision 2 with one submission", revision, got)
	}
	revision, err = s.Revision(ctx, scope)
	if err != nil || revision != 2 {
		t.Fatalf("after Update revision = %d err=%v, want 2", revision, err)
	}

	boom := errors.New("boom")
	if _, _, err := s.Update(ctx, scope, func(Domain) (Domain, error) { return Domain{}, boom }); !errors.Is(err, boom) {
		t.Fatalf("failed Update err = %v, want the fn error unchanged", err)
	}
	revision, err = s.Revision(ctx, scope)
	if err != nil || revision != 2 {
		t.Fatalf("after failed Update revision = %d err=%v, want 2", revision, err)
	}

	revision, err = s.Revision(ctx, other)
	if err != nil || revision != 0 {
		t.Fatalf("independent scope revision = %d err=%v, want 0", revision, err)
	}
}

func TestMemoryDomainStoreRevisionContract(t *testing.T) {
	assertDomainStoreRevisionContract(t, NewMemoryDomainStore())
}

func TestMemoryDomainStoreRevisionRejectsInvalidScope(t *testing.T) {
	s := NewMemoryDomainStore()
	if _, err := s.Revision(t.Context(), Scope{}); err == nil {
		t.Fatal("invalid scope must error")
	}
}

// snapshotDomainStore isolates domainSnapshot's read ordering from the full stores.
type snapshotDomainStore struct {
	DomainStore
	revision    int64
	revisionErr error
	getErr      error
}

func (s snapshotDomainStore) Revision(context.Context, Scope) (int64, error) {
	return s.revision, s.revisionErr
}

func (s snapshotDomainStore) Get(context.Context, Scope) (Domain, error) {
	return Domain{}, s.getErr
}

func TestDomainSnapshotReadErrors(t *testing.T) {
	scope := Scope{Org: "acme", Stream: "main"}
	boom := errors.New("boom")

	if _, _, err := domainSnapshot(t.Context(), snapshotDomainStore{revisionErr: boom}, scope); !errors.Is(err, boom) {
		t.Fatalf("revision error = %v, want boom", err)
	}
	if _, _, err := domainSnapshot(t.Context(), snapshotDomainStore{getErr: boom}, scope); !errors.Is(err, boom) {
		t.Fatalf("get error = %v, want boom", err)
	}

	domain, revision, err := domainSnapshot(t.Context(), snapshotDomainStore{
		revisionErr: ErrNotFound,
		getErr:      ErrNotFound,
	}, scope)
	if err != nil || revision != 0 || len(domain.Entities) != 0 {
		t.Fatalf("missing snapshot = %#v revision=%d err=%v, want empty domain and 0", domain, revision, err)
	}
}
