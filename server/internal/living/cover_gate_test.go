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
		t.Fatal("expected load after demo seed")
	}
	c := demoSeedCommitter(domains, configs)
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
	_, err = c.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{JobID: "intake", JobInstanceID: "ghost", GroupID: "lots"})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("unknown instance = %v", err)
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

	truckID, _, _ := commitIntakeThroughLotsClosed(t, c)
	boomPut := &Committer{
		Domains: &getOKPutErrDomainStore{
			get: func(ctx context.Context, org string) (Domain, error) { return domains.Get(ctx, org) },
			err: errors.New("put fail"),
		},
		Configs:    c.Configs,
		Authorizer: AllowAllAuthorizer{},
	}
	_, err = boomPut.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "intake", JobInstanceID: truckID, GroupID: "lots",
	})
	// lots already closed → CanClose false before Put
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("already closed = %v", err)
	}

	c2, domains2, _ := testCommitter(t, AllowAllAuthorizer{})
	truckID2 := commitTruckArrive(t, c2)
	_ = commitLotRegister(t, c2, truckID2, registerLotValue())
	boomPut2 := &Committer{
		Domains: &getOKPutErrDomainStore{
			get: func(ctx context.Context, org string) (Domain, error) { return domains2.Get(ctx, org) },
			err: errors.New("put fail"),
		},
		Configs:    c2.Configs,
		Authorizer: AllowAllAuthorizer{},
	}
	_, err = boomPut2.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "intake", JobInstanceID: truckID2, GroupID: "lots",
	})
	if err == nil || !strings.Contains(err.Error(), "put fail") {
		t.Fatalf("put fail = %v", err)
	}

	boomGet := &Committer{
		Domains:    &errDomainStore{err: errors.New("get boom")},
		Configs:    c.Configs,
		Authorizer: AllowAllAuthorizer{},
	}
	_, err = boomGet.CloseGroup(t.Context(), "feragame", op, CloseGroupRequest{
		JobID: "intake", JobInstanceID: truckID, GroupID: "lots",
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
		JobID: "intake", JobInstanceID: "ghost", GroupID: "lots",
	})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("empty domain unknown instance = %v", err)
	}
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
