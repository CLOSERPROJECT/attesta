package main

import "testing"

func TestApplicationVersionUsesBuildCommit(t *testing.T) {
	original := buildCommit
	t.Cleanup(func() { buildCommit = original })
	buildCommit = "1234567890abcdef"

	if got, want := applicationVersion(), "1.1234567890ab"; got != want {
		t.Fatalf("applicationVersion() = %q, want %q", got, want)
	}
}

func TestPageBaseIncludesBuildVersion(t *testing.T) {
	server := &Server{buildVersion: "1.abc1234"}

	if got, want := server.pageBase("home_body", "", "").BuildVersion, "1.abc1234"; got != want {
		t.Fatalf("pageBase().BuildVersion = %q, want %q", got, want)
	}
}
