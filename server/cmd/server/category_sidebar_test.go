package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestCategorySidebarTemplateHTMXLeaf(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := CategorySidebarView{
		Title: "Stream Categories",
		Categories: []CategorySidebarCategoryView{{
			Slug: "supply-chain", Name: "Supply Chain", IconURL: "/c.svg", Expanded: true,
			SubCategories: []CategorySidebarLeafView{{
				Slug: "procurement", Name: "Procurement", Active: true,
				PartialURL: "/streams/public?category=supply-chain&subCategory=procurement",
				PushURL:    "/?category=supply-chain&subCategory=procurement",
			}},
		}},
	}
	if err := tmpl.ExecuteTemplate(&out, "category_sidebar", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		`class="category-sidebar"`,
		`class="category-sidebar-title">Stream Categories<`,
		`hx-get="/streams/public?category=supply-chain&amp;subCategory=procurement"`,
		`hx-push-url="/?category=supply-chain&amp;subCategory=procurement"`,
		`is-active`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %q in %s", want, body)
		}
	}
	if strings.Contains(body, `href="#`) {
		t.Fatalf("HTMX leaf must not use href anchor, got %s", body)
	}
}

func TestCategorySidebarTemplateAnchorLeaf(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var out bytes.Buffer
	view := CategorySidebarView{
		Title: "Stream Categories",
		Categories: []CategorySidebarCategoryView{{
			Slug: "supply-chain", Name: "Supply Chain", Expanded: true,
			SubCategories: []CategorySidebarLeafView{{
				Slug: "procurement", Name: "Procurement",
				Href: "#cat-supply-chain--procurement",
			}},
		}},
	}
	if err := tmpl.ExecuteTemplate(&out, "category_sidebar", view); err != nil {
		t.Fatalf("render: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, `href="#cat-supply-chain--procurement"`) {
		t.Fatalf("missing anchor href, got %s", body)
	}
	if strings.Contains(body, "hx-get=") || strings.Contains(body, "<button") {
		t.Fatalf("anchor leaf must not render HTMX button, got %s", body)
	}
}
