package living

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// groupIntakeConfigWithLabels mirrors groupIntakeConfig but authors /fake/labels on the
// lot register leaf — same Sidecar path flat Jobs use on rework.
func groupIntakeConfigWithLabels() OrgConfig {
	cfg := groupIntakeConfig()
	intake := cfg.Jobs.Values["intake"].(Job)
	lots := intake.Tasks[1]
	register := lots["tasks"].([]TaskConfig)[0]
	register["onSubmit"] = map[string]any{
		"transform": []any{},
		"http": []any{
			map[string]any{
				"method": `"POST"`,
				"url":    `"/fake/labels"`,
				"label":  `"Stampa etichette"`,
				"action": "link",
				"body":   `{"lotId":"lot-sidecar"}`,
			},
		},
	}
	lots["tasks"] = []TaskConfig{register}
	intake.Tasks[1] = lots
	cfg.Jobs.Values["intake"] = intake
	return cfg
}

func groupSidecarCommitter(t *testing.T, httpPort HttpPort) (*Committer, *Loader, *MemoryDomainStore) {
	t.Helper()
	domains := NewMemoryDomainStore()
	configs := NewMemoryConfigStore()
	cfg := groupIntakeConfigWithLabels()
	if err := configs.Put(t.Context(), "grouporg", "v1", cfg); err != nil {
		t.Fatal(err)
	}
	if err := configs.SetActiveVersion(t.Context(), "grouporg", "v1"); err != nil {
		t.Fatal(err)
	}
	c := &Committer{
		Domains:    domains,
		Configs:    configs,
		Authorizer: AllowAllAuthorizer{},
		HttpPort:   httpPort,
		Now:        func() time.Time { return time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC) },
	}
	loader := &Loader{Domains: domains, Configs: configs}
	return c, loader, domains
}

func TestCommitGroupLeafSidecarLabelsAndProxy(t *testing.T) {
	const artifactID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	var labelsCalls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/fake/labels":
			labelsCalls++
			if got := r.Header.Get("Authorization"); got != "Bearer tok" {
				t.Errorf("labels auth = %q", got)
			}
			if got := r.Header.Get("X-Attesta-Org"); got != "grouporg" {
				t.Errorf("labels org = %q", got)
			}
			_, _ = io.WriteString(w, "/artifacts/"+artifactID)
		case r.Method == http.MethodGet && r.URL.Path == "/artifacts/"+artifactID:
			if got := r.Header.Get("Authorization"); got != "Bearer tok" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.Header().Set("X-Attesta-Org", "grouporg")
			w.Header().Set("Content-Type", "application/pdf")
			_, _ = io.WriteString(w, "%PDF-group-leaf")
		default:
			http.Error(w, "unexpected "+r.Method+" "+r.URL.Path, http.StatusNotFound)
		}
	}))
	t.Cleanup(upstream.Close)

	port := NewSidecarHttpPort(upstream.URL, "tok", upstream.Client())
	c, _, domains := groupSidecarCommitter(t, port)
	op := OperatorIdentity{UserID: "ada", Org: "grouporg", Roles: []string{"administration"}}

	arrive, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath: "intake/truckArrive", Value: map[string]any{}, Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID

	spawned, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         map[string]any{},
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if labelsCalls != 1 {
		t.Fatalf("Sidecar /fake/labels calls = %d, want 1", labelsCalls)
	}
	if len(spawned.HttpErrors) != 0 {
		t.Fatalf("httpErrors = %#v", spawned.HttpErrors)
	}
	if len(spawned.Links) != 1 {
		t.Fatalf("links = %#v", spawned.Links)
	}
	wantURL := ArtifactProxyPathPrefix + artifactID
	if spawned.Links[0].URL != wantURL || spawned.Links[0].Label != "Stampa etichette" || spawned.Links[0].Action != HttpActionLink {
		t.Fatalf("link = %#v, want url %q", spawned.Links[0], wantURL)
	}
	if len(spawned.Domain.GroupInstances) != 1 || spawned.Domain.GroupInstances[0].ID != "lots-1" {
		t.Fatalf("groupInstances = %#v", spawned.Domain.GroupInstances)
	}
	stored, err := domains.Get(t.Context(), "grouporg")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Submissions) == 0 {
		t.Fatal("Domain missing nested submission after Sidecar Commit")
	}

	h := &Handler{
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return op, true
		},
		Artifacts: port,
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, wantURL, nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "%PDF-group-leaf" {
		t.Fatalf("artifact proxy = %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("content-type = %q", ct)
	}
}

func TestCommitGroupLeafUnconfiguredHttpSoftFail(t *testing.T) {
	c, _, domains := groupSidecarCommitter(t, UnconfiguredHttpPort{})
	op := OperatorIdentity{UserID: "ada", Org: "grouporg", Roles: []string{"administration"}}

	arrive, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath: "intake/truckArrive", Value: map[string]any{}, Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID

	spawned, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         map[string]any{},
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatalf("Commit must succeed when living-http unset: %v", err)
	}
	if len(spawned.Links) != 0 {
		t.Fatalf("links = %#v", spawned.Links)
	}
	if len(spawned.HttpErrors) != 1 || !strings.Contains(spawned.HttpErrors[0], "not configured") {
		t.Fatalf("httpErrors = %#v", spawned.HttpErrors)
	}
	if len(spawned.Domain.GroupInstances) != 1 {
		t.Fatalf("Domain rolled back group spawn? %#v", spawned.Domain.GroupInstances)
	}
	stored, err := domains.Get(t.Context(), "grouporg")
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.GroupInstances) != 1 || stored.GroupInstances[0].ID != "lots-1" {
		t.Fatalf("DomainStore rolled back: %#v", stored.GroupInstances)
	}
}

func TestCommitGroupLeafSidecarViaHTTPHandler(t *testing.T) {
	const artifactID = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fake/labels" {
			http.Error(w, "nope", http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, "/artifacts/"+artifactID)
	}))
	t.Cleanup(upstream.Close)

	port := NewSidecarHttpPort(upstream.URL, "tok", upstream.Client())
	c, loader, _ := groupSidecarCommitter(t, port)
	op := OperatorIdentity{UserID: "ada", Org: "grouporg", Roles: []string{"administration"}}

	arrive, err := c.Commit(t.Context(), "grouporg", op, CommitRequest{
		TaskPath: "intake/truckArrive", Value: map[string]any{}, Mint: startingNewMint(),
	})
	if err != nil {
		t.Fatal(err)
	}
	truckID := arrive.JobInstanceID

	h := &Handler{
		Loader:    loader,
		Committer: c,
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return op, true
		},
		Artifacts: port,
	}
	body, err := json.Marshal(map[string]any{
		"taskPath":      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		"jobInstanceId": truckID,
		"mint":          map[string]any{"kind": "spawningNew"},
		"value":         map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/living/commit", strings.NewReader(string(body))))
	if rec.Code != http.StatusOK {
		t.Fatalf("commit status = %d body=%s", rec.Code, rec.Body.String())
	}
	var wire map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	links, _ := wire["links"].([]any)
	if len(links) != 1 {
		t.Fatalf("wire links = %#v", wire["links"])
	}
	link0, _ := links[0].(map[string]any)
	if link0["url"] != ArtifactProxyPathPrefix+artifactID {
		t.Fatalf("wire link0 = %#v", link0)
	}
}
