package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLayoutHidesLeaveOrganizationFromAccountMenu(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	view := PageBase{
		ShowLogout:    true,
		ShowMyOrgLink: true,
	}
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if strings.Contains(body, "Leave organization") {
		t.Fatalf("expected no leave organization in account menu, got:\n%s", body)
	}
	if strings.Contains(body, `action="/my/leave-organization"`) {
		t.Fatalf("expected no leave organization form in account menu, got:\n%s", body)
	}
}
