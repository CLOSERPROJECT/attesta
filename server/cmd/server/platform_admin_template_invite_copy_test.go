package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlatformAdminTemplateInviteCopyButton(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))
	view := PlatformAdminView{
		InviteLink: "/invite/platform-token-pending",
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "platform_admin_body", view); err != nil {
		t.Fatalf("render platform admin template: %v", err)
	}
	body := out.String()

	if strings.Contains(body, "Invite link:") {
		t.Fatalf("raw invite link text should be hidden, got body: %s", body)
	}
	if !strings.Contains(body, `class="secondary js-invite-copy"`) {
		t.Fatalf("expected invite copy button class, got body: %s", body)
	}
	if !strings.Contains(body, `data-copy-invite-link="/invite/platform-token-pending"`) {
		t.Fatalf("expected invite copy button, got body: %s", body)
	}
	if !strings.Contains(body, `data-copy-icon-default style="display: inline-block;"`) {
		t.Fatalf("expected default copy icon visible by default, got body: %s", body)
	}
	if !strings.Contains(body, `data-copy-icon-done style="display: none;"`) {
		t.Fatalf("expected done copy icon hidden by default, got body: %s", body)
	}
	if !strings.Contains(body, "<span data-copy-label>Copy invite link</span>") {
		t.Fatalf("expected copy label in button, got body: %s", body)
	}
}
