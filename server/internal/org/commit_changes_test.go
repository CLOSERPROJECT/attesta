package org

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type domainChangeCall struct {
	Scope    Scope
	Revision int64
}

type recordingNotifier struct {
	mu    sync.Mutex
	calls []domainChangeCall
}

func (n *recordingNotifier) DomainChanged(scope Scope, revision int64) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.calls = append(n.calls, domainChangeCall{Scope: scope, Revision: revision})
}

func (n *recordingNotifier) recorded() []domainChangeCall {
	n.mu.Lock()
	defer n.mu.Unlock()
	return append([]domainChangeCall(nil), n.calls...)
}

func TestCommitNotifiesDomainChange(t *testing.T) {
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	notifier := &recordingNotifier{}
	c.Changes = notifier

	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath: "intake/truckArrive",
		Value:    truckArriveValue(),
		Mint:     startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}

	calls := notifier.recorded()
	if len(calls) != 1 {
		t.Fatalf("notifications = %#v, want exactly one", calls)
	}
	if calls[0].Scope != FeragameScope() {
		t.Fatalf("notified scope = %#v, want %#v", calls[0].Scope, FeragameScope())
	}
	if calls[0].Revision != result.Revision {
		t.Fatalf("notified revision = %d, want commit revision %d", calls[0].Revision, result.Revision)
	}
}

func TestCommitDoesNotNotifyWhenNothingWasWritten(t *testing.T) {
	t.Run("authorizer denies", func(t *testing.T) {
		c, _, _ := testTaskCommitter(t, DenyAllAuthorizer{})
		notifier := &recordingNotifier{}
		c.Changes = notifier

		_, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
			TaskPath: "intake/truckArrive",
			Value:    truckArriveValue(),
			Mint:     startingNewMint(),
		})
		if err == nil {
			t.Fatal("Commit succeeded, want authorization error")
		}
		if calls := notifier.recorded(); len(calls) != 0 {
			t.Fatalf("notifications = %#v, want none", calls)
		}
	})

	t.Run("domain conflict", func(t *testing.T) {
		c, domains, _ := testTaskCommitter(t, AllowAllAuthorizer{})
		c.Domains = conflictDomainStore{domains}
		notifier := &recordingNotifier{}
		c.Changes = notifier

		_, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
			TaskPath: "intake/truckArrive",
			Value:    truckArriveValue(),
			Mint:     startingNewMint(),
		})
		if err == nil {
			t.Fatal("Commit succeeded, want conflict error")
		}
		if calls := notifier.recorded(); len(calls) != 0 {
			t.Fatalf("notifications = %#v, want none", calls)
		}
	})
}

func TestCommitNotifiesWhenLaterSideRecordFails(t *testing.T) {
	c, domains, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	c.DPPs = &createErrDPPStore{err: errors.New("boom")}
	c.Now = func() time.Time { return time.Date(2026, 11, 4, 14, 0, 0, 0, time.UTC) }
	loadID, _ := seedFullyProcessedLoad(t, c)

	notifier := &recordingNotifier{}
	c.Changes = notifier

	_, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "lot_processing/lock",
		JobInstanceID: loadID,
		Value:         map[string]any{"ack": true},
	})
	if err == nil {
		t.Fatal("Commit succeeded, want DPP create error")
	}

	calls := notifier.recorded()
	if len(calls) != 1 {
		t.Fatalf("notifications = %#v, want exactly one", calls)
	}
	if calls[0].Scope != FeragameScope() {
		t.Fatalf("notified scope = %#v, want %#v", calls[0].Scope, FeragameScope())
	}
	revision, revisionErr := domains.Revision(t.Context(), FeragameScope())
	if revisionErr != nil {
		t.Fatal(revisionErr)
	}
	if calls[0].Revision != revision {
		t.Fatalf("notified revision = %d, want the written revision %d", calls[0].Revision, revision)
	}
}
