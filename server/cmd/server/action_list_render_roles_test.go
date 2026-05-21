package main

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestBuildProcessActionListViewDoesNotFilterByCurrentActiveRole(t *testing.T) {
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
	server := &Server{}
	actor := Actor{
		Role:      "dep2",
		RoleSlugs: []string{"dep1", "dep2"},
	}
	view := server.buildProcessActionListView(t.Context(), testRuntimeConfig(), "workflow", process, actor, "", "", false)

	if view.Action == nil || view.Action.SubstepID != "1.1" {
		t.Fatalf("expected dep1 step to remain visible after dep2 completion context, got %#v", view.Action)
	}
}
