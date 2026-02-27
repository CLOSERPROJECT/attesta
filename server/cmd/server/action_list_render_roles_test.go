package main

import (
	"html/template"
	"net/http/httptest"
	"strings"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRenderActionListDoesNotFilterByCurrentActiveRole(t *testing.T) {
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"1.2": {State: "pending"},
			"1.3": {State: "pending"},
			"2.1": {State: "pending"},
			"2.2": {State: "pending"},
			"3.1": {State: "pending"},
			"3.2": {State: "pending"},
		},
	}
	server := &Server{
		tmpl: template.Must(template.New("test").Parse(`{{define "action_list.html"}}{{range .Actions}}{{.SubstepID}}|{{end}}{{end}}`)),
		configProvider: func() (RuntimeConfig, error) {
			return testRuntimeConfig(), nil
		},
	}
	actor := Actor{
		UserID:    "u-session",
		Role:      "dep2",
		RoleSlugs: []string{"dep1", "dep2"},
	}
	rec := httptest.NewRecorder()

	server.renderActionList(rec, nil, process, actor, "")

	if rec.Code != 200 {
		t.Fatalf("status = %d, want %d", rec.Code, 200)
	}
	if !strings.Contains(rec.Body.String(), "1.1|") {
		t.Fatalf("expected dep1 step to remain visible after dep2 completion context, got %q", rec.Body.String())
	}
}
