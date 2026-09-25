package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOrgAdminTemplateRolePillRendersCSSVariables(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := OrgAdminView{
		ActivePanel: "members",
		Roles: []Role{
			{Slug: "qa-reviewer", Name: "QA Reviewer"},
		},
		Users: []OrgAdminUserRow{
			{
				UserID:     "user-1",
				Email:      "user@example.com",
				Activated:  true,
				IsOrgAdmin: true,
				RoleOptions: []OrgAdminRoleOption{
					{
						Slug:     "qa-reviewer",
						Name:     "QA Reviewer",
						Palette:  "emerald",
						Selected: true,
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
		t.Fatalf("render org admin template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, "ZgotmplZ") {
		t.Fatalf("unexpected escaped css marker in output: %s", body)
	}
	if !strings.Contains(body, `data-role-palette="emerald"`) {
		t.Fatalf("expected role pill palette attribute in output, got body: %s", body)
	}

	emailStart := strings.Index(body, `<span class="user-email">`)
	if emailStart < 0 {
		t.Fatalf("expected user-email block in output, got body: %s", body)
	}
	emailEnd := strings.Index(body[emailStart:], `</span>`)
	if emailEnd < 0 {
		t.Fatalf("expected user-email closing tag in output, got body: %s", body)
	}
	emailBlock := body[emailStart : emailStart+emailEnd]
	if !strings.Contains(emailBlock, "<svg") {
		t.Fatalf("expected org-admin icon in user-email block, got: %s", emailBlock)
	}
	if got := strings.Count(emailBlock, "<svg"); got != 1 {
		t.Fatalf("expected exactly one icon in user-email block, got %d in %s", got, emailBlock)
	}

	tagsStart := strings.Index(body, `class="role-pill-row"`)
	if tagsStart < 0 {
		t.Fatalf("expected role-pill-row block in output, got body: %s", body)
	}
	tagsEnd := strings.Index(body[tagsStart:], `</div>`)
	if tagsEnd < 0 {
		t.Fatalf("expected role-pill-row closing tag in output, got body: %s", body)
	}
	tagsBlock := body[tagsStart : tagsStart+tagsEnd]
	if strings.Contains(tagsBlock, "Org Admin") {
		t.Fatalf("org-admin pill should be hidden from role-pill-row, got: %s", tagsBlock)
	}
	if !strings.Contains(tagsBlock, "QA Reviewer") {
		t.Fatalf("expected non-admin role pill in role-pill-row, got: %s", tagsBlock)
	}

	manageStart := strings.Index(body, `id="manage-user-user-1"`)
	if manageStart < 0 {
		t.Fatalf("expected manage-user dialog, got body: %s", body)
	}
	manageEnd := strings.Index(body[manageStart:], `</dialog>`)
	if manageEnd < 0 {
		t.Fatalf("expected manage-user dialog close, got body: %s", body)
	}
	manageDialog := body[manageStart : manageStart+manageEnd]
	for _, want := range []string{
		`name="is_org_admin"`,
		`value="1"`,
		"checked",
		"Organization roles",
	} {
		if !strings.Contains(manageDialog, want) {
			t.Fatalf("expected %q in manage-user dialog, got: %s", want, manageDialog)
		}
	}
	if strings.Contains(manageDialog, `data-value="org-admin"`) {
		t.Fatalf("manage-user roles picker must not list Org admin, got: %s", manageDialog)
	}
}

func TestOrgAdminTemplateLocksSoleOrgAdminStanding(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := OrgAdminView{
		ActivePanel: "members",
		Users: []OrgAdminUserRow{
			{
				UserID:                 "user-1",
				Email:                  "owner@example.com",
				Activated:              true,
				IsOrgAdmin:             true,
				OrgAdminStandingLocked: true,
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
		t.Fatalf("render org admin template: %v", err)
	}
	body := out.String()
	manageStart := strings.Index(body, `id="manage-user-user-1"`)
	if manageStart < 0 {
		t.Fatalf("expected manage-user dialog, got:\n%s", body)
	}
	manageEnd := strings.Index(body[manageStart:], `</dialog>`)
	if manageEnd < 0 {
		t.Fatal("expected manage-user dialog close")
	}
	manageDialog := body[manageStart : manageStart+manageEnd]
	for _, want := range []string{
		`type="hidden" name="is_org_admin" value="1"`,
		"disabled",
		"is-disabled",
		"This is the only Org admin",
	} {
		if !strings.Contains(manageDialog, want) {
			t.Fatalf("expected %q in locked manage-user dialog, got:\n%s", want, manageDialog)
		}
	}
}

func TestBuildOrgAdminUserRowsLocksSoleOrgAdmin(t *testing.T) {
	sole := buildOrgAdminUserRowsFromIdentity(nil, []IdentityUser{
		{ID: "admin-1", Email: "owner@example.com", IsOrgAdmin: true, Status: "active"},
		{ID: "member-1", Email: "member@example.com", IsOrgAdmin: false, Status: "active"},
	})
	if len(sole) != 2 {
		t.Fatalf("rows = %d, want 2", len(sole))
	}
	if !sole[0].OrgAdminStandingLocked || sole[1].OrgAdminStandingLocked {
		t.Fatalf("standing lock = admin:%v member:%v", sole[0].OrgAdminStandingLocked, sole[1].OrgAdminStandingLocked)
	}

	shared := buildOrgAdminUserRowsFromIdentity(nil, []IdentityUser{
		{ID: "admin-1", Email: "owner@example.com", IsOrgAdmin: true, Status: "active"},
		{ID: "admin-2", Email: "co@example.com", IsOrgAdmin: true, Status: "active"},
	})
	if shared[0].OrgAdminStandingLocked || shared[1].OrgAdminStandingLocked {
		t.Fatalf("shared admins must not lock standing: %#v", shared)
	}
}

func TestOrgAdminTemplateLastInviteCopyButton(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := OrgAdminView{
		ActivePanel: "members",
		InviteLink:  "/invite/token-pending",
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
		t.Fatalf("render org admin template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	if strings.Contains(body, "See all invites") {
		t.Fatalf("invites modal trigger should be hidden, got body: %s", body)
	}
	if strings.Contains(body, "All invites") {
		t.Fatalf("invites modal should not render, got body: %s", body)
	}
	if strings.Contains(compactBody, `class="secondary js-invite-copy"`) || strings.Contains(compactBody, `data-copy-invite-link="/invite/token-pending"`) {
		t.Fatalf("did not expect invite copy button markup, got body: %s", body)
	}
	if !strings.Contains(compactBody, `id="add-user-dialog" class="dialog dialog-overflow" data-auto-open`) {
		t.Fatalf("expected add-user dialog data-auto-open when invite link is present, got body: %s", body)
	}
	if strings.Contains(body, "Last invite:") {
		t.Fatalf("last invite text should be hidden, got body: %s", body)
	}
}
