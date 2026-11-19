package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	revisionSourceCommit = "0123456789abcdef0123456789abcdef01234567"
	revisionBuildCommit  = "89abcdef0123456789abcdef0123456789abcdef"
	revisionBuildInfo    = "fedcba9876543210fedcba9876543210fedcba98"
)

func setRevisionSources(t *testing.T, sourceCommit, built, buildInfo string) {
	t.Helper()
	t.Setenv("SOURCE_COMMIT", sourceCommit)
	previousBuild := buildCommit
	previousInfo := buildInfoRevision
	buildCommit = built
	buildInfoRevision = func() string { return buildInfo }
	t.Cleanup(func() {
		buildCommit = previousBuild
		buildInfoRevision = previousInfo
	})
}

func TestDeployedRevisionPrefersSourceCommit(t *testing.T) {
	cases := []struct {
		name         string
		sourceCommit string
		built        string
		buildInfo    string
		want         string
	}{
		{name: "source commit", sourceCommit: revisionSourceCommit, built: revisionBuildCommit, buildInfo: revisionBuildInfo, want: revisionSourceCommit},
		{name: "build commit", sourceCommit: "HEAD", built: revisionBuildCommit, buildInfo: revisionBuildInfo, want: revisionBuildCommit},
		{name: "build info", sourceCommit: "", built: "HEAD", buildInfo: revisionBuildInfo, want: revisionBuildInfo},
		{name: "unavailable", sourceCommit: "HEAD", built: "", buildInfo: "HEAD", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setRevisionSources(t, tc.sourceCommit, tc.built, tc.buildInfo)
			if got := deployedRevision(); got != tc.want {
				t.Fatalf("deployedRevision() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestIsFullRevision(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{revisionSourceCommit, true},
		{strings.ToUpper(revisionSourceCommit), true},
		{"0123456789abcdef0123456789abcdef0123456", false},
		{"0123456789abcdef0123456789abcdef012345678", false},
		{"0123456789abcdef0123456789abcdef0123456g", false},
		{"HEAD", false},
	}
	for _, tc := range cases {
		if got := isFullRevision(tc.value); got != tc.want {
			t.Fatalf("isFullRevision(%q) = %v, want %v", tc.value, got, tc.want)
		}
	}
}

func TestHandleRevision(t *testing.T) {
	t.Run("serves the deployed commit", func(t *testing.T) {
		setRevisionSources(t, revisionSourceCommit, "", "")
		recorder := httptest.NewRecorder()
		handleRevision(recorder, httptest.NewRequest(http.MethodGet, "/revision", nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", recorder.Code)
		}
		if got, want := recorder.Body.String(), revisionSourceCommit+"\n"; got != want {
			t.Fatalf("body = %q, want %q", got, want)
		}
		if got := recorder.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
			t.Fatalf("content type = %q", got)
		}
		if got := recorder.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("cache control = %q", got)
		}
	})

	t.Run("rejects other methods", func(t *testing.T) {
		setRevisionSources(t, revisionSourceCommit, "", "")
		recorder := httptest.NewRecorder()
		handleRevision(recorder, httptest.NewRequest(http.MethodPost, "/revision", nil))
		if recorder.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want 405", recorder.Code)
		}
		if got := recorder.Header().Get("Allow"); got != http.MethodGet {
			t.Fatalf("allow = %q", got)
		}
	})

	t.Run("reports an unavailable revision", func(t *testing.T) {
		setRevisionSources(t, "HEAD", "", "HEAD")
		recorder := httptest.NewRecorder()
		handleRevision(recorder, httptest.NewRequest(http.MethodGet, "/revision", nil))
		if recorder.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want 503", recorder.Code)
		}
	})
}
