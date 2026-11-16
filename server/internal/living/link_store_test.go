package living

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryLinkStorePutGetByAddressAndTimestamp(t *testing.T) {
	store := NewMemoryLinkStore()
	addr := SubmissionAddress{
		JobID: "intake", JobInstanceID: "T1", TaskID: "register",
		GroupID: "lots", GroupInstanceID: "lots-1",
	}
	ts := "2026-11-13T12:00:00.000000000Z"
	links := []HttpLinkItem{
		{Label: "Stampa", URL: "blob:labels", Action: HttpActionLink},
		{Label: "CSV", URL: "blob:csv", Action: HttpActionDownload, Filename: "reg.csv"},
	}
	if err := store.Put(t.Context(), "feragame", addr, ts, links); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(t.Context(), "feragame", addr, ts)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].URL != "blob:labels" || got[1].Filename != "reg.csv" {
		t.Fatalf("got = %#v", got)
	}
	got[0].URL = "mutated"
	again, err := store.Get(t.Context(), "feragame", addr, ts)
	if err != nil {
		t.Fatal(err)
	}
	if again[0].URL != "blob:labels" {
		t.Fatalf("store leaked mutation: %#v", again)
	}
	if _, err := store.Get(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: "T1", TaskID: "register",
	}, ts); !errors.Is(err, ErrNotFound) {
		t.Fatalf("flat address should miss nested record: %v", err)
	}
	if _, err := store.Get(t.Context(), "feragame", addr, "2026-11-13T12:00:01Z"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("wrong timestamp should miss: %v", err)
	}
	if err := store.Put(t.Context(), "feragame", addr, "", links); err == nil {
		t.Fatal("empty timestamp must fail Put")
	}
}

func TestCommitPersistsLinksAndReadableByAddressAndTimestamp(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg.csv"`}},
		},
	}
	links := NewMemoryLinkStore()
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.LinkStore = links

	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	if len(result.Links) != 2 {
		t.Fatalf("commit response links = %#v", result.Links)
	}

	var addr SubmissionAddress
	var ts string
	found := false
	for _, sub := range result.Domain.Submissions {
		if sub.TaskID == "register" {
			addr = SubmissionAddress{
				JobID: sub.JobID, JobInstanceID: sub.JobInstanceID, TaskID: sub.TaskID,
				GroupID: sub.GroupID, GroupInstanceID: sub.GroupInstanceID,
			}
			ts = sub.Timestamp
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("register submission missing: %#v", result.Domain.Submissions)
	}
	if addr.GroupID != "" || addr.GroupInstanceID != "" {
		t.Fatalf("flat register address must omit group: %#v", addr)
	}
	if ts == "" {
		t.Fatal("register submission missing timestamp")
	}

	stored, err := links.Get(t.Context(), "feragame", addr, ts)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 2 || stored[0].Label != "Stampa etichette" || stored[1].Filename != "reg.csv" {
		t.Fatalf("stored links = %#v", stored)
	}

	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	for _, sub := range domain.Submissions {
		if sub.TaskID == "register" {
			if _, has := sub.Facts["links"]; has {
				t.Fatalf("links must not fold into Submission facts: %#v", sub.Facts)
			}
		}
	}
}

func TestCommitManySubmitPreservesLinksPerTimestamp(t *testing.T) {
	clock := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels-1"},
			{URL: "blob:csv-1", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg1.csv"`}},
			{URL: "blob:labels-2"},
			{URL: "blob:csv-2", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg2.csv"`}},
		},
	}
	store := NewMemoryLinkStore()
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.LinkStore = store
	c.Now = func() time.Time {
		now := clock
		clock = clock.Add(time.Second)
		return now
	}

	truckID := commitTruckArrive(t, c)
	first := commitLotRegister(t, c, truckID, registerLotValue())
	lot2 := registerLotValue()
	lot2["client"] = "DA"
	lot2["numeroFir"] = "FIR-2"
	second := commitLotRegister(t, c, truckID, lot2)

	addr := SubmissionAddress{JobID: "intake", JobInstanceID: truckID, TaskID: "register"}
	var ts1, ts2 string
	for _, sub := range second.Domain.Submissions {
		if sub.TaskID != "register" || sub.JobInstanceID != truckID {
			continue
		}
		if ts1 == "" {
			ts1 = sub.Timestamp
			continue
		}
		ts2 = sub.Timestamp
	}
	if ts1 == "" || ts2 == "" || ts1 == ts2 {
		t.Fatalf("want two distinct register timestamps, got %q %q (subs=%#v)", ts1, ts2, second.Domain.Submissions)
	}

	got1, err := store.Get(t.Context(), "feragame", addr, ts1)
	if err != nil {
		t.Fatal(err)
	}
	got2, err := store.Get(t.Context(), "feragame", addr, ts2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got1) != 2 || got1[0].URL != "blob:labels-1" || got1[1].Filename != "reg1.csv" {
		t.Fatalf("first links = %#v (commit links=%#v)", got1, first.Links)
	}
	if len(got2) != 2 || got2[0].URL != "blob:labels-2" || got2[1].Filename != "reg2.csv" {
		t.Fatalf("second links = %#v (commit links=%#v)", got2, second.Links)
	}
	if len(store.byKey) != 2 {
		t.Fatalf("link docs = %d, want 2", len(store.byKey))
	}
}

func TestCommitLinkStoreFailureDoesNotFailOrRollback(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg.csv"`}},
		},
	}
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.LinkStore = &errLinkStore{err: errors.New("links down")}

	truckID := commitTruckArrive(t, c)
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         registerLotValue(),
	})
	if err != nil {
		t.Fatalf("Commit must succeed when LinkStore fails: %v", err)
	}
	if len(result.Links) != 2 {
		t.Fatalf("response must still return links: %#v", result.Links)
	}
	loadID := firstLoadID(result.Domain)
	if result.Domain.Kind("lot")[loadID]["client"] != "CO" {
		t.Fatalf("domain missing after link store failure: %#v", result.Domain.Kind("lot"))
	}
	stored, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("lot")[loadID]["client"] != "CO" {
		t.Fatalf("DomainStore rolled back: %#v", stored)
	}
}

func TestCommitSkipsLinkStoreWhenLinksEmpty(t *testing.T) {
	store := &countingLinkStore{inner: NewMemoryLinkStore()}
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	c.LinkStore = store

	_ = commitTruckArrive(t, c)
	if store.puts != 0 {
		t.Fatalf("truckArrive has no http links; puts=%d", store.puts)
	}
}

type errLinkStore struct{ err error }

func (e *errLinkStore) Put(context.Context, string, SubmissionAddress, string, []HttpLinkItem) error {
	return e.err
}
func (e *errLinkStore) Get(context.Context, string, SubmissionAddress, string) ([]HttpLinkItem, error) {
	return nil, e.err
}

type countingLinkStore struct {
	inner LinkStore
	puts  int
}

func (c *countingLinkStore) Put(ctx context.Context, org string, addr SubmissionAddress, timestamp string, links []HttpLinkItem) error {
	c.puts++
	return c.inner.Put(ctx, org, addr, timestamp, links)
}
func (c *countingLinkStore) Get(ctx context.Context, org string, addr SubmissionAddress, timestamp string) ([]HttpLinkItem, error) {
	return c.inner.Get(ctx, org, addr, timestamp)
}
