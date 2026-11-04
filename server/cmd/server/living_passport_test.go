package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/CLOSERPROJECT/attesta/server/internal/living"
	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

func livingPassportTestRecord() living.PassportRecord {
	return living.PassportRecord{
		Org:         "synth",
		SubjectID:   "W-9",
		GTIN:        "09506000134352",
		Lot:         "LOT-9",
		Serial:      "W-9",
		ProductName: "Synth Widget",
		Description: "Locked synthetic widget",
		Snapshot: map[string]any{
			"widget": map[string]any{"color": "blue"},
			"subjectId": "W-9",
		},
		ConfigVersion: "v1",
		IssuedAt:      time.Date(2026, 11, 4, 10, 0, 0, 0, time.UTC).Format(time.RFC3339),
		DigitalLink:   untp.URL("09506000134352", "LOT-9", "W-9"),
	}
}

func livingPassportTestServer(t *testing.T) (*Server, living.PassportStore, living.PassportRecord) {
	t.Helper()
	store := living.NewMemoryPassportStore()
	rec := livingPassportTestRecord()
	if err := store.Create(t.Context(), rec); err != nil {
		t.Fatal(err)
	}
	server := &Server{
		store:           NewMemoryStore(),
		tmpl:            testTemplates(),
		livingPassports: store,
	}
	return server, store, rec
}

func TestHandleLivingPassportNotLocked404(t *testing.T) {
	server := &Server{
		store:           NewMemoryStore(),
		tmpl:            testTemplates(),
		livingPassports: living.NewMemoryPassportStore(),
	}
	link := untp.URL("09506000134352", "LOT-missing", "SER-missing")
	req := httptest.NewRequest(http.MethodGet, link+"?format=json", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleLivingPassportEvents410(t *testing.T) {
	server, _, rec := livingPassportTestServer(t)
	req := httptest.NewRequest(http.MethodGet, rec.DigitalLink+"/events", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)
	if rr.Code != http.StatusGone {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusGone)
	}
}

func TestHandleLivingPassportJSONHappyPath(t *testing.T) {
	server, _, rec := livingPassportTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com"+rec.DigitalLink+"?format=json", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	var cred untp.Credential
	if err := json.Unmarshal(rr.Body.Bytes(), &cred); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := strings.Join(cred.Type, ","); !strings.Contains(got, "DigitalProductPassport") {
		t.Fatalf("type = %q", got)
	}
	subject, ok := cred.CredentialSubject.(map[string]any)
	if !ok {
		t.Fatalf("subject type = %T", cred.CredentialSubject)
	}
	if subject["name"] != "Synth Widget" {
		t.Fatalf("name = %#v", subject["name"])
	}
	chars, _ := subject["characteristics"].(map[string]any)
	if chars["org"] != "synth" || chars["subjectId"] != "W-9" {
		t.Fatalf("characteristics = %#v", chars)
	}
	snap, _ := chars["snapshot"].(map[string]any)
	widget, _ := snap["widget"].(map[string]any)
	if widget["color"] != "blue" {
		t.Fatalf("snapshot = %#v", snap)
	}
	if cred.ID != "https://dl.example.com"+rec.DigitalLink {
		t.Fatalf("id = %q", cred.ID)
	}
}

func TestHandleLivingPassportHTMLHappyPath(t *testing.T) {
	server, _, rec := livingPassportTestServer(t)
	req := httptest.NewRequest(http.MethodGet, rec.DigitalLink, nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rr.Code, rr.Body.String())
	}
	body := rr.Body.String()
	if !strings.Contains(body, "LIVING_PASSPORT") || !strings.Contains(body, rec.GTIN) || !strings.Contains(body, "Synth Widget") {
		t.Fatalf("body = %q", body)
	}
}

func TestHandleLivingPassportLinkset(t *testing.T) {
	server, _, rec := livingPassportTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com"+rec.DigitalLink+"?linkType=linkset", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d", rr.Code)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/linkset+json" {
		t.Fatalf("content-type = %q", got)
	}
	var linkset untp.Linkset
	if err := json.Unmarshal(rr.Body.Bytes(), &linkset); err != nil {
		t.Fatalf("decode: %v", err)
	}
	anchor := "https://dl.example.com" + rec.DigitalLink
	if linkset.Linkset[0].Anchor != anchor {
		t.Fatalf("anchor = %q", linkset.Linkset[0].Anchor)
	}
}

func TestHandleDigitalLinkStreamStillServedWithoutLiving(t *testing.T) {
	server, _, process := untpTestServer(t)
	server.livingPassports = living.NewMemoryPassportStore()
	link := digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)
	req := httptest.NewRequest(http.MethodGet, link+"?format=json", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want stream DPP still served", rr.Code)
	}
}
