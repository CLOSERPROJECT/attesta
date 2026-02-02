package main

import (
	"net/http"
	"net/http/httptest"
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
		{name: "valid cookie", cookie: &http.Cookie{Name: "demo_user", Value: "u1|dep1"}, want: Actor{UserID: "u1", Role: "dep1"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.cookie != nil {
				req.AddCookie(tc.cookie)
			}
			if got := readActor(req); got != tc.want {
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
