package org

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDemoCommitHelpersSeedLotProcessingReady(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	if err := seedLotProcessingReadyDomain(t.Context(), domains, configs, FeragameScope()); err != nil {
		t.Fatal(err)
	}
	domain, err := domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	loadID := firstLoadID(domain)
	if loadID == "" {
		t.Fatal("expected lot after demo seed")
	}
	var sawLotProc bool
	for _, inst := range domain.JobInstances {
		if inst.JobID == "lot_processing" && inst.ID == loadID {
			sawLotProc = true
		}
		if inst.JobID == "workingDay" || inst.JobID == "line" {
			t.Fatalf("obsolete Job instance present: %#v", inst)
		}
	}
	if !sawLotProc {
		t.Fatalf("lot_processing instance missing: %#v", domain.JobInstances)
	}
}

func TestDecodeScanEntityIDSubstringAndBadURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"prefix/entityId=NO-1.1&trail=1", "NO-1.1"},
		{"entityId=a%2Fb#frag", "a/b"},
		{"entityId=%zz", "%zz"},
		{"?foo=1", ""},
		{"http://x/y?z=1", ""},
	}
	for _, tc := range cases {
		if got := DecodeScanEntityID(tc.raw); got != tc.want {
			t.Fatalf("DecodeScanEntityID(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestHandlerTaskFormRecapQrErrorEdges(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	gateOK := func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}, true
	}
	h := &API{DomainLoader: loader, TaskCommitter: c, Gate: gateOK}

	for _, path := range []string{"/api/org/task-form", "/api/org/task-recap"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s method = %d", path, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	(&API{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("task-form unconfigured = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	(&API{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("task-recap unconfigured = %d", rec.Code)
	}

	emptyOrg := &API{
		DomainLoader: loader, TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada"}, true
		},
	}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("task-form empty org = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("task-recap empty org = %d", rec.Code)
	}

	denyGate := &API{
		DomainLoader: loader, TaskCommitter: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			http.Error(w, "no", http.StatusUnauthorized)
			return OperatorIdentity{}, false
		},
	}
	rec = httptest.NewRecorder()
	denyGate.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("task-form gate = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	denyGate.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("task-recap gate = %d", rec.Code)
	}

	missingCfg := &API{
		DomainLoader: &DomainLoader{Domains: NewMemoryDomainStore(), Configs: NewMemoryConfigStore()},
		Gate:         gateOK,
	}
	rec = httptest.NewRecorder()
	missingCfg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("task-form missing config = %d", rec.Code)
	}

	boomForm := &API{
		DomainLoader: &DomainLoader{Domains: &errDomainStore{err: errors.New("boom")}, Configs: c.Configs},
		Gate:         gateOK,
	}
	rec = httptest.NewRecorder()
	boomForm.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("task-form boom = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("task-form HEAD = %d body=%q", rec.Code, rec.Body.String())
	}

	truckID := commitTruckArrive(t, c)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead,
		"/api/org/task-recap?stream=recycling&instance="+truckID+"&jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("task-recap HEAD = %d body=%q", rec.Code, rec.Body.String())
	}

	boomRecap := &API{
		DomainLoader: &DomainLoader{Domains: &errDomainStore{err: errors.New("boom")}, Configs: c.Configs},
		Gate:         gateOK,
	}
	rec = httptest.NewRecorder()
	boomRecap.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&instance="+truckID+"&jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusNotFound {
		t.Fatalf("task-recap boom = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&API{Gate: gateOK}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/close", strings.NewReader(`{}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("close gone = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&API{Gate: gateOK}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve", strings.NewReader(`{}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("qr unconfigured = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve?stream=recycling", strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("qr bad json = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	missingCfg.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve?stream=recycling", strings.NewReader(`{"raw":"x"}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("qr missing config = %d", rec.Code)
	}
	boomQR := &API{
		DomainLoader: &DomainLoader{Domains: &errDomainStore{err: errors.New("boom")}, Configs: c.Configs},
		Gate:         gateOK,
	}
	rec = httptest.NewRecorder()
	boomQR.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve?stream=recycling", strings.NewReader(`{"raw":"x"}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("qr boom = %d", rec.Code)
	}
}

func TestPassportValidateAndOptInErrorEdges(t *testing.T) {
	if err := validateDPPBlock(nil); err == nil {
		t.Fatal("nil block")
	}
	for _, p := range []*DPPConfig{
		{Subject: DPPSubjectConfig{}, Identity: DPPIdentityConfig{GTIN: `"1"`, Lot: `"l"`, Serial: `"s"`}, Snapshot: DPPSnapshotConfig{Expr: `{"a":1}`}},
		{Subject: DPPSubjectConfig{Entity: "widget"}, Identity: DPPIdentityConfig{Lot: `"l"`, Serial: `"s"`}, Snapshot: DPPSnapshotConfig{Expr: `{"a":1}`}},
		{Subject: DPPSubjectConfig{Entity: "widget"}, Identity: DPPIdentityConfig{GTIN: `"1"`, Serial: `"s"`}, Snapshot: DPPSnapshotConfig{Expr: `{"a":1}`}},
		{Subject: DPPSubjectConfig{Entity: "widget"}, Identity: DPPIdentityConfig{GTIN: `"1"`, Lot: `"l"`}, Snapshot: DPPSnapshotConfig{Expr: `{"a":1}`}},
	} {
		if err := validateDPPBlock(p); err == nil {
			t.Fatalf("expected validate fail for %#v", p)
		}
	}

	if orgConfigHasDPPOptIn(OrgConfig{}) {
		t.Fatal("nil jobs values")
	}
	if orgConfigHasDPPOptIn(OrgConfig{
		Jobs: OrderedObject{Keys: []string{"x"}, Values: map[string]any{"x": "not-a-job"}},
	}) {
		t.Fatal("non-job value")
	}
	leafOptIn := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"issue"},
			Values: map[string]any{
				"issue": Job{
					Tasks: []TaskConfig{{
						"id":       "lock",
						"onSubmit": map[string]any{"passport": "yes"},
					}},
				},
			},
		},
	}
	if !orgConfigHasDPPOptIn(leafOptIn) {
		t.Fatal("leaf passport opt-in")
	}
	if !taskOnSubmitDPP(map[string]any{"onSubmit": map[string]any{"passport": 1}}) {
		t.Fatal("passport non-bool non-map")
	}
}

type createErrDPPStore struct{ err error }

func (s *createErrDPPStore) Get(context.Context, Scope, string) (DPPRecord, error) {
	return DPPRecord{}, ErrNotFound
}
func (s *createErrDPPStore) GetByIdentity(context.Context, string, string, string) (DPPRecord, error) {
	return DPPRecord{}, ErrNotFound
}
func (s *createErrDPPStore) Create(context.Context, DPPRecord) error   { return s.err }
func (s *createErrDPPStore) DeleteByOrg(context.Context, string) error { return nil }

func TestIssuePassportErrorEdges(t *testing.T) {
	cfg := syntheticDPPConfig()
	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{"W-1": {"lot": "L1", "color": "blue"}})
	var ce *CommitError

	_, err := issueDPP(t.Context(), nil, cfg, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("nil store = %v", err)
	}

	bad := cfg
	bad.DPP = &DPPConfig{Subject: DPPSubjectConfig{Entity: "widget"}}
	_, err = issueDPP(t.Context(), NewMemoryDPPStore(), bad, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("invalid block = %v", err)
	}

	noRoot := cfg
	noRoot.DPP = nil
	noRoot.Jobs = OrderedObject{Keys: []string{"issue"}, Values: map[string]any{
		"issue": Job{Tasks: []TaskConfig{{"id": "lock"}}},
	}}
	_, err = issueDPP(t.Context(), NewMemoryDPPStore(), noRoot, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("nil root = %v", err)
	}

	_, err = issueDPP(t.Context(), NewMemoryDPPStore(), cfg, domain, "  ", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("empty subject = %v", err)
	}

	_, err = issueDPP(t.Context(), NewMemoryDPPStore(), cfg, domain, "missing", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("missing subject = %v", err)
	}

	badIdent := cfg
	badIdent.DPP = &DPPConfig{
		Subject:  DPPSubjectConfig{Entity: "widget"},
		Identity: DPPIdentityConfig{GTIN: `$missing`, Lot: `"L"`, Serial: `"S"`},
		Snapshot: DPPSnapshotConfig{Expr: `{"ok":true}`},
	}
	_, err = issueDPP(t.Context(), NewMemoryDPPStore(), badIdent, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("identity eval = %v", err)
	}

	badSnap := cfg
	badSnap.DPP = &DPPConfig{
		Subject:  DPPSubjectConfig{Entity: "widget"},
		Identity: DPPIdentityConfig{GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`},
		Snapshot: DPPSnapshotConfig{Expr: `"not-an-object"`},
	}
	_, err = issueDPP(t.Context(), NewMemoryDPPStore(), badSnap, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("snapshot type = %v", err)
	}

	boomSnap := cfg
	boomSnap.DPP = &DPPConfig{
		Subject:  DPPSubjectConfig{Entity: "widget"},
		Identity: DPPIdentityConfig{GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`},
		Snapshot: DPPSnapshotConfig{Expr: `$bogus(`},
	}
	_, err = issueDPP(t.Context(), NewMemoryDPPStore(), boomSnap, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("snapshot expr = %v", err)
	}

	_, err = issueDPP(t.Context(), &createErrDPPStore{err: ErrDPPExists}, cfg, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "passport-exists" {
		t.Fatalf("exists = %v", err)
	}
	_, err = issueDPP(t.Context(), &createErrDPPStore{err: errors.New("create boom")}, cfg, domain, "W-1", nil, nil, "t")
	if err == nil || !strings.Contains(err.Error(), "create boom") {
		t.Fatalf("create boom = %v", err)
	}
}

func TestEvalPassportIdentityAndStringEdges(t *testing.T) {
	bindings := ExprBindings{}
	if _, err := evalDPPIdentity(DPPIdentityConfig{
		GTIN: `$bogus(`, Lot: `"L"`, Serial: `"S"`,
	}, bindings); err == nil {
		t.Fatal("gtin expr")
	}
	if _, err := evalDPPIdentity(DPPIdentityConfig{
		GTIN: `"not-a-gtin"`, Lot: `"L"`, Serial: `"S"`,
	}, bindings); err == nil {
		t.Fatal("gtin normalize")
	}
	if _, err := evalDPPIdentity(DPPIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `$bogus(`, Serial: `"S"`,
	}, bindings); err == nil {
		t.Fatal("lot expr")
	}
	if _, err := evalDPPIdentity(DPPIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `$bogus(`,
	}, bindings); err == nil {
		t.Fatal("serial expr")
	}
	if _, err := evalDPPIdentity(DPPIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`, ProductName: `$bogus(`,
	}, bindings); err == nil {
		t.Fatal("productName expr")
	}
	if _, err := evalDPPIdentity(DPPIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`, Description: `$bogus(`,
	}, bindings); err == nil {
		t.Fatal("description expr")
	}

	if _, err := evalDPPString(`$bogus(`, "label", bindings); err == nil {
		t.Fatal("evalScalar err")
	}
	if _, err := evalDPPString(nil, "label", bindings); err == nil {
		t.Fatal("undefined")
	}
	if _, err := evalDPPString(42, "label", bindings); err == nil {
		t.Fatal("non-string")
	}
	if _, err := evalDPPString(`""`, "label", bindings); err == nil {
		t.Fatal("empty string")
	}
	if cloneAnyMap(nil) != nil {
		t.Fatal("clone nil")
	}
}

func TestSchemaPropertyTitlesEarlyReturns(t *testing.T) {
	if schemaPropertyTitles(nil) != nil {
		t.Fatal("nil schema")
	}
	if schemaPropertyTitles(map[string]any{"properties": map[string]any{}}) != nil {
		t.Fatal("empty props")
	}
	if schemaPropertyTitles(map[string]any{
		"properties": map[string]any{"a": "not-map", "b": map[string]any{"title": "  "}},
	}) != nil {
		t.Fatal("no usable titles")
	}
	got := schemaPropertyTitles(map[string]any{
		"properties": map[string]any{"a": map[string]any{"title": " Alpha "}},
	})
	if got["a"] != "Alpha" {
		t.Fatalf("title = %#v", got)
	}
}
