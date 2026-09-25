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

// The UNTP Product subject requires productCategory, producedAtFacility and
// countryOfProduction. A stream that configures none of them must still yield
// all three, or the published credential is schema-invalid.
func TestBuildUNTPDPPCredentialDerivesRequiredSubjectFields(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Workflow.CategorySlug = "supply-chain"
	cfg.Workflow.SubCategorySlug = "battery-cells"
	cfg.Workflow.Steps[0].OrganizationSlug = "org1"
	cfg.Organizations = []WorkflowOrganization{{Slug: "org1", Name: "Organization 1"}}
	cfg.DPP.ProductCategory = nil
	cfg.DPP.ProducedAtFacility = nil
	cfg.DPP.CountryOfProduction = nil

	credential := buildUNTPDPPCredential("https://dl.example.com", cfg, "workflow", untpTestProcess(), untpTestLink())
	subject := credential.CredentialSubject.(UNTPProduct)

	if len(subject.ProductCategory) != 1 {
		t.Fatalf("productCategory = %#v", subject.ProductCategory)
	}
	category := subject.ProductCategory[0]
	if category.Code != "battery-cells" || category.Name != "Battery cells" {
		t.Fatalf("derived category = %#v, want the stream taxonomy path", category)
	}
	if category.SchemeID != "https://dl.example.com/vocabulary/untp/categories" || category.SchemeName != untpCategorySchemeName {
		t.Fatalf("derived scheme = %#v, want the Attesta scheme, not UN CPC", category)
	}
	if subject.ProducedAtFacility == nil || subject.ProducedAtFacility.Name != "Organization 1" {
		t.Fatalf("derived facility = %#v, want the step organization", subject.ProducedAtFacility)
	}
	if subject.ProducedAtFacility.ID != "https://dl.example.com/organization/org1" {
		t.Fatalf("derived facility id = %q", subject.ProducedAtFacility.ID)
	}
	if subject.CountryOfProduction == nil || subject.CountryOfProduction.CountryCode != untpUnknownCountryCode {
		t.Fatalf("derived country = %#v, want an explicit unspecified code", subject.CountryOfProduction)
	}
}

func TestBuildUNTPDPPCredentialUsesDeploymentCountry(t *testing.T) {
	t.Setenv("DPP_COUNTRY_OF_PRODUCTION", "NL,Netherlands")
	cfg := testRuntimeConfig()
	cfg.DPP.CountryOfProduction = nil

	credential := buildUNTPDPPCredential("https://dl.example.com", cfg, "workflow", untpTestProcess(), untpTestLink())
	country := credential.CredentialSubject.(UNTPProduct).CountryOfProduction

	if country == nil || country.CountryCode != "NL" || country.CountryName != "Netherlands" {
		t.Fatalf("country = %#v, want the deployment default", country)
	}
}

func TestBuildUNTPDPPCredentialUncategorizedStreamCategory(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Workflow.CategorySlug = ""
	cfg.Workflow.SubCategorySlug = ""
	cfg.DPP.ProductCategory = nil

	credential := buildUNTPDPPCredential("https://dl.example.com", cfg, "workflow", untpTestProcess(), untpTestLink())
	category := credential.CredentialSubject.(UNTPProduct).ProductCategory

	if len(category) != 1 || category[0].Code != untpUnclassifiedCode || category[0].Name != untpUnclassifiedName {
		t.Fatalf("category = %#v, want an explicit unclassified value", category)
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
	if event.Type != "ObjectEvent" || event.Action != "OBSERVE" || event.BizStep != "inspecting" {
		t.Fatalf("epcis event dispatch = %#v", event)
	}
	wantID := "https://dl.example.com" + link + "/events#1.1"
	if event.EventID != wantID {
		t.Fatalf("eventID = %q, want %q", event.EventID, wantID)
	}
	if event.EventTime != "2026-03-05T14:30:00Z" || event.EventTimeZoneOffset != "+00:00" {
		t.Fatalf("event time = %q %q", event.EventTime, event.EventTimeZoneOffset)
	}
	if len(event.EPCList) != 1 || event.EPCList[0] != "https://dl.example.com"+link {
		t.Fatalf("epcList = %#v", event.EPCList)
	}
	if event.ReadPoint == nil || event.ReadPoint.ID != "https://dl.example.com/facility/refinery" {
		t.Fatalf("readPoint = %#v", event.ReadPoint)
	}
	if event.BizLocation == nil || event.BizLocation.ID != "https://dl.example.com/facility/refinery" {
		t.Fatalf("bizLocation = %#v", event.BizLocation)
	}
	if event.Substep.SubstepID != "1.1" || event.Substep.Title != "A" || event.Substep.StepID != "1" {
		t.Fatalf("substep record = %#v", event.Substep)
	}
	if event.Substep.Role != "dep1" || event.Substep.UNTPRole != "operator" || event.Substep.CompletedBy != "u1" {
		t.Fatalf("substep actor = %#v", event.Substep)
	}
	input, ok := event.Substep.Input.(map[string]interface{})
	if !ok || input["value"] != float64(1) {
		t.Fatalf("substep input = %#v, want entered value preserved", event.Substep.Input)
	}
	if _, present := input["attachment"]; present {
		t.Fatalf("substep input = %#v, want attachment metadata excluded", input)
	}
	if event.Substep.Digest != "sha256:"+digestPayload(step.Data) {
		t.Fatalf("substep digest = %q", event.Substep.Digest)
	}
	if len(event.BizTransactionList) != 1 {
		t.Fatalf("bizTransactionList = %#v", event.BizTransactionList)
	}
	txn := event.BizTransactionList[0]
	if txn.BizTransaction != "https://dl.example.com"+link+"/attachment/65f2a79b8e7f7d8f3c7c99aa/file" {
		t.Fatalf("attachment transaction = %#v", txn)
	}
	if txn.Type != "https://dl.example.com/vocabulary/untp/attachment" {
		t.Fatalf("attachment transaction type = %q", txn.Type)
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

	if len(events) != 1 || events[0].Substep.UNTPRole != "operator" {
		t.Fatalf("events = %#v, want substep role fallback mapped via untpRole", events)
	}
}

func TestBuildUNTPTraceabilityEventsOmitsUNTPRoleForUnmappedRole(t *testing.T) {
	cfg := testRuntimeConfig()
	cfg.Roles = append(cfg.Roles, WorkflowRole{OrgSlug: "org1", Slug: "dep2", Name: "Department 2"})
	process := untpTestProcess()
	step := process.Progress["1.1"]
	step.DoneBy = &Actor{ID: "u2", Role: "dep2"}
	process.Progress["1.1"] = step
	link := untpTestLink()

	events := buildUNTPTraceabilityEvents("https://dl.example.com", cfg.Workflow, cfg, process, link)

	if len(events) != 1 || events[0].Substep.Role != "dep2" || events[0].Substep.UNTPRole != "" {
		t.Fatalf("events = %#v, want untpRole omitted for role without mapping", events)
	}
}

// EPCIS requires eventTime on every event, and substeps completed before
// DoneAt was recorded have none.
func TestBuildUNTPTraceabilityEventsFallsBackToPassportTimeWithoutDoneAt(t *testing.T) {
	cfg := testRuntimeConfig()
	process := untpTestProcess()
	step := process.Progress["1.1"]
	step.DoneAt = nil
	process.Progress["1.1"] = step
	link := untpTestLink()

	events := buildUNTPTraceabilityEvents("https://dl.example.com", cfg.Workflow, cfg, process, link)

	if len(events) != 1 || events[0].EventTime != "2026-03-06T08:00:00Z" {
		t.Fatalf("events = %#v, want passport generation time as eventTime", events)
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
		Context           []interface{}     `json:"@context"`
		ID                string            `json:"id"`
		ValidFrom         string            `json:"validFrom"`
		CredentialSubject []UNTPObjectEvent `json:"credentialSubject"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode events credential: %v", err)
	}
	if strings.Join(payload.Type, ",") != "DigitalTraceabilityEvent,VerifiableCredential" {
		t.Fatalf("type = %#v", payload.Type)
	}
	if len(payload.Context) != 4 || payload.Context[2] != untpEPCISContext {
		t.Fatalf("context = %#v, want VCDM + UNTP + EPCIS + extension namespace", payload.Context)
	}
	namespace, ok := payload.Context[3].(map[string]interface{})
	if !ok || namespace["attesta"] != "http://dl.example.com/vocabulary/untp/" {
		t.Fatalf("inline context = %#v", payload.Context[3])
	}
	if payload.ID != "http://dl.example.com"+link+"/events" {
		t.Fatalf("credential id = %q", payload.ID)
	}
	if len(payload.CredentialSubject) != 1 {
		t.Fatalf("events = %d, want 1", len(payload.CredentialSubject))
	}
	event := payload.CredentialSubject[0]
	if event.EventTime != "2026-03-05T14:30:00Z" || event.Action != "OBSERVE" {
		t.Fatalf("event = %#v", event)
	}
	if len(event.EPCList) != 1 || event.EPCList[0] != "http://dl.example.com"+link {
		t.Fatalf("epcList = %#v", event.EPCList)
	}
	input, ok := event.Substep.Input.(map[string]interface{})
	if !ok || input["value"] != float64(1) {
		t.Fatalf("substep input = %#v, want served entered values", event.Substep.Input)
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
