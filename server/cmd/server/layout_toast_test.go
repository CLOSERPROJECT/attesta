package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestLayoutIncludesToastHost(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", PageBase{}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if !strings.Contains(body, `id="toast-host"`) {
		t.Fatalf("expected #toast-host in layout, got:\n%s", body)
	}
	if !strings.Contains(body, `class="toast-host"`) {
		t.Fatalf("expected .toast-host class in layout, got:\n%s", body)
	}
	if !strings.Contains(body, `aria-live="polite"`) {
		t.Fatalf("expected aria-live=polite on toast host, got:\n%s", body)
	}
}

func TestLayoutRendersToastError(t *testing.T) {
	tmpl := parseTestTemplates(t)
	var rendered bytes.Buffer
	if err := tmpl.ExecuteTemplate(&rendered, "layout.html", PageBase{ToastError: "Instance name is required"}); err != nil {
		t.Fatalf("ExecuteTemplate() error = %v", err)
	}
	body := rendered.String()
	if !strings.Contains(body, `class="error" data-toast`) {
		t.Fatalf("expected error data-toast, got:\n%s", body)
	}
	if !strings.Contains(body, "Instance name is required") {
		t.Fatalf("expected toast message, got:\n%s", body)
	}
}
