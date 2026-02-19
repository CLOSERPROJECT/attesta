package main

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestReadActor(t *testing.T) {
	tests := []struct {
		name   string
		cookie *http.Cookie
		want   Actor
	}{
		{name: "missing cookie", cookie: nil, want: Actor{}},
		{name: "malformed cookie", cookie: &http.Cookie{Name: "demo_user", Value: "broken"}, want: Actor{}},
		{name: "valid legacy cookie", cookie: &http.Cookie{Name: "demo_user", Value: "u1|dep1"}, want: Actor{UserID: "u1", Role: "dep1", WorkflowKey: "workflow"}},
		{name: "valid scoped cookie", cookie: &http.Cookie{Name: "demo_user", Value: "u1|dep1|wf-a"}, want: Actor{UserID: "u1", Role: "dep1", WorkflowKey: "wf-a"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			if got := readActor(req, "workflow"); got != tc.want {
				t.Fatalf("readActor() = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestIsHTMXRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("HX-Request", "TrUe")
	if !isHTMXRequest(req) {
		t.Fatal("expected HX-Request header to be case-insensitive true")
	}

	notHTMX := httptest.NewRequest(http.MethodGet, "/", nil)
	notHTMX.Header.Set("HX-Request", "false")
	if isHTMXRequest(notHTMX) {
		t.Fatal("expected HX-Request false to return false")
	}

	missing := httptest.NewRequest(http.MethodGet, "/", nil)
	if isHTMXRequest(missing) {
		t.Fatal("expected missing HX-Request header to return false")
	}
}

func TestEnvOr(t *testing.T) {
	if got := envOr("ATTESTA_TEST_ENV_OR_UNSET", "fallback"); got != "fallback" {
		t.Fatalf("envOr for unset value = %q, want fallback", got)
	}

	t.Setenv("ATTESTA_TEST_ENV_OR_SET", "configured")
	if got := envOr("ATTESTA_TEST_ENV_OR_SET", "fallback"); got != "configured" {
		t.Fatalf("envOr for set value = %q, want configured", got)
	}

	t.Setenv("ATTESTA_TEST_ENV_OR_EMPTY", "")
	if got := envOr("ATTESTA_TEST_ENV_OR_EMPTY", "fallback"); got != "fallback" {
		t.Fatalf("envOr for empty value = %q, want fallback", got)
	}
}

func TestEnvBool(t *testing.T) {
	if got := envBool("ATTESTA_TEST_ENV_BOOL_UNSET", true); !got {
		t.Fatalf("envBool unset = %t, want true fallback", got)
	}

	t.Setenv("ATTESTA_TEST_ENV_BOOL_SET", "false")
	if got := envBool("ATTESTA_TEST_ENV_BOOL_SET", true); got {
		t.Fatalf("envBool set false = %t, want false", got)
	}

	t.Setenv("ATTESTA_TEST_ENV_BOOL_INVALID", "nope")
	if got := envBool("ATTESTA_TEST_ENV_BOOL_INVALID", false); got {
		t.Fatalf("envBool invalid = %t, want false fallback", got)
	}
}

func TestLogRequests(t *testing.T) {
	var logs bytes.Buffer
	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldPrefix := log.Prefix()
	log.SetOutput(&logs)
	log.SetFlags(0)
	log.SetPrefix("")
	t.Cleanup(func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		log.SetPrefix(oldPrefix)
	})

	handler := logRequests(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body = %q, want %q", rec.Body.String(), "ok")
	}
	line := strings.TrimSpace(logs.String())
	if !strings.Contains(line, "GET /ping") {
		t.Fatalf("log line = %q, want method and path", line)
	}
	if !strings.Contains(line, "/ping ") {
		t.Fatalf("log line = %q, want elapsed duration", line)
	}
}

func TestPrefersJSONResponse(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		accept string
		want   bool
	}{
		{name: "format json query", path: "/dpp?format=json", want: true},
		{name: "format json case insensitive", path: "/dpp?format=JSON", want: true},
		{name: "accept json header", path: "/dpp", accept: "text/html, application/json", want: true},
		{name: "accept text only", path: "/dpp", accept: "text/html", want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.accept != "" {
				req.Header.Set("Accept", tc.accept)
			}
			if got := prefersJSONResponse(req); got != tc.want {
				t.Fatalf("prefersJSONResponse() = %t, want %t", got, tc.want)
			}
		})
	}
}
