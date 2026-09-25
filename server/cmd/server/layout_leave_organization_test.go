package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLayoutLeaveOnlyInsideAccountSettings(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	view := PageBase{
		ShowLogout:          true,
		ShowAccountSettings: true,
		AccountOrgName:      "Acme Org",
		LeavePath:           leaveOrganizationPath(),
		CanLeave:            true,
	}
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if !strings.Contains(body, `id="account-settings-dialog"`) {
		t.Fatal("expected account settings dialog")
	}
	if !strings.Contains(body, `action="/my/leave-organization"`) {
		t.Fatal("expected leave form inside leave confirm dialog")
	}
	// Leave must not be a direct account-menu form row.
	menuStart := strings.Index(body, `class="account-dropdown"`)
	menuEnd := strings.Index(body, `id="account-settings-dialog"`)
	if menuStart == -1 || menuEnd == -1 || menuEnd <= menuStart {
		t.Fatalf("expected account dropdown before settings dialog, got:\n%s", body)
	}
	menu := body[menuStart:menuEnd]
	if strings.Contains(menu, `action="/my/leave-organization"`) {
		t.Fatalf("leave form must not be in account dropdown, got:\n%s", menu)
	}
}
