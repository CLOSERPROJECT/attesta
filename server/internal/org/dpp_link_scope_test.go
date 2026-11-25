package org

import (
	"errors"
	"testing"
)

func TestStreamsKeepSeparateDPPsAndLinks(t *testing.T) {
	passports := NewMemoryDPPStore()
	links := NewMemoryLinkStore()

	scopeA := Scope{Org: "feragame", Stream: "pv-recovery"}
	scopeB := Scope{Org: "feragame", Stream: "other"}
	addr := SubmissionAddress{JobID: "intake", JobInstanceID: "T1", TaskID: "register"}
	ts := "2026-11-19T12:00:00.000000000Z"

	if err := passports.Create(t.Context(), DPPRecord{
		Org: scopeA.Org, Stream: scopeA.Stream, SubjectID: "L-A",
		GTIN: "09506000134352", Lot: "LOT-A", Serial: "L-A",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/a",
	}); err != nil {
		t.Fatal(err)
	}
	if err := passports.Create(t.Context(), DPPRecord{
		Org: scopeB.Org, Stream: scopeB.Stream, SubjectID: "L-A",
		GTIN: "09506000134352", Lot: "LOT-B", Serial: "L-A",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/b",
	}); err != nil {
		t.Fatal(err)
	}
	gotA, err := passports.Get(t.Context(), scopeA, "L-A")
	if err != nil || gotA.Lot != "LOT-A" || gotA.Stream != scopeA.Stream {
		t.Fatalf("passport A = %#v err=%v", gotA, err)
	}
	gotB, err := passports.Get(t.Context(), scopeB, "L-A")
	if err != nil || gotB.Lot != "LOT-B" || gotB.Stream != scopeB.Stream {
		t.Fatalf("passport B = %#v err=%v", gotB, err)
	}
	if _, err := passports.Get(t.Context(), scopeA, "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing A = %v", err)
	}

	linkA := []HttpLinkItem{{Label: "A", URL: "blob:a", Action: HttpActionLink}}
	linkB := []HttpLinkItem{{Label: "B", URL: "blob:b", Action: HttpActionDownload, Filename: "b.csv"}}
	if err := links.Put(t.Context(), scopeA, addr, ts, linkA); err != nil {
		t.Fatal(err)
	}
	if err := links.Put(t.Context(), scopeB, addr, ts, linkB); err != nil {
		t.Fatal(err)
	}
	storedA, err := links.Get(t.Context(), scopeA, addr, ts)
	if err != nil || len(storedA) != 1 || storedA[0].URL != "blob:a" {
		t.Fatalf("links A = %#v err=%v", storedA, err)
	}
	storedB, err := links.Get(t.Context(), scopeB, addr, ts)
	if err != nil || len(storedB) != 1 || storedB[0].Filename != "b.csv" {
		t.Fatalf("links B = %#v err=%v", storedB, err)
	}
}

func TestCreateRejectsIdentityAlreadyIssuedInAnotherStream(t *testing.T) {
	store := NewMemoryDPPStore()
	rec := DPPRecord{
		Org: "feragame", Stream: "pv-recovery", SubjectID: "L1",
		GTIN: "09506000134352", Lot: "SAME", Serial: "S1",
		Snapshot: map[string]any{}, IssuedAt: "t", DigitalLink: "/x",
	}
	if err := store.Create(t.Context(), rec); err != nil {
		t.Fatal(err)
	}
	dup := rec
	dup.Stream = "other"
	dup.SubjectID = "L2"
	if err := store.Create(t.Context(), dup); !errors.Is(err, ErrDPPExists) {
		t.Fatalf("cross-stream identity = %v, want ErrDPPExists", err)
	}
	got, err := store.GetByIdentity(t.Context(), "09506000134352", "SAME", "S1")
	if err != nil || got.Stream != "pv-recovery" || got.SubjectID != "L1" {
		t.Fatalf("GetByIdentity = %#v err=%v", got, err)
	}
}
