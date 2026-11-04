package living

import (
	"context"
	"errors"
	"testing"
)

func TestMemoryLinkStorePutGetByAddress(t *testing.T) {
	store := NewMemoryLinkStore()
	addr := SubmissionAddress{
		JobID: "intake", JobInstanceID: "T1", TaskID: "register",
		GroupID: "lots", GroupInstanceID: "lots-1",
	}
	links := []HttpLinkItem{
		{Label: "Stampa", URL: "blob:labels", Action: HttpActionLink},
		{Label: "CSV", URL: "blob:csv", Action: HttpActionDownload, Filename: "reg.csv"},
	}
	if err := store.Put(t.Context(), "feragame", addr, links); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(t.Context(), "feragame", addr)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].URL != "blob:labels" || got[1].Filename != "reg.csv" {
		t.Fatalf("got = %#v", got)
	}
	got[0].URL = "mutated"
	again, err := store.Get(t.Context(), "feragame", addr)
	if err != nil {
		t.Fatal(err)
	}
	if again[0].URL != "blob:labels" {
		t.Fatalf("store leaked mutation: %#v", again)
	}
	if _, err := store.Get(t.Context(), "feragame", SubmissionAddress{
		JobID: "intake", JobInstanceID: "T1", TaskID: "register",
	}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("flat address should miss nested record: %v", err)
	}
}

func TestCommitPersistsLinksAndReadableByAddress(t *testing.T) {
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
	found := false
	for _, sub := range result.Domain.Submissions {
		if sub.TaskID == "register" {
			addr = SubmissionAddress{
				JobID: sub.JobID, JobInstanceID: sub.JobInstanceID, TaskID: sub.TaskID,
				GroupID: sub.GroupID, GroupInstanceID: sub.GroupInstanceID,
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("register submission missing: %#v", result.Domain.Submissions)
	}
	if addr.GroupID == "" || addr.GroupInstanceID == "" || addr.GroupInstanceID == ProvisionalGroupInstanceID {
		t.Fatalf("address must use minted group occurrence: %#v", addr)
	}

	stored, err := links.Get(t.Context(), "feragame", addr)
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
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         registerLotValue(),
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatalf("Commit must succeed when LinkStore fails: %v", err)
	}
	if len(result.Links) != 2 {
		t.Fatalf("response must still return links: %#v", result.Links)
	}
	loadID := firstLoadID(result.Domain)
	if result.Domain.Kind("load")[loadID]["client"] != "CO" {
		t.Fatalf("domain missing after link store failure: %#v", result.Domain.Kind("load"))
	}
	stored, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("load")[loadID]["client"] != "CO" {
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

func (e *errLinkStore) Put(context.Context, string, SubmissionAddress, []HttpLinkItem) error {
	return e.err
}
func (e *errLinkStore) Get(context.Context, string, SubmissionAddress) ([]HttpLinkItem, error) {
	return nil, e.err
}

type countingLinkStore struct {
	inner LinkStore
	puts  int
}

func (c *countingLinkStore) Put(ctx context.Context, org string, addr SubmissionAddress, links []HttpLinkItem) error {
	c.puts++
	return c.inner.Put(ctx, org, addr, links)
}
func (c *countingLinkStore) Get(ctx context.Context, org string, addr SubmissionAddress) ([]HttpLinkItem, error) {
	return c.inner.Get(ctx, org, addr)
}
