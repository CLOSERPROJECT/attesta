package main

import (
	"bytes"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

func TestTimelineTemplateRendersStepSummaryLayout(t *testing.T) {
	tmpl := template.Must(template.ParseGlob(filepath.Join("..", "..", "templates", "*.html")))

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "timeline.html", []TimelineStep{{
		StepID:     "1",
		OrgName:    "Acme Org",
		OrgLogoURL: "/organization/logo/acme",
		Title:      "Review batch",
		Expanded:   true,
	}}); err != nil {
		t.Fatalf("render timeline template: %v", err)
	}
	body := out.String()

	if !strings.Contains(body, `class="timeline-step-summary-main"`) {
		t.Fatalf("expected summary main layout, got: %s", body)
	}
	if !strings.Contains(body, `class="timeline-step-org-mark"`) {
		t.Fatalf("expected org mark, got: %s", body)
	}
	if !strings.Contains(body, `src="/organization/logo/acme"`) {
		t.Fatalf("expected org logo url, got: %s", body)
	}
	if !strings.Contains(body, `class="pill timeline-step-number"`) {
		t.Fatalf("expected step number pill, got: %s", body)
	}
	if !strings.Contains(body, "Acme Org") || !strings.Contains(body, "Review batch") {
		t.Fatalf("expected org and title copy, got: %s", body)
	}
}
