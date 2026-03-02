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
			{Slug: "qa-reviewer", Name: "QA Reviewer"},
		},
		Users: []OrgAdminUserRow{
			{
				UserID:    "user-1",
				Email:     "user@example.com",
				Activated: true,
				RoleOptions: []OrgAdminRoleOption{
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
}
