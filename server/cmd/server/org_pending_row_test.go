package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOrgPendingRowTemplate(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := OrgPendingRowView{
		Email:     "member@example.com",
		DateLabel: "Requested on",
		Date:      "20 Mar 2026 at 10:00 UTC",
		Roles: RolePillRowView{
			Size: "sm",
			Pills: []RolePillView{
				{Label: "QA Reviewer", Palette: "emerald"},
			},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "org_pending_row", view); err != nil {
		t.Fatalf("render org_pending_row: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="list-row-main list-row-main-stack"`,
		"member@example.com",
		`class="role-pill-row"`,
		`data-role-palette="emerald"`,
		"QA Reviewer",
		"Requested on: 20 Mar 2026 at 10:00 UTC",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in org_pending_row, got:\n%s", want, body)
		}
	}
}

func TestOrgPendingRowTemplateOmitsEmptyDate(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := OrgPendingRowView{
		Email:     "member@example.com",
		DateLabel: "Invited on",
	}
	if err := tmpl.ExecuteTemplate(&out, "org_pending_row", view); err != nil {
		t.Fatalf("render org_pending_row: %v", err)
	}
	body := out.String()
	if strings.Contains(body, "Invited on") {
		t.Fatalf("empty date must omit date label, got:\n%s", body)
	}
}

func TestOrgPendingRowFromJoinRequest(t *testing.T) {
	got := orgPendingRowFromJoinRequest(OrgAdminJoinRequestRow{
		RequesterEmail: " join@example.com ",
		CreatedAt:      " 20 Mar 2026 at 10:00 UTC ",
		Roles: []OrgAdminRoleOption{
			{Slug: "qa-reviewer", Name: "QA Reviewer", Palette: "emerald"},
		},
	})
	if got.Email != "join@example.com" || got.DateLabel != "Requested on" || got.Date != "20 Mar 2026 at 10:00 UTC" {
		t.Fatalf("unexpected join row: %#v", got)
	}
	if got.Roles.Size != "sm" || len(got.Roles.Pills) != 1 || got.Roles.Pills[0].Label != "QA Reviewer" {
		t.Fatalf("unexpected join roles: %#v", got.Roles)
	}
}

func TestOrgPendingRowFromInvite(t *testing.T) {
	got := orgPendingRowFromInvite(OrgAdminInviteRow{
		Email:          " pending@example.com ",
		CreatedAtLabel: " 21 Mar 2026 at 11:00 UTC ",
		Roles: []OrgAdminRoleOption{
			{Slug: "approver", Name: "Approver", Palette: "orange"},
		},
	})
	if got.Email != "pending@example.com" || got.DateLabel != "Invited on" || got.Date != "21 Mar 2026 at 11:00 UTC" {
		t.Fatalf("unexpected invite row: %#v", got)
	}
	if got.Roles.Size != "sm" || len(got.Roles.Pills) != 1 || got.Roles.Pills[0].Label != "Approver" {
		t.Fatalf("unexpected invite roles: %#v", got.Roles)
	}
}
