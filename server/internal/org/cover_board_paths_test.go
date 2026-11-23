package org

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAnchorStartRoles(t *testing.T) {
	if got := AnchorStartRoles(OrgConfig{}); got != nil {
		t.Fatalf("nil stream = %#v", got)
	}
	if got := AnchorStartRoles(OrgConfig{Stream: &StreamConfig{}}); got != nil {
		t.Fatalf("empty anchor job = %#v", got)
	}
	if got := AnchorStartRoles(OrgConfig{
		Stream: &StreamConfig{Instance: StreamInstance{Job: "missing"}},
		Jobs:   OrderedObject{Keys: []string{}, Values: map[string]any{}},
	}); got != nil {
		t.Fatalf("missing job = %#v", got)
	}
	if got := AnchorStartRoles(OrgConfig{
		Stream: &StreamConfig{Instance: StreamInstance{Job: "intake"}},
		Jobs: OrderedObject{
			Keys:   []string{"intake"},
			Values: map[string]any{"intake": "nope"},
		},
	}); got != nil {
		t.Fatalf("invalid job = %#v", got)
	}
	if got := AnchorStartRoles(OrgConfig{
		Stream: &StreamConfig{Instance: StreamInstance{Job: "intake"}},
		Jobs: OrderedObject{
			Keys:   []string{"intake"},
			Values: map[string]any{"intake": Job{Title: "Intake"}},
		},
	}); got != nil {
		t.Fatalf("job with no tasks = %#v", got)
	}

	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	got := AnchorStartRoles(cfg)
	if len(got) != 1 || got[0] != "administration" {
		t.Fatalf("feragame anchor roles = %#v, want [administration]", got)
	}
}

func TestTimelineHelpersEdgeCoverage(t *testing.T) {
	if got := timelineStatusLabel("all"); got != "All" {
		t.Fatalf("all = %q", got)
	}
	if got := timelineStatusLabel(" terminated "); got != "TERMINATED" {
		t.Fatalf("terminated = %q", got)
	}
	if got := timelineStatusLabel(" available "); got != "available" {
		t.Fatalf("default = %q", got)
	}
	if got := humanReadableTimelineTime(time.Time{}); got != "" {
		t.Fatalf("zero time = %q", got)
	}
	if got := humanReadableTimelineTime(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC)); got == "" {
		t.Fatal("expected formatted time")
	}

	if got := orgJobSeedField(Job{}); got != "" {
		t.Fatalf("empty InstanceID = %q", got)
	}
	if got := orgJobSeedField(Job{InstanceID: "lotId"}); got != "" {
		t.Fatalf("non-$ = %q", got)
	}
	if got := orgJobSeedField(Job{InstanceID: "$."}); got != "" {
		t.Fatalf("empty field = %q", got)
	}
	if got := orgJobSeedField(Job{InstanceID: "$.lot.nested"}); got != "" {
		t.Fatalf("nested = %q", got)
	}
	if got := orgJobSeedField(Job{InstanceID: "$.lotId"}); got != "lotId" {
		t.Fatalf("simple = %q", got)
	}

	if got := AppOrgSeededStartTaskPath("recycling", "T1", "lot_processing", "startLot", "", "L1"); got != "/app/recycling/i/T1/lot_processing/new/startLot" {
		t.Fatalf("empty seed field = %q", got)
	}

	got, ok := aggregateStreamSubstepStates([]StreamSubstepState{StreamSubstepState("weird")})
	if ok || got != "" {
		t.Fatalf("unknown states = %q ok=%v", got, ok)
	}
	if kids := timelineChildStatusesFromSubsteps([]TimelineSubstepDTO{{Status: ""}, {Status: "available"}}); len(kids) != 1 || kids[0] != StreamSubstepAvailable {
		t.Fatalf("substep kids = %#v", kids)
	}
	if kids := timelineChildStatusesFromRows([]TimelineRowDTO{{Status: ""}, {Status: "locked"}}); len(kids) != 1 || kids[0] != StreamSubstepLocked {
		t.Fatalf("row kids = %#v", kids)
	}
}

func TestStreamPlayStartEdges(t *testing.T) {
	if got := streamPlayStart(OrgConfig{}); got != nil {
		t.Fatalf("nil stream = %#v", got)
	}
	if got := streamPlayStart(OrgConfig{Stream: &StreamConfig{}}); got != nil {
		t.Fatalf("empty job = %#v", got)
	}
	if got := streamPlayStart(OrgConfig{
		Stream: &StreamConfig{Instance: StreamInstance{Job: "missing"}},
		Jobs:   OrderedObject{Keys: nil, Values: map[string]any{}},
	}); got != nil {
		t.Fatalf("missing job = %#v", got)
	}
	if got := streamPlayStart(OrgConfig{
		Stream: &StreamConfig{Instance: StreamInstance{Job: "intake"}},
		Jobs: OrderedObject{
			Keys:   []string{"intake"},
			Values: map[string]any{"intake": "nope"},
		},
	}); got != nil {
		t.Fatalf("bad job = %#v", got)
	}
	if got := streamPlayStart(OrgConfig{
		Stream: &StreamConfig{Instance: StreamInstance{Job: "intake"}},
		Jobs: OrderedObject{
			Keys:   []string{"intake"},
			Values: map[string]any{"intake": Job{Title: "Intake"}},
		},
	}); got != nil {
		t.Fatalf("no tasks = %#v", got)
	}
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	got := streamPlayStart(cfg)
	if got == nil || got.JobID != "intake" || got.TaskID != "truckArrive" {
		t.Fatalf("feragame play = %#v", got)
	}
}

type activeVersionErrStore struct {
	ConfigStore
	err error
}

func (s activeVersionErrStore) ActiveVersion(ctx context.Context, scope Scope) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.ConfigStore.ActiveVersion(ctx, scope)
}

type getConfigErrStore struct {
	ConfigStore
	err error
}

func (s getConfigErrStore) Get(ctx context.Context, scope Scope, version string) (OrgConfig, error) {
	if s.err != nil {
		return OrgConfig{}, s.err
	}
	return s.ConfigStore.Get(ctx, scope, version)
}

func TestHandleTimelineAndResolveScopeEdges(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truck := commitTruckArrive(t, c)
	gateOK := func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}
	h := &API{DomainLoader: loader, Gate: gateOK}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/timeline?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&API{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/timeline?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unconfigured = %d", rec.Code)
	}

	emptyOrg := &API{DomainLoader: loader, Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada"}, true
	}}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/timeline?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("empty org = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/timeline?stream=recycling", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("missing instance = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/api/org/timeline?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HEAD = %d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("HEAD body = %q", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/timeline?stream=nope&instance="+truck, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown stream = %d", rec.Code)
	}

	// resolveStreamScope: nil configs
	rec = httptest.NewRecorder()
	scope, _, ok := h.resolveStreamScope(rec, t.Context(), "feragame", "recycling", nil)
	if ok || scope.Stream != "" || rec.Code != http.StatusInternalServerError {
		t.Fatalf("nil configs = ok=%v scope=%#v code=%d", ok, scope, rec.Code)
	}

	// ActiveVersion hard error
	rec = httptest.NewRecorder()
	_, _, ok = h.resolveStreamScope(rec, t.Context(), "feragame", "recycling", activeVersionErrStore{
		ConfigStore: loader.Configs,
		err:         errors.New("boom"),
	})
	if ok || rec.Code != http.StatusInternalServerError {
		t.Fatalf("active version boom = ok=%v code=%d", ok, rec.Code)
	}

	// Get config hard error after a real active version
	version, err := loader.Configs.ActiveVersion(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	_ = version
	rec = httptest.NewRecorder()
	_, _, ok = h.resolveStreamScope(rec, t.Context(), "feragame", "recycling", getConfigErrStore{
		ConfigStore: loader.Configs,
		err:         errors.New("get boom"),
	})
	if ok || rec.Code != http.StatusInternalServerError {
		t.Fatalf("get boom = ok=%v code=%d", ok, rec.Code)
	}

	// Domain loader Get hard error (not ErrNotFound) — ensureStreamInstance path.
	badDomainAPI := &API{
		DomainLoader: &DomainLoader{Domains: &errDomainStore{err: errors.New("domain boom")}, Configs: loader.Configs},
		Gate:         gateOK,
	}
	rec = httptest.NewRecorder()
	badDomainAPI.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/timeline?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("domain boom = %d", rec.Code)
	}

	// ensureStreamInstance: missing domain store / ErrNotFound domain
	rec = httptest.NewRecorder()
	if (&API{DomainLoader: &DomainLoader{Configs: loader.Configs}}).ensureStreamInstance(rec, t.Context(), FeragameScope(), OrgConfig{}, truck) {
		t.Fatal("nil domains must fail")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("nil domains code = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	if (&API{DomainLoader: &DomainLoader{Domains: &errDomainStore{err: ErrNotFound}, Configs: loader.Configs}}).
		ensureStreamInstance(rec, t.Context(), FeragameScope(), OrgConfig{}, truck) {
		t.Fatal("domain not found must fail")
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("domain not found code = %d", rec.Code)
	}

	// resolveStreamScope: Get returns ErrNotFound
	rec = httptest.NewRecorder()
	_, _, ok = h.resolveStreamScope(rec, t.Context(), "feragame", "recycling", getConfigErrStore{
		ConfigStore: loader.Configs,
		err:         ErrNotFound,
	})
	if ok || rec.Code != http.StatusNotFound {
		t.Fatalf("get not found = ok=%v code=%d", ok, rec.Code)
	}

	// resolveStreamScope: ActiveVersion ErrNotFound
	rec = httptest.NewRecorder()
	_, _, ok = h.resolveStreamScope(rec, t.Context(), "feragame", "recycling", activeVersionErrStore{
		ConfigStore: loader.Configs,
		err:         ErrNotFound,
	})
	if ok || rec.Code != http.StatusNotFound {
		t.Fatalf("active not found = ok=%v code=%d", ok, rec.Code)
	}

	// Normalize failure (blank stream key)
	rec = httptest.NewRecorder()
	_, _, ok = h.resolveStreamScope(rec, t.Context(), "feragame", "   ", loader.Configs)
	if ok || rec.Code != http.StatusBadRequest {
		t.Fatalf("blank stream = ok=%v code=%d", ok, rec.Code)
	}
}

func TestHandleStreamsEdges(t *testing.T) {
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	gateOK := func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}
	h := &API{DomainLoader: &DomainLoader{Domains: NewMemoryDomainStore(), Configs: configs}, Gate: gateOK}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/streams", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&API{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/streams", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unconfigured = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{Configs: configs},
		Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada"}, true
		},
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/streams", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("empty org = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/api/org/streams", nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD = %d body=%q", rec.Code, rec.Body.String())
	}

	// ListStreams error
	rec = httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{Configs: listStreamsErrStore{ConfigStore: configs, err: errors.New("list boom")}},
		Gate:         gateOK,
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/streams", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("list boom = %d", rec.Code)
	}

	// Skip streams whose active version / get fails
	rec = httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{Configs: activeVersionErrStore{ConfigStore: configs, err: errors.New("skip")}},
		Gate:         gateOK,
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/streams", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("skip active errors = %d", rec.Code)
	}
}

type listStreamsErrStore struct {
	ConfigStore
	err error
}

func (s listStreamsErrStore) ListStreams(ctx context.Context, org string) ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.ConfigStore.ListStreams(ctx, org)
}

func TestOrgTimelineSubmissionsTimestampFormats(t *testing.T) {
	domain := EmptyDomain()
	domain.Submissions = []Submission{
		{JobID: "intake", TaskID: "arrive", JobInstanceID: "T1", Timestamp: "2026-11-19T10:00:00.123456789Z", UserID: "a"},
		{JobID: "intake", TaskID: "arrive", JobInstanceID: "T1", Timestamp: "2026-11-19T11:00:00Z", UserID: "b"},
		{JobID: "intake", TaskID: "arrive", JobInstanceID: "T1", Timestamp: "not-a-time", UserID: "c"},
		{JobID: "intake", TaskID: "arrive", JobInstanceID: "T1", Timestamp: "", UserID: "d"},
	}
	got := OrgTimelineSubmissions(domain, "intake", "T1", "arrive")
	if len(got) != 4 {
		t.Fatalf("got %#v", got)
	}
	var sawParsed, sawFallback bool
	for _, s := range got {
		if s.Timestamp == "not-a-time" && s.TimestampHuman == "not-a-time" {
			sawFallback = true
		}
		if s.TimestampHuman != "" && s.Timestamp != "not-a-time" && s.Timestamp != "" {
			sawParsed = true
		}
	}
	if !sawParsed || !sawFallback {
		t.Fatalf("formats = %#v", got)
	}
}

func TestRequireStreamAndMissingStreamQuery(t *testing.T) {
	_, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	gateOK := func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}
	h := &API{DomainLoader: loader, TaskCommitter: &TaskCommitter{Configs: loader.Configs, Domains: loader.Domains, Authorizer: AllowAllAuthorizer{}}, Gate: gateOK}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/org/domain?instance=T1", nil)
	_, _, ok := h.requireStreamAndInstance(rec, req, "feragame", loader.Configs, true)
	if ok || rec.Code != http.StatusBadRequest {
		t.Fatalf("missing stream = ok=%v code=%d", ok, rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/task-form?jobId=intake&taskId=truckArrive&instance=T1", nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("task-form missing stream = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/commit", strings.NewReader(`{"taskPath":"intake/truckArrive"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("commit missing stream = %d body=%s", rec.Code, rec.Body.String())
	}

	if fakeHTTPPath("") != "" || fakeHTTPPath("https://ex.test/path?q=1") != "/path" {
		t.Fatalf("fakeHTTPPath url = %q %q", fakeHTTPPath(""), fakeHTTPPath("https://ex.test/path?q=1"))
	}
	if fakeHTTPPath("/only?x=1") != "/only" || fakeHTTPPath("/plain") != "/plain" {
		t.Fatal("fakeHTTPPath query strip")
	}
	_, _ = registerCSV(map[string]any{"palletIds": []string{"a", "b"}, "client": "c"})
	_, _ = registerCSV(map[string]any{"palletIds": 42})
}

func TestBuildOrgTimelineMissingJobFallsBack(t *testing.T) {
	cfg := OrgConfig{Jobs: OrderedObject{Keys: []string{}, Values: map[string]any{}}}
	steps := BuildOrgTimelineSteps(cfg, EmptyDomain(), StreamTimeline{Steps: []StreamTimelineStep{
		{JobID: "missing", Title: "", Tasks: []StreamTimelineTask{{ID: "t", Title: "", State: StreamSubstepAvailable}}},
	}}, "recycling", "T1")
	if len(steps) != 1 || steps[0].Title != "missing" {
		t.Fatalf("empty step title falls back to job id: %#v", steps)
	}
	if len(steps[0].Substeps) != 1 || steps[0].Substeps[0].Title != "t" {
		t.Fatalf("empty task title falls back to id: %#v", steps[0].Substeps)
	}

	cfg.Jobs = OrderedObject{Keys: []string{"each"}, Values: map[string]any{"each": Job{
		Title: "Each", InstanceID: "$.lotId",
		Tasks: []TaskConfig{{"id": "start"}},
	}}}
	steps = BuildOrgTimelineSteps(cfg, EmptyDomain(), StreamTimeline{Steps: []StreamTimelineStep{
		{JobID: "each", Title: "Each", Rows: []StreamTimelineRow{
			{ID: "L1", Label: "L1", Tasks: []StreamTimelineTask{{ID: "start", Title: "Start", State: StreamSubstepLocked}}},
		}},
	}}, "recycling", "T1")
	if len(steps) != 1 || len(steps[0].Rows) != 1 || steps[0].Rows[0].Status != string(StreamSubstepLocked) {
		t.Fatalf("each-row status = %#v", steps)
	}
}

func TestWriteCommitErrorKinds(t *testing.T) {
	cases := []struct {
		err  error
		code int
		body string
	}{
		{&CommitError{Kind: "unauthorized"}, http.StatusForbidden, "forbidden"},
		{&CommitError{Kind: "unknown-target", Message: "bad target"}, http.StatusBadRequest, "bad target"},
		{&CommitError{Kind: "prep", Message: "prep fail"}, http.StatusBadRequest, "prep fail"},
		{&CommitError{Kind: "forbidden", Message: "nope"}, http.StatusForbidden, "nope"},
		{&CommitError{Kind: "mutate", Message: "mut"}, http.StatusConflict, "mut"},
		{&CommitError{Kind: "passport-exists", Message: "dup"}, http.StatusConflict, "dup"},
		{&CommitError{Kind: "conflict", Message: "race"}, http.StatusConflict, "race"},
		{&CommitError{Kind: "other", Message: "x"}, http.StatusInternalServerError, "commit failed"},
		{ErrNotFound, http.StatusNotFound, "active config not found"},
		{errors.New("plain"), http.StatusInternalServerError, "failed to commit"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		writeCommitError(rec, tc.err)
		if rec.Code != tc.code || !strings.Contains(rec.Body.String(), tc.body) {
			t.Fatalf("%v → %d %q, want %d containing %q", tc.err, rec.Code, rec.Body.String(), tc.code, tc.body)
		}
	}
}

func TestProjectOnceTasksMissingInstance(t *testing.T) {
	job := Job{Title: "Intake", Tasks: []TaskConfig{{"id": "arrive", "title": "Arrive"}}}
	wlJob := WorklistJob{JobID: "intake", Title: "Intake", TaskIDs: []string{"arrive"}}
	tasks := projectOnceTasks(job, wlJob, "missing")
	if len(tasks) == 0 {
		t.Fatal("expected chrome tasks for missing instance")
	}
}

type nthActiveVersionStore struct {
	ConfigStore
	n, failAt int
	err       error
}

func (s *nthActiveVersionStore) ActiveVersion(ctx context.Context, scope Scope) (string, error) {
	s.n++
	if s.failAt > 0 && s.n == s.failAt {
		return "", s.err
	}
	return s.ConfigStore.ActiveVersion(ctx, scope)
}

type nthDomainGetStore struct {
	DomainStore
	n, failAt int
	err       error
}

func (s *nthDomainGetStore) Get(ctx context.Context, scope Scope) (Domain, error) {
	s.n++
	if s.failAt > 0 && s.n == s.failAt {
		return Domain{}, s.err
	}
	return s.DomainStore.Get(ctx, scope)
}

func TestHandleLoadTimelineQrErrorBranches(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truck := commitTruckArrive(t, c)
	gateOK := func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}

	// Load: requireStreamAndInstance OK, then ActiveVersion fails as ErrNotFound / boom
	for _, tc := range []struct {
		err  error
		code int
	}{
		{ErrNotFound, http.StatusNotFound},
		{errors.New("load boom"), http.StatusInternalServerError},
	} {
		rec := httptest.NewRecorder()
		(&API{
			DomainLoader: &DomainLoader{
				Domains: loader.Domains,
				Configs: &nthActiveVersionStore{ConfigStore: loader.Configs, failAt: 2, err: tc.err},
			},
			Gate: gateOK,
		}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truck, nil))
		if rec.Code != tc.code {
			t.Fatalf("load second active %v = %d, want %d", tc.err, rec.Code, tc.code)
		}
	}

	// Timeline: ensureStreamInstance OK, second Domains.Get fails
	for _, tc := range []struct {
		err  error
		code int
	}{
		{ErrNotFound, http.StatusOK}, // EmptyDomain fallback
		{errors.New("domain boom"), http.StatusInternalServerError},
	} {
		rec := httptest.NewRecorder()
		(&API{
			DomainLoader: &DomainLoader{
				Domains: &nthDomainGetStore{DomainStore: loader.Domains, failAt: 2, err: tc.err},
				Configs: loader.Configs,
			},
			Gate: gateOK,
		}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/timeline?stream=recycling&instance="+truck, nil))
		if rec.Code != tc.code {
			t.Fatalf("timeline second get %v = %d, want %d", tc.err, rec.Code, tc.code)
		}
	}

	h := &API{DomainLoader: loader, Gate: gateOK}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve", strings.NewReader(`{"raw":"x"}`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("qr missing stream = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve?stream=recycling", strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("qr bad json = %d", rec.Code)
	}

	// Task form HEAD already covered elsewhere; hit NotFound / forbidden-ish via nth active fail after scope resolve
	rec = httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{
			Domains: loader.Domains,
			Configs: &nthActiveVersionStore{ConfigStore: loader.Configs, failAt: 2, err: ErrNotFound},
		},
		Gate: gateOK,
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-form?stream=recycling&jobId=intake&taskId=truckArrive&instance="+truck, nil))
	if rec.Code != http.StatusNotFound && rec.Code != http.StatusInternalServerError {
		t.Fatalf("task-form second active = %d", rec.Code)
	}

	// Task recap: invalid address + domain boom + HEAD
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&jobId=&taskId=truckArrive&jobInstanceId="+truck+"&instance="+truck, nil))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("recap missing jobId = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{Domains: &errDomainStore{err: errors.New("recap boom")}, Configs: loader.Configs},
		Gate:         gateOK,
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&jobId=intake&taskId=truckArrive&jobInstanceId="+truck+"&instance="+truck, nil))
	// ensureStreamInstance fails first with domain boom → 500
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("recap domain boom = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead,
		"/api/org/task-recap?stream=recycling&jobId=intake&taskId=truckArrive&jobInstanceId="+truck+"&instance="+truck, nil))
	if rec.Code != http.StatusOK && rec.Code != http.StatusNotFound {
		t.Fatalf("recap HEAD = %d", rec.Code)
	}
}

func TestWorklistAndPathAvailabilityEdges(t *testing.T) {
	if got := EvaluatePathAvailability(OrgConfig{}, EmptyDomain(), TaskPathRef{JobID: "x", TaskID: "y"}, "T"); got.Done {
		t.Fatal("missing job")
	}
	cfg := OrgConfig{Jobs: OrderedObject{Keys: []string{"j"}, Values: map[string]any{"j": "nope"}}}
	if EvaluatePathAvailability(cfg, EmptyDomain(), TaskPathRef{JobID: "j", TaskID: "t"}, "T").Done {
		t.Fatal("bad job")
	}
	cfg.Jobs.Values["j"] = Job{Tasks: []TaskConfig{{"id": "once"}, {"id": "many", "submit": "many"}}}
	if EvaluatePathAvailability(cfg, EmptyDomain(), TaskPathRef{JobID: "j", TaskID: "missing"}, "T").Done {
		t.Fatal("missing task")
	}
	domain := EmptyDomain()
	domain.Submissions = []Submission{{JobID: "j", TaskID: "once", JobInstanceID: "T"}}
	if !EvaluatePathAvailability(cfg, domain, TaskPathRef{JobID: "j", TaskID: "once"}, "T").Done {
		t.Fatal("once done")
	}
	if EvaluatePathAvailability(cfg, domain, TaskPathRef{JobID: "j", TaskID: "many"}, "T").Done {
		t.Fatal("many never done")
	}
	if sequenceOkForPath(OrgConfig{}, EmptyDomain(), TaskPathRef{JobID: "j", TaskID: "once"}, "T") {
		t.Fatal("sequence missing job")
	}
	if sequenceOkForPath(cfg, EmptyDomain(), TaskPathRef{JobID: "j", TaskID: "nope"}, "T") {
		t.Fatal("sequence missing task")
	}
	if !startsWhenOk(cfg, Job{}, EmptyDomain()) {
		t.Fatal("nil startsWhen")
	}

	if _, ok := NextAdvanceTaskRef(OrgConfig{}, Worklist{}, "", "T"); ok {
		t.Fatal("empty job")
	}
	if _, ok := NextAdvanceTaskRef(OrgConfig{}, Worklist{}, "j", ""); ok {
		t.Fatal("empty instance")
	}
	if _, ok := NextAdvanceTaskRef(OrgConfig{}, Worklist{}, "j", "T"); ok {
		t.Fatal("missing wl job")
	}
	wl := Worklist{Jobs: []WorklistJob{{JobID: "j", TaskIDs: []string{"once"}, Instances: []WorklistInstance{}}}}
	if _, ok := NextAdvanceTaskRef(cfg, wl, "j", "T"); ok {
		t.Fatal("missing instance")
	}
	if worklistInstanceHasRemainingWork(cfg, Worklist{}, "j", "T") {
		t.Fatal("missing wl job remaining")
	}
	if worklistInstanceHasRemainingWork(cfg, wl, "j", "T") {
		t.Fatal("missing instance remaining")
	}
	wl.Jobs[0].Instances = []WorklistInstance{{ID: "T", Tasks: map[string]TaskStatus{"once": TaskStatusDone, "many": TaskStatusActionable}}}
	wl.Jobs[0].TaskIDs = []string{"once", "many"}
	if worklistInstanceHasRemainingWork(cfg, wl, "j", "T") {
		t.Fatal("many skipped; once done → no remaining")
	}
	wl.Jobs[0].Instances[0].Tasks["once"] = TaskStatusActionable
	if !worklistInstanceHasRemainingWork(cfg, wl, "j", "T") {
		t.Fatal("actionable once → remaining")
	}

	cfg.Jobs.Keys = []string{"missing", "bad", "j"}
	cfg.Jobs.Values["bad"] = 1
	refs := CollectActionableTaskRefs(cfg, Worklist{Jobs: []WorklistJob{
		{JobID: "j", CanStart: true, TaskIDs: []string{"once"}, Instances: []WorklistInstance{
			{ID: "T", Tasks: map[string]TaskStatus{"once": TaskStatusActionable, "ghost": TaskStatusActionable}},
		}},
	}})
	if len(refs) < 1 {
		t.Fatalf("collect = %#v", refs)
	}
	if !rolesIntersect([]string{"a", "b"}, []string{"c", "b"}) {
		t.Fatal("intersect")
	}
	if rolesIntersect([]string{"a"}, []string{"b"}) {
		t.Fatal("no intersect")
	}
	if rolesIntersect(nil, []string{"a"}) || rolesIntersect([]string{"a"}, nil) {
		t.Fatal("nil roles")
	}
	if rolesIntersect([]string{"", "x"}, []string{"", "y"}) {
		t.Fatal("empty role strings")
	}
	if !rolesIntersect([]string{"", "x"}, []string{"x"}) {
		t.Fatal("skip empty still match")
	}
	if _, ok := worklistTaskByID(WorklistJob{Tasks: []WorklistTask{{ID: "a"}}}, "missing"); ok {
		t.Fatal("missing task")
	}
}

func TestDecodeTaskConfigAndTransformTags(t *testing.T) {
	if got := decodeTaskConfigList([]TaskConfig{{"id": "a"}}); len(got) != 1 || got[0]["id"] != "a" {
		t.Fatalf("typed list = %#v", got)
	}
	if got := decodeTaskConfigList([]any{map[string]any{"id": "b"}, "skip", TaskConfig{"id": "c"}}); len(got) != 2 {
		t.Fatalf("any list = %#v", got)
	}
	if got := decodeTaskConfigList(42); got != nil {
		t.Fatalf("default = %#v", got)
	}
	if asTaskConfig(nil) != nil || asTaskConfig(1) != nil {
		t.Fatal("asTaskConfig reject")
	}
	if got := asTaskConfig(TaskConfig{"id": "x"}); got["id"] != "x" {
		t.Fatalf("asTaskConfig typed = %#v", got)
	}
	if got := asTaskConfig(map[string]any{"id": "y"}); got["id"] != "y" {
		t.Fatalf("asTaskConfig map = %#v", got)
	}

	transformMintOp{}.transformOpTag()
	transformMintManyOp{}.transformOpTag()
	transformPatchEachOp{}.transformOpTag()
	transformPatchOp{}.transformOpTag()
	transformBindOp{}.transformOpTag()
	MintOp{}.mutateOp()
	PatchOp{}.mutateOp()

	_ = domainBindingsWithToday(EmptyDomain(), "T1", nil, "2026-11-23")
}

func TestHandleLoadEdges(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truck := commitTruckArrive(t, c)
	gateOK := func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}
	h := &API{DomainLoader: loader, Gate: gateOK}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/domain?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	(&API{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unconfigured = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	(&API{DomainLoader: loader, Gate: func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada"}, true
	}}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("empty org = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, "/api/org/domain?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD = %d body=%q", rec.Code, rec.Body.String())
	}

	// LoadForStreamInstance ErrNotFound via broken active config
	rec = httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{Domains: loader.Domains, Configs: activeVersionErrStore{ConfigStore: loader.Configs, err: ErrNotFound}},
		Gate:         gateOK,
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truck, nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("active not found = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{Domains: loader.Domains, Configs: activeVersionErrStore{ConfigStore: loader.Configs, err: errors.New("load boom")}},
		Gate:         gateOK,
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/org/domain?stream=recycling&instance="+truck, nil))
	// requireStreamAndInstance → resolveStreamScope fails with 500 before Load
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("load boom = %d", rec.Code)
	}
}

func TestEnsureDPPSnapshotChromeAndScopeFromConfig(t *testing.T) {
	if _, err := ScopeFromConfig(OrgConfig{}); err == nil {
		t.Fatal("missing stream")
	}
	if _, err := ScopeFromConfig(OrgConfig{Org: "o", Stream: &StreamConfig{Key: "s"}}); err != nil {
		t.Fatal(err)
	}

	chrome, raw := ensureDPPSnapshotChrome(OrgConfig{}, map[string]any{"onSubmit": map[string]any{"passport": 1}}, nil, nil)
	if chrome != nil || raw != nil {
		t.Fatalf("no dpp cfg = %#v %#v", chrome, raw)
	}
	cfg := OrgConfig{DPP: &DPPConfig{Snapshot: DPPSnapshotConfig{Expr: "passport.snapshot"}}}
	chrome, raw = ensureDPPSnapshotChrome(cfg, map[string]any{}, nil, nil)
	if chrome != nil || raw != nil {
		t.Fatalf("no passport hook = %#v %#v", chrome, raw)
	}
	cfg.DPP.Snapshot.Expr = "   "
	chrome, raw = ensureDPPSnapshotChrome(cfg, map[string]any{"onSubmit": map[string]any{"passport": true}}, nil, nil)
	if chrome != nil || raw != nil {
		t.Fatalf("empty expr = %#v %#v", chrome, raw)
	}
	cfg.DPP.Snapshot.Expr = "passport.snapshot"
	chrome, raw = ensureDPPSnapshotChrome(cfg, map[string]any{"onSubmit": map[string]any{"passport": true}}, nil, nil)
	if len(chrome) != 1 || chrome[0].ID != DPPSnapshotChromeID || raw[DPPSnapshotChromeID] == nil {
		t.Fatalf("inject = %#v %#v", chrome, raw)
	}
	existing := []ChromeRule{{ID: DPPSnapshotChromeID, Kind: "readonly", Expr: "old"}}
	existingRaw := map[string]any{DPPSnapshotChromeID: map[string]any{"kind": "readonly", "expr": "old"}}
	chrome, raw = ensureDPPSnapshotChrome(cfg, map[string]any{"onSubmit": map[string]any{"passport": true}}, existing, existingRaw)
	if chrome[0].Kind != "dataReport" || chrome[0].Expr != "passport.snapshot" {
		t.Fatalf("rewrite = %#v", chrome)
	}
	entry, _ := raw[DPPSnapshotChromeID].(map[string]any)
	if entry["kind"] != "dataReport" || entry["expr"] != "passport.snapshot" {
		t.Fatalf("rewrite raw = %#v", entry)
	}
	// nil chromeRaw with existing chrome id forces map alloc + empty entry branch
	chrome, raw = ensureDPPSnapshotChrome(cfg, map[string]any{"onSubmit": map[string]any{"passport": true}},
		[]ChromeRule{{ID: DPPSnapshotChromeID}}, nil)
	if raw[DPPSnapshotChromeID] == nil || chrome[0].Kind != "dataReport" {
		t.Fatalf("nil raw rewrite = %#v %#v", chrome, raw)
	}

	if got := evaluateBlockedReason(EmptyDomain(), nil, "T1", nil, nil); got != "" {
		t.Fatalf("nil task = %q", got)
	}
	if got := evaluateBlockedReason(EmptyDomain(), map[string]any{}, "T1", nil, nil); got != "" {
		t.Fatalf("missing reason = %q", got)
	}
	if got := evaluateBlockedReason(EmptyDomain(), map[string]any{"blockedReason": 1}, "T1", nil, nil); got != "" {
		t.Fatalf("non-string = %q", got)
	}
	if got := evaluateBlockedReason(EmptyDomain(), map[string]any{"blockedReason": "   "}, "T1", nil, nil); got != "" {
		t.Fatalf("blank = %q", got)
	}
	if got := evaluateBlockedReason(EmptyDomain(), map[string]any{"blockedReason": "1 + 1"}, "T1", nil, nil); got != "" {
		t.Fatalf("non-string result = %q", got)
	}
	if got := evaluateBlockedReason(EmptyDomain(), map[string]any{"blockedReason": "'blocked'"}, "T1", nil, nil); got != "blocked" {
		t.Fatalf("string result = %q", got)
	}
	if got := evaluateBlockedReason(EmptyDomain(), map[string]any{"blockedReason": "$nowhere"}, "T1", nil, nil); got != "" {
		t.Fatalf("expr error = %q", got)
	}
}

func TestMapApplyCommitErrorAndStreamDefaults(t *testing.T) {
	if got := MapWorklistStatusToStreamState(TaskStatus("nope")); got != StreamSubstepLocked {
		t.Fatalf("default map = %q", got)
	}
	if ProjectStreamTimeline(OrgConfig{}, EmptyDomain(), OperatorIdentity{}, "T1").Steps != nil {
		t.Fatal("nil stream timeline")
	}
	if ProjectStreamTimeline(OrgConfig{Stream: &StreamConfig{}}, EmptyDomain(), OperatorIdentity{}, "").Steps != nil {
		t.Fatal("empty instance timeline")
	}
	if ListStreamInstances(OrgConfig{}, EmptyDomain(), OperatorIdentity{}) != nil {
		t.Fatal("nil stream instances")
	}

	ce := &CommitError{Kind: "prep", Message: "x"}
	if got := mapApplyCommitError(ce); got != ce {
		t.Fatalf("commit error passthrough = %#v", got)
	}
	if got := mapApplyCommitError(TransformError{Kind: "bad", Message: "m"}); got == nil {
		t.Fatal("transform wrap")
	} else if c, ok := got.(*CommitError); !ok || c.Kind != "prep" {
		t.Fatalf("transform wrap = %#v", got)
	}
	if got := mapApplyCommitError(InstanceExprError{Kind: "type", Message: "boom"}); got == nil {
		t.Fatal("instance wrap")
	} else if c, ok := got.(*CommitError); !ok || c.Kind != "prep" {
		t.Fatalf("instance wrap = %#v", got)
	}
	if got := mapApplyCommitError(MutateError{Kind: "missing-id", Entity: EntityType("lot"), ID: "L1"}); got == nil {
		t.Fatal("mutate wrap")
	} else if c, ok := got.(*CommitError); !ok || c.Kind != "mutate" {
		t.Fatalf("mutate wrap = %#v", got)
	}
	plain := errors.New("plain")
	if got := mapApplyCommitError(plain); !errors.Is(got, plain) {
		t.Fatalf("plain = %#v", got)
	}
}

func TestCoverageGateRemainingPaths(t *testing.T) {
	c, _, loader := testTaskCommitter(t, AllowAllAuthorizer{})
	truck := commitTruckArrive(t, c)
	gateOK := func(http.ResponseWriter, *http.Request) (OperatorIdentity, bool) {
		return adminOp(), true
	}

	// qr-resolve: scope OK, then Load ActiveVersion ErrNotFound / boom
	for _, tc := range []struct {
		err  error
		code int
	}{
		{ErrNotFound, http.StatusNotFound},
		{errors.New("qr load boom"), http.StatusInternalServerError},
	} {
		rec := httptest.NewRecorder()
		(&API{
			DomainLoader: &DomainLoader{
				Domains: loader.Domains,
				Configs: &nthActiveVersionStore{ConfigStore: loader.Configs, failAt: 2, err: tc.err},
			},
			Gate: gateOK,
		}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/org/qr-resolve?stream=recycling",
			strings.NewReader(`{"raw":"x"}`)))
		if rec.Code != tc.code {
			t.Fatalf("qr load %v = %d, want %d", tc.err, rec.Code, tc.code)
		}
	}

	// task-recap: ensure OK, LoadTaskRecap Domains.Get boom → 500
	rec := httptest.NewRecorder()
	(&API{
		DomainLoader: &DomainLoader{
			Domains: &nthDomainGetStore{DomainStore: loader.Domains, failAt: 2, err: errors.New("recap get boom")},
			Configs: loader.Configs,
		},
		Gate: gateOK,
	}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet,
		"/api/org/task-recap?stream=recycling&instance="+truck+
			"&jobId=intake&taskId=truckArrive&jobInstanceId="+truck, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("recap second get = %d", rec.Code)
	}

	// LoadTaskRecap direct edges (HTTP cannot reach "submission address required")
	l := &DomainLoader{Domains: loader.Domains, Configs: loader.Configs}
	if _, err := l.LoadTaskRecap(t.Context(), Scope{}, SubmissionAddress{}); err == nil {
		t.Fatal("empty scope")
	}
	if _, err := l.LoadTaskRecap(t.Context(), FeragameScope(), SubmissionAddress{}); err == nil ||
		!strings.Contains(err.Error(), "submission address required") {
		t.Fatalf("addr required = %v", err)
	}
	if _, err := (&DomainLoader{}).LoadTaskRecap(t.Context(), FeragameScope(), SubmissionAddress{
		JobID: "j", JobInstanceID: "i", TaskID: "t",
	}); err == nil || !strings.Contains(err.Error(), "stores required") {
		t.Fatalf("stores = %v", err)
	}
	if _, err := (&DomainLoader{Domains: &errDomainStore{err: ErrNotFound}}).LoadTaskRecap(
		t.Context(), FeragameScope(), SubmissionAddress{JobID: "j", JobInstanceID: "i", TaskID: "t"},
	); !errors.Is(err, ErrNotFound) {
		t.Fatalf("domain not found = %v", err)
	}
	if _, err := (&DomainLoader{Domains: &errDomainStore{err: errors.New("get boom")}}).LoadTaskRecap(
		t.Context(), FeragameScope(), SubmissionAddress{JobID: "j", JobInstanceID: "i", TaskID: "t"},
	); err == nil || !strings.Contains(err.Error(), "get boom") {
		t.Fatalf("domain boom = %v", err)
	}

	got := projectWorklistTasks([]TaskConfig{{"id": "a"}}, []string{"a", "ghost"})
	if len(got) != 2 || got[1].ID != "ghost" || got[1].Title != "ghost" {
		t.Fatalf("missing task chrome = %#v", got)
	}
	if worklistInstanceIsFinished(Job{}, WorklistInstance{}) {
		t.Fatal("no tasks finished")
	}

	domain := EmptyDomain()
	putKind(&domain, "lot", EntityMap{"L1": {"x": 1}})
	cfg := OrgConfig{
		Stream: &StreamConfig{
			Key: "truckId",
			Steps: []StreamStep{{
				Job:  "lot_processing",
				Each: &StreamEach{Entity: "lot", Where: `$bogus(`},
			}},
		},
	}
	prog := projectStepProgress(cfg, domain, Worklist{}, "T1")
	if len(prog) != 1 || prog[0].Total != 0 {
		t.Fatalf("each err progress = %#v", prog)
	}

	rows := buildOrgTimelineRows(Job{Title: "Lot"}, StreamTimelineStep{
		JobID: "lot_processing",
		Rows:  []StreamTimelineRow{{ID: "L1", Label: "  ", Finished: true}},
	}, domain, "recycling", truck)
	if len(rows) != 1 || rows[0].Label != "L1" {
		t.Fatalf("empty label = %#v", rows)
	}

	domain.Submissions = []Submission{{
		JobID: "j", JobInstanceID: "i", TaskID: "t",
		Timestamp: "not-a-time", UserID: "ada",
	}}
	subs := OrgTimelineSubmissions(domain, "j", "i", "t")
	if len(subs) != 1 || subs[0].TimestampHuman != "not-a-time" {
		t.Fatalf("raw timestamp = %#v", subs)
	}

	cfg.Jobs = OrderedObject{Keys: []string{"ghost"}, Values: map[string]any{}}
	if len(ProjectWorklistAll(cfg, EmptyDomain(), adminOp()).Jobs) != 0 {
		t.Fatal("missing job key")
	}

	labelCfg := OrgConfig{Catalogs: nil, Stream: &StreamConfig{Key: "truckId"}}
	if got := labelForPendingEachRow(labelCfg, EmptyDomain(), Job{Label: `42`}, "lot", "R1", nil); got != "R1" {
		t.Fatalf("non-string label = %q", got)
	}
	if got := labelForPendingEachRow(labelCfg, EmptyDomain(), Job{Label: `""`}, "lot", "R1", nil); got != "R1" {
		t.Fatalf("blank label = %q", got)
	}
	provisional := provisionalStartTasks(OrgConfig{}, EmptyDomain(), Job{Tasks: []TaskConfig{{"id": "t"}}},
		[]string{"t"}, WorklistJob{}, adminOp(), nil)
	if len(provisional) != 1 || provisional[0].Title != "t" {
		t.Fatalf("provisional title = %#v", provisional)
	}

	ui := withDefaultPlaceholders(
		map[string]any{"arr": map[string]any{}},
		map[string]any{"properties": map[string]any{
			"arr": map[string]any{
				"type":  "array",
				"items": map[string]any{"type": "string"},
			},
		}},
		nil,
	)
	itemsUi, _ := ui["arr"].(map[string]any)["items"].(map[string]any)
	if itemsUi == nil {
		t.Fatalf("items placeholder ui = %#v", ui["arr"])
	}
	if !stampPlaceholder(map[string]any{"ui:options": map[string]any{"placeholder": ""}}, "", false) {
		t.Fatal("stamp empty widget slots")
	}
	if expandPlaceholderToShadcn(map[string]any{}) {
		t.Fatal("expand empty")
	}
	if acceptsPlaceholder(map[string]any{"type": []any{"boolean", "string"}}) != true {
		t.Fatal("type list accepts")
	}

	if err := validateTaskBlockedReason(TaskConfig{"id": "t", "blockedReason": 1}, "job"); err == nil {
		t.Fatal("blockedReason type")
	}
	if err := validateTaskBlockedReason(TaskConfig{"id": "t", "blockedReason": `$bogus(`}, "job"); err == nil {
		t.Fatal("blockedReason expr")
	}
	if err := rejectGroupAuthoring(TaskConfig{"id": "t", "kind": "group"}, "job"); err == nil {
		t.Fatal("group kind")
	}
	if err := rejectUnlockFrom(TaskConfig{"id": "t", "unlockFrom": "x"}, "job"); err == nil {
		t.Fatal("unlockFrom")
	}
	if taskSubmit(nil) != submitOnce || taskSubmit(TaskConfig{"submit": 1}) != submitOnce {
		t.Fatal("taskSubmit defaults")
	}
	if taskOnCommit(nil) != nil {
		t.Fatal("nil onCommit")
	}
	if taskCommitNavigation(map[string]any{"onCommit": map[string]any{"navigation": "nope"}}) != commitNavAdvance {
		t.Fatal("nav default")
	}
	if !taskAdvanceTarget(map[string]any{"onCommit": map[string]any{"advanceTarget": "x"}}) {
		t.Fatal("advanceTarget non-bool")
	}

	if _, err := RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{
			map[string]any{"op": "bind", "expr": `"hello"`},
			map[string]any{"op": "patchEach", "entity": "lot", "set": map[string]any{"x": 1}},
		},
	}); err == nil {
		t.Fatal("patchEach type")
	}
	if _, err := RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "patch", "entity": "lot", "id": `$bogus(`, "set": map[string]any{"x": 1}}},
	}); err == nil {
		t.Fatal("patch id expr")
	}
	if _, err := RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "mint", "entity": "lot", "id": `"L1"`, "set": map[string]any{"x": `$bogus(`}}},
	}); err == nil {
		t.Fatal("mint set expr")
	}
	if _, err := RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "mintMany", "entity": "lot", "count": "1", "id": `$bogus(`, "set": map[string]any{}}},
	}); err == nil {
		t.Fatal("mintMany id expr")
	}
	if _, err := RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "mintMany", "entity": "lot", "count": "1", "id": `"L1"`, "set": map[string]any{"x": `$bogus(`}}},
	}); err == nil {
		t.Fatal("mintMany set expr")
	}
	if _, err := RunTransform(EmptyDomain(), RunTransformInput{
		Facts: map[string]any{"a": map[string]any{}},
		Transform: []any{map[string]any{"op": "patchEach", "entity": "lot", "set": map[string]any{"x": `$bogus(`}}},
	}); err == nil {
		t.Fatal("patchEach set expr")
	}
	if got, err := evalScalar("$value", nil, ExprBindings{}); err != nil || got.Defined {
		t.Fatalf("$value omit = %#v %v", got, err)
	}
	if set, err := evalSet(map[string]any{"k": "$value"}, nil, ExprBindings{}); err != nil || len(set) != 0 {
		t.Fatalf("evalSet omit = %#v %v", set, err)
	}
}
