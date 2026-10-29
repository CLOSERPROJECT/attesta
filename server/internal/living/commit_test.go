package living

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func registerValue() map[string]any {
	return map[string]any{
		"client":             "NO",
		"date":               "2026-10-19",
		"wasteCode":          "191212",
		"idoneitaRiutilizzo": "NIR",
		"truckWeight":        100.0,
		"truckTare":          200.0,
		"palletCount":        2.0,
		"notes":              "",
	}
}

func testCommitter(t *testing.T, auth Authorizer) (*Committer, *MemoryDomainStore, *Loader) {
	t.Helper()
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	if _, err := SeedFixtureConfigStore(configs); err != nil {
		t.Fatal(err)
	}
	if auth == nil {
		auth = AllowAllAuthorizer{}
	}
	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: auth,
		Now:        func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) },
	}
	return c, domains, &Loader{Domains: domains, Configs: configs}
}

func TestCommitRegisterPersistsAndLoadReflects(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	op := OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}
	instanceID := "NIR-261019-NO-1"

	result, err := c.Commit(t.Context(), "feragame", op, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: instanceID,
		Value:         registerValue(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.JobInstanceID != instanceID {
		t.Fatalf("jobInstanceId = %q", result.JobInstanceID)
	}
	if result.Domain.Load[instanceID]["client"] != "NO" {
		t.Fatalf("load = %#v", result.Domain.Load)
	}
	if len(result.Domain.Pallet) != 2 {
		t.Fatalf("pallets = %#v", result.Domain.Pallet)
	}
	if len(result.Domain.Submissions) != 1 {
		t.Fatalf("submissions = %#v", result.Domain.Submissions)
	}
	sub := result.Domain.Submissions[0]
	if sub.JobID != "intake" || sub.TaskID != "register" || sub.JobInstanceID != instanceID {
		t.Fatalf("submission path keys = %#v", sub)
	}
	if TaskPath(sub.JobID, sub.TaskID) != "intake/register" {
		t.Fatalf("task path = %s/%s", sub.JobID, sub.TaskID)
	}

	loaded, err := loader.Load(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Domain.Load[instanceID]["truckWeight"] != 100.0 {
		t.Fatalf("loaded domain = %#v", loaded.Domain)
	}
	if len(loaded.Domain.JobInstances) != 1 || loaded.Domain.JobInstances[0].ID != instanceID {
		t.Fatalf("jobInstances = %#v", loaded.Domain.JobInstances)
	}
}

func TestCommitRejectsWhenAuthorizerDenies(t *testing.T) {
	c, domains, _ := testCommitter(t, DenyAllAuthorizer{})
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{UserID: "gio", Org: "feragame"}, CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         registerValue(),
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unauthorized" {
		t.Fatalf("err = %v", err)
	}
	_, getErr := domains.Get(t.Context(), "feragame")
	if getErr == nil {
		t.Fatal("domain should remain unset when Authorizer denies")
	}
}

func TestCommitRejectsRunIdShapedUnsupportedPath(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	_, err := c.Commit(t.Context(), "feragame", OperatorIdentity{UserID: "ada", Org: "feragame"}, CommitRequest{
		TaskPath:      "intake/codiceFir",
		JobInstanceID: "NIR-261019-NO-1",
		Value:         map[string]any{"codiceFir": "FIR-1"},
	})
	var ce *CommitError
	if err == nil || !asCommitError(err, &ce) || ce.Kind != "unknown-target" {
		t.Fatalf("err = %v", err)
	}
}

func TestHandleCommitHappyPath(t *testing.T) {
	c, _, loader := testCommitter(t, AllowAllAuthorizer{})
	h := &Handler{
		Loader:    loader,
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada", Org: "feragame", Roles: []string{"administration"}}, true
		},
	}

	body := `{"taskPath":"intake/register","jobInstanceId":"NIR-261019-NO-1","value":{"client":"NO","date":"2026-10-19","wasteCode":"191212","idoneitaRiutilizzo":"NIR","truckWeight":100,"truckTare":200,"palletCount":1,"notes":""}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(body))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	var got CommitResult
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Domain.Load["NIR-261019-NO-1"]["client"] != "NO" {
		t.Fatalf("got = %#v", got)
	}

	loadRec := httptest.NewRecorder()
	loadReq := httptest.NewRequest(http.MethodGet, "/api/living/domain", nil)
	h.ServeHTTP(loadRec, loadReq)
	if loadRec.Code != http.StatusOK {
		t.Fatalf("load status = %d", loadRec.Code)
	}
	var loaded LoadResult
	if err := json.Unmarshal(loadRec.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Domain.Submissions) != 1 {
		t.Fatalf("loaded submissions = %#v", loaded.Domain.Submissions)
	}
}

func TestHandleCommitRejectsWhenGateDenies(t *testing.T) {
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	h := &Handler{
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return OperatorIdentity{}, false
		},
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(`{}`))
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}

func asCommitError(err error, dest **CommitError) bool {
	if err == nil {
		return false
	}
	ce, ok := err.(*CommitError)
	if !ok {
		return false
	}
	*dest = ce
	return true
}
