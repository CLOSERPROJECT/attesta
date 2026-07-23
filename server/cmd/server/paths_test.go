package main

import "testing"

func TestStreamPath(t *testing.T) {
	if got := streamPath("  wf-a  "); got != "/my/streams/wf-a" {
		t.Fatalf("streamPath = %q", got)
	}
}

func TestStreamInstancePath(t *testing.T) {
	if got := streamInstancePath("wf-a", "abc123"); got != "/my/streams/wf-a/instance/abc123" {
		t.Fatalf("streamInstancePath = %q", got)
	}
}

func TestOrganizationPath(t *testing.T) {
	cases := []struct {
		rest string
		want string
	}{
		{"", "/my/organization"},
		{"profile", "/my/organization/profile"},
		{"/roles", "/my/organization/roles"},
		{"formata-builder?stream=x", "/my/organization/formata-builder?stream=x"},
	}
	for _, tc := range cases {
		if got := organizationPath(tc.rest); got != tc.want {
			t.Fatalf("organizationPath(%q) = %q, want %q", tc.rest, got, tc.want)
		}
	}
}

func TestAppHomePath(t *testing.T) {
	if appHomePath != "/my" {
		t.Fatalf("appHomePath = %q", appHomePath)
	}
}
