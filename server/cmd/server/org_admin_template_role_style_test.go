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
				UserMongoID: "507f1f77bcf86cd799439011",
				Email:       "user@example.com",
				Activated:   true,
				IsOrgAdmin:  true,
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
