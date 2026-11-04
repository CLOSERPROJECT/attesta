package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testServer(t *testing.T) (*server, string) {
	t.Helper()
	dir := t.TempDir()
	store, err := newDiskStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	return &server{store: store, token: "test-token"}, dir
}

func authReq(t *testing.T, method, path, token string, body any) *http.Request {
	t.Helper()
	var r io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		r = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, r)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestAuthRejectsMissingAndWrongToken(t *testing.T) {
	srv, _ := testServer(t)
	h := srv.routes()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authReq(t, http.MethodPost, "/fake/labels", "", map[string]any{"loadId": "x"}))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("missing token status = %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authReq(t, http.MethodPost, "/fake/labels", "wrong", map[string]any{"loadId": "x"}))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token status = %d", rr.Code)
	}
}

func TestLabelsAndRegisterCSVRoundTrip(t *testing.T) {
	srv, dataDir := testServer(t)
	h := srv.routes()

	rr := httptest.NewRecorder()
	req := authReq(t, http.MethodPost, "/fake/labels", "test-token", map[string]any{
		"loadId": "NIR-1",
		"labels": []any{map[string]any{"id": "NIR-1.1"}},
	})
	req.Header.Set(orgHeader, "org-feragame")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("labels status = %d body=%q", rr.Code, rr.Body.String())
	}
	labelsURL := strings.TrimSpace(rr.Body.String())
	if !strings.HasPrefix(labelsURL, "/artifacts/") || len(labelsURL) != len("/artifacts/")+32 {
		t.Fatalf("labels url = %q", labelsURL)
	}

	rr = httptest.NewRecorder()
	req = authReq(t, http.MethodPost, "/fake/register-csv", "test-token", map[string]any{
		"client":    "NO",
		"loadId":    "NIR-1",
		"palletIds": []any{"NIR-1.1", "NIR-1.2"},
	})
	req.Header.Set(orgHeader, "org-feragame")
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("csv status = %d body=%q", rr.Code, rr.Body.String())
	}
	if got := rr.Header().Get("Content-Disposition"); got != `attachment; filename="NIR-1.csv"` {
		t.Fatalf("disposition = %q", got)
	}
	csvURL := strings.TrimSpace(rr.Body.String())
	if !strings.HasPrefix(csvURL, "/artifacts/") {
		t.Fatalf("csv url = %q", csvURL)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authReq(t, http.MethodGet, csvURL, "test-token", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get csv status = %d", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "text/csv" {
		t.Fatalf("csv content-type = %q", ct)
	}
	if got := rr.Header().Get("Content-Disposition"); got != `attachment; filename="NIR-1.csv"` {
		t.Fatalf("get disposition = %q", got)
	}
	text := rr.Body.String()
	if !strings.Contains(text, "NIR-1.1 NIR-1.2") || !strings.HasPrefix(text, "client,date,") {
		t.Fatalf("csv body = %q", text)
	}

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authReq(t, http.MethodGet, labelsURL, "test-token", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("get pdf status = %d", rr.Code)
	}
	if got := rr.Header().Get("X-Attesta-Org"); got != "org-feragame" {
		t.Fatalf("get org header = %q", got)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("pdf content-type = %q", ct)
	}
	pdfBody := rr.Body.Bytes()
	if !bytes.HasPrefix(pdfBody, []byte("%PDF-")) {
		t.Fatalf("pdf body prefix = %q", string(pdfBody[:min(20, len(pdfBody))]))
	}
	if !bytes.Contains(pdfBody, []byte("%%EOF")) {
		t.Fatal("pdf missing end-of-file marker")
	}
	// go-pdf/fpdf emits a full xref + page content stream; plant stub had neither.
	if !bytes.Contains(pdfBody, []byte("xref")) || !bytes.Contains(pdfBody, []byte("stream")) {
		t.Fatalf("pdf missing structure (len=%d)", len(pdfBody))
	}
	if !bytes.Contains(pdfBody, []byte("NIR-1")) || !bytes.Contains(pdfBody, []byte("NIR-1.1")) {
		t.Fatalf("pdf missing load/label text (len=%d)", len(pdfBody))
	}

	id := strings.TrimPrefix(labelsURL, "/artifacts/")
	metaRaw, err := os.ReadFile(filepath.Join(dataDir, id+".meta.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta artifactMeta
	if err := json.Unmarshal(metaRaw, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.OrgID != "org-feragame" || meta.ContentType != "application/pdf" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestGetArtifactRequiresToken(t *testing.T) {
	srv, _ := testServer(t)
	h := srv.routes()

	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, authReq(t, http.MethodPost, "/fake/labels", "test-token", map[string]any{"loadId": "x"}))
	url := strings.TrimSpace(rr.Body.String())

	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, authReq(t, http.MethodGet, url, "", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rr.Code)
	}
}

func TestCreateDataDirOnStart(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nested", "data")
	if _, err := newDiskStore(root); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		t.Fatalf("data dir not created: %v", err)
	}
}
