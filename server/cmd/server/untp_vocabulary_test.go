package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func vocabularyTestServer(t *testing.T) *Server {
	t.Helper()
	return &Server{tmpl: testTemplates()}
}

// The namespace has to resolve through the real mux: a term IRI that 404s in
// production defeats the point of owning it.
func TestVocabularyRouteIsRegistered(t *testing.T) {
	mux := (&Server{tmpl: testTemplates()}).newMux()

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com/vocabulary/untp/substep", nil)
	req.Header.Set("Accept", "application/ld+json")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK || rr.Header().Get("Content-Type") != "application/ld+json" {
		t.Fatalf("status = %d content-type = %q", rr.Code, rr.Header().Get("Content-Type"))
	}
}

func TestHandleUNTPVocabularyServesJSONLDContext(t *testing.T) {
	server := vocabularyTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com/vocabulary/untp/", nil)
	req.Header.Set("Accept", "application/ld+json")
	rr := httptest.NewRecorder()
	server.handleUNTPVocabulary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/ld+json" {
		t.Fatalf("content-type = %q", got)
	}
	var document struct {
		Context map[string]interface{} `json:"@context"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &document); err != nil {
		t.Fatalf("decode context: %v", err)
	}
	if document.Context["attesta"] != "http://dl.example.com/vocabulary/untp/" {
		t.Fatalf("namespace = %#v", document.Context["attesta"])
	}
	substep, ok := document.Context["attesta:substep"].(map[string]interface{})
	if !ok || substep["@id"] != "attesta:substep" {
		t.Fatalf("substep term = %#v", document.Context["attesta:substep"])
	}
	scoped, ok := substep["@context"].(map[string]interface{})
	if !ok || scoped["digest"] != "attesta:digest" {
		t.Fatalf("scoped terms = %#v", substep["@context"])
	}
	// Without @json, a nested payload of undeclared keys expands to an empty
	// node and the entered values are silently lost.
	input, ok := scoped["input"].(map[string]interface{})
	if !ok || input["@id"] != "attesta:input" || input["@type"] != "@json" {
		t.Fatalf("input term = %#v, want a @json literal mapping", scoped["input"])
	}
}

// The context a DTE carries inline must be the one this namespace publishes,
// or consumers expanding against the hosted context get different terms.
func TestDTEInlineContextMatchesPublishedVocabulary(t *testing.T) {
	inline := untpDTEContext("https://dl.example.com")
	if len(inline) != 4 {
		t.Fatalf("context = %#v, want 4 entries", inline)
	}
	published := buildUNTPVocabularyContext("https://dl.example.com")["@context"]
	inlineJSON, _ := json.Marshal(inline[3])
	publishedJSON, _ := json.Marshal(published)
	if string(inlineJSON) != string(publishedJSON) {
		t.Fatalf("inline context %s != published context %s", inlineJSON, publishedJSON)
	}
}

func TestHandleUNTPVocabularyServesHTMLReference(t *testing.T) {
	server := vocabularyTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com/vocabulary/untp/", nil)
	rr := httptest.NewRecorder()
	server.handleUNTPVocabulary(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{
		"UNTP_VOCABULARY NS http://dl.example.com/vocabulary/untp/",
		"TERM input http://dl.example.com/vocabulary/untp/input",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in vocabulary page, got: %s", want, body)
		}
	}
}

func TestHandleUNTPVocabularyTermIRIs(t *testing.T) {
	server := vocabularyTestServer(t)

	for path, wantStatus := range map[string]int{
		"/vocabulary/untp/substep":    http.StatusOK,
		"/vocabulary/untp/input":      http.StatusOK,
		"/vocabulary/untp/attachment": http.StatusOK,
		"/vocabulary/untp/nonsense":   http.StatusNotFound,
	} {
		req := httptest.NewRequest(http.MethodGet, "http://dl.example.com"+path, nil)
		rr := httptest.NewRecorder()
		server.handleUNTPVocabulary(rr, req)
		if rr.Code != wantStatus {
			t.Fatalf("%s status = %d, want %d", path, rr.Code, wantStatus)
		}
	}
}

func TestHandleUNTPVocabularyRedirectsBarePath(t *testing.T) {
	server := vocabularyTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com/vocabulary/untp", nil)
	rr := httptest.NewRecorder()
	server.handleUNTPVocabulary(rr, req)

	if rr.Code != http.StatusMovedPermanently || rr.Header().Get("Location") != "/vocabulary/untp/" {
		t.Fatalf("status = %d location = %q", rr.Code, rr.Header().Get("Location"))
	}
}

// Term IRIs emitted in credentials must resolve, which is the whole point of
// owning the namespace.
func TestUNTPVocabularyCoversEmittedExtensionTerms(t *testing.T) {
	cfg := testRuntimeConfig()
	process := untpTestProcess()
	events := buildUNTPTraceabilityEvents("https://dl.example.com", cfg.Workflow, cfg, process, untpTestLink())
	if len(events) != 1 {
		t.Fatalf("events = %d", len(events))
	}
	encoded, err := json.Marshal(events[0])
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	raw, ok := decoded["attesta:substep"]
	if !ok {
		t.Fatalf("event has no attesta:substep extension: %s", encoded)
	}
	var substep map[string]json.RawMessage
	if err := json.Unmarshal(raw, &substep); err != nil {
		t.Fatalf("decode substep: %v", err)
	}
	for field := range substep {
		if !untpVocabularyTermExists(field) {
			t.Fatalf("emitted term %q is not published at /vocabulary/untp/", field)
		}
	}
	if !untpVocabularyTermExists("attachment") {
		t.Fatal("bizTransaction attachment type is not published")
	}
}

func TestHandleUNTPVocabularyRejectsNonGET(t *testing.T) {
	server := vocabularyTestServer(t)

	req := httptest.NewRequest(http.MethodPost, "http://dl.example.com/vocabulary/untp/", nil)
	rr := httptest.NewRecorder()
	server.handleUNTPVocabulary(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}
