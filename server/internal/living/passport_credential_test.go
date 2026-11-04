package living

import (
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

func TestBuildPassportDPPCredentialFromRecord(t *testing.T) {
	rec := PassportRecord{
		Org:         "synth",
		SubjectID:   "W-1",
		GTIN:        "09506000134352",
		Lot:         "LOT-1",
		Serial:      "W-1",
		ProductName: "Synth Widget",
		Description: "A synthetic widget",
		Snapshot: map[string]any{
			"color": "blue",
			"n":     float64(2),
		},
		ConfigVersion: "v1",
		IssuedAt:      time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		DigitalLink:   untp.URL("09506000134352", "LOT-1", "W-1"),
	}
	cred := BuildPassportDPPCredential("https://dl.example.com", rec)
	subject, ok := cred.CredentialSubject.(untp.Product)
	if !ok {
		t.Fatalf("subject type = %T", cred.CredentialSubject)
	}
	if subject.Name != "Synth Widget" || subject.BatchNumber != "LOT-1" || subject.ItemNumber != "W-1" {
		t.Fatalf("subject = %#v", subject)
	}
	if subject.Characteristics["org"] != "synth" || subject.Characteristics["subjectId"] != "W-1" {
		t.Fatalf("characteristics = %#v", subject.Characteristics)
	}
	snap, _ := subject.Characteristics["snapshot"].(map[string]any)
	if snap["color"] != "blue" {
		t.Fatalf("snapshot = %#v", snap)
	}
	if cred.ValidFrom != "2026-11-04T10:00:00Z" {
		t.Fatalf("validFrom = %q", cred.ValidFrom)
	}
}

func TestMemoryPassportStoreGetByIdentity(t *testing.T) {
	store := NewMemoryPassportStore()
	rec := PassportRecord{
		Org: "synth", SubjectID: "W1", GTIN: "09506000134352",
		Lot: "L1", Serial: "W1", Snapshot: map[string]any{"n": 1}, IssuedAt: "t",
		DigitalLink: untp.URL("09506000134352", "L1", "W1"),
	}
	if err := store.Create(t.Context(), rec); err != nil {
		t.Fatal(err)
	}
	got, err := store.GetByIdentity(t.Context(), "09506000134352", "L1", "W1")
	if err != nil {
		t.Fatal(err)
	}
	if got.SubjectID != "W1" || got.Snapshot["n"] != 1 {
		t.Fatalf("got = %#v", got)
	}
	if _, err := store.GetByIdentity(t.Context(), "09506000134352", "L1", "missing"); err != ErrNotFound {
		t.Fatalf("missing = %v", err)
	}
}
