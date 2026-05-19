package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleDocsRoutes(t *testing.T) {
	server := &Server{}

	tests := []struct {
		name             string
		path             string
		wantStatus       int
		wantLocation     string
		wantContentType  string
		wantBodyContains string
	}{
		{
			name:         "redirect docs root",
			path:         "/docs",
			wantStatus:   http.StatusMovedPermanently,
			wantLocation: "/docs/",
		},
		{
			name:             "swagger ui html",
			path:             "/docs/",
			wantStatus:       http.StatusOK,
			wantContentType:  "text/html; charset=utf-8",
			wantBodyContains: "SwaggerUIBundle",
		},
		{
			name:             "openapi json",
			path:             "/docs/openapi3.json",
			wantStatus:       http.StatusOK,
			wantContentType:  "application/json; charset=utf-8",
			wantBodyContains: "{",
		},
		{
			name:             "openapi yaml",
			path:             "/docs/openapi3.yaml",
			wantStatus:       http.StatusOK,
			wantContentType:  "application/yaml; charset=utf-8",
			wantBodyContains: "openapi:",
		},
		{
			name:       "not found",
			path:       "/docs/unknown",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			server.handleDocs(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantStatus)
			}
			if tc.wantLocation != "" {
				if got := rec.Header().Get("Location"); got != tc.wantLocation {
					t.Fatalf("location = %q, want %q", got, tc.wantLocation)
				}
			}
			if tc.wantContentType != "" {
				if got := rec.Header().Get("Content-Type"); got != tc.wantContentType {
					t.Fatalf("content-type = %q, want %q", got, tc.wantContentType)
				}
			}
			if tc.wantBodyContains != "" && !strings.Contains(rec.Body.String(), tc.wantBodyContains) {
				t.Fatalf("response body missing %q", tc.wantBodyContains)
			}
		})
	}
}

func TestOpenAPIDocCandidates(t *testing.T) {
	got := openAPIDocCandidates("openapi3.json")
	want := []string{
		filepath.Join("gen", "http", "openapi3.json"),
		filepath.Join("..", "gen", "http", "openapi3.json"),
		filepath.Join("..", "..", "gen", "http", "openapi3.json"),
	}
	if len(got) != len(want) {
		t.Fatalf("candidate count = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("candidate[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestOpenAPIRequestOriginUsesForwardedHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://internal:3000/docs/openapi3.json", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "attesta.example.com")

	if got := openAPIRequestOrigin(req); got != "https://attesta.example.com" {
		t.Fatalf("origin = %q, want %q", got, "https://attesta.example.com")
	}
}

func TestListenAddrFromEnv(t *testing.T) {
	t.Setenv("ADDR", "")
	t.Setenv("PORT", "3001")
	if got := listenAddrFromEnv(); got != ":3001" {
		t.Fatalf("listenAddrFromEnv() = %q, want %q", got, ":3001")
	}

	t.Setenv("ADDR", "127.0.0.1:3002")
	if got := listenAddrFromEnv(); got != "127.0.0.1:3002" {
		t.Fatalf("listenAddrFromEnv() = %q, want %q", got, "127.0.0.1:3002")
	}
}

func TestRewriteOpenAPIServersJSON(t *testing.T) {
	data, err := rewriteOpenAPIServers([]byte(`{"openapi":"3.0.3","servers":[{"url":"http://localhost:3000"}]}`), "openapi3.json", "https://attesta.example.com")
	if err != nil {
		t.Fatalf("rewriteOpenAPIServers: %v", err)
	}
	var doc struct {
		Servers []struct {
			URL string `json:"url"`
		} `json:"servers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if len(doc.Servers) != 1 || doc.Servers[0].URL != "https://attesta.example.com" {
		t.Fatalf("servers = %+v, want deployed origin", doc.Servers)
	}
}

func TestServeOpenAPIFileNotFound(t *testing.T) {
	server := &Server{}
	req := httptest.NewRequest(http.MethodGet, "/docs/missing", nil)
	rec := httptest.NewRecorder()

	server.serveOpenAPIFile(rec, req, "missing-openapi3-file.json", "application/json; charset=utf-8")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
	if !strings.Contains(rec.Body.String(), "OpenAPI spec not found") {
		t.Fatalf("expected missing OpenAPI error body, got %q", rec.Body.String())
	}
}
