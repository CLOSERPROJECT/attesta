package main

import (
	"fmt"
	"html/template"
	"path/filepath"
	"testing"
)

var templateGlobPatterns = []string{
	"templates/*.html",
	"templates/pages/*.html",
	"templates/components/*.html",
}

func parseTemplates() (*template.Template, error) {
	tmpl := template.New("")
	var err error
	for _, pattern := range templateGlobPatterns {
		tmpl, err = tmpl.ParseGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", pattern, err)
		}
	}
	return tmpl, nil
}

func parseTestTemplates(t testing.TB) *template.Template {
	t.Helper()
	tmpl := template.New("")
	for _, pattern := range templateGlobPatterns {
		fullPattern := filepath.Join("..", "..", pattern)
		var err error
		tmpl, err = tmpl.ParseGlob(fullPattern)
		if err != nil {
			t.Fatalf("parse templates %s: %v", fullPattern, err)
		}
	}
	return tmpl
}
