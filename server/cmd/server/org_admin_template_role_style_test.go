package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrgAdminTemplateRolePillRendersCSSVariables(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := OrgAdminView{
		Roles: []Role{
			{Slug: "org-admin", Name: "Org Admin"},
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
						Slug:       "org-admin",
						Name:       "Org Admin",
						RoleColor:  template.CSS("var(--role-red-bg)"),
						RoleBorder: template.CSS("var(--role-red-border)"),
						Selected:   true,
					},
					{
						Slug:       "qa-reviewer",
						Name:       "QA Reviewer",
						RoleColor:  template.CSS("var(--role-emerald-bg)"),
						RoleBorder: template.CSS("var(--role-emerald-border)"),
						Selected:   true,
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
	if !strings.Contains(body, `style="--pill-bg: var(--role-emerald-bg); --border: var(--role-emerald-border)"`) {
		t.Fatalf("expected role pill css variables in output, got body: %s", body)
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

	tagsStart := strings.Index(body, `<div class="user-tags">`)
	if tagsStart < 0 {
		t.Fatalf("expected user-tags block in output, got body: %s", body)
	}
	tagsEnd := strings.Index(body[tagsStart:], `</div>`)
	if tagsEnd < 0 {
		t.Fatalf("expected user-tags closing tag in output, got body: %s", body)
	}
	tagsBlock := body[tagsStart : tagsStart+tagsEnd]
	if strings.Contains(tagsBlock, "Org Admin") {
		t.Fatalf("org-admin pill should be hidden from user-tags block, got: %s", tagsBlock)
	}
	if !strings.Contains(tagsBlock, "QA Reviewer") {
		t.Fatalf("expected non-admin role pill in user-tags block, got: %s", tagsBlock)
	}
}

func TestOrgAdminTemplateLastInviteCopyButton(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	view := OrgAdminView{
		InviteLink: "/invite/token-pending",
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
	if !strings.Contains(compactBody, `class="secondary js-invite-copy"`) || !strings.Contains(compactBody, `data-copy-invite-link="/invite/token-pending"`) {
		t.Fatalf("expected last invite copy button with invite link, got body: %s", body)
	}
	if strings.Contains(body, "Last invite:") {
		t.Fatalf("last invite text should be hidden, got body: %s", body)
	}
	if !strings.Contains(compactBody, `data-copy-icon-default style="display: inline-block;"`) {
		t.Fatalf("expected default copy icon visible by default, got body: %s", body)
	}
	if !strings.Contains(compactBody, `data-copy-icon-done style="display: none;"`) {
		t.Fatalf("expected done copy icon hidden by default, got body: %s", body)
	}
	if !strings.Contains(body, `<span data-copy-label>`) {
		t.Fatalf("expected copy label span, got body: %s", body)
	}
	if !strings.Contains(body, "data-copy-icon-default") || !strings.Contains(body, "data-copy-icon-done") {
		t.Fatalf("expected copy state icons in invite copy button, got body: %s", body)
	}
}
