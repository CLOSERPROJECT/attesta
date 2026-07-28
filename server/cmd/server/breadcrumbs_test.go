package main

import (
	"testing"
)

func TestBuildStreamBreadcrumbs(t *testing.T) {
	got := buildStreamBreadcrumbs("wf-a", "Alpha Stream")
	if len(got.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(got.Items))
	}
	if got.Items[0].Label != "Dashboard" || got.Items[0].Href != appHomePath {
		t.Fatalf("root = %+v", got.Items[0])
	}
	if got.Items[1].Label != "Stream: Alpha Stream" || got.Items[1].Href != streamPath("wf-a") || !got.Items[1].Current {
		t.Fatalf("current = %+v", got.Items[1])
	}
}

func TestBuildStreamBreadcrumbsFallsBackToKey(t *testing.T) {
	got := buildStreamBreadcrumbs("wf-a", "  ")
	if got.Items[1].Label != "Stream: wf-a" {
		t.Fatalf("label = %q, want Stream: workflow key", got.Items[1].Label)
	}
}

func TestBuildProcessBreadcrumbsUsesInstanceName(t *testing.T) {
	got := buildProcessBreadcrumbs("wf-a", "Alpha Stream", "Batch 1", "abc123")
	if len(got.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(got.Items))
	}
	if got.Items[0].Label != "Dashboard" || got.Items[0].Href != appHomePath {
		t.Fatalf("root = %+v", got.Items[0])
	}
	if got.Items[1].Label != "Stream: Alpha Stream" || got.Items[1].Href != streamPath("wf-a") {
		t.Fatalf("stream crumb = %+v", got.Items[1])
	}
	if got.Items[2].Label != "Instance: Batch 1" || got.Items[2].Href != streamInstancePath("wf-a", "abc123") || !got.Items[2].Current {
		t.Fatalf("instance crumb = %+v", got.Items[2])
	}
}

func TestBuildProcessBreadcrumbsFallsBackToProcessID(t *testing.T) {
	got := buildProcessBreadcrumbs("wf-a", "Alpha Stream", "", "abc123")
	if got.Items[2].Label != "Instance: abc123" {
		t.Fatalf("label = %q", got.Items[2].Label)
	}
}

func TestBuildOrgAdminBreadcrumbs(t *testing.T) {
	got := buildOrgAdminBreadcrumbs("members")
	if len(got.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(got.Items))
	}
	if got.Items[0].Label != "Dashboard" || got.Items[0].Href != appHomePath {
		t.Fatalf("root = %+v", got.Items[0])
	}
	if got.Items[1].Label != "Organization admin" || got.Items[1].Href != organizationPath("profile") {
		t.Fatalf("middle = %+v", got.Items[1])
	}
	if got.Items[2].Label != "Members" || got.Items[2].Href != organizationPath("members") || !got.Items[2].Current {
		t.Fatalf("section = %+v", got.Items[2])
	}
}

func TestBuildOrgAdminBreadcrumbsSections(t *testing.T) {
	cases := map[string]struct {
		label string
		href  string
	}{
		"profile": {label: "Profile", href: "/my/organization/profile"},
		"roles":   {label: "Roles", href: "/my/organization/roles"},
		"members": {label: "Members", href: "/my/organization/members"},
		"":        {label: "Profile", href: "/my/organization/profile"},
		"other":   {label: "Profile", href: "/my/organization/profile"},
	}
	for panel, want := range cases {
		got := buildOrgAdminBreadcrumbs(panel)
		if got.Items[2].Label != want.label || got.Items[2].Href != want.href {
			t.Fatalf("panel %q: got %+v, want label=%q href=%q", panel, got.Items[2], want.label, want.href)
		}
		if href := orgAdminSectionHref(panel); href != want.href {
			t.Fatalf("orgAdminSectionHref(%q) = %q, want %q", panel, href, want.href)
		}
	}
}

func TestBuildPlatformAdminBreadcrumbsOrgsPanel(t *testing.T) {
	cases := []string{"orgs", "", "other"}
	for _, panel := range cases {
		got := buildPlatformAdminBreadcrumbs(panel)
		if len(got.Items) != 2 {
			t.Fatalf("panel %q: len(Items) = %d, want 2", panel, len(got.Items))
		}
		if got.Items[0].Label != "Dashboard" || got.Items[0].Href != appHomePath {
			t.Fatalf("panel %q: root = %+v", panel, got.Items[0])
		}
		if got.Items[1].Label != "Platform admin" || got.Items[1].Href != "/admin/orgs" || !got.Items[1].Current {
			t.Fatalf("panel %q: current = %+v", panel, got.Items[1])
		}
	}
}

func TestBuildPlatformAdminBreadcrumbsCategoriesPanel(t *testing.T) {
	got := buildPlatformAdminBreadcrumbs("categories")
	if len(got.Items) != 3 {
		t.Fatalf("len(Items) = %d, want 3", len(got.Items))
	}
	if got.Items[0].Label != "Dashboard" || got.Items[0].Href != appHomePath {
		t.Fatalf("root = %+v", got.Items[0])
	}
	if got.Items[1].Label != "Platform admin" || got.Items[1].Href != "/admin/orgs" || got.Items[1].Current {
		t.Fatalf("middle = %+v", got.Items[1])
	}
	if got.Items[2].Label != "Categories" || got.Items[2].Href != "/admin/categories" || !got.Items[2].Current {
		t.Fatalf("current = %+v", got.Items[2])
	}
}
