package living

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOrderedObjectJSONRoundTripAndErrors(t *testing.T) {
	var empty OrderedObject
	raw, err := empty.MarshalJSON()
	if err != nil || string(raw) != "{}" {
		t.Fatalf("empty marshal = %s err=%v", raw, err)
	}
	if _, ok := empty.Get("x"); ok {
		t.Fatal("Get on nil Values")
	}

	var o OrderedObject
	if err := o.UnmarshalJSON([]byte("null")); err != nil {
		t.Fatal(err)
	}
	if err := o.UnmarshalJSON([]byte(`{"b":2,"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if o.Keys[0] != "b" || o.Keys[1] != "a" {
		t.Fatalf("keys = %#v", o.Keys)
	}
	if got, _ := o.Get("a"); got != float64(1) {
		t.Fatalf("a = %#v", got)
	}
	o.Set("a", 3)
	if o.Len() != 2 {
		t.Fatalf("len after overwrite = %d", o.Len())
	}
	raw, err = o.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"b":2`) || !strings.Contains(string(raw), `"a":3`) {
		t.Fatalf("marshal = %s", raw)
	}

	for _, bad := range []string{`[]`, `"x"`, `{`, `{"k":`} {
		if err := (&OrderedObject{}).UnmarshalJSON([]byte(bad)); err == nil {
			t.Fatalf("expected error for %s", bad)
		}
	}
}

func TestErrorStringsAndMarkerMethods(t *testing.T) {
	if (&CommitError{Kind: "prep"}).Error() != "living: prep" {
		t.Fatal("CommitError empty message")
	}
	if !strings.Contains((&CommitError{Kind: "prep", Message: "x"}).Error(), "prep: x") {
		t.Fatal("CommitError with message")
	}
	if (MutateError{Kind: "missing-id", Entity: EntityType("load"), ID: "L1"}).Error() == "" {
		t.Fatal("MutateError")
	}
	if (InstanceExprError{Kind: "type", Message: "boom"}).Error() == "" {
		t.Fatal("InstanceExprError")
	}
	if (TransformError{Kind: "unsupported-op", Op: "fly"}).Error() != "unsupported op: fly" {
		t.Fatal("TransformError unsupported")
	}
	if !strings.Contains((TransformError{Kind: "bad"}).Error(), "transform bad") {
		t.Fatal("TransformError kind only")
	}
	if !strings.Contains((TransformError{Kind: "bad", Message: "m"}).Error(), "bad: m") {
		t.Fatal("TransformError message")
	}
	MintOp{}.mutateOp()
	PatchOp{}.mutateOp()
	transformMintOp{}.transformOpTag()
	transformMintManyOp{}.transformOpTag()
	transformPatchEachOp{}.transformOpTag()
	transformPatchOp{}.transformOpTag()
}

func TestHelperScalarsTruthyIntFloatHeader(t *testing.T) {
	if truthy(nil) || !truthy(true) || truthy(false) {
		t.Fatal("bool/nil truthy")
	}
	if truthy("") || !truthy("x") || truthy(0) || !truthy(1) || truthy(int64(0)) || !truthy(int64(2)) {
		t.Fatal("string/int truthy")
	}
	if truthy(0.0) || !truthy(1.5) || !truthy(struct{}{}) {
		t.Fatal("float/default truthy")
	}

	for _, tc := range []struct {
		in   any
		want float64
		ok   bool
	}{
		{float64(1.5), 1.5, true},
		{float32(2), 2, true},
		{3, 3, true},
		{int32(4), 4, true},
		{int64(5), 5, true},
		{"no", 0, false},
	} {
		got, ok := floatFromAny(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Fatalf("floatFromAny(%T)=%v,%v", tc.in, got, ok)
		}
	}
	for _, tc := range []struct {
		in   any
		want int
		ok   bool
	}{
		{1, 1, true},
		{int32(2), 2, true},
		{int64(3), 3, true},
		{float64(4), 4, true},
		{float64(4.5), 0, false},
		{float32(5), 5, true},
		{float32(5.5), 0, false},
		{"x", 0, false},
	} {
		got, ok := intFromAny(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Fatalf("intFromAny(%#v)=%v,%v", tc.in, got, ok)
		}
	}

	if headerValue(nil, "X") != "" {
		t.Fatal("nil headers")
	}
	if headerValue(map[string]string{"Content-Type": "text/plain"}, "content-type") != "text/plain" {
		t.Fatal("case-insensitive header")
	}
	if headerValue(map[string]string{"A": "1"}, "B") != "" {
		t.Fatal("missing header")
	}
	if _, err := requirePlainTextURL(""); err == nil {
		t.Fatal("empty url")
	}
	if _, err := requirePlainTextURL(" ok "); err == nil {
		t.Fatal("padded url")
	}
	if _, err := requirePlainTextURL("http://x\ny"); err == nil {
		t.Fatal("multiline url")
	}
	if got, err := requirePlainTextURL("http://x"); err != nil || got != "http://x" {
		t.Fatalf("plain url = %q %v", got, err)
	}
	if fakeHTTPPath("") != "" || fakeHTTPPath("https://h/p?q=1") != "/p" || fakeHTTPPath("/raw?x=1") != "/raw" {
		t.Fatal("fakeHTTPPath")
	}
	if !schemaTypeIncludesString("string") || schemaTypeIncludesString("number") {
		t.Fatal("schemaType string")
	}
	if !schemaTypeIncludesString([]any{"null", "string"}) || schemaTypeIncludesString([]any{"number"}) {
		t.Fatal("schemaType array")
	}
	if acceptsPlaceholder(map[string]any{"readOnly": true, "type": "string"}) {
		t.Fatal("readonly placeholder")
	}
	if !acceptsPlaceholder(map[string]any{"type": "number"}) || !acceptsPlaceholder(map[string]any{"type": []any{"null", "integer"}}) {
		t.Fatal("acceptsPlaceholder types")
	}
	if acceptsPlaceholder(map[string]any{"type": "object"}) {
		t.Fatal("object placeholder")
	}
}

func TestParseTaskPathAndStartsWhenAll(t *testing.T) {
	if _, err := ParseTaskPath("nope"); err == nil {
		t.Fatal("missing slash")
	}
	if _, err := ParseTaskPath("a/b/c"); err == nil {
		t.Fatal("nested path")
	}
	ref, err := ParseTaskPath(" intake/register ")
	if err != nil || ref.JobID != "intake" || ref.TaskID != "register" {
		t.Fatalf("parse = %#v %v", ref, err)
	}

	eligible := map[string]any{
		"dynamicProperties": map[string]any{
			"weights": map[string]any{"read": "rows"},
		},
	}
	gate := &StartsWhen{Task: StartsWhenRef{JobID: "intake", TaskID: "palletWeight"}, Quantifier: "all"}
	instances := []JobInstance{{JobID: "intake", ID: "a"}, {JobID: "intake", ID: "b"}}
	if IsStartsWhenSatisfied(gate, nil, instances, eligible) {
		t.Fatal("all with no submissions")
	}
	if IsStartsWhenSatisfied(gate, []Submission{sub("intake", "palletWeight", "a")}, instances, eligible) {
		t.Fatal("all missing one instance")
	}
	subs := []Submission{sub("intake", "palletWeight", "a"), sub("intake", "palletWeight", "b")}
	if !IsStartsWhenSatisfied(gate, subs, instances, eligible) {
		t.Fatal("all satisfied")
	}
	if IsStartsWhenSatisfied(gate, subs, nil, eligible) {
		t.Fatal("all with zero instances")
	}
	if !IsStartsWhenSatisfied(nil, nil, nil, eligible) {
		t.Fatal("nil startsWhen")
	}

	job := Job{Repeatable: false, StartsWhen: gate}
	if CanStartJob(job, "intake", nil, instances, eligible) {
		t.Fatal("non-repeatable with existing instances")
	}
	if !CanStartJob(Job{Repeatable: true, StartsWhen: gate}, "intake", subs, instances, eligible) {
		t.Fatal("repeatable can start when gate ok")
	}
}

func TestEntityMapsCloneDomainMutateEntities(t *testing.T) {
	d := EmptyDomain()
	putKind(&d, "kind.x", nil)
	putKind(&d, "kind.y", EntityMap{"id": {"k": 1}})
	maps := d.EntityMaps()
	if len(maps["kind.x"]) != 0 || maps["kind.y"]["id"]["k"] != 1 {
		t.Fatalf("EntityMaps = %#v", maps)
	}
	cloned := cloneDomain(d)
	if len(cloned.Kind("kind.x")) != 0 || cloned.Kind("kind.y")["id"]["k"] != 1 {
		t.Fatalf("cloneDomain kinds = %#v", cloned.Entities)
	}

	got, err := Mutate(EmptyDomain(), MutateRequest{
		UserID: "u", JobID: "j", TaskID: "t", JobInstanceID: "i", Timestamp: "ts",
		Facts: map[string]any{},
		Ops: []MutateOp{
			MintOp{Entity: EntityType("custom"), ID: "c1", Set: map[string]any{"a": 1}},
			PatchOp{Entity: EntityType("custom"), ID: "c1", Set: map[string]any{"a": 2, "b": nil}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Kind("custom")["c1"]["a"] != 2 {
		t.Fatalf("extra mutate = %#v", got.Entities)
	}
	if _, ok := got.Kind("custom")["c1"]["b"]; ok {
		t.Fatal("nil patch should unset")
	}

	noSub := false
	_, err = Mutate(EmptyDomain(), MutateRequest{
		AppendSubmission: &noSub,
		Ops:              []MutateOp{MintOp{Entity: EntityType(""), ID: "x"}},
	})
	if err == nil {
		t.Fatal("empty entity kind")
	}
	_, err = Mutate(EmptyDomain(), MutateRequest{
		AppendSubmission: &noSub,
		Ops:              []MutateOp{PatchOp{Entity: EntityType("load"), ID: "missing", Set: map[string]any{"x": 1}}},
	})
	var me MutateError
	if !asMutateError(err, &me) || me.Kind != "missing-id" {
		t.Fatalf("missing-id = %v", err)
	}

	_, err = Mutate(Domain{}, MutateRequest{
		UserID: "u", JobID: "j", TaskID: "t", JobInstanceID: "i", Timestamp: "ts",
		Ops: []MutateOp{MintOp{Entity: EntityType("load"), ID: "L1", Set: map[string]any{"x": 1}}},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestInjectEnumOnSchemaPropertyBranches(t *testing.T) {
	props := map[string]any{
		"sourceLoadId": map[string]any{"type": "string"},
		"rows": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nestedId": map[string]any{"type": "string"},
				},
			},
		},
		"noise": "skip",
	}
	if !injectEnumOnSchemaProperty(props, "sourceLoadId", []string{"A", "B"}) {
		t.Fatal("root enum")
	}
	root := props["sourceLoadId"].(map[string]any)
	if got, _ := root["enum"].([]string); len(got) != 2 || got[0] != "A" {
		// json-ish: enum may be []string from assignment
		enum, _ := root["enum"].([]string)
		if len(enum) != 2 {
			t.Fatalf("root enum = %#v", root["enum"])
		}
	}
	if !injectEnumOnSchemaProperty(props, "nestedId", []string{"N1"}) {
		t.Fatal("nested enum")
	}
	rows := props["rows"].(map[string]any)
	items := rows["items"].(map[string]any)
	itemProps := items["properties"].(map[string]any)
	nested := itemProps["nestedId"].(map[string]any)
	if enum, _ := nested["enum"].([]string); len(enum) != 1 || enum[0] != "N1" {
		t.Fatalf("nested enum = %#v", nested["enum"])
	}
	if injectEnumOnSchemaProperty(props, "missing", []string{"x"}) {
		t.Fatal("missing property")
	}

	projected, err := projectTaskForm(domainFromKinds(map[string]EntityMap{
		"load": {"L1": {"client": "NO"}},
	}), map[string]any{
		"enumReads": map[string]any{"sourceLoadId": "loads"},
		"reads": map[string]any{
			"loads": map[string]any{"entity": "load", "key": "id"},
		},
		"schema": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"sourceLoadId": map[string]any{"type": "string"},
			},
		},
	}, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	prop := projected.Schema["properties"].(map[string]any)["sourceLoadId"].(map[string]any)
	if enum, _ := prop["enum"].([]string); len(enum) != 1 || enum[0] != "L1" {
		t.Fatalf("projected enum = %#v", prop["enum"])
	}
}

func TestFirstActionableTaskAndCompileDerivedEdges(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	domain := EmptyDomain()
	wl := ProjectWorklist(cfg, domain, FixtureAllRolesOp())
	ref, ok := FirstActionableTask(cfg, wl)
	if !ok || ref.JobID != "intake" || ref.TaskID != "truckArrive" || ref.JobInstanceID != "" {
		t.Fatalf("first = %#v ok=%v", ref, ok)
	}
	ref, ok = FirstActionableTask(OrgConfig{}, Worklist{})
	if ok || ref.JobID != "" {
		t.Fatalf("empty first = %#v", ref)
	}

	if compileDerivedRules(map[string]any{}, nil) != nil {
		t.Fatal("empty derived")
	}
	if compileDerivedRules(map[string]any{"derived": map[string]any{"x": "bad"}}, nil) != nil {
		t.Fatal("non-map derived entry")
	}
	if compileDerivedRules(map[string]any{"derived": map[string]any{"x": map[string]any{}}}, nil) != nil {
		t.Fatal("empty expr")
	}
	over := true
	got := compileDerivedRules(map[string]any{
		"derived": map[string]any{
			"leaf":       map[string]any{"expr": "1", "overridable": over},
			"row.*.cell": map[string]any{"expr": "2"},
		},
	}, nil)
	if got == nil || got["leaf"].Expr != "1" || got["leaf"].Overridable == nil || !*got["leaf"].Overridable {
		t.Fatalf("concrete derived = %#v", got)
	}
	if _, has := got["row.*.cell"]; has {
		t.Fatal("map-star without rows should drop")
	}
	expanded := compileDerivedRules(map[string]any{
		"derived": map[string]any{"row.*.cell": map[string]any{"expr": "2"}},
	}, []string{"r1"})
	if expanded["row.r1.cell"].Expr != "2" {
		t.Fatalf("expanded = %#v", expanded)
	}
}

func TestAsJobNormalizeAndConfigJSONEdges(t *testing.T) {
	if _, ok := AsJob(nil); ok {
		t.Fatal("nil AsJob")
	}
	if _, ok := AsJob(42); ok {
		t.Fatal("bad AsJob")
	}
	raw := json.RawMessage(`{"repeatable":true,"instanceId":"$.id","label":"$.id","tasks":[{"id":"t1"}]}`)
	job, ok := AsJob(raw)
	if !ok || !job.Repeatable || len(job.Tasks) != 1 || job.Tasks[0].ID() != "t1" {
		t.Fatalf("AsJob raw = %#v ok=%v", job, ok)
	}
	if _, ok := AsJob(json.RawMessage(`{`)); ok {
		t.Fatal("bad raw AsJob")
	}

	cfg := OrgConfig{Jobs: OrderedObject{Keys: []string{"j"}, Values: map[string]any{"j": raw}}}
	normalizeOrgConfigJobs(&cfg)
	if _, ok := cfg.Jobs.Values["j"].(Job); !ok {
		t.Fatalf("normalize = %#v", cfg.Jobs.Values["j"])
	}
	normalizeOrgConfigJobs(nil)
	normalizeOrgConfigJobs(&OrgConfig{})

	var decoded OrgConfig
	if err := json.Unmarshal([]byte(`{"id":"x","version":"1","jobs":null}`), &decoded); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(`{"id":"x","version":"1","jobs":[]}`), &decoded); err == nil {
		t.Fatal("jobs must be object")
	}
	if err := json.Unmarshal([]byte(`{"repeatable":true,"instanceId":"a","label":"b","tasks":[{"id":""}]}`), &Job{}); err == nil {
		t.Fatal("empty task id")
	}
}

func TestHandlerHTTPErrorPaths(t *testing.T) {
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	loader := &Loader{Domains: domains, Configs: configs}
	c := &Committer{Domains: domains, Configs: configs, Authorizer: AllowAllAuthorizer{}}
	gateOK := func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}, true
	}
	h := &Handler{Loader: loader, Committer: c, Gate: gateOK}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("not found = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/domain", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("load method = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/commit", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("commit method = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&Handler{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/domain", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("load unconfigured = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	(&Handler{}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(`{}`)))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("commit unconfigured = %d", rec.Code)
	}

	emptyOrg := &Handler{
		Loader: loader, Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada"}, true
		},
	}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/domain", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("load empty org = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(`{}`)))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("commit empty org = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/api/living/domain", nil)
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD load = %d body=%q", rec.Code, rec.Body.String())
	}

	missingCfg := &Handler{
		Loader: &Loader{Domains: NewMemoryDomainStore(), Configs: NewMemoryConfigStore()},
		Gate:   gateOK,
	}
	rec = httptest.NewRecorder()
	missingCfg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/domain", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("load missing config = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(`{`)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("bad json = %d", rec.Code)
	}

	deny := &Handler{Loader: loader, Committer: &Committer{Domains: domains, Configs: configs, Authorizer: DenyAllAuthorizer{}}, Gate: gateOK}
	rec = httptest.NewRecorder()
	deny.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(
		`{"taskPath":"intake/truckArrive","jobInstanceId":"","mint":{"kind":"startingNew"},"value":{"date":"2026-10-19","turno":"1st","truckWeight":1000}}`,
	)))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unauthorized commit = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(
		`{"taskPath":"nope","value":{}}`,
	)))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unknown-target = %d body=%s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(
		`{"taskPath":"intake/weightCheck","jobInstanceId":"x","mint":{"kind":"none"},"value":{}}`,
	)))
	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusForbidden && rec.Code != http.StatusConflict {
		t.Fatalf("prep/forbidden path = %d body=%s", rec.Code, rec.Body.String())
	}

	writeCommitError(httptest.NewRecorder(), &CommitError{Kind: "mutate", Message: "dup"})
	writeCommitError(httptest.NewRecorder(), &CommitError{Kind: "forbidden", Message: "no"})
	writeCommitError(httptest.NewRecorder(), &CommitError{Kind: "other", Message: "x"})
	writeCommitError(httptest.NewRecorder(), ErrNotFound)
	writeCommitError(httptest.NewRecorder(), errors.New("boom"))
}

func TestCommitterValidationBranches(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}

	_, err := c.Commit(t.Context(), "  ", op, CommitRequest{})
	var ce *CommitError
	if !asCommitError(err, &ce) || ce.Kind != "forbidden" {
		t.Fatalf("empty org = %v", err)
	}
	_, err = (&Committer{}).Commit(t.Context(), "feragame", op, CommitRequest{})
	if err == nil {
		t.Fatal("nil stores")
	}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{TaskPath: "bad", Value: map[string]any{}})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("bad path = %v", err)
	}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", Mint: &CommitMint{Kind: "weird"}, Value: map[string]any{},
	})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("bad mint = %v", err)
	}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: "_new", Value: map[string]any{},
	})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("_new = %v", err)
	}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", Mint: &CommitMint{Kind: MintKindNone}, Value: map[string]any{},
	})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("missing instance = %v", err)
	}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/register", JobInstanceID: "x", Mint: &CommitMint{Kind: MintKindNone},
	})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("nil value = %v", err)
	}
}

func TestNewCerbosAuthorizerDefaults(t *testing.T) {
	a := NewCerbosAuthorizer("http://example", nil, nil)
	if a.client == nil || a.now == nil {
		t.Fatal("defaults")
	}
}

func TestEvaluateJobInstanceExprErrors(t *testing.T) {
	_, _, err := EvaluateJobInstanceExprs(Job{InstanceID: "1+", Label: `"ok"`}, nil, nil)
	if err == nil {
		t.Fatal("bad instanceId expr")
	}
	_, _, err = EvaluateJobInstanceExprs(Job{InstanceID: `""`, Label: `"ok"`}, nil, nil)
	if err == nil {
		t.Fatal("empty instanceId")
	}
	_, _, err = EvaluateJobInstanceExprs(Job{InstanceID: `"id"`, Label: "1+"}, nil, nil)
	if err == nil {
		t.Fatal("bad label expr")
	}
	_, _, err = EvaluateJobInstanceExprs(Job{InstanceID: `"id"`, Label: `""`}, nil, nil)
	if err == nil {
		t.Fatal("empty label")
	}
}

func TestParseReadDefAndResolveReadEdges(t *testing.T) {
	if _, err := parseReadDef("x"); err == nil {
		t.Fatal("non-map read")
	}
	if _, err := parseReadDef(map[string]any{}); err == nil {
		t.Fatal("missing entity")
	}
	if _, err := parseReadDef(map[string]any{"entity": "load"}); err == nil {
		t.Fatal("missing key")
	}
	def, err := parseReadDef(map[string]any{"entity": "load", "key": "client", "where": "client = 'NO'"})
	if err != nil {
		t.Fatal(err)
	}
	domain := domainFromKinds(map[string]EntityMap{
		"load": {"L1": {"client": "NO"}, "L2": {"client": "SU"}, "L3": {}},
	})
	ids, err := ResolveRead(domain, def, "inst", "label", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "NO" {
		t.Fatalf("keyed ids = %#v", ids)
	}
	ids, err = ResolveRead(domain, ReadDef{Entity: EntityType("load"), Key: "id"}, "", "", nil)
	if err != nil || len(ids) != 3 {
		t.Fatalf("id key = %#v %v", ids, err)
	}
	extra, err := listEntities(domainFromKinds(map[string]EntityMap{"custom": nil}), EntityType("custom"))
	if err != nil || len(extra) != 0 {
		t.Fatalf("nil extra list = %#v %v", extra, err)
	}
}

func TestParseTransformOpsAndRunTransformErrors(t *testing.T) {
	allowed := map[string]struct{}{"load": {}, "pallet": {}}
	if _, err := parseTransformOps("nope", allowed); err == nil {
		t.Fatal("non-array")
	}
	if _, err := parseTransformOps([]any{"x"}, allowed); err == nil {
		t.Fatal("non-map op")
	}
	if _, err := parseTransformOps([]any{map[string]any{}}, allowed); err == nil {
		t.Fatal("missing op")
	}
	if _, err := parseTransformOps([]any{map[string]any{"op": "fly"}}, allowed); err == nil {
		t.Fatal("unsupported")
	}
	if _, err := parseTransformOps([]any{map[string]any{"op": "mint", "entity": "nope", "set": map[string]any{}}}, allowed); err == nil {
		t.Fatal("entity not allowed")
	}
	if _, err := parseTransformOps([]any{map[string]any{"op": "mint", "entity": "load"}}, allowed); err == nil {
		t.Fatal("missing set")
	}
	if _, err := parseTransformOps([]any{map[string]any{"op": "mint", "set": map[string]any{}}}, nil); err == nil {
		t.Fatal("missing entity")
	}
	if _, err := parseTransformOps([]any{map[string]any{"op": "patch", "entity": "load", "set": map[string]any{}}}, nil); err == nil {
		t.Fatal("patch missing id")
	}
	ops, err := parseTransformOps([]any{
		map[string]any{"op": "mint", "entity": "load", "id": `"L1"`, "as": "load", "set": map[string]any{"client": `"NO"`}},
		map[string]any{"op": "mintMany", "entity": "pallet", "count": "2", "id": `"L1." & $string($index+1)`, "set": map[string]any{"loadId": `"L1"`}},
		map[string]any{"op": "patch", "entity": "load", "id": `"L1"`, "set": map[string]any{"client": `"SU"`}},
		map[string]any{"op": "patchEach", "entity": "pallet", "set": map[string]any{"marked": true}},
	}, allowed)
	if err != nil || len(ops) != 4 {
		t.Fatalf("parse ok = %#v %v", ops, err)
	}

	_, err = RunTransform(EmptyDomain(), RunTransformInput{Transform: "bad"})
	if err == nil {
		t.Fatal("RunTransform bad")
	}
	_, err = RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "mint", "entity": "load", "id": `""`, "set": map[string]any{"x": 1}}},
	})
	if err == nil {
		t.Fatal("empty mint id")
	}
	_, err = RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "mint", "entity": "load", "id": "1+", "set": map[string]any{"x": 1}}},
	})
	if err == nil {
		t.Fatal("bad mint id expr")
	}
	_, err = RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "patch", "entity": "load", "id": `""`, "set": map[string]any{"x": 1}}},
	})
	if err == nil {
		t.Fatal("empty patch id")
	}
	_, err = RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "mintMany", "entity": "pallet", "count": `-1`, "id": `"x"`, "set": map[string]any{}}},
	})
	if err == nil {
		t.Fatal("negative count")
	}
	_, err = RunTransform(EmptyDomain(), RunTransformInput{
		Transform: []any{map[string]any{"op": "mintMany", "entity": "pallet", "count": `"no"`, "id": `"x"`, "set": map[string]any{}}},
	})
	if err == nil {
		t.Fatal("bad count type")
	}
	domain := EmptyDomain()
	putKind(&domain, "load", EntityMap{"L1": {"client": "NO"}})
	putKind(&domain, "pallet", EntityMap{"L1.1": {"loadId": "L1"}})
	got, err := RunTransform(domain, RunTransformInput{
		JobID: "intake", TaskID: "register", JobInstanceID: "L1", UserID: "ada",
		Facts:   map[string]any{"L1.1": map[string]any{"weight": 1}},
		Allowed: allowed,
		Transform: []any{
			map[string]any{"op": "mint", "entity": "load", "id": `"L2"`, "as": "load", "set": map[string]any{"client": `"X"`}},
			map[string]any{"op": "mintMany", "entity": "pallet", "count": "1", "id": `"L2.1"`, "set": map[string]any{"loadId": `"L2"`}},
			map[string]any{"op": "patch", "entity": "load", "id": `"L1"`, "set": map[string]any{"client": `"SU"`}},
			map[string]any{"op": "patchEach", "entity": "pallet", "set": map[string]any{"ok": true}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Request.Ops) < 4 {
		t.Fatalf("ops = %#v", got.Request.Ops)
	}
	applyMintLocally(&Domain{}, EntityType(""), "x", nil) // empty kind no-op path
	applyPatchLocally(&Domain{}, EntityType("load"), "missing", map[string]any{"a": 1})

	if EntityKindsFromConfig(OrgConfig{}) != nil {
		t.Fatal("nil domain")
	}
	if EntityKindsFromConfig(OrgConfig{Domain: map[string]any{}}) != nil {
		t.Fatal("no entities")
	}
	if EntityKindsFromConfig(OrgConfig{Domain: map[string]any{"entities": "x"}}) != nil {
		t.Fatal("bad entities")
	}
	kinds := EntityKindsFromConfig(OrgConfig{Domain: map[string]any{"entities": map[string]any{"load": map[string]any{}}}})
	if _, ok := kinds["load"]; !ok {
		t.Fatalf("kinds = %#v", kinds)
	}
}

func TestSubmitHTTPParseAndEvalErrors(t *testing.T) {
	if taskOnSubmitHTTP(nil) != nil || taskOnSubmitHTTP(map[string]any{}) != nil {
		t.Fatal("no onSubmit http")
	}
	if parseSubmitHTTP(nil) != nil || parseSubmitHTTP("x") != nil {
		t.Fatal("bad http list")
	}
	if parseSubmitHTTP([]any{"x"}) != nil {
		t.Fatal("non-map effect")
	}
	if parseSubmitHTTP([]any{map[string]any{"url": "u", "label": "l", "action": "link"}}) != nil {
		t.Fatal("missing method")
	}
	if parseSubmitHTTP([]any{map[string]any{"method": "POST", "label": "l", "action": "link"}}) != nil {
		t.Fatal("missing url")
	}
	if parseSubmitHTTP([]any{map[string]any{"method": "POST", "url": "u", "action": "link"}}) != nil {
		t.Fatal("missing label")
	}
	if parseSubmitHTTP([]any{map[string]any{"method": "POST", "url": "u", "label": "l"}}) != nil {
		t.Fatal("missing action")
	}
	effects := parseSubmitHTTP([]any{map[string]any{
		"method": `"POST"`, "url": `"/x"`, "label": `"L"`, "action": "link", "body": `{"a":1}`,
	}})
	if len(effects) != 1 || !effects[0].HasBody {
		t.Fatalf("effects = %#v", effects)
	}
	if len(runSubmitHTTPEffects(t.Context(), nil, nil, nil, nil, EmptyDomain(), "", nil, "").Errors) != 0 {
		t.Fatal("empty effects")
	}

	port := &recordingHttpPort{responses: []HttpPortResponse{{URL: "blob:ok"}}}
	item, msg := runOneSubmitHTTP(t.Context(), port, SubmitHttpEffect{
		Method: "1+", URL: `"/x"`, Label: `"L"`, Action: "link",
	}, nil, nil, "")
	if item != nil || msg == "" {
		t.Fatalf("bad method expr item=%v msg=%q", item, msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), port, SubmitHttpEffect{
		Method: `""`, URL: `"/x"`, Label: `"L"`, Action: "link",
	}, nil, nil, "")
	if item != nil || msg == "" {
		t.Fatalf("empty method msg=%q", msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), port, SubmitHttpEffect{
		Method: `"POST"`, URL: "1+", Label: `"L"`, Action: "link",
	}, nil, nil, "")
	if item != nil || msg == "" {
		t.Fatalf("bad url expr msg=%q", msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), port, SubmitHttpEffect{
		Method: `"POST"`, URL: `"/x"`, Label: `""`, Action: "link",
	}, nil, nil, "")
	if item != nil || msg == "" {
		t.Fatalf("empty label msg=%q", msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), port, SubmitHttpEffect{
		Method: `"POST"`, URL: `"/x"`, Label: `"L"`, Action: `"nope"`,
	}, nil, nil, "")
	if item != nil || msg == "" {
		t.Fatalf("bad action msg=%q", msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), nil, SubmitHttpEffect{
		Method: `"POST"`, URL: `"/x"`, Label: `"L"`, Action: "link",
	}, nil, nil, "")
	if item != nil || msg != "http port not configured" {
		t.Fatalf("nil port msg=%q", msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), &recordingHttpPort{responses: []HttpPortResponse{{URL: ""}}}, SubmitHttpEffect{
		Method: `"POST"`, URL: `"/x"`, Label: `"L"`, Action: "download",
	}, nil, nil, "")
	if item != nil || msg == "" {
		t.Fatalf("empty resp url msg=%q", msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), &recordingHttpPort{responses: []HttpPortResponse{{
		URL: "blob:x", Headers: map[string]string{"Content-Disposition": `attachment; filename="a.csv"`},
	}}}, SubmitHttpEffect{
		Method: `"POST"`, URL: `"/x"`, Label: `"L"`, Action: "download",
	}, nil, nil, "")
	if msg != "" || item == nil || item.Filename != "a.csv" {
		t.Fatalf("download filename = %#v msg=%q", item, msg)
	}
	item, msg = runOneSubmitHTTP(t.Context(), port, SubmitHttpEffect{
		Method: `"POST"`, URL: `"https://evil.example/x"`, Label: `"L"`, Action: "link",
	}, nil, nil, "feragame")
	if item != nil || !strings.Contains(msg, "absolute URL") {
		t.Fatalf("ssrf reject msg=%q", msg)
	}

	if filenameFromContentDisposition("") != "" {
		t.Fatal("empty cd")
	}
	if got := filenameFromContentDisposition(`attachment; filename*=UTF-8''na%20me.csv`); got != "na me.csv" {
		t.Fatalf("star filename = %q", got)
	}
	if got := filenameFromContentDisposition(`attachment; filename=plain.csv`); got != "plain.csv" {
		t.Fatalf("plain filename = %q", got)
	}
	if filenameFromContentDisposition(`attachment; foo=bar`) != "" {
		t.Fatal("no filename")
	}
}

func TestWeighFilterAndValidateEdges(t *testing.T) {
	schema := feragameWeighRowSchema()
	_, err := filterFilledObjectRows(map[string]any{"a": "not-empty"}, schema)
	if err == nil {
		t.Fatal("invalid cell type")
	}
	_, err = filterFilledObjectRows(map[string]any{"a": map[string]any{"weight": "x", "panelCount": 1, "tareKind": "euro", "idoneita": "NIR"}}, schema)
	if err == nil {
		t.Fatal("bad weight")
	}
	_, err = filterFilledObjectRows(map[string]any{"a": map[string]any{"weight": 1, "panelCount": "x", "tareKind": "euro", "idoneita": "NIR"}}, schema)
	if err == nil {
		t.Fatal("bad panel")
	}
	_, err = filterFilledObjectRows(map[string]any{"a": map[string]any{"weight": 1, "panelCount": 1, "tareKind": "t", "idoneita": "NIR"}}, schema)
	if err == nil {
		t.Fatal("bad tare enum")
	}
	_, err = filterFilledObjectRows(map[string]any{"a": map[string]any{"weight": 1, "panelCount": 1, "tareKind": "euro", "idoneita": "ZZ"}}, schema)
	if err == nil {
		t.Fatal("bad idoneita")
	}
	filled, err := filterFilledObjectRows(map[string]any{
		"skip":  nil,
		"empty": "",
		"ok":    map[string]any{"weight": 2, "panelCount": 1, "tareKind": "euro", "idoneita": "IR"},
		"blank": map[string]any{},
	}, schema)
	if err != nil || len(filled) != 1 {
		t.Fatalf("filled = %#v err=%v", filled, err)
	}
	row, _ := filled["ok"].(map[string]any)
	if row["weight"] != 2.0 {
		t.Fatalf("filled = %#v", filled)
	}
	if msg := validateObjectRowFacts(nil, []string{"ok"}, schema); msg == "" {
		t.Fatal("empty facts")
	}
	if msg := validateObjectRowFacts(filled, []string{"other"}, schema); !strings.Contains(msg, "rows") {
		t.Fatalf("not in rows = %q", msg)
	}
	filled["ok"] = map[string]any{"weight": 0.0, "tareKind": "euro", "idoneita": "IR", "panelCount": 1.0}
	if msg := validateObjectRowFacts(filled, []string{"ok"}, schema); !strings.Contains(msg, "> 0") {
		t.Fatalf("weight>0 = %q", msg)
	}
	if isEmptyScalar("  ") != true || isEmptyScalar(1) {
		t.Fatal("isEmptyScalar")
	}
}

func TestProjectTaskFormErrorPathsAndPlaceholder(t *testing.T) {
	_, err := projectTaskForm(EmptyDomain(), map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
	}, "", "", nil)
	if err == nil {
		t.Fatal("dyn without reads")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{}},
		"reads":             map[string]any{},
	}, "", "", nil)
	if err == nil {
		t.Fatal("invalid dyn def")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "missing"}},
		"reads":             map[string]any{},
	}, "", "", nil)
	if err == nil {
		t.Fatal("missing read")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"enumReads": map[string]any{"sourceLoadId": "loads"},
	}, "", "", nil)
	if err == nil {
		t.Fatal("enumReads without reads")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"enumReads": map[string]any{"sourceLoadId": ""},
		"reads":     map[string]any{},
	}, "", "", nil)
	if err == nil {
		t.Fatal("empty enum read name")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"enumReads": map[string]any{"sourceLoadId": "loads"},
		"reads":     map[string]any{},
	}, "", "", nil)
	if err == nil {
		t.Fatal("missing enum read")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"enumReads": map[string]any{"sourceLoadId": "loads"},
		"reads":     map[string]any{"loads": map[string]any{"entity": "load", "key": "id"}},
		"schema":    map[string]any{"type": "object", "properties": map[string]any{}},
	}, "", "", nil)
	if err == nil {
		t.Fatal("enum property missing")
	}
	if cols := readFormColumns(nil); cols.ID != "Id" {
		t.Fatalf("default cols = %#v", cols)
	}
	if cols := readFormColumns(map[string]any{"ui:options": map[string]any{"columns": map[string]any{"id": "X", "value": "Y"}}}); cols.ID != "X" || cols.Value != "Y" {
		t.Fatalf("cols = %#v", cols)
	}
	if fieldHasPlaceholder(map[string]any{"ui:placeholder": "x"}) != true {
		t.Fatal("ui placeholder")
	}
	if fieldHasPlaceholder(map[string]any{"ui:options": map[string]any{"placeholder": "x"}}) != true {
		t.Fatal("opts placeholder")
	}
	if fieldHasPlaceholder(map[string]any{"ui:options": map[string]any{"shadcn4Text": map[string]any{"placeholder": "x"}}}) != true {
		t.Fatal("widget placeholder")
	}
	if fieldHasPlaceholder(map[string]any{}) {
		t.Fatal("no placeholder")
	}
	if defaultPlaceholderText("k", map[string]any{}) != "Enter k…" {
		t.Fatal("default placeholder key")
	}
	painted := paintAuthoredFieldMarkers(map[string]any{
		"ui:field": "mapTable",
		"additionalProperties": map[string]any{
			"child": map[string]any{"ui:field": "mapTable"},
			"raw":   1,
		},
	})
	if painted["ui:field"] != nil {
		t.Fatalf("painted field remains: %#v", painted)
	}
}

func TestParseFeragameConfigFixtureErrorPaths(t *testing.T) {
	if _, err := ParseFeragameConfigFixture([]byte(`:`)); err == nil {
		t.Fatal("bad yaml")
	}
	if _, err := ParseFeragameConfigFixture([]byte(`[]`)); err == nil {
		t.Fatal("root not mapping")
	}
	if _, err := ParseFeragameConfigFixture([]byte(`org: x`)); err == nil {
		t.Fatal("missing id")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\n")); err == nil {
		t.Fatal("missing steps")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps: []\n")); err == nil {
		t.Fatal("steps not mapping")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j: []\n")); err == nil {
		t.Fatal("job not mapping")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: maybe\n")); err == nil {
		t.Fatal("bad repeatable")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: true\n")); err == nil {
		t.Fatal("missing instanceId")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: true\n    instanceId: $.id\n")); err == nil {
		t.Fatal("missing label")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: true\n    instanceId: $.id\n    label: $.id\n    substeps: {}\n")); err == nil {
		t.Fatal("empty tasks")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: true\n    instanceId: $.id\n    label: $.id\n    substeps:\n      - {}\n")); err == nil {
		t.Fatal("task missing id")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: false\n    instanceId: $.id\n    label: $.id\n    startsWhen: []\n    substeps:\n      - id: t1\n")); err == nil {
		t.Fatal("bad startsWhen")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: false\n    instanceId: $.id\n    label: $.id\n    startsWhen:\n      quantifier: some\n    substeps:\n      - id: t1\n")); err == nil {
		t.Fatal("startsWhen missing substep")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: false\n    instanceId: $.id\n    label: $.id\n    startsWhen:\n      substep: bad\n    substeps:\n      - id: t1\n")); err == nil {
		t.Fatal("startsWhen bad path")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: false\n    instanceId: $.id\n    label: $.id\n    startsWhen:\n      substep: a/b\n      quantifier: maybe\n    substeps:\n      - id: t1\n")); err == nil {
		t.Fatal("startsWhen bad quantifier")
	}
	if _, err := ParseFeragameConfigFixture([]byte("id: x\nsteps:\n  j:\n    repeatable: true\n    instanceId: $.id\n    label: $.id\n    substeps:\n      - id: t1\n      - id: t1\n")); err == nil {
		t.Fatal("duplicate task id")
	}
	cfg, err := ParseFeragameConfigFixture([]byte(`
id: mini
org: acme
domain:
  entities:
    load: {}
steps:
  intake:
    repeatable: true
    instanceId: $.id
    label: $.id
    startLabel: Start
    substeps:
      - id: register
        roles: [administration]
`))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Org != "acme" || cfg.Domain == nil || len(cfg.Roles) != 1 {
		t.Fatalf("mini cfg = %#v", cfg)
	}

	// SeedFixtureConfigStore with missing org
	noOrg, err := ParseFeragameConfigFixture([]byte(`
id: mini
steps:
  intake:
    repeatable: true
    instanceId: $.id
    label: $.id
    substeps:
      - id: register
`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(noOrg.Org) != "" {
		t.Fatalf("expected empty org, got %q", noOrg.Org)
	}
}

func TestPrepStripsChromeFields(t *testing.T) {
	got, err := prepTaskSubmission(map[string]any{
		"chrome": map[string]any{
			"hint":          map[string]any{"kind": "readonly", "expr": "1"},
			"resultsReport": map[string]any{"kind": "dataReport", "expr": "1", "title": "R"},
			"ignoredNoExpr": map[string]any{"kind": "readonly"},
		},
		"schema": map[string]any{"properties": map[string]any{
			"codiceFir": map[string]any{"type": "string"},
		}},
	}, map[string]any{
		"codiceFir":     "FIR-1",
		"hint":          "x",
		"resultsReport": map[string]any{"sections": []any{}},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got.Facts["hint"]; ok {
		t.Fatalf("readonly chrome kept: %#v", got.Facts)
	}
	if _, ok := got.Facts["resultsReport"]; ok {
		t.Fatalf("dataReport chrome kept: %#v", got.Facts)
	}
	if got.Facts["codiceFir"] != "FIR-1" {
		t.Fatalf("facts = %#v", got.Facts)
	}
}

func TestPrepRowAndLoaderEdges(t *testing.T) {
	got, err := prepTaskSubmission(map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"reads":             map[string]any{"rows": map[string]any{"entity": "pallet", "key": "id"}},
		"schema": map[string]any{
			"required":   []any{"weights"},
			"properties": map[string]any{"weights": map[string]any{}},
		},
	}, map[string]any{}, nil)
	if err != nil || len(got.Facts) != 0 {
		t.Fatalf("full-map+empty rows = %#v %v", got, err)
	}
	progressSchema := map[string]any{"type": "string", "enum": []any{"partial", "done"}}
	_, err = prepTaskSubmission(map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"schema": map[string]any{"properties": map[string]any{
			"weights": map[string]any{"additionalProperties": progressSchema},
		}},
	}, map[string]any{"weights": map[string]any{"a": "done"}}, []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = prepTaskSubmission(map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"schema": map[string]any{"properties": map[string]any{
			"weights": map[string]any{"additionalProperties": progressSchema},
		}},
	}, map[string]any{"weights": map[string]any{"a": "nope"}}, []string{"a"})
	if err == nil {
		t.Fatal("bad progress")
	}
	_, err = prepTaskSubmission(map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"schema": map[string]any{
			"required":   []any{"weights"},
			"properties": map[string]any{"weights": map[string]any{}},
		},
	}, map[string]any{"weights": map[string]any{
		"a": map[string]any{"weight": 1.0, "panelCount": 1.0, "tareKind": "wood", "idoneita": "NIR"},
	}}, []string{"a", "b"})
	if err == nil {
		t.Fatal("full-map required missing b")
	}
	_, err = filterFilledCells(map[string]any{"a": "nope"}, map[string]any{"type": "string", "enum": []any{"partial", "done"}})
	if err == nil {
		t.Fatal("filterFilledCells")
	}
	_, err = filterFilledCells(map[string]any{"a": "done"}, map[string]any{"type": "string"})
	if err == nil {
		t.Fatal("missing enum must not silently accept progress")
	}
	if msg := validateCellFacts(nil, []string{"a"}); msg == "" {
		t.Fatal("empty progress facts")
	}
	if msg := validateCellFacts(map[string]any{"z": "done"}, []string{"a"}); msg == "" {
		t.Fatal("progress not in rows")
	}
	if rowMapValue(nil, map[string]any{"a": 1}, "")["a"] != 1 {
		t.Fatal("primary empty uses value")
	}
	if len(rowMapValue(nil, map[string]any{}, "weights")) != 0 {
		t.Fatal("missing nested map")
	}
	if additionalPropertiesSchema(map[string]any{}, "x") != nil {
		t.Fatal("missing additionalProperties")
	}
	if !additionalPropertiesIsCellMap(map[string]any{"type": "string"}) {
		t.Fatal("progress type string")
	}
	if !additionalPropertiesIsCellMap(map[string]any{"type": []any{"string", "null"}}) {
		t.Fatal("progress type union")
	}
	if additionalPropertiesIsCellMap(map[string]any{"type": "object"}) {
		t.Fatal("object not progress")
	}
	if enforceRequiredMapKeys(map[string]any{}, nil) != "" {
		t.Fatal("empty rows ok")
	}
	if msg := enforceRequiredMapKeys(map[string]any{}, []string{"a"}); msg == "" {
		t.Fatal("missing keys")
	}

	if _, err := (&Loader{}).Load(t.Context(), "", OperatorIdentity{}); err == nil {
		t.Fatal("empty org")
	}
	if _, err := (&Loader{}).Load(t.Context(), "x", OperatorIdentity{}); err == nil {
		t.Fatal("nil stores")
	}
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	failing := &Loader{
		Domains: &errDomainStore{err: errors.New("boom")},
		Configs: configs,
	}
	if _, err := failing.Load(t.Context(), "feragame", FixtureAllRolesOp()); err == nil {
		t.Fatal("domain get error")
	}
}

type errDomainStore struct{ err error }

func (e *errDomainStore) Get(context.Context, string) (Domain, error) { return Domain{}, e.err }
func (e *errDomainStore) Put(context.Context, string, Domain) error   { return e.err }

func TestChromeCompileHelpersAndUnlockEdges(t *testing.T) {
	rules := readChromeRules(map[string]any{
		"chrome": map[string]any{
			"a": map[string]any{"kind": "readonly"},
			"b": map[string]any{"kind": "readonly", "expr": "1", "title": "B"},
			"c": map[string]any{"kind": "dataReport", "expr": "  ", "title": "C"},
			"d": map[string]any{"kind": "dataReport", "expr": "1", "title": "D"},
			"e": "skip",
		},
	})
	if len(rules) != 2 {
		t.Fatalf("rules = %#v", rules)
	}
	if chromeTitle(map[string]any{"title": "T"}, "id", ChromeRule{Kind: "dataReport", Title: "R"}) != "R" {
		t.Fatal("dataReport title")
	}
	if chromeTitle(map[string]any{}, "id", ChromeRule{}) != "id" {
		t.Fatal("fallback title")
	}
	if sf, ok := stringFieldForChrome("readonly"); !ok || sf != "readonlyField" {
		t.Fatal("stringFieldForChrome readonly")
	}
	if sf, ok := stringFieldForChrome("dataReport"); !ok || sf != "dataReportField" {
		t.Fatal("stringFieldForChrome dataReport")
	}
	if sf, ok := stringFieldForChrome("x"); ok || sf != "" {
		t.Fatal("stringFieldForChrome unknown")
	}
	order := placeChromeInOrder([]string{"x"}, ChromeRule{ID: "x"})
	if len(order) != 1 {
		t.Fatal("already present")
	}
	order = placeChromeInOrder([]string{"a", "c"}, ChromeRule{ID: "b", Before: "c"})
	if strings.Join(order, ",") != "a,b,c" {
		t.Fatalf("before = %v", order)
	}
	order = placeChromeInOrder([]string{"a", "c"}, ChromeRule{ID: "b", After: "a"})
	if strings.Join(order, ",") != "a,b,c" {
		t.Fatalf("after = %v", order)
	}
	if encodeFormPath([]string{"row", "a.b", "cell"}) == "row.a.b.cell" {
		t.Fatal("dotted path should use json pointer style")
	}
	if sequenceOkForCommit(OrgConfig{}, EmptyDomain(), "missing", "t", "") {
		t.Fatal("unknown job sequence")
	}
	configs := NewMemoryConfigStore()
	if err := configs.SetActiveVersion(t.Context(), "nope", "1"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetActiveVersion = %v", err)
	}
	domains := NewMemoryDomainStore()
	if err := domains.Put(t.Context(), "o", Domain{}); err != nil {
		t.Fatal(err)
	}
	if _, err := domains.Get(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get missing = %v", err)
	}
}

func TestCommitErrorWrappingViaApply(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	var ce *CommitError
	_, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/nope", JobInstanceID: "x", Value: map[string]any{"a": 1},
	})
	if !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("unknown task = %v", err)
	}
	if _, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/truckArrive", Value: truckArriveValue(), Mint: startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}
	_, err = c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: "", Value: map[string]any{}, Mint: startingNewMint(),
	})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("startingNew non-first = %v", err)
	}
	lineOp := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"line"}}
	if _, err := c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno", Value: openTurnoValue(), Mint: startingNewMint(),
	}); err != nil {
		t.Fatal(err)
	}
	_, err = c.Commit(t.Context(), "feragame", lineOp, CommitRequest{
		TaskPath: "line/openTurno", Value: openTurnoValue(), Mint: startingNewMint(),
	})
	if !asCommitError(err, &ce) || ce.Kind != "prep" {
		t.Fatalf("duplicate openTurno instance = %v", err)
	}
}

func TestReferencedStartsWhenAndRolesHelpers(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if referencedStartsWhenTask(cfg, nil) != nil {
		t.Fatal("nil sw")
	}
	if referencedStartsWhenTask(cfg, &StartsWhen{Task: StartsWhenRef{JobID: "missing", TaskID: "x"}}) != nil {
		t.Fatal("missing job")
	}
	if referencedStartsWhenTask(cfg, &StartsWhen{Task: StartsWhenRef{JobID: "intake", TaskID: "missing"}}) != nil {
		t.Fatal("missing task")
	}
	if got := referencedStartsWhenTask(cfg, &StartsWhen{Task: StartsWhenRef{JobID: "intake", TaskID: "truckArrive"}}); got == nil {
		t.Fatal("expected task")
	}
	if rolesAllowedForTask(cfg, "missing", "x") != nil {
		t.Fatal("roles missing job")
	}
	roles := rolesAllowedForTask(cfg, "intake", "truckArrive")
	if len(roles) == 0 {
		t.Fatal("truckArrive roles")
	}
	nestedRoles := rolesAllowedForPath(cfg, TaskPathRef{JobID: "intake", GroupID: "lots", TaskID: "register"})
	if len(nestedRoles) == 0 {
		t.Fatal("lots/register roles")
	}
}

type errConfigStore struct {
	ConfigStore
	putErr error
	setErr error
}

func (e errConfigStore) Put(ctx context.Context, org, version string, cfg OrgConfig) error {
	if e.putErr != nil {
		return e.putErr
	}
	return e.ConfigStore.Put(ctx, org, version, cfg)
}

func (e errConfigStore) SetActiveVersion(ctx context.Context, org, version string) error {
	if e.setErr != nil {
		return e.setErr
	}
	return e.ConfigStore.SetActiveVersion(ctx, org, version)
}

func TestSeedFixtureConfigStorePutErrors(t *testing.T) {
	base := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(errConfigStore{ConfigStore: base, putErr: errors.New("put")}); err == nil {
		t.Fatal("put error")
	}
	base = NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(errConfigStore{ConfigStore: base, setErr: errors.New("set")}); err == nil {
		t.Fatal("set error")
	}
}

func TestBuildTaskFormBundleErrorsAndFakeHTTP(t *testing.T) {
	cfg, err := LoadFeragameConfigFixture()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{JobID: "missing", TaskID: "x"}); err == nil {
		t.Fatal("unknown job")
	}
	if _, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{JobID: "intake", TaskID: "missing"}); err == nil {
		t.Fatal("unknown task")
	}
	cfg.Jobs.Values["intake"] = "nope"
	if _, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{JobID: "intake", TaskID: "truckArrive"}); err == nil {
		t.Fatal("invalid job")
	}
	cfg, _ = LoadFeragameConfigFixture()
	intake := cfg.Jobs.Values["intake"].(Job)
	task := map[string]any(intake.Tasks[0])
	task["chrome"] = map[string]any{
		"client": map[string]any{"kind": "readonly", "expr": `"x"`},
	}
	intake.Tasks[0] = TaskConfig(task)
	cfg.Jobs.Values["intake"] = intake
	if _, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{JobID: "intake", TaskID: "register", JobInstanceID: ""}); err == nil {
		t.Fatal("chrome collision")
	}

	cfg, _ = LoadFeragameConfigFixture()
	intake = cfg.Jobs.Values["intake"].(Job)
	task = map[string]any(intake.Tasks[0])
	task["chrome"] = map[string]any{
		"report": map[string]any{"kind": "dataReport", "expr": `"not-object"`, "title": "R"},
	}
	intake.Tasks[0] = TaskConfig(task)
	cfg.Jobs.Values["intake"] = intake
	if _, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{JobID: "intake", TaskID: "truckArrive"}); err == nil {
		t.Fatal("bad dataReport dto")
	}
	task["chrome"] = map[string]any{
		"report": map[string]any{"kind": "dataReport", "expr": "1+", "title": "R"},
	}
	intake.Tasks[0] = TaskConfig(task)
	cfg.Jobs.Values["intake"] = intake
	if _, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{JobID: "intake", TaskID: "truckArrive"}); err == nil {
		t.Fatal("bad dataReport expr")
	}
	task["chrome"] = map[string]any{
		"report": map[string]any{"kind": "dataReport", "expr": `{"rows":[]}`, "title": "R"},
	}
	intake.Tasks[0] = TaskConfig(task)
	cfg.Jobs.Values["intake"] = intake
	bundle, err := BuildTaskFormBundle(cfg, EmptyDomain(), ActionableTaskRef{JobID: "intake", TaskID: "truckArrive"})
	if err != nil || bundle == nil {
		t.Fatalf("good dataReport = %v %#v", err, bundle)
	}

	port := NewFakeHttpPort()
	if _, err := port.Request(t.Context(), HttpRequest{Method: "POST", URL: "/fake/register-csv", Body: "nope"}); err == nil {
		t.Fatal("csv body type")
	}
	if _, err := port.Request(t.Context(), HttpRequest{Method: "GET", URL: "/fake/labels"}); err == nil {
		t.Fatal("unknown method")
	}
	resp, err := port.Request(t.Context(), HttpRequest{Method: "POST", URL: "http://h/fake/labels?x=1", Body: map[string]any{}})
	if err != nil || !strings.HasPrefix(resp.URL, "data:application/pdf") {
		t.Fatalf("labels = %#v %v", resp, err)
	}
	resp, err = port.Request(t.Context(), HttpRequest{Method: "POST", URL: "/fake/register-csv", Body: map[string]any{
		"client": "NO", "palletIds": []any{"a", "b"},
	}})
	if err != nil || resp.Headers["Content-Disposition"] == "" {
		t.Fatalf("csv = %#v %v", resp, err)
	}
	if deepCloneJSON(nil) != nil {
		t.Fatal("clone nil")
	}
	if cloneMapJSON(nil) == nil {
		t.Fatal("cloneMap nil -> empty")
	}
}

func TestCerbosAuthorizerErrorBranches(t *testing.T) {
	var nilAuth *CerbosAuthorizer
	if _, err := nilAuth.AllowCommit(t.Context(), AuthorizeCommit{}); err == nil {
		t.Fatal("nil authorizer")
	}
	pdp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not-json`))
	}))
	t.Cleanup(pdp.Close)
	a := NewCerbosAuthorizer(pdp.URL, pdp.Client(), nil)
	if _, err := a.AllowCommit(t.Context(), AuthorizeCommit{TaskPath: "", JobID: "j", TaskID: "t"}); err == nil {
		t.Fatal("decode error")
	}
	pdp2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	t.Cleanup(pdp2.Close)
	a2 := NewCerbosAuthorizer(pdp2.URL, pdp2.Client(), nil)
	if _, err := a2.AllowCommit(t.Context(), AuthorizeCommit{Operator: OperatorIdentity{UserID: "u"}, TaskPath: "a/b"}); err == nil {
		t.Fatal("status error")
	}
	pdp3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"resourceInstances": map[string]any{}})
	}))
	t.Cleanup(pdp3.Close)
	a3 := NewCerbosAuthorizer(pdp3.URL, pdp3.Client(), nil)
	ok, err := a3.AllowCommit(t.Context(), AuthorizeCommit{Operator: OperatorIdentity{UserID: "u"}, TaskPath: "a/b"})
	if err != nil || ok {
		t.Fatalf("missing action = %v %v", ok, err)
	}
}

type errAuthorizer struct{}

func (errAuthorizer) AllowCommit(context.Context, AuthorizeCommit) (bool, error) {
	return false, errors.New("auth boom")
}

func TestCommitAuthorizerErrorAndReadyWhen(t *testing.T) {
	c, _, _ := testCommitter(t, errAuthorizer{})
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{UserID: "ada", Org: "feragame"}, CommitRequest{
		TaskPath: "intake/truckArrive", Value: truckArriveValue(), Mint: startingNewMint(),
	})
	if err == nil || !strings.Contains(err.Error(), "auth boom") {
		t.Fatalf("authorizer err = %v", err)
	}
	if readyWhenFromTask(nil) != nil {
		t.Fatal("nil task readyWhen")
	}
	if readyWhenOkForCommit(EmptyDomain(), map[string]any{"readyWhen": 1}, "", nil) {
		t.Fatal("non-string readyWhen")
	}
	if !readyWhenOkForCommit(EmptyDomain(), map[string]any{"readyWhen": "   "}, "", nil) {
		t.Fatal("blank readyWhen")
	}
	if readyWhenOkForCommit(EmptyDomain(), map[string]any{"readyWhen": "1+"}, "", nil) {
		t.Fatal("bad readyWhen expr")
	}
	noSub := false
	_, err = Mutate(EmptyDomain(), MutateRequest{
		AppendSubmission: &noSub,
		Ops: []MutateOp{
			MintOp{Entity: EntityType("pallet"), ID: "p1", Set: map[string]any{"x": 1}},
			MintOp{Entity: EntityType("container"), ID: "c1", Set: map[string]any{"x": 1}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestConfigJSONDecodeErrorLines(t *testing.T) {
	var cfg OrgConfig
	if err := json.Unmarshal([]byte(`{"id":`), &cfg); err == nil {
		t.Fatal("truncated json")
	}
	if err := json.Unmarshal([]byte(`{"id":"x","version":"1","jobs":{"j":{"repeatable":true,"instanceId":"a","label":"b","tasks":[{"id":"t1"},{"id":"t1"}]}}}`), &cfg); err == nil {
		t.Fatal("dup tasks via job unmarshal")
	}
	jobs, err := decodeOrderedJobs(json.RawMessage(`{"k":`))
	if err == nil {
		t.Fatalf("truncated job value: %#v", jobs)
	}
	_, err = decodeOrderedJobs(json.RawMessage(`{"k": true}`))
	if err == nil {
		t.Fatal("job not object")
	}
}

func TestRowReadHelpersAndWorklistEdges(t *testing.T) {
	_, ok, err := rowReadFromTask(map[string]any{})
	if err != nil || ok {
		t.Fatalf("no primary = %v %v", ok, err)
	}
	_, ok, err = rowReadFromTask(map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
	})
	if err == nil || ok {
		t.Fatal("missing reads")
	}
	_, ok, err = rowReadFromTask(map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"reads":             map[string]any{},
	})
	if err == nil || ok {
		t.Fatal("missing def read")
	}
	def, ok, err := rowReadFromTask(map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"reads":             map[string]any{"rows": map[string]any{"entity": "pallet", "key": "id"}},
	})
	if err != nil || !ok || def.Entity != EntityType("pallet") {
		t.Fatalf("row read = %#v %v %v", def, ok, err)
	}
	if _, err := rowIDsForTask(EmptyDomain(), "intake", map[string]any{
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"reads":             map[string]any{"rows": "bad"},
	}, "", nil); err == nil {
		t.Fatal("bad row read def")
	}
}

func TestCoverageBufferHelpersAndHTTP(t *testing.T) {
	if fakeHTTPPath("") != "" || fakeHTTPPath("  ") != "" {
		t.Fatal("empty fakeHTTPPath")
	}
	if fakeHTTPPath("https://ex.test/a/b?x=1") != "/a/b" {
		t.Fatal("url path")
	}
	if fakeHTTPPath("/local?x=1") != "/local" {
		t.Fatal("query strip")
	}
	if fakeHTTPPath("plain") != "plain" {
		t.Fatal("raw path")
	}
	if csvEscape(nil) != "" || csvEscape(`a"b,c`) != `"a""b,c"` {
		t.Fatal("csvEscape")
	}
	if deepCloneJSON(nil) != nil {
		t.Fatal("nil clone")
	}
	if deepCloneJSON(make(chan int)) != nil {
		t.Fatal("unmarshalable clone")
	}
	if TaskConfig(nil).ID() != "" {
		t.Fatal("nil task id")
	}
	em, err := listEntities(EmptyDomain(), EntityType("nope"))
	if err != nil || len(em) != 0 {
		t.Fatalf("missing entity map = %#v %v", em, err)
	}
	if _, ok := dynamicPropertyDef(map[string]any{}, "x"); ok {
		t.Fatal("dyn nil")
	}
	if _, ok := dynamicPropertyDef(map[string]any{"dynamicProperties": map[string]any{"x": "bad"}}, "x"); ok {
		t.Fatal("dyn not object")
	}
	if additionalPropertiesIsCellMap(map[string]any{"anyOf": []any{"nope", map[string]any{"type": "number"}}}) {
		t.Fatal("anyOf must not classify")
	}
	if !additionalPropertiesIsCellMap(map[string]any{"anyOf": []any{map[string]any{"type": "string"}}}) {
		t.Fatal("anyOf string must classify as cell map")
	}
	if msg := enforceRequiredMapKeys(map[string]any{}, []string{"a", "b"}); !strings.Contains(msg, "a, b") {
		t.Fatalf("joinComma via required keys = %q", msg)
	}
	if _, ok := taskOnSubmitTransform(map[string]any{}); ok {
		t.Fatal("no onSubmit")
	}
	_, _, err = lookupTask(OrgConfig{}, "missing", "t")
	if err == nil {
		t.Fatal("unknown job")
	}
	badJobs := newOrderedObject()
	badJobs.Set("j", "not-a-job")
	_, _, err = lookupTask(OrgConfig{Jobs: badJobs}, "j", "t")
	if err == nil {
		t.Fatal("invalid job")
	}
	goodJobs := newOrderedObject()
	goodJobs.Set("j", Job{Tasks: []TaskConfig{{"id": "only"}}})
	_, _, err = lookupTask(OrgConfig{Jobs: goodJobs}, "j", "missing")
	if err == nil {
		t.Fatal("unknown task")
	}
	var o OrderedObject
	o.Set("k", 1)
	if got, _ := o.Get("k"); got != 1 || o.Len() != 1 {
		t.Fatal("Set on nil Values")
	}
	ui := map[string]any{}
	paintHideRootFormTitle(ui)
	if opts, _ := ui["ui:options"].(map[string]any); opts["hideTitle"] != true {
		t.Fatalf("hideTitle = %#v", ui)
	}
	paintHideRootFormTitle(ui)
	expanded := expandMapDerived(map[string]DerivedRule{
		"plain":          {Expr: "1"},
		"weights.*.mass": {Expr: "$lookup(weights, __ROW_ID__).mass"},
	}, []string{"p1"})
	if expanded["plain"].Expr != "1" || expanded[encodeFormPath([]string{"weights", "p1", "mass"})].Expr == "" {
		t.Fatalf("expandMapDerived = %#v", expanded)
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"schema":            map[string]any{"type": "object", "properties": "bad"},
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"reads":             map[string]any{"rows": map[string]any{"entity": "pallet", "key": "id"}},
	}, "", "", nil)
	if err == nil {
		t.Fatal("props not map")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"schema":            map[string]any{"type": "object", "properties": map[string]any{}},
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"reads":             map[string]any{"rows": "bad"},
	}, "", "", nil)
	if err == nil {
		t.Fatal("bad read def in project")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"schema":            map[string]any{"type": "object", "properties": map[string]any{}},
		"dynamicProperties": map[string]any{"weights": map[string]any{"read": "rows"}},
		"reads":             map[string]any{"rows": map[string]any{"entity": "pallet", "key": "id"}},
	}, "", "", nil)
	if err == nil {
		t.Fatal("missing property schema")
	}
	_, err = projectTaskForm(EmptyDomain(), map[string]any{
		"enumReads": map[string]any{"sourceLoadId": "loads"},
		"reads":     map[string]any{"loads": "bad"},
		"schema":    map[string]any{"type": "object", "properties": map[string]any{"sourceLoadId": map[string]any{"type": "string"}}},
	}, "", "", nil)
	if err == nil {
		t.Fatal("bad enum read def")
	}
	if rolesAllowedForTask(OrgConfig{}, "missing", "t") != nil {
		t.Fatal("roles missing job")
	}
	if rolesAllowedForTask(OrgConfig{Jobs: badJobs}, "j", "t") != nil {
		t.Fatal("roles invalid job")
	}
	if rolesAllowedForTask(OrgConfig{Jobs: goodJobs}, "j", "missing") != nil {
		t.Fatal("roles missing task")
	}
	if rolesFromTask(nil) != nil || rolesFromTask(map[string]any{}) != nil {
		t.Fatal("rolesFromTask empty")
	}
	if rolesFromTask(map[string]any{"roles": "bad"}) != nil {
		t.Fatal("roles not list")
	}
	if got := rolesFromTask(map[string]any{"roles": []any{"a", 1, ""}}); len(got) != 1 || got[0] != "a" {
		t.Fatalf("rolesFromTask filter = %#v", got)
	}
	ids := palletWeightRowIDs(EmptyDomain(), "nope")
	if ids == nil {
		ids = []string{}
	}
	_ = lineLoadRowIDs(EmptyDomain())
	wl := ProjectWorklist(OrgConfig{Jobs: badJobs}, EmptyDomain(), FixtureAllRolesOp())
	if len(wl.Jobs) != 0 {
		t.Fatalf("skip invalid jobs = %#v", wl)
	}
	cfg := OrgConfig{Jobs: goodJobs}
	wl = ProjectWorklist(cfg, Domain{JobInstances: []JobInstance{{JobID: "other", ID: "x"}}}, FixtureAllRolesOp())
	if len(wl.Jobs) != 1 || len(wl.Jobs[0].Instances) != 0 {
		t.Fatalf("filter instances = %#v", wl)
	}

	gateOK := func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada", Org: "feragame"}, true
	}
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	boomLoad := &Handler{
		Loader: &Loader{Domains: &errDomainStore{err: errors.New("boom")}, Configs: configs},
		Gate:   gateOK,
	}
	rec := httptest.NewRecorder()
	boomLoad.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/domain", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("load boom = %d", rec.Code)
	}
	domains := NewMemoryDomainStore()
	c := &Committer{Domains: domains, Configs: configs, Authorizer: AllowAllAuthorizer{}}
	h := &Handler{Loader: &Loader{Domains: domains, Configs: configs}, Committer: c, Gate: gateOK}
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", errReader{})
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("commit read err = %d", rec.Code)
	}
	denyGate := &Handler{Loader: &Loader{Domains: domains, Configs: configs}, Committer: c, Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		http.Error(w, "no", http.StatusUnauthorized)
		return OperatorIdentity{}, false
	}}
	rec = httptest.NewRecorder()
	denyGate.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/living/domain", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("gate deny load = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	denyGate.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(`{}`)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("gate deny commit = %d", rec.Code)
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("read fail") }
