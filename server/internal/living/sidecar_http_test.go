package living

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSidecarHttpPortSuccessRewritesArtifactURL(t *testing.T) {
	var gotAuth, gotOrg, gotPath, gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotOrg = r.Header.Get("X-Attesta-Org")
		gotPath = r.URL.Path
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Disposition", `attachment; filename="reg.csv"`)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, "/artifacts/0123456789abcdef0123456789abcdef")
	}))
	t.Cleanup(srv.Close)

	port := NewSidecarHttpPort(srv.URL, "secret", srv.Client())
	resp, err := port.Request(t.Context(), HttpRequest{
		Method: "POST",
		URL:    "/fake/register-csv",
		Org:    "feragame",
		Body:   map[string]any{"lotId": "NIR-1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer secret" || gotOrg != "feragame" || gotPath != "/fake/register-csv" {
		t.Fatalf("request auth=%q org=%q path=%q", gotAuth, gotOrg, gotPath)
	}
	if gotCT != "application/json" || !strings.Contains(string(gotBody), `"lotId":"NIR-1"`) {
		t.Fatalf("body ct=%q body=%s", gotCT, gotBody)
	}
	want := ArtifactProxyPathPrefix + "0123456789abcdef0123456789abcdef"
	if resp.URL != want {
		t.Fatalf("rewritten url = %q want %q", resp.URL, want)
	}
	if resp.Headers["Content-Disposition"] == "" {
		t.Fatalf("headers = %#v", resp.Headers)
	}
}

func TestSidecarHttpPortRejectsAbsoluteURL(t *testing.T) {
	port := NewSidecarHttpPort("http://living-http:8090", "t", http.DefaultClient)
	_, err := port.Request(t.Context(), HttpRequest{Method: "POST", URL: "https://evil.example/x"})
	if err == nil || !strings.Contains(err.Error(), "absolute URL") {
		t.Fatalf("err = %v", err)
	}
	_, err = port.Request(t.Context(), HttpRequest{Method: "POST", URL: "//evil.example/x"})
	if err == nil || !strings.Contains(err.Error(), "absolute URL") {
		t.Fatalf("protocol-relative err = %v", err)
	}
}

func TestSidecarHttpPortSoftErrorOnUpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusBadGateway)
	}))
	t.Cleanup(srv.Close)
	port := NewSidecarHttpPort(srv.URL, "t", srv.Client())
	_, err := port.Request(t.Context(), HttpRequest{Method: "POST", URL: "/fake/labels"})
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("err = %v", err)
	}
}

func TestSidecarHttpPortTimeoutSoftError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		_, _ = io.WriteString(w, "/artifacts/0123456789abcdef0123456789abcdef")
	}))
	t.Cleanup(srv.Close)
	client := srv.Client()
	client.Timeout = 20 * time.Millisecond
	port := NewSidecarHttpPort(srv.URL, "t", client)
	_, err := port.Request(t.Context(), HttpRequest{Method: "POST", URL: "/fake/labels"})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestUnconfiguredHttpPortSoftError(t *testing.T) {
	_, err := UnconfiguredHttpPort{}.Request(t.Context(), HttpRequest{Method: "POST", URL: "/fake/labels"})
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("err = %v", err)
	}
}

func TestCommitHTTPUnconfiguredDoesNotRollbackDomain(t *testing.T) {
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = UnconfiguredHttpPort{}

	truckID := commitTruckArrive(t, c)
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         registerLotValue(),
	})
	if err != nil {
		t.Fatalf("Commit must succeed: %v", err)
	}
	if result.Domain.Kind("lot")[firstLoadID(result.Domain)]["client"] != "CO" {
		t.Fatalf("domain missing: %#v", result.Domain.Kind("lot"))
	}
	if len(result.Links) != 0 {
		t.Fatalf("links = %#v", result.Links)
	}
	if len(result.HttpErrors) != 2 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}
	stored, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("lot")[firstLoadID(stored)]["client"] != "CO" {
		t.Fatalf("DomainStore rolled back: %#v", stored)
	}
}

func TestCommitSidecarHttpPortSuccessLinks(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		id := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
		if n == 2 {
			id = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
			w.Header().Set("Content-Disposition", `attachment; filename="NIR-261019-CO-1.csv"`)
		}
		_, _ = io.WriteString(w, "/artifacts/"+id)
	}))
	t.Cleanup(srv.Close)

	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = NewSidecarHttpPort(srv.URL, "tok", srv.Client())

	truckID := commitTruckArrive(t, c)
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         registerLotValue(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.HttpErrors) != 0 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}
	if len(result.Links) != 2 {
		t.Fatalf("links = %#v", result.Links)
	}
	if result.Links[0].URL != ArtifactProxyPathPrefix+"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("link0 = %#v", result.Links[0])
	}
	if result.Links[1].URL != ArtifactProxyPathPrefix+"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("link1 = %#v", result.Links[1])
	}
	if result.Links[1].Filename != "NIR-261019-CO-1.csv" {
		t.Fatalf("filename = %q", result.Links[1].Filename)
	}
	if n != 2 {
		t.Fatalf("upstream calls = %d", n)
	}
}

func TestHandlerArtifactProxyOrgGate(t *testing.T) {
	id := "cccccccccccccccccccccccccccccccc"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-Attesta-Org", "feragame")
		w.Header().Set("Content-Type", "application/pdf")
		_, _ = io.WriteString(w, "%PDF-proxy")
	}))
	t.Cleanup(upstream.Close)

	h := &Handler{
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "ada", Org: "feragame"}, true
		},
		Artifacts: NewSidecarHttpPort(upstream.URL, "secret", upstream.Client()),
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "%PDF-proxy" {
		t.Fatalf("proxy = %d %q", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("content-type = %q", ct)
	}

	wrongOrg := &Handler{
		Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
			return OperatorIdentity{UserID: "bob", Org: "other"}, true
		},
		Artifacts: NewSidecarHttpPort(upstream.URL, "secret", upstream.Client()),
	}
	rec = httptest.NewRecorder()
	wrongOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("wrong org status = %d", rec.Code)
	}
}

func TestFetchArtifact(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Attesta-Org", "feragame")
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", `attachment; filename="a.csv"`)
		_, _ = io.WriteString(w, "a,b\n")
	}))
	t.Cleanup(srv.Close)
	port := NewSidecarHttpPort(srv.URL, "t", srv.Client())
	art, err := port.FetchArtifact(context.Background(), "dddddddddddddddddddddddddddddddd")
	if err != nil {
		t.Fatal(err)
	}
	if art.OrgID != "feragame" || art.Filename != "a.csv" || string(art.Body) != "a,b\n" {
		t.Fatalf("artifact = %#v", art)
	}
}

func TestRequirePathOnlyURL(t *testing.T) {
	if _, err := requirePathOnlyURL("/fake/labels"); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"http://x/y", "https://x/y", "//x/y", "fake/labels", ""} {
		if _, err := requirePathOnlyURL(bad); err == nil {
			t.Fatalf("expected reject for %q", bad)
		}
	}
}

func TestHttpPortFromEnv(t *testing.T) {
	t.Setenv("LIVING_HTTP_URL", "")
	t.Setenv("LIVING_HTTP_TOKEN", "")
	if _, ok := HttpPortFromEnv().(UnconfiguredHttpPort); !ok {
		t.Fatalf("empty URL type = %T", HttpPortFromEnv())
	}

	t.Setenv("LIVING_HTTP_URL", " http://living-http:8090/ ")
	t.Setenv("LIVING_HTTP_TOKEN", " secret ")
	port, ok := HttpPortFromEnv().(*SidecarHttpPort)
	if !ok || port.BaseURL != "http://living-http:8090" || port.Token != "secret" {
		t.Fatalf("env port = %#v", port)
	}
	if port.HTTP == nil || port.HTTP.Timeout != defaultSidecarTimeout {
		t.Fatalf("default client = %#v", port.HTTP)
	}
}

func TestNewSidecarHttpPortDefaultClient(t *testing.T) {
	port := NewSidecarHttpPort("http://example:8090", "t", nil)
	if port.HTTP == nil || port.HTTP.Timeout != defaultSidecarTimeout {
		t.Fatalf("client = %#v", port.HTTP)
	}
}

func TestSidecarHttpPortRequestEdgeErrors(t *testing.T) {
	var nilPort *SidecarHttpPort
	if _, err := nilPort.Request(t.Context(), HttpRequest{Method: "GET", URL: "/x"}); err == nil {
		t.Fatal("nil port")
	}
	empty := &SidecarHttpPort{BaseURL: "", HTTP: http.DefaultClient}
	if _, err := empty.Request(t.Context(), HttpRequest{Method: "GET", URL: "/x"}); err == nil {
		t.Fatal("empty base")
	}
	if _, err := NewSidecarHttpPort("://bad", "t", http.DefaultClient).Request(t.Context(), HttpRequest{
		Method: "GET", URL: "/x",
	}); err == nil {
		t.Fatal("invalid base")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	t.Cleanup(srv.Close)
	port := NewSidecarHttpPort(srv.URL, "", srv.Client())
	resp, err := port.Request(t.Context(), HttpRequest{Method: "GET", URL: "/ping"})
	if err != nil || resp.URL != "ok" {
		t.Fatalf("no-body GET = %#v err=%v", resp, err)
	}
}

func TestFetchArtifactErrors(t *testing.T) {
	var nilPort *SidecarHttpPort
	if _, err := nilPort.FetchArtifact(t.Context(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err == nil {
		t.Fatal("nil port")
	}
	port := NewSidecarHttpPort("http://example:8090", "t", http.DefaultClient)
	if _, err := port.FetchArtifact(t.Context(), "not-hex"); err == nil || !strings.Contains(err.Error(), "invalid artifact") {
		t.Fatalf("invalid id err = %v", err)
	}
	if _, err := (&SidecarHttpPort{BaseURL: "://bad", HTTP: http.DefaultClient}).FetchArtifact(
		t.Context(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	); err == nil {
		t.Fatal("invalid base")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/artifacts/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa":
			http.NotFound(w, r)
		case "/artifacts/bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb":
			http.Error(w, "nope", http.StatusBadGateway)
		default:
			http.Error(w, "unexpected", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	port = NewSidecarHttpPort(srv.URL, "t", srv.Client())
	if _, err := port.FetchArtifact(t.Context(), "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("404 err = %v", err)
	}
	if _, err := port.FetchArtifact(t.Context(), "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"); err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("502 err = %v", err)
	}
}

func TestRewriteJoinValidArtifactHelpers(t *testing.T) {
	if got := rewriteSidecarArtifactURL("/artifacts/not-valid-id"); got != "/artifacts/not-valid-id" {
		t.Fatalf("rewrite invalid = %q", got)
	}
	if got := rewriteSidecarArtifactURL("  /artifacts/cccccccccccccccccccccccccccccccc  "); got != ArtifactProxyPathPrefix+"cccccccccccccccccccccccccccccccc" {
		t.Fatalf("rewrite = %q", got)
	}
	if _, err := joinBasePath("", "/x"); err == nil {
		t.Fatal("empty base")
	}
	if _, err := joinBasePath("not-a-url", "/x"); err == nil {
		t.Fatal("invalid base")
	}
	got, err := joinBasePath("http://living-http:8090", "/artifacts/x")
	if err != nil || got != "http://living-http:8090/artifacts/x" {
		t.Fatalf("join = %q err=%v", got, err)
	}
	if validArtifactID("short") || validArtifactID("gggggggggggggggggggggggggggggggg") {
		t.Fatal("validArtifactID should reject")
	}
	if !validArtifactID("0123456789abcdef0123456789abcdef") {
		t.Fatal("validArtifactID should accept")
	}
}

func TestHandlerArtifactProxyErrorArms(t *testing.T) {
	id := "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	gateOK := Gate(func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada", Org: "feragame"}, true
	})

	rec := httptest.NewRecorder()
	(&Handler{Gate: gateOK}).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("method = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&Handler{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("nil gate = %d", rec.Code)
	}

	deny := &Handler{Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		http.Error(w, "no", http.StatusUnauthorized)
		return OperatorIdentity{}, false
	}}
	rec = httptest.NewRecorder()
	deny.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("deny gate = %d", rec.Code)
	}

	emptyOrg := &Handler{Gate: func(w http.ResponseWriter, r *http.Request) (OperatorIdentity, bool) {
		return OperatorIdentity{UserID: "ada"}, true
	}}
	rec = httptest.NewRecorder()
	emptyOrg.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("empty org = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&Handler{Gate: gateOK}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+"bad-id", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bad id = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	(&Handler{Gate: gateOK}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil artifacts = %d", rec.Code)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/ffffffffffffffffffffffffffffffff"):
			http.NotFound(w, r)
		case strings.HasSuffix(r.URL.Path, "/11111111111111111111111111111111"):
			http.Error(w, "boom", http.StatusBadGateway)
		case strings.HasSuffix(r.URL.Path, "/22222222222222222222222222222222"):
			w.Header().Set("X-Attesta-Org", "")
			_, _ = io.WriteString(w, "orphan")
		case strings.HasSuffix(r.URL.Path, "/"+id):
			w.Header().Set("X-Attesta-Org", "feragame")
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Content-Disposition", `attachment; filename="x.txt"`)
			_, _ = io.WriteString(w, "body")
		default:
			http.Error(w, "unexpected "+r.URL.Path, http.StatusInternalServerError)
		}
	}))
	t.Cleanup(upstream.Close)
	arts := NewSidecarHttpPort(upstream.URL, "secret", upstream.Client())
	h := &Handler{Gate: gateOK, Artifacts: arts}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+"ffffffffffffffffffffffffffffffff", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("upstream 404 = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+"11111111111111111111111111111111", nil))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("upstream 502 = %d", rec.Code)
	}
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, ArtifactProxyPathPrefix+"22222222222222222222222222222222", nil))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("empty artifact org = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodHead, ArtifactProxyPathPrefix+id, nil))
	if rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Fatalf("HEAD = %d body=%q", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Content-Disposition") == "" || rec.Header().Get("Content-Type") != "text/plain" {
		t.Fatalf("HEAD headers = %v", rec.Header())
	}
}
