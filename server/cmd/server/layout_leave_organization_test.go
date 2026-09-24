package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLayoutShowsLeaveOrganizationWhenAffiliated(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	view := PageBase{
		ShowLogout:            true,
		ShowLeaveOrganization: true,
		LeaveOrganizationPath: leaveOrganizationPath(),
	}
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", view); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if !strings.Contains(body, `action="/my/leave-organization"`) {
		t.Fatalf("expected leave organization form action, got:\n%s", body)
	}
	if !strings.Contains(body, "Leave organization") {
		t.Fatalf("expected Leave organization label, got:\n%s", body)
	}
}

func TestLayoutHidesLeaveOrganizationWhenNotAffiliated(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", PageBase{ShowLogout: true}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if strings.Contains(body, "Leave organization") {
		t.Fatalf("expected no leave organization when not affiliated, got:\n%s", body)
	}
	if strings.Contains(body, `action="/my/leave-organization"`) {
		t.Fatalf("expected no leave organization form when not affiliated, got:\n%s", body)
	}
}
