package main

import (
	"bytes"
	"fmt"
	"html/template"
	"path/filepath"
	"strings"
	"testing"
)

var templateGlobPatterns = []string{
	"templates/*.html",
	"templates/pages/*.html",
	"templates/components/*.html",
}

func templateFuncs() template.FuncMap {
	return template.FuncMap{
		"streamTimelineStep": func(step TimelineStep, hideStatus bool) StreamTimelineStepView {
			return StreamTimelineStepView{Step: step, HideStatus: hideStatus}
		},
		"streamTimelineSubstep": func(substep TimelineSubstep, hideStatus bool) StreamTimelineSubstepView {
			return StreamTimelineSubstepView{Substep: substep, HideStatus: hideStatus}
		},
		"substepShellDisplay": substepShellDisplay,
		"effectiveSubstepBodyMode": effectiveSubstepBodyMode,
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict: odd number of arguments")
			}
			out := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key at index %d is not a string", i/2)
				}
				out[key] = values[i+1]
			}
			return out, nil
		},
		"replace": func(s, old, new string) string {
			return strings.ReplaceAll(s, old, new)
		},
	}
}

// withTemplateFuncs registers shared funcs plus render, which executes a named
// template by string (Go's {{ template }} action only accepts constant names).
func withTemplateFuncs(tmpl *template.Template) *template.Template {
	funcs := templateFuncs()
	funcs["render"] = func(name string, data any) (template.HTML, error) {
		var buf bytes.Buffer
		if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
			return "", err
		}
		return template.HTML(buf.String()), nil
	}
	return tmpl.Funcs(funcs)
}

func parseTemplates() (*template.Template, error) {
	tmpl := withTemplateFuncs(template.New(""))
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
	tmpl := withTemplateFuncs(template.New(""))
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
