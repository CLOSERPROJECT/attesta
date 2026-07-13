package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTemplates(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	serverRoot := filepath.Join(wd, "..", "..")
	if err := os.Chdir(serverRoot); err != nil {
		t.Fatalf("chdir to server root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(wd)
	})

	tmpl, err := parseTemplates()
	if err != nil {
		t.Fatalf("parseTemplates: %v", err)
	}
	if tmpl == nil {
		t.Fatal("expected non-nil template set")
	}

	for _, name := range []string{
		"layout.html",
		"page_header",
		"home_body",
		"process_body",
		"stream.html",
	} {
		if tmpl.Lookup(name) == nil {
			t.Errorf("missing template %q", name)
		}
	}
}
