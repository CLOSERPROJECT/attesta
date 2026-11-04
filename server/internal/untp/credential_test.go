package untp

import (
	"strings"
	"testing"
	"time"
)

func TestBuildDPPCredential(t *testing.T) {
	validFrom := time.Date(2026, 3, 6, 8, 0, 0, 0, time.UTC)
	credential := BuildDPPCredential(DPPInput{
		BaseURL: "https://dl.example.com",
		Link:    "/01/09506000134352/10/LOT-001/21/SERIAL-001",
		Product: ProductIdentity{
			Name:        "Recycled Gallium Batch",
			Description: "Gallium intake and refinement",
			OwnerName:   "Attesta Demo Operator",
			BatchNumber: "LOT-001",
			ItemNumber:  "SERIAL-001",
			GTIN:        "09506000134352",
			ValidFrom:   validFrom,
		},
		Characteristics: map[string]interface{}{
			"workflow":     "workflow",
			"workflowName": "Demo",
			"processId":    "abc",
		},
	})

	if got, want := strings.Join(credential.Type, ","), "DigitalProductPassport,VerifiableCredential"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	subject, ok := credential.CredentialSubject.(Product)
	if !ok {
		t.Fatalf("credentialSubject type = %T", credential.CredentialSubject)
	}
	if subject.BatchNumber != "LOT-001" || subject.ItemNumber != "SERIAL-001" {
		t.Fatalf("batch/serial = %#v", subject)
	}
	if got := subject.Characteristics["gs1Element"]; got != "(01)09506000134352(10)LOT-001(21)SERIAL-001" {
		t.Fatalf("gs1Element = %v", got)
	}
	if credential.ValidFrom != "2026-03-06T08:00:00Z" {
		t.Fatalf("validFrom = %q", credential.ValidFrom)
	}
}

func TestBuildLinksetAndFilter(t *testing.T) {
	link := "/01/09506000134352/10/LOT-001/21/SERIAL-001"
	linkset := BuildLinkset("https://dl.example.com", link)
	if len(linkset.Linkset) != 1 {
		t.Fatalf("contexts = %d", len(linkset.Linkset))
	}
	filtered, ok := FilterLinkset(linkset, "dpp")
	if !ok || filtered.Linkset[0].DPP == nil || filtered.Linkset[0].DTE != nil {
		t.Fatalf("filtered = %#v", filtered)
	}
	if scope, ok := LinksetScope("linkset"); !ok || scope != "all" {
		t.Fatalf("LinksetScope(linkset) = (%q, %v)", scope, ok)
	}
}

func TestBuildModifyEventsAndDTE(t *testing.T) {
	eventDate := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	events := BuildModifyEvents("https://dl.example.com", "/01/g/10/l/21/s", ProductIdentity{
		Name:        "Product",
		BatchNumber: "LOT-001",
		ItemNumber:  "SERIAL-001",
	}, "Demo workflow", []EventInput{{
		ID:        "1.1",
		Title:     "A",
		EventDate: eventDate,
		ActorID:   "u1",
		ActorRole: "dep1",
		Attachments: []AttachmentRef{{
			ID:          "att-1",
			Filename:    "cert.pdf",
			ContentType: "application/pdf",
		}},
	}})
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}
	if events[0].EventDate != "2026-03-05T14:30:00Z" {
		t.Fatalf("eventDate = %q", events[0].EventDate)
	}
	if len(events[0].RelatedDocument) != 1 {
		t.Fatalf("relatedDocument = %#v", events[0].RelatedDocument)
	}

	credential := BuildDTECredential(DTEInput{
		BaseURL: "https://dl.example.com",
		Link:    "/01/g/10/l/21/s",
		Product: ProductIdentity{Name: "Product", OwnerName: "Owner", ValidFrom: eventDate},
		Scheme:  "Demo",
		Events:  []EventInput{{ID: "1.1", Title: "A", EventDate: eventDate}},
	})
	if got, want := strings.Join(credential.Type, ","), "DigitalTraceabilityEvent,VerifiableCredential"; got != want {
		t.Fatalf("type = %q", got)
	}
}
