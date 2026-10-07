package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaginationTemplateRendersPrevPagesNext(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PaginationView{
		AriaLabel:       "Demo pagination",
		Links:           []PaginationLink{{Page: 1, URL: "/p?page=1"}, {Page: 2, URL: "/p?page=2", IsCurrent: true}},
		HasPreviousPage: true,
		HasNextPage:     false,
		PreviousURL:     "/p?page=1",
		NextURL:         "/p?page=2",
	}
	if err := tmpl.ExecuteTemplate(&out, "pagination", view); err != nil {
		t.Fatalf("render pagination: %v", err)
	}
	body := out.String()
	compact := strings.Join(strings.Fields(body), " ")

	for _, want := range []string{
		`class="pagination"`,
		`aria-label="Demo pagination"`,
		`class="pagination-pages"`,
		`href="/p?page=1"`,
		`href="/p?page=2"`,
		`class="btn btn-primary"`,
		`class="btn btn-secondary"`,
		`is-disabled`,
		">1<",
		">2<",
	} {
		if !strings.Contains(compact, want) && !strings.Contains(body, want) {
			t.Fatalf("expected %q in pagination, got:\n%s", want, body)
		}
	}
	if !strings.Contains(compact, `pagination-btn is-disabled`) {
		t.Fatalf("expected next link disabled when no next page, got:\n%s", body)
	}
	if strings.Contains(compact, `pagination--inline`) {
		t.Fatalf("did not expect inline modifier when Inline=false, got:\n%s", body)
	}
}

func TestPaginationTemplateCurrentPageUsesPrimaryButton(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PaginationView{
		AriaLabel: "Pages",
		Links: []PaginationLink{
			{Page: 1, URL: "/a"},
			{Page: 2, URL: "/b", IsCurrent: true},
		},
		HasPreviousPage: true,
		HasNextPage:     true,
		PreviousURL:     "/a",
		NextURL:         "/b",
	}
	if err := tmpl.ExecuteTemplate(&out, "pagination", view); err != nil {
		t.Fatalf("render pagination: %v", err)
	}
	body := out.String()
	currentIdx := strings.Index(body, `href="/b"`)
	if currentIdx == -1 {
		t.Fatalf("missing current page link, got:\n%s", body)
	}
	// Class is on the same <a> as href="/b"
	anchorStart := strings.LastIndex(body[:currentIdx], "<a")
	anchorEnd := strings.Index(body[currentIdx:], ">")
	if anchorStart == -1 || anchorEnd == -1 {
		t.Fatalf("could not locate current page anchor, got:\n%s", body)
	}
	anchor := body[anchorStart : currentIdx+anchorEnd]
	if !strings.Contains(anchor, `btn-primary`) {
		t.Fatalf("expected current page to use btn-primary, got:\n%s", anchor)
	}
	if strings.Contains(anchor, `btn-secondary`) {
		t.Fatalf("current page must not use btn-secondary, got:\n%s", anchor)
	}
}

func TestPaginationTemplateOmitsHTMXWhenUnset(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PaginationView{
		AriaLabel:       "Plain",
		Links:           []PaginationLink{{Page: 1, URL: "/plain", IsCurrent: true}},
		HasPreviousPage: false,
		HasNextPage:     false,
		PreviousURL:     "/plain",
		NextURL:         "/plain",
	}
	if err := tmpl.ExecuteTemplate(&out, "pagination", view); err != nil {
		t.Fatalf("render pagination: %v", err)
	}
	body := out.String()
	for _, forbidden := range []string{"hx-get", "hx-target", "hx-select", "hx-swap", "hx-push-url"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected no %q when HxTarget unset, got:\n%s", forbidden, body)
		}
	}
}

func TestPaginationTemplateIncludesHTMXWhenSet(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PaginationView{
		AriaLabel:       "HTMX",
		Links:           []PaginationLink{{Page: 1, URL: "/p1"}, {Page: 2, URL: "/p2", IsCurrent: true}},
		HasPreviousPage: true,
		HasNextPage:     false,
		PreviousURL:     "/p1",
		NextURL:         "/p2",
		HxTarget:        "#results",
		HxSelect:        "#results",
		PushURL:         true,
	}
	if err := tmpl.ExecuteTemplate(&out, "pagination", view); err != nil {
		t.Fatalf("render pagination: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`hx-get="/p1"`,
		`hx-get="/p2"`,
		`hx-target="#results"`,
		`hx-select="#results"`,
		`hx-swap="outerHTML"`,
		`hx-push-url="true"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q when HTMX set, got:\n%s", want, body)
		}
	}
}

func TestPaginationTemplateOmitsPushURLWhenFalse(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PaginationView{
		AriaLabel:       "No push",
		Links:           []PaginationLink{{Page: 1, URL: "/x", IsCurrent: true}},
		HasPreviousPage: false,
		HasNextPage:     false,
		PreviousURL:     "/x",
		NextURL:         "/x",
		HxTarget:        "#frag",
		HxSelect:        "#frag",
		PushURL:         false,
	}
	if err := tmpl.ExecuteTemplate(&out, "pagination", view); err != nil {
		t.Fatalf("render pagination: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `hx-target="#frag"`) {
		t.Fatalf("expected hx-target when HxTarget set, got:\n%s", body)
	}
	if strings.Contains(body, `hx-push-url`) {
		t.Fatalf("did not expect hx-push-url when PushURL=false, got:\n%s", body)
	}
}

func TestPaginationTemplateInlineModifier(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := PaginationView{
		AriaLabel:       "Inline",
		Inline:          true,
		Links:           []PaginationLink{{Page: 1, URL: "/i", IsCurrent: true}},
		HasPreviousPage: false,
		HasNextPage:     false,
		PreviousURL:     "/i",
		NextURL:         "/i",
	}
	if err := tmpl.ExecuteTemplate(&out, "pagination", view); err != nil {
		t.Fatalf("render pagination: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `class="pagination pagination--inline"`) {
		t.Fatalf("expected inline modifier class, got:\n%s", body)
	}
}

func TestBuildHomeProcessGroupCarriesPaginationView(t *testing.T) {
	items := make([]StreamInstanceCard, homeProcessesPerPage+1)
	for i := range items {
		items[i] = StreamInstanceCard{ID: "p", Status: "active"}
	}
	group := buildHomeProcessGroupForStatus("/my/streams/wf", items, homeProcessesByStatus(items), "active", "status", 1)
	p := group.Pagination
	if p.AriaLabel != "Active stream instances pagination" {
		t.Fatalf("Pagination.AriaLabel = %q", p.AriaLabel)
	}
	if !p.Inline {
		t.Fatal("expected stream dashboard pagination Inline=true")
	}
	if p.HxTarget != "#stream-dashboard-results" || p.HxSelect != "#stream-dashboard-results" {
		t.Fatalf("unexpected HTMX targets: %#v", p)
	}
	if !p.PushURL {
		t.Fatal("expected PushURL=true for stream dashboard")
	}
	if !p.HasNextPage || p.HasPreviousPage {
		t.Fatalf("page 1 of 2: HasPrevious=%v HasNext=%v", p.HasPreviousPage, p.HasNextPage)
	}
	if len(p.Links) != 2 || !p.Links[0].IsCurrent || p.Links[1].IsCurrent {
		t.Fatalf("unexpected Links: %#v", p.Links)
	}
}

func TestBuildOnboardingJoinViewCarriesPaginationView(t *testing.T) {
	identity := &fakeIdentityStore{}
	identity.listOrganizationsPageFunc = func(_ context.Context, opts IdentityOrgListOptions) (IdentityOrgPage, error) {
		if opts.Search != "acme" {
			t.Fatalf("search = %q, want acme", opts.Search)
		}
		return IdentityOrgPage{
			Organizations: []IdentityOrg{{Slug: "acme", Name: "Acme"}},
			Total:         onboardingJoinSearchLimit + 1,
		}, nil
	}
	server := &Server{identity: identity, tmpl: parseTestTemplates(t)}
	req := httptest.NewRequest(http.MethodGet, "/my/onboarding/join?q=acme", nil)
	rec := httptest.NewRecorder()
	view, ok := server.buildOnboardingJoinView(rec, req, &AccountUser{IdentityUserID: "u"}, "", "acme", "", 1)
	if !ok {
		t.Fatal("expected buildOnboardingJoinView ok")
	}
	p := view.Pagination
	if p.AriaLabel != "Organizations pagination" {
		t.Fatalf("Pagination.AriaLabel = %q", p.AriaLabel)
	}
	if p.Inline {
		t.Fatal("expected onboarding join pagination Inline=false")
	}
	if p.HxTarget != "#onboarding-join-results" || p.HxSelect != "#onboarding-join-results" {
		t.Fatalf("unexpected HTMX targets: %#v", p)
	}
	if !p.PushURL {
		t.Fatal("expected PushURL=true for onboarding join")
	}
	if !p.HasNextPage || p.HasPreviousPage {
		t.Fatalf("page 1 of 2: HasPrevious=%v HasNext=%v", p.HasPreviousPage, p.HasNextPage)
	}
	if p.PreviousURL != "/my/onboarding/join?q=acme" {
		t.Fatalf("PreviousURL = %q", p.PreviousURL)
	}
	if p.NextURL != "/my/onboarding/join?page=2&q=acme" {
		t.Fatalf("NextURL = %q", p.NextURL)
	}
	if len(p.Links) != 2 || !p.Links[0].IsCurrent || p.Links[1].IsCurrent {
		t.Fatalf("unexpected Links: %#v", p.Links)
	}
	if p.Links[0].URL != "/my/onboarding/join?q=acme" || p.Links[1].URL != "/my/onboarding/join?page=2&q=acme" {
		t.Fatalf("unexpected link URLs: %#v", p.Links)
	}
}
