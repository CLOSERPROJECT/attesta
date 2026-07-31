package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestMyHomeStreamGroupTemplateRendersCategoryHeaders(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	group := MyHomeStreamGroupView{
		CategoryName:           "Supply Chain",
		CategoryIconURL:        "/static/taxonomy/batch-traceability.svg",
		SubCategoryName:        "Procurement",
		SubCategoryIconURL:     "/static/taxonomy/procurement-workflow.svg",
		SubCategoryDescription: "PO management",
		Streams: []ManagedPublicStreamCardView{
			{Card: PublicStreamCardView{Name: "Example Stream"}},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "my_home_stream_group", group); err != nil {
		t.Fatalf("render my_home_stream_group template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-home-results-category-header"`,
		`class="public-home-results-category-name">Supply Chain<`,
		`class="public-home-results-subcategory-header"`,
		`class="public-home-results-subcategory-name"`,
		"Procurement",
		"PO management",
		`class="public-home-stream-grid"`,
		"Example Stream",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered group, got: %s", want, body)
		}
	}
}

func TestMyHomeStreamGroupTemplateUncategorizedOmitsSubcategoryHeader(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	group := MyHomeStreamGroupView{
		CategoryName:  "Uncategorized",
		Uncategorized: true,
		Streams: []ManagedPublicStreamCardView{
			{Card: PublicStreamCardView{Name: "Loose Stream"}},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "my_home_stream_group", group); err != nil {
		t.Fatalf("render my_home_stream_group template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="public-home-results-category-name">Uncategorized<`,
		`class="public-home-stream-grid"`,
		"Loose Stream",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in rendered uncategorized group, got: %s", want, body)
		}
	}
	if strings.Contains(body, `class="public-home-results-subcategory-header"`) {
		t.Fatalf("uncategorized group must not render subcategory header, got: %s", body)
	}
}

func TestMyHomeStreamGroupTemplateOmitsCategoryHeaderWhenEmpty(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	group := MyHomeStreamGroupView{
		SubCategoryName: "Order Fulfillment",
		Streams: []ManagedPublicStreamCardView{
			{Card: PublicStreamCardView{Name: "Follow-on Stream"}},
		},
	}
	if err := tmpl.ExecuteTemplate(&out, "my_home_stream_group", group); err != nil {
		t.Fatalf("render my_home_stream_group template: %v", err)
	}
	body := out.String()
	if strings.Contains(body, `class="public-home-results-category-header"`) {
		t.Fatalf("empty CategoryName must omit category header, got: %s", body)
	}
	if !strings.Contains(body, `class="public-home-results-subcategory-name"`) || !strings.Contains(body, "Order Fulfillment") {
		t.Fatalf("expected subcategory header, got: %s", body)
	}
}

func TestHomePickerBodyTemplateRendersEmptyState(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := HomeWorkflowPickerView{
		PageBase: PageBase{Body: "home_picker_body"},
	}
	if err := tmpl.ExecuteTemplate(&out, "home_picker_body", view); err != nil {
		t.Fatalf("render home_picker_body template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`my-home"`,
		"Choose a stream",
		`class="empty-state"`,
		`class="empty-state-title">No streams available<`,
		"Streams for your organization and roles will appear here.",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in empty home picker, got: %s", want, body)
		}
	}
	if strings.Contains(body, `class="my-home-catalog"`) {
		t.Fatalf("empty home picker must not render catalog, got: %s", body)
	}
}

func TestHomePickerBodyTemplateRendersCreateStreamAction(t *testing.T) {
	tmpl := parseTestTemplates(t)

	var out bytes.Buffer
	view := HomeWorkflowPickerView{
		PageBase:         PageBase{Body: "home_picker_body"},
		ShowCreateStream: true,
	}
	if err := tmpl.ExecuteTemplate(&out, "home_picker_body", view); err != nil {
		t.Fatalf("render home_picker_body template: %v", err)
	}
	body := out.String()

	for _, want := range []string{
		`class="page-header-actions"`,
		`href="/my/organization/formata-builder"`,
		"Create a stream",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in home picker with create CTA, got: %s", want, body)
		}
	}
}
