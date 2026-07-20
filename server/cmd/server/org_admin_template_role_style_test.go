package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestOrgAdminTemplateRolePillRendersCSSVariables(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := OrgAdminView{
		Header: PageHeaderView{
			Title:       "Organization admin dashboard",
			Description: "Switch between organization settings, roles, and members.",
			BackHref:    "/",
		},
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
						Slug:     "org-admin",
						Name:     "Org Admin",
						Palette:  "red",
						Selected: true,
					},
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
	tmpl := parseTestTemplates(t)
	view := OrgAdminView{
		Header: PageHeaderView{
			Title:       "Organization admin dashboard",
			Description: "Switch between organization settings, roles, and members.",
			BackHref:    "/",
		},
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
	if strings.Contains(compactBody, `class="secondary js-invite-copy"`) || strings.Contains(compactBody, `data-copy-invite-link="/invite/token-pending"`) {
		t.Fatalf("did not expect invite copy button markup, got body: %s", body)
	}
	if !strings.Contains(compactBody, `if (addUserDialog && true && !addUserDialog.open)`) {
		t.Fatalf("expected add-user dialog reopen script when invite link is present, got body: %s", body)
	}
	if strings.Contains(body, "Last invite:") {
		t.Fatalf("last invite text should be hidden, got body: %s", body)
	}
}
