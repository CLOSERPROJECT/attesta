package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrgAdminTemplateRoleUsageStates(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	view := OrgAdminView{
		RoleRows: []OrgAdminRoleRow{
			{
				Slug:       "approver",
				Name:       "Approver",
				RoleColor:  template.CSS("var(--role-blue-bg)"),
				RoleBorder: template.CSS("var(--role-blue-border)"),
				InUse:      true,
			},
			{
				Slug:       "qa-reviewer",
				Name:       "QA Reviewer",
				RoleColor:  template.CSS("var(--role-emerald-bg)"),
				RoleBorder: template.CSS("var(--role-emerald-border)"),
				InUse:      false,
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "org_admin_body", view); err != nil {
		t.Fatalf("render org admin template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	if !strings.Contains(compactBody, `aria-label="Edit role"`) || !strings.Contains(compactBody, `title="Role in use"`) || !strings.Contains(compactBody, `disabled`) {
		t.Fatalf("expected in-use edit button to be disabled, got: %s", body)
	}
	if !strings.Contains(compactBody, `aria-label="Delete role"`) || !strings.Contains(compactBody, `title="Role in use"`) || !strings.Contains(compactBody, `disabled`) {
		t.Fatalf("expected in-use delete button to be disabled, got: %s", body)
	}
	if strings.Contains(body, `id="edit-role-approver"`) || strings.Contains(body, `id="delete-role-approver"`) {
		t.Fatalf("did not expect dialogs for in-use role, got: %s", body)
	}
	if !strings.Contains(compactBody, `Not used`) {
		t.Fatalf("expected unused role helper text, got: %s", body)
	}
	if !strings.Contains(body, `id="edit-role-qa-reviewer"`) || !strings.Contains(body, `id="delete-role-qa-reviewer"`) {
		t.Fatalf("expected dialogs for unused role, got: %s", body)
	}
	if strings.Contains(body, `<p class="manage-dialog-danger-title">Role in use</p>`) {
		t.Fatalf("did not expect role-in-use dialog title, got: %s", body)
	}
	if strings.Contains(body, "Remove the role from the users that have it before continuing with the action.") {
		t.Fatalf("did not expect role-in-use dialog copy, got: %s", body)
	}
}
