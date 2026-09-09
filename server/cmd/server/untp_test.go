package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func untpTestProcess() *Process {
	doneAt := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	return &Process{
		ID:          primitive.NewObjectID(),
		WorkflowKey: "workflow",
		Status:      "done",
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", DoneAt: &doneAt, DoneBy: &Actor{ID: "u1", Role: "dep1"}, Data: map[string]interface{}{"value": float64(1)}},
			"1.2": {State: "pending"},
		},
		DPP: &ProcessDPP{
			GTIN:        "09506000134352",
			Lot:         "LOT-001",
			Serial:      "SERIAL-001",
			GeneratedAt: time.Date(2026, 3, 6, 8, 0, 0, 0, time.UTC),
		},
	}
}

func untpTestLink() string {
	return digitalLinkURL("09506000134352", "LOT-001", "SERIAL-001")
}

func TestBuildUNTPDPPCredential(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.DPP = DPPConfig{
		ProductName:        "Recycled Gallium Batch",
		ProductDescription: "Gallium intake and refinement",
		OwnerName:          "Attesta Demo Operator",
		ProductCategory: &DPPProductCategory{
			Code:       "41601",
			Name:       "Gallium, unwrought",
			SchemeID:   untpCPCSchemeID,
			SchemeName: untpCPCSchemeName,
		},
		ProducedAtFacility: &DPPFacility{
			ID:           "https://dl.example.com/facility/refinery",
			Name:         "Demo Refinery",
			RegisteredID: "ref-001",
		},
		CountryOfProduction: &DPPCountry{CountryCode: "NL", CountryName: "Netherlands"},
	}
	process := untpTestProcess()
	link := untpTestLink()

	credential := buildUNTPDPPCredential("https://dl.example.com", cfg, "workflow", process, link)

	if got, want := strings.Join(credential.Type, ","), "DigitalProductPassport,VerifiableCredential"; got != want {
		t.Fatalf("type = %q, want %q", got, want)
	}
	if len(credential.Context) != 2 || credential.Context[0] != untpVCCredentialsContext || credential.Context[1] != untpVocabularyContext {
		t.Fatalf("context = %#v", credential.Context)
	}
	wantURL := "https://dl.example.com" + link
	if credential.ID != wantURL {
		t.Fatalf("credential id = %q, want %q", credential.ID, wantURL)
	}
	if credential.Issuer.ID != "https://dl.example.com" || credential.Issuer.Name != "Attesta Demo Operator" {
		t.Fatalf("issuer = %#v", credential.Issuer)
	}
	if credential.Name != "Digital Product Passport — Recycled Gallium Batch" {
		t.Fatalf("name = %q", credential.Name)
	}
	if credential.ValidFrom != "2026-03-06T08:00:00Z" {
		t.Fatalf("validFrom = %q", credential.ValidFrom)
	}
	subject, ok := credential.CredentialSubject.(UNTPProduct)
	if !ok {
		t.Fatalf("credentialSubject type = %T", credential.CredentialSubject)
	}
	if subject.ID != wantURL || subject.Name != "Recycled Gallium Batch" || subject.Description != "Gallium intake and refinement" {
		t.Fatalf("subject identification = %#v", subject)
	}
	if subject.BatchNumber != "LOT-001" || subject.ItemNumber != "SERIAL-001" || subject.IDGranularity != "item" {
		t.Fatalf("subject batch/serial/granularity = %#v", subject)
	}
	if len(subject.ProductCategory) != 1 || subject.ProductCategory[0].Code != "41601" || subject.ProductCategory[0].SchemeID != untpCPCSchemeID {
		t.Fatalf("productCategory = %#v", subject.ProductCategory)
	}
	if subject.ProducedAtFacility == nil || subject.ProducedAtFacility.ID != "https://dl.example.com/facility/refinery" || subject.ProducedAtFacility.Name != "Demo Refinery" {
		t.Fatalf("producedAtFacility = %#v", subject.ProducedAtFacility)
	}
	if subject.CountryOfProduction == nil || subject.CountryOfProduction.CountryCode != "NL" {
		t.Fatalf("countryOfProduction = %#v", subject.CountryOfProduction)
	}
	if subject.IDScheme.ID != gs1DigitalLinkSchemeID || subject.IDScheme.Name != "GS1 Digital Link" {
		t.Fatalf("idScheme = %#v", subject.IDScheme)
	}
	if len(subject.RelatedDocument) != 1 || subject.RelatedDocument[0].LinkURL != wantURL+"/events" || subject.RelatedDocument[0].LinkType != untpLinkTypeDTE {
		t.Fatalf("relatedDocument = %#v", subject.RelatedDocument)
	}
	if subject.Characteristics["workflow"] != "workflow" || subject.Characteristics["processId"] != process.ID.Hex() {
		t.Fatalf("characteristics = %#v", subject.Characteristics)
	}
	if got := subject.Characteristics["gs1Element"]; got != "(01)09506000134352(10)LOT-001(21)SERIAL-001" {
		t.Fatalf("gs1Element = %v", got)
	}
}

func TestBuildUNTPDPPCredentialFallbacks(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Workflow.Description = "Demo workflow description"
	link := untpTestLink()

	credential := buildUNTPDPPCredential("", cfg, "workflow", nil, link)

	subject, ok := credential.CredentialSubject.(UNTPProduct)
	if !ok {
		t.Fatalf("credentialSubject type = %T", credential.CredentialSubject)
	}
	if subject.Name != "Demo workflow" || subject.Description != "Demo workflow description" {
		t.Fatalf("subject fallbacks = %#v", subject)
	}
	if credential.Issuer.Name != "Demo workflow" {
		t.Fatalf("issuer fallback = %#v", credential.Issuer)
	}
	if credential.ValidFrom != "" {
		t.Fatalf("validFrom = %q, want empty for nil process", credential.ValidFrom)
	}
	if subject.BatchNumber != "" || subject.ItemNumber != "" {
		t.Fatalf("batch/serial = %q/%q, want empty", subject.BatchNumber, subject.ItemNumber)
	}
	if subject.ID != link {
		t.Fatalf("subject id = %q, want relative link when no base URL", subject.ID)
	}
}

func TestBuildUNTPTraceabilityEvents(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Roles = []WorkflowRole{{OrgSlug: "org1", Slug: "dep1", Name: "Department 1", UNTpRole: "operator"}}
	cfg.DPP.ProducedAtFacility = &DPPFacility{ID: "https://dl.example.com/facility/refinery", Name: "Demo Refinery"}

	process := untpTestProcess()
	step := process.Progress["1.1"]
	step.Data = map[string]interface{}{
		"value": float64(1),
		"attachment": map[string]interface{}{
			"attachmentId": "65f2a79b8e7f7d8f3c7c99aa",
			"filename":     "cert.pdf",
			"contentType":  "application/pdf",
		},
	}
	process.Progress["1.1"] = step
	link := untpTestLink()

	events := buildUNTPTraceabilityEvents("https://dl.example.com", cfg.Workflow, cfg, process, link)

	if len(events) != 1 {
		t.Fatalf("events = %d, want 1 (only done substeps)", len(events))
	}
	event := events[0]
	if strings.Join(event.Type, ",") != "ModifyEvent,LifecycleEvent" {
		t.Fatalf("event type = %#v", event.Type)
	}
	wantID := "https://dl.example.com" + link + "/events#1.1"
	if event.ID != wantID {
		t.Fatalf("event id = %q, want %q", event.ID, wantID)
	}
	if event.Name != "A" {
		t.Fatalf("event name = %q", event.Name)
	}
	if event.EventDate != "2026-03-05T14:30:00Z" {
		t.Fatalf("eventDate = %q", event.EventDate)
	}
	if event.ActivityType.Code != "1.1" || event.ActivityType.Name != "A" || event.ActivityType.SchemeID != "https://dl.example.com" || event.ActivityType.SchemeName != "Demo workflow" {
		t.Fatalf("activityType = %#v", event.ActivityType)
	}
	if len(event.ModifiedProduct) != 1 || event.ModifiedProduct[0].Disposition != "active" {
		t.Fatalf("modifiedProduct = %#v", event.ModifiedProduct)
	}
	product := event.ModifiedProduct[0].Product
	if product.ID != "https://dl.example.com"+link || product.BatchNumber != "LOT-001" || product.ItemNumber != "SERIAL-001" || product.IDGranularity != "item" {
		t.Fatalf("event product = %#v", product)
	}

	if event.ModifiedAtFacility == nil || event.ModifiedAtFacility.ID != "https://dl.example.com/facility/refinery" || event.ModifiedAtFacility.Name != "Demo Refinery" {
		t.Fatalf("modifiedAtFacility = %#v", event.ModifiedAtFacility)
	}
	if len(event.RelatedParty) != 1 || event.RelatedParty[0].Role != "operator" || event.RelatedParty[0].Party.Name != "u1" {
		t.Fatalf("relatedParty = %#v", event.RelatedParty)
	}
	if event.RelatedParty[0].Party.ID != "https://dl.example.com/#actor=u1" {
		t.Fatalf("party id = %q", event.RelatedParty[0].Party.ID)
	}
	if len(event.RelatedDocument) != 1 || event.RelatedDocument[0].LinkURL != "https://dl.example.com"+link+"/attachment/65f2a79b8e7f7d8f3c7c99aa/file" {
		t.Fatalf("relatedDocument = %#v", event.RelatedDocument)
	}
	if event.RelatedDocument[0].LinkName != "cert.pdf" || event.RelatedDocument[0].MediaType != "application/pdf" {
		t.Fatalf("relatedDocument meta = %#v", event.RelatedDocument[0])
	}
}

func TestBuildUNTPTraceabilityEventsRoleFallback(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Roles = []WorkflowRole{{OrgSlug: "org1", Slug: "dep1", Name: "Department 1", UNTpRole: "operator"}}
	process := untpTestProcess()
	step := process.Progress["1.1"]
	step.DoneBy = &Actor{ID: "u1"}
	process.Progress["1.1"] = step
	link := untpTestLink()

	events := buildUNTPTraceabilityEvents("", cfg.Workflow, cfg, process, link)

	if len(events) != 1 || events[0].RelatedParty[0].Role != "operator" {
		t.Fatalf("events = %#v, want DoneBy role fallback mapped via untpRole", events)
	}
}

func TestBuildUNTPTraceabilityEventsOmitsRelatedPartyForUnmappedRole(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Roles = append(cfg.Roles, WorkflowRole{OrgSlug: "org1", Slug: "dep2", Name: "Department 2"})
	process := untpTestProcess()
	step := process.Progress["1.1"]
	step.DoneBy = &Actor{ID: "u2", Role: "dep2"}
	process.Progress["1.1"] = step
	link := untpTestLink()

	events := buildUNTPTraceabilityEvents("https://dl.example.com", cfg.Workflow, cfg, process, link)

	if len(events) != 1 || len(events[0].RelatedParty) != 0 {
		t.Fatalf("events = %#v, want relatedParty omitted for role without untpRole", events)
	}
}

func TestBuildUNTPLinksetAndFilter(t *testing.T) {
	link := untpTestLink()
	linkset := buildUNTPLinkset("https://dl.example.com", link)

	if len(linkset.Linkset) != 1 {
		t.Fatalf("linkset contexts = %d, want 1", len(linkset.Linkset))
	}
	context := linkset.Linkset[0]
	anchor := "https://dl.example.com" + link
	if context.Anchor != anchor {
		t.Fatalf("anchor = %q, want %q", context.Anchor, anchor)
	}
	if len(context.DPP) != 1 || context.DPP[0].Href != anchor+"?format=json" || context.DPP[0].Type != "application/json" {
		t.Fatalf("dpp relation = %#v", context.DPP)
	}
	if len(context.DTE) != 1 || context.DTE[0].Href != anchor+"/events" {
		t.Fatalf("dte relation = %#v", context.DTE)
	}
	if len(context.PIP) != 1 || context.PIP[0].Href != anchor || context.PIP[0].Type != "text/html" {
		t.Fatalf("pip relation = %#v", context.PIP)
	}

	filtered, ok := filterUNTPLinkset(linkset, "dpp")
	if !ok {
		t.Fatal("expected dpp filter to succeed")
	}
	if filtered.Linkset[0].DPP == nil || filtered.Linkset[0].DTE != nil || filtered.Linkset[0].PIP != nil {
		t.Fatalf("filtered linkset = %#v, want only dpp", filtered.Linkset[0])
	}
	if filtered.Linkset[0].Anchor != anchor {
		t.Fatalf("filtered anchor = %q", filtered.Linkset[0].Anchor)
	}

	if _, ok := filterUNTPLinkset(linkset, "unknown"); ok {
		t.Fatal("expected unknown relation filter to fail")
	}
	if _, ok := filterUNTPLinkset(UNTPLinkset{}, "dpp"); ok {
		t.Fatal("expected empty linkset filter to fail")
	}
}

func TestUntpLinksetScope(t *testing.T) {
	cases := []struct {
		linkType  string
		wantScope string
		wantOK    bool
	}{
		{"linkset", "all", true},
		{" ALL ", "all", true},
		{"dpp", "dpp", true},
		{"DTE", "dte", true},
		{"pip", "pip", true},
		{"", "", false},
		{"bogus", "", false},
	}
	for _, tc := range cases {
		scope, ok := untpLinksetScope(tc.linkType)
		if scope != tc.wantScope || ok != tc.wantOK {
			t.Fatalf("untpLinksetScope(%q) = (%q, %v), want (%q, %v)", tc.linkType, scope, ok, tc.wantScope, tc.wantOK)
		}
	}
}
func untpTestServer(t *testing.T) (*Server, *MemoryStore, Process) {
	t.Helper()
	tempDir := t.TempDir()
	writeWorkflowConfig(t, tempDir+"/workflow.yaml", "Demo workflow", "string")
	store := NewMemoryStore()
	process := seedDPPProcess(store)
	server := &Server{
		store:     store,
		tmpl:      testTemplates(),
		configDir: tempDir,
	}
	return server, store, process
}

func TestHandleDigitalLinkLinksetParam(t *testing.T) {
	server, _, process := untpTestServer(t)
	link := digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com"+link+"?linkType=linkset", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/linkset+json" {
		t.Fatalf("content-type = %q", got)
	}
	var linkset UNTPLinkset
	if err := json.Unmarshal(rr.Body.Bytes(), &linkset); err != nil {
		t.Fatalf("decode linkset: %v", err)
	}
	anchor := "https://dl.example.com" + link
	if linkset.Linkset[0].Anchor != anchor {
		t.Fatalf("anchor = %q, want %q", linkset.Linkset[0].Anchor, anchor)
	}
	if linkset.Linkset[0].DPP[0].Href != anchor+"?format=json" {
		t.Fatalf("dpp href = %q", linkset.Linkset[0].DPP[0].Href)
	}
	if linkset.Linkset[0].DTE[0].Href != anchor+"/events" {
		t.Fatalf("dte href = %q", linkset.Linkset[0].DTE[0].Href)
	}
}

func TestHandleDigitalLinkLinksetFilteredByLinkType(t *testing.T) {
	server, _, process := untpTestServer(t)
	link := digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com"+link+"?linkType=dte", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	var linkset UNTPLinkset
	if err := json.Unmarshal(rr.Body.Bytes(), &linkset); err != nil {
		t.Fatalf("decode linkset: %v", err)
	}
	context := linkset.Linkset[0]
	if context.DTE == nil || context.DPP != nil || context.PIP != nil {
		t.Fatalf("filtered context = %#v, want only dte", context)
	}
}

func TestHandleDigitalLinkLinksetUnknownLinkType(t *testing.T) {
	server, _, process := untpTestServer(t)
	link := digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)

	req := httptest.NewRequest(http.MethodGet, link+"?linkType=bogus", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestHandleDigitalLinkLinksetAcceptHeader(t *testing.T) {
	server, _, process := untpTestServer(t)
	link := digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com"+link, nil)
	req.Header.Set("Accept", "application/linkset+json")
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "application/linkset+json" {
		t.Fatalf("content-type = %q", got)
	}
	var linkset UNTPLinkset
	if err := json.Unmarshal(rr.Body.Bytes(), &linkset); err != nil {
		t.Fatalf("decode linkset: %v", err)
	}
	if len(linkset.Linkset) != 1 || linkset.Linkset[0].DPP == nil || linkset.Linkset[0].DTE == nil || linkset.Linkset[0].PIP == nil {
		t.Fatalf("linkset = %#v, want all relations", linkset)
	}
}

func TestHandleDigitalLinkEvents(t *testing.T) {
	server, _, process := untpTestServer(t)
	link := digitalLinkURL(process.DPP.GTIN, process.DPP.Lot, process.DPP.Serial)

	req := httptest.NewRequest(http.MethodGet, "http://dl.example.com"+link+"/events", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); !strings.Contains(got, "application/json") {
		t.Fatalf("content-type = %q", got)
	}
	var payload struct {
		Type              []string          `json:"type"`
		ID                string            `json:"id"`
		ValidFrom         string            `json:"validFrom"`
		CredentialSubject []UNTPModifyEvent `json:"credentialSubject"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode events credential: %v", err)
	}
	if strings.Join(payload.Type, ",") != "DigitalTraceabilityEvent,VerifiableCredential" {
		t.Fatalf("type = %#v", payload.Type)
	}
	if payload.ID != "http://dl.example.com"+link+"/events" {
		t.Fatalf("credential id = %q", payload.ID)
	}
	if len(payload.CredentialSubject) != 1 {
		t.Fatalf("events = %d, want 1", len(payload.CredentialSubject))
	}
	event := payload.CredentialSubject[0]
	if event.EventDate != "2026-03-05T14:30:00Z" {
		t.Fatalf("eventDate = %q", event.EventDate)
	}
	if len(event.RelatedParty) != 1 || event.RelatedParty[0].Role != "operator" || event.RelatedParty[0].Party.Name != "u1" {
		t.Fatalf("relatedParty = %#v", event.RelatedParty)
	}
	if len(event.ModifiedProduct) != 1 || event.ModifiedProduct[0].Product.BatchNumber != "LOT-001" {
		t.Fatalf("modifiedProduct = %#v", event.ModifiedProduct)
	}
}

func TestHandleDigitalLinkEventsUnknownProcess(t *testing.T) {
	server, _, _ := untpTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/01/09506000134352/10/NOPE/21/NOPE/events", nil)
	rr := httptest.NewRecorder()
	server.handleDigitalLinkDPP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestParseDigitalLinkEventsPath(t *testing.T) {
	gtin, lot, serial, ok, err := parseDigitalLinkEventsPath("/01/09506000134352/10/LOT-1/21/SER-1/events")
	if !ok || err != nil || gtin != "09506000134352" || lot != "LOT-1" || serial != "SER-1" {
		t.Fatalf("parseDigitalLinkEventsPath = (%q, %q, %q, %v, %v)", gtin, lot, serial, ok, err)
	}
	if _, _, _, ok, err := parseDigitalLinkEventsPath("/01/09506000134352/10/LOT-1/21/SER-1"); ok || err != nil {
		t.Fatalf("expected no match for plain digital link, got ok=%v err=%v", ok, err)
	}
	if _, _, _, ok, err := parseDigitalLinkEventsPath("/01/09506000134352/10/LOT-1/21/SER-1/nope"); ok || err != nil {
		t.Fatalf("expected no match for unknown suffix, got ok=%v err=%v", ok, err)
	}
	if _, _, _, ok, err := parseDigitalLinkEventsPath("/01/bad/10/LOT-1/21/SER-1/events"); !ok || err == nil {
		t.Fatalf("expected ok=true with error for invalid parts, got ok=%v err=%v", ok, err)
	}
}
