package org

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type conflictDomainStore struct{ DomainStore }

func (conflictDomainStore) Update(context.Context, Scope, func(Domain) (Domain, error)) (Domain, error) {
	return Domain{}, ErrDomainConflict
}

func TestMemoryDomainStoreUpdateMissingOrgPassesEmptyDomain(t *testing.T) {
	s := NewMemoryDomainStore()
	var seen Domain
	got, err := s.Update(t.Context(), Scope{Org: "acme", Stream: "main"}, func(d Domain) (Domain, error) {
		seen = d
		d.Submissions = append(d.Submissions, Submission{JobID: "j", TaskID: "t"})
		return d, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(seen.Submissions) != 0 || seen.Entities == nil {
		t.Fatalf("fn input = %#v, want EmptyDomain", seen)
	}
	stored, err := s.Get(t.Context(), Scope{Org: "acme", Stream: "main"})
	if err != nil || len(stored.Submissions) != 1 || len(got.Submissions) != 1 {
		t.Fatalf("stored = %#v err=%v", stored, err)
	}
}

func TestMemoryDomainStoreUpdateFnErrorWritesNothing(t *testing.T) {
	s := NewMemoryDomainStore()
	boom := errors.New("boom")
	_, err := s.Update(t.Context(), Scope{Org: "acme", Stream: "main"}, func(d Domain) (Domain, error) {
		d.Submissions = append(d.Submissions, Submission{JobID: "j"})
		return d, boom
	})
	if err != boom {
		t.Fatalf("err = %v, want the fn error unchanged", err)
	}
	if _, err := s.Get(t.Context(), Scope{Org: "acme", Stream: "main"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get after failed Update = %v, want ErrNotFound", err)
	}
}

func TestMemoryDomainStoreUpdateConcurrentAppendsAllLand(t *testing.T) {
	s := NewMemoryDomainStore()
	const n = 50
	var wg sync.WaitGroup
	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Update(context.Background(), Scope{Org: "acme", Stream: "main"}, func(d Domain) (Domain, error) {
				d.Submissions = append(d.Submissions, Submission{JobID: "j", TaskID: "t", JobInstanceID: string(rune('a' + i))})
				return d, nil
			})
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := s.Get(t.Context(), Scope{Org: "acme", Stream: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Submissions) != n {
		t.Fatalf("submissions = %d, want %d", len(got.Submissions), n)
	}
}

func TestCommitConcurrentMintsDistinctIDs(t *testing.T) {
	c, domains, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	const n = 10
	var wg sync.WaitGroup
	ids := make(chan string, n)
	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			res, err := c.Commit(context.Background(), FeragameScope(), op, CommitRequest{
				TaskPath: "intake/truckArrive",
				Value:    truckArriveValue(),
				Mint:     startingNewMint(),
			})
			if err != nil {
				t.Error(err)
				return
			}
			ids <- res.JobInstanceID
		}()
	}
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate minted id %q", id)
		}
		seen[id] = true
	}
	got, err := domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Submissions) != n || len(got.Kind("truck")) != n {
		t.Fatalf("submissions=%d trucks=%d, want %d each", len(got.Submissions), len(got.Kind("truck")), n)
	}
}

func TestCommitMapsDomainConflict(t *testing.T) {
	c, domains, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	c.Domains = conflictDomainStore{domains}
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	req := CommitRequest{TaskPath: "intake/truckArrive", Value: truckArriveValue(), Mint: startingNewMint()}

	_, err := c.Commit(t.Context(), FeragameScope(), op, req)
	var ce *CommitError
	if !asCommitError(err, &ce) || ce.Kind != "conflict" {
		t.Fatalf("err = %v, want CommitError conflict", err)
	}

	h := &API{
		DomainLoader:  loader,
		TaskCommitter: c,
		Gate:          func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) { return op, true },
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/commit", strings.NewReader(
		`{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"truckWeight":1000}}`,
	)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
}
