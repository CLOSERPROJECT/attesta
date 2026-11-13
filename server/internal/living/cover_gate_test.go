package living

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDemoCommitHelpersSeedLineLoadReady(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	if err := seedLineLoadReadyDomain(t.Context(), domains, configs, "feragame"); err != nil {
		t.Fatal(err)
	}
	domain, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if firstLoadID(domain) == "" {
		t.Fatal("expected lot after demo seed")
	}
	c := demoSeedCommitter(domains, configs)
	if err := commitCloseTurnoCtx(t.Context(), c, "feragame", DemoLineShiftID); err != nil {
		t.Fatal(err)
	}
	if _, err := commitLineShiftNew(t.Context(), c, "feragame", "2nd"); err != nil {
		t.Fatal(err)
	}
	if _, err := commitLineShiftNew(t.Context(), c, "feragame", ""); err == nil {
		t.Fatal("duplicate 1st openTurno should fail")
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

func TestPrepGroupLifecycleAndCloseErrorPaths(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	op := adminOp()
	var ce *CommitError

	_, err := c.CloseGroup(t.Context(), "  ", op, CloseGroupRequest{JobID: "intake", JobInstanceID: "T", GroupID: "lots"})
	if !asCommitError(err, &ce) || ce.Kind != "forbidden" {
		t.Fatalf("empty org = %v", err)
	}
	_, err = (&Committer{}).CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{JobID: "intake", JobInstanceID: "T", GroupID: "lots"})
	if err == nil {
		t.Fatal("nil stores")
	}
	_, err = c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{JobID: "", JobInstanceID: "T", GroupID: "lots"})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("missing ids = %v", err)
	}
	_, err = c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{JobID: "missing", JobInstanceID: "T", GroupID: "lots"})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("unknown job = %v", err)
	}
	_, err = c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{JobID: "line", JobInstanceID: "ghost", GroupID: "weigh"})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("line leaf CloseGroup = %v, want unknown-target", err)
	}
	_, err = c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{JobID: "intake", JobInstanceID: "T", GroupID: "truckArrive"})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("non-group = %v", err)
	}

	configs := NewMemoryConfigStore()
	if err := configs.Put(t.Context(), "badorg", "v1", OrgConfig{
		ID: "bad", Org: "badorg", Version: "v1",
		Jobs: OrderedObject{Keys: []string{"j"}, Values: map[string]any{"j": "not-a-job"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "badorg", "v1"); err != nil {
		t.Fatal(err)
	}
	bad := &Committer{Domains: NewMemoryDomainStore(), Configs: configs, Authorizer: AllowAllAuthorizer{}}
	_, err = bad.CloseGroup(t.Context(), "badorg", op, CloseGroupRequest{JobID: "j", JobInstanceID: "T", GroupID: "g"})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("invalid job = %v", err)
	}

	noActive := &Committer{Domains: NewMemoryDomainStore(), Configs: NewMemoryConfigStore(), Authorizer: AllowAllAuthorizer{}}
	if _, err := noActive.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{JobID: "intake", JobInstanceID: "T", GroupID: "lots"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing active = %v", err)
	}

	_, lineID := seedLineReadyForLineLoad(t, c)
	_, err = c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "line", JobInstanceID: lineID, GroupID: "weigh",
	})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("feragame line has no groups: CloseGroup = %v", err)
	}

	// Feragame has no groups; probe config so Domain Get runs after group lookup.
	probeConfigs := NewMemoryConfigStore()
	if err := probeConfigs.Put(t.Context(), "probe", "v1", OrgConfig{
		ID: "probe", Org: "probe", Version: "v1",
		Jobs: OrderedObject{
			Keys: []string{"j"},
			Values: map[string]any{
				"j": Job{Tasks: []TaskConfig{{
					"id": "g", "kind": "group",
					"tasks": []any{map[string]any{"id": "child"}},
				}}},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := probeConfigs.SetActiveVersion(t.Context(), "probe", "v1"); err != nil {
		t.Fatal(err)
	}
	boomGet := &Committer{
		Domains:    &errDomainStore{err: errors.New("get boom")},
		Configs:    probeConfigs,
		Authorizer: AllowAllAuthorizer{},
	}
	_, err = boomGet.CloseGroup(t.Context(), "probe", op, CloseGroupRequest{
		JobID: "j", JobInstanceID: "T", GroupID: "g",
	})
	if err == nil || !strings.Contains(err.Error(), "get boom") {
		t.Fatalf("get boom = %v", err)
	}

	emptyDomain := &Committer{
		Domains:    &errDomainStore{err: ErrNotFound},
		Configs:    c.Configs,
		Authorizer: AllowAllAuthorizer{},
	}
	_, err = emptyDomain.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "line", JobInstanceID: "ghost", GroupID: "weigh",
	})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("empty domain CloseGroup leaf = %v", err)
	}
	_ = domains
}

type getOKPutErrDomainStore struct {
	get func(context.Context, string) (Domain, error)
	err error
}

func (s *getOKPutErrDomainStore) Get(ctx context.Context, org string) (Domain, error) {
	return s.get(ctx, org)
}

func (s *getOKPutErrDomainStore) Put(context.Context, string, Domain) error { return s.err }

func TestHandlerTaskFormRecapCloseQrErrorEdges(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	gateOK := func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}, true
	}
	h := &Handler{Loader: loader, Committer: c, Gate: gateOK}

	for _, path := range []string{"/api/living/task-form", "/api/living/task-recap"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s method = %d", path, rec.Code)
		}
	}

	rec := httptest.NewRecorder()
	(&Handler{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("task-form unconfigured = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	(&Handler{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("task-recap unconfigured = %d", rec.Code)
	}

	emptyOrg := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada"}, true
		},
	}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("task-form empty org = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("task-recap empty org = %d", rec.Code)
	}

	denyGate := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			http.Error(w, "no", http.StatusUnauthorized)
			return OperatorIdentity{}, false
		},
	}
	rec = httptest.NewRecorder()
	denyGate.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("task-form gate = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	denyGate.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId=T", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("task-recap gate = %d", rec.Code)
	}

	missingCfg := &Handler{
		Loader: &Loader{Domains: NewMemoryDomainStore(), Configs: NewMemoryConfigStore()},
		Gate:   gateOK,
	}
	rec = httptest.NewRecorder()
	missingCfg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("task-form missing config = %d", rec.Code)
	}

	boomForm := &Handler{
		Loader: &Loader{Domains: &errDomainStore{err: errors.New("boom")}, Configs: c.Configs},
		Gate:   gateOK,
	}
	rec = httptest.NewRecorder()
	boomForm.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/task-form?jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("task-form boom = %d", rec.Code)
	}

	ensureOpenWorkingDay(t, c)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/api/living/task-form?jobId=intake&taskId=truckArrive", nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("task-form HEAD = %d body=%q", rec.Code, rec.Body.String())
	}

	truckID := commitTruckArrive(t, c)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead,
		"/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("task-recap HEAD = %d body=%q", rec.Code, rec.Body.String())
	}

	boomRecap := &Handler{
		Loader: &Loader{Domains: &errDomainStore{err: errors.New("boom")}, Configs: c.Configs},
		Gate:   gateOK,
	}
	rec = httptest.NewRecorder()
	boomRecap.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/living/task-recap?jobId=intake&taskId=truckArrive&jobInstanceId="+truckID, nil))
	if rec.Code != http.StatusInternalServerError && rec.Code != http.StatusNotFound {
		t.Fatalf("task-recap boom = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&Handler{Gate: gateOK}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/close", strings.NewReader(`{}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("close unconfigured = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/close", strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("close bad json = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/close", strings.NewReader(
		`{"jobId":"intake","jobInstanceId":"ghost","groupId":"lots"}`,
	)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("close prep err = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	(&Handler{Gate: gateOK}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/qr-resolve", strings.NewReader(`{}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("qr unconfigured = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/qr-resolve", strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("qr bad json = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	missingCfg.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/qr-resolve", strings.NewReader(`{"raw":"x"}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("qr missing config = %d", rec.Code)
	}
	boomQR := &Handler{
		Loader: &Loader{Domains: &errDomainStore{err: errors.New("boom")}, Configs: c.Configs},
		Gate:   gateOK,
	}
	rec = httptest.NewRecorder()
	boomQR.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/qr-resolve", strings.NewReader(`{"raw":"x"}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("qr boom = %d", rec.Code)
	}
}

func TestPassportValidateAndOptInErrorEdges(t *testing.T) {
	if err := validatePassportBlock(nil); err == nil {
		t.Fatal("nil block")
	}
	for _, p := range []*PassportConfig{
		{Subject: PassportSubjectConfig{}, Identity: PassportIdentityConfig{GTIN: `"1"`, Lot: `"l"`, Serial: `"s"`}, Snapshot: PassportSnapshotConfig{Expr: `{"a":1}`}},
		{Subject: PassportSubjectConfig{Entity: "widget"}, Identity: PassportIdentityConfig{Lot: `"l"`, Serial: `"s"`}, Snapshot: PassportSnapshotConfig{Expr: `{"a":1}`}},
		{Subject: PassportSubjectConfig{Entity: "widget"}, Identity: PassportIdentityConfig{GTIN: `"1"`, Serial: `"s"`}, Snapshot: PassportSnapshotConfig{Expr: `{"a":1}`}},
		{Subject: PassportSubjectConfig{Entity: "widget"}, Identity: PassportIdentityConfig{GTIN: `"1"`, Lot: `"l"`}, Snapshot: PassportSnapshotConfig{Expr: `{"a":1}`}},
	} {
		if err := validatePassportBlock(p); err == nil {
			t.Fatalf("expected validate fail for %#v", p)
		}
	}

	if orgConfigHasPassportOptIn(OrgConfig{}) {
		t.Fatal("nil jobs values")
	}
	if orgConfigHasPassportOptIn(OrgConfig{
		Jobs: OrderedObject{Keys: []string{"x"}, Values: map[string]any{"x": "not-a-job"}},
	}) {
		t.Fatal("non-job value")
	}
	nested := OrgConfig{
		Jobs: OrderedObject{
			Keys: []string{"issue"},
			Values: map[string]any{
				"issue": Job{
					Tasks: []TaskConfig{{
						"id": "g", "kind": "group",
						"tasks": []any{
							map[string]any{
								"id": "lock",
								"onSubmit": map[string]any{"passport": "yes"},
							},
						},
					}},
				},
			},
		},
	}
	if !orgConfigHasPassportOptIn(nested) {
		t.Fatal("nested group opt-in")
	}
	if !taskOnSubmitPassport(map[string]any{"onSubmit": map[string]any{"passport": 1}}) {
		t.Fatal("passport non-bool non-map")
	}
}

type createErrPassportStore struct{ err error }

func (s *createErrPassportStore) Get(context.Context, string, string) (PassportRecord, error) {
	return PassportRecord{}, ErrNotFound
}
func (s *createErrPassportStore) GetByIdentity(context.Context, string, string, string) (PassportRecord, error) {
	return PassportRecord{}, ErrNotFound
}
func (s *createErrPassportStore) Create(context.Context, PassportRecord) error { return s.err }
func (s *createErrPassportStore) DeleteByOrg(context.Context, string) error   { return nil }

func TestIssuePassportErrorEdges(t *testing.T) {
	cfg := syntheticPassportConfig()
	domain := EmptyDomain()
	putKind(&domain, "widget", EntityMap{"W-1": {"lot": "L1", "color": "blue"}})
	var ce *CommitError

	_, err := issuePassport(t.Context(), nil, cfg, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("nil store = %v", err)
	}

	bad := cfg
	bad.Passport = &PassportConfig{Subject: PassportSubjectConfig{Entity: "widget"}}
	_, err = issuePassport(t.Context(), NewMemoryPassportStore(), bad, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("invalid block = %v", err)
	}

	noRoot := cfg
	noRoot.Passport = nil
	noRoot.Jobs = OrderedObject{Keys: []string{"issue"}, Values: map[string]any{
		"issue": Job{Tasks: []TaskConfig{{"id": "lock"}}},
	}}
	_, err = issuePassport(t.Context(), NewMemoryPassportStore(), noRoot, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("nil root = %v", err)
	}

	_, err = issuePassport(t.Context(), NewMemoryPassportStore(), cfg, domain, "  ", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("empty subject = %v", err)
	}

	_, err = issuePassport(t.Context(), NewMemoryPassportStore(), cfg, domain, "missing", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("missing subject = %v", err)
	}

	badIdent := cfg
	badIdent.Passport = &PassportConfig{
		Subject:  PassportSubjectConfig{Entity: "widget"},
		Identity: PassportIdentityConfig{GTIN: `$missing`, Lot: `"L"`, Serial: `"S"`},
		Snapshot: PassportSnapshotConfig{Expr: `{"ok":true}`},
	}
	_, err = issuePassport(t.Context(), NewMemoryPassportStore(), badIdent, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("identity eval = %v", err)
	}

	badSnap := cfg
	badSnap.Passport = &PassportConfig{
		Subject:  PassportSubjectConfig{Entity: "widget"},
		Identity: PassportIdentityConfig{GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`},
		Snapshot: PassportSnapshotConfig{Expr: `"not-an-object"`},
	}
	_, err = issuePassport(t.Context(), NewMemoryPassportStore(), badSnap, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("snapshot type = %v", err)
	}

	boomSnap := cfg
	boomSnap.Passport = &PassportConfig{
		Subject:  PassportSubjectConfig{Entity: "widget"},
		Identity: PassportIdentityConfig{GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`},
		Snapshot: PassportSnapshotConfig{Expr: `$bogus(`},
	}
	_, err = issuePassport(t.Context(), NewMemoryPassportStore(), boomSnap, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("snapshot expr = %v", err)
	}

	_, err = issuePassport(t.Context(), &createErrPassportStore{err: ErrPassportExists}, cfg, domain, "W-1", nil, nil, "t")
	if !asCommitError(err, &ce) || ce.Kind != "passport-exists" {
		t.Fatalf("exists = %v", err)
	}
	_, err = issuePassport(t.Context(), &createErrPassportStore{err: errors.New("create boom")}, cfg, domain, "W-1", nil, nil, "t")
	if err == nil || !strings.Contains(err.Error(), "create boom") {
		t.Fatalf("create boom = %v", err)
	}
}

func TestEvalPassportIdentityAndStringEdges(t *testing.T) {
	bindings := ExprBindings{}
	if _, err := evalPassportIdentity(PassportIdentityConfig{
		GTIN: `$bogus(`, Lot: `"L"`, Serial: `"S"`,
	}, bindings); err == nil {
		t.Fatal("gtin expr")
	}
	if _, err := evalPassportIdentity(PassportIdentityConfig{
		GTIN: `"not-a-gtin"`, Lot: `"L"`, Serial: `"S"`,
	}, bindings); err == nil {
		t.Fatal("gtin normalize")
	}
	if _, err := evalPassportIdentity(PassportIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `$bogus(`, Serial: `"S"`,
	}, bindings); err == nil {
		t.Fatal("lot expr")
	}
	if _, err := evalPassportIdentity(PassportIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `$bogus(`,
	}, bindings); err == nil {
		t.Fatal("serial expr")
	}
	if _, err := evalPassportIdentity(PassportIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`, ProductName: `$bogus(`,
	}, bindings); err == nil {
		t.Fatal("productName expr")
	}
	if _, err := evalPassportIdentity(PassportIdentityConfig{
		GTIN: `"9506000134352"`, Lot: `"L"`, Serial: `"S"`, Description: `$bogus(`,
	}, bindings); err == nil {
		t.Fatal("description expr")
	}

	if _, err := evalPassportString(`$bogus(`, "label", bindings); err == nil {
		t.Fatal("evalScalar err")
	}
	if _, err := evalPassportString(nil, "label", bindings); err == nil {
		t.Fatal("undefined")
	}
	if _, err := evalPassportString(42, "label", bindings); err == nil {
		t.Fatal("non-string")
	}
	if _, err := evalPassportString(`""`, "label", bindings); err == nil {
		t.Fatal("empty string")
	}
	if cloneAnyMap(nil) != nil {
		t.Fatal("clone nil")
	}
}

func TestFactTitlesFromConfigAndSchemaEarlyReturns(t *testing.T) {
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

	if (&Loader{}).factTitlesFromConfig(t.Context(), "org", SubmissionAddress{JobID: "j", TaskID: "t"}) != nil {
		t.Fatal("nil configs")
	}
	configs := NewMemoryConfigStore()
	l := &Loader{Configs: configs}
	if l.factTitlesFromConfig(t.Context(), "org", SubmissionAddress{JobID: "j", TaskID: "t"}) != nil {
		t.Fatal("no active")
	}
	if err := configs.Put(t.Context(), "org", "v1", OrgConfig{
		ID: "c", Org: "org", Version: "v1",
		Jobs: OrderedObject{Keys: []string{"j"}, Values: map[string]any{"j": "bad"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "org", "v1"); err != nil {
		t.Fatal(err)
	}
	if l.factTitlesFromConfig(t.Context(), "org", SubmissionAddress{JobID: "j", TaskID: "t"}) != nil {
		t.Fatal("non-job")
	}
	if err := configs.Put(t.Context(), "org", "v1", OrgConfig{
		ID: "c", Org: "org", Version: "v1",
		Jobs: OrderedObject{Keys: []string{"j"}, Values: map[string]any{
			"j": Job{Tasks: []TaskConfig{{"id": "other"}}},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if l.factTitlesFromConfig(t.Context(), "org", SubmissionAddress{JobID: "missing", TaskID: "t"}) != nil {
		t.Fatal("missing job")
	}
	if l.factTitlesFromConfig(t.Context(), "org", SubmissionAddress{JobID: "j", TaskID: "t"}) != nil {
		t.Fatal("missing leaf")
	}
	if err := configs.Put(t.Context(), "org", "v1", OrgConfig{
		ID: "c", Org: "org", Version: "v1",
		Jobs: OrderedObject{Keys: []string{"j"}, Values: map[string]any{
			"j": Job{Tasks: []TaskConfig{{"id": "t"}}},
		}},
	}); err != nil {
		t.Fatal(err)
	}
	if l.factTitlesFromConfig(t.Context(), "org", SubmissionAddress{JobID: "j", TaskID: "t"}) != nil {
		t.Fatal("nil payload schema")
	}

	boomGet := &Loader{Configs: &activeOKGetErrConfigStore{
		active: "v1",
		err:    errors.New("get boom"),
	}}
	if boomGet.factTitlesFromConfig(t.Context(), "org", SubmissionAddress{JobID: "j", TaskID: "t"}) != nil {
		t.Fatal("get boom")
	}
}

type activeOKGetErrConfigStore struct {
	active string
	err    error
}

func (s *activeOKGetErrConfigStore) Get(context.Context, string, string) (OrgConfig, error) {
	return OrgConfig{}, s.err
}
func (s *activeOKGetErrConfigStore) Put(context.Context, string, string, OrgConfig) error { return nil }
func (s *activeOKGetErrConfigStore) ActiveVersion(context.Context, string) (string, error) {
	return s.active, nil
}
func (s *activeOKGetErrConfigStore) SetActiveVersion(context.Context, string, string) error {
	return nil
}
