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

func TestPublicStreamPath(t *testing.T) {
	if got := publicStreamPath("  wf-a  "); got != "/streams/wf-a" {
		t.Fatalf("publicStreamPath = %q, want /streams/wf-a", got)
	}
	if got := publicStreamPath(""); got != "/streams/" {
		t.Fatalf("empty key = %q, want /streams/", got)
	}
}

func TestAdminPath(t *testing.T) {
	cases := []struct {
		rest string
		want string
	}{
		{"", "/admin"},
		{"organizations", "/admin/organizations"},
		{"/categories", "/admin/categories"},
		{"organizations/logo/logo-1", "/admin/organizations/logo/logo-1"},
		{"  organizations  ", "/admin/organizations"},
	}
	for _, tc := range cases {
		if got := adminPath(tc.rest); got != tc.want {
			t.Fatalf("adminPath(%q) = %q, want %q", tc.rest, got, tc.want)
		}
	}
}

func TestPublicHomePath(t *testing.T) {
	cases := []struct {
		cat, sub, want string
	}{
		{"supply-chain", "procurement", "/?category=supply-chain&subCategory=procurement"},
		{"  supply-chain  ", "  procurement  ", "/?category=supply-chain&subCategory=procurement"},
		{"", "procurement", "/"},
		{"supply-chain", "", "/"},
		{"", "", "/"},
		{"  ", "  ", "/"},
	}
	for _, tc := range cases {
		if got := publicHomePath(tc.cat, tc.sub); got != tc.want {
			t.Fatalf("publicHomePath(%q, %q) = %q, want %q", tc.cat, tc.sub, got, tc.want)
		}
	}
}
