package living

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseTaskPathFlatAndNested(t *testing.T) {
	flat, err := ParseTaskPath("intake/truckArrive")
	if err != nil {
		t.Fatal(err)
	}
	if flat.JobID != "intake" || flat.TaskID != "truckArrive" || flat.Nested() {
		t.Fatalf("flat = %#v", flat)
	}

	nested, err := ParseTaskPath("intake/lots/L1/register")
	if err != nil {
		t.Fatal(err)
	}
	if nested.JobID != "intake" || nested.GroupID != "lots" || nested.GroupInstanceID != "L1" || nested.TaskID != "register" {
		t.Fatalf("nested = %#v", nested)
	}
	if !nested.Nested() {
		t.Fatal("expected Nested")
	}
}

func TestParseTaskPathRejectsBadShapes(t *testing.T) {
	for _, path := range []string{
		"",
		"solo",
		"/task",
		"job/",
		"a/b/c",
		"a/b/c/d/e",
		"intake//L1/register",
		"intake/lots//register",
	} {
		if _, err := ParseTaskPath(path); err == nil {
			t.Fatalf("ParseTaskPath(%q) should fail", path)
		}
	}
}

func TestTaskConfigGroupHelpers(t *testing.T) {
	leaf := TaskConfig{"id": "register", "schema": map[string]any{"type": "object"}}
	if leaf.IsGroup() {
		t.Fatal("leaf IsGroup")
	}
	if len(leaf.ChildTasks()) != 0 {
		t.Fatal("leaf children")
	}

	group := TaskConfig{
		"id":         "lots",
		"kind":       "group",
		"repeatable": true,
		"startLabel": "Nuovo lotto",
		"closeLabel": "Chiudi scarico",
		"tasks": []any{
			map[string]any{"id": "register", "roles": []any{"administration"}},
		},
	}
	if !group.IsGroup() {
		t.Fatal("group IsGroup")
	}
	children := group.ChildTasks()
	if len(children) != 1 || children[0].ID() != "register" {
		t.Fatalf("ChildTasks = %#v", children)
	}

	yamlShaped := TaskConfig{
		"id":   "lots",
		"kind": "group",
		"substeps": []any{
			map[string]any{"id": "register"},
		},
	}
	if got := yamlShaped.ChildTasks(); len(got) != 1 || got[0].ID() != "register" {
		t.Fatalf("substeps ChildTasks = %#v", got)
	}
}

func TestLeafTaskByPath(t *testing.T) {
	tasks := []TaskConfig{
		{"id": "truckArrive", "schema": map[string]any{}},
		{
			"id":   "lots",
			"kind": "group",
			"tasks": []TaskConfig{
				{"id": "register", "schema": map[string]any{}},
			},
		},
		{"id": "weightCheck", "schema": map[string]any{}},
	}

	flat, err := ParseTaskPath("intake/truckArrive")
	if err != nil {
		t.Fatal(err)
	}
	leaf, ok := LeafTaskByPath(tasks, flat)
	if !ok || leaf.ID() != "truckArrive" {
		t.Fatalf("flat leaf = %#v ok=%v", leaf, ok)
	}

	nested, err := ParseTaskPath("intake/lots/L1/register")
	if err != nil {
		t.Fatal(err)
	}
	leaf, ok = LeafTaskByPath(tasks, nested)
	if !ok || leaf.ID() != "register" {
		t.Fatalf("nested leaf = %#v ok=%v", leaf, ok)
	}

	if _, ok := LeafTaskByPath(tasks, TaskPathRef{JobID: "intake", TaskID: "lots"}); ok {
		t.Fatal("group id must not resolve as leaf")
	}
	if _, ok := LeafTaskByPath(tasks, TaskPathRef{
		JobID: "intake", GroupID: "missing", GroupInstanceID: "L1", TaskID: "register",
	}); ok {
		t.Fatal("missing group")
	}
}

func TestValidateTaskConfigsUnlockFrom(t *testing.T) {
	if err := validateTaskConfigs([]TaskConfig{
		{"id": "open"},
		{"id": "mid"},
		{"id": "side", "unlockFrom": "open"},
	}); err != nil {
		t.Fatalf("valid unlockFrom = %v", err)
	}

	cases := []struct {
		name string
		tasks []TaskConfig
		want string
	}{
		{
			name:  "first_task",
			tasks: []TaskConfig{{"id": "open", "unlockFrom": "ghost"}},
			want:  "first task must not declare unlockFrom",
		},
		{
			name:  "non_string",
			tasks: []TaskConfig{{"id": "open"}, {"id": "side", "unlockFrom": 1}},
			want:  "unlockFrom must be a string",
		},
		{
			name:  "empty",
			tasks: []TaskConfig{{"id": "open"}, {"id": "side", "unlockFrom": ""}},
			want:  "non-empty",
		},
		{
			name:  "self",
			tasks: []TaskConfig{{"id": "open"}, {"id": "side", "unlockFrom": "side"}},
			want:  "self",
		},
		{
			name:  "forward",
			tasks: []TaskConfig{{"id": "open"}, {"id": "side", "unlockFrom": "later"}, {"id": "later"}},
			want:  "earlier sibling",
		},
		{
			name:  "unknown",
			tasks: []TaskConfig{{"id": "open"}, {"id": "side", "unlockFrom": "missing"}},
			want:  "not a sibling",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTaskConfigs(tc.tasks)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v want substring %q", err, tc.want)
			}
		})
	}
}

func TestValidateTaskConfigsGroups(t *testing.T) {
	if err := validateTaskConfigs([]TaskConfig{
		{"id": "a"},
		{
			"id":   "lots",
			"kind": "group",
			"tasks": []TaskConfig{
				{"id": "register"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{"id": "lots", "kind": "group"},
	}); err == nil || !strings.Contains(err.Error(), "non-empty children") {
		t.Fatalf("empty group err = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{
			"id":     "lots",
			"kind":   "group",
			"schema": map[string]any{"type": "object"},
			"tasks":  []TaskConfig{{"id": "register"}},
		},
	}); err == nil || !strings.Contains(err.Error(), "leaf Commit fields") {
		t.Fatalf("group+schema err = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{
			"id": "fake",
			"tasks": []TaskConfig{
				{"id": "child"},
			},
		},
	}); err == nil || !strings.Contains(err.Error(), "kind: group") {
		t.Fatalf("leaf with children err = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{"id": "a"},
		{"id": "a"},
	}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("dup err = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{
			"id":   "lots",
			"kind": "group",
			"tasks": []TaskConfig{
				{"id": "register"},
				{"id": "register"},
			},
		},
	}); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("nested dup err = %v", err)
	}

	if err := validateTaskConfigs([]TaskConfig{
		{
			"id":   "lots",
			"kind": "group",
			"tasks": []TaskConfig{
				{
					"id":   "inner",
					"kind": "group",
					"tasks": []TaskConfig{
						{"id": "register"},
					},
				},
			},
		},
	}); err == nil || !strings.Contains(err.Error(), "one nesting level") {
		t.Fatalf("group-under-group err = %v", err)
	}
}

func TestStartsWhenStillRejectsNestedTaskPath(t *testing.T) {
	yaml := []byte(`
id: nested-starts-when
org: test
steps:
  intake:
    repeatable: true
    instanceId: $truck.id
    label: $truck.id
    substeps:
      - id: register
        roles: [administration]
  line:
    repeatable: false
    instanceId: $shift.id
    label: $shift.id
    startsWhen:
      substep: intake/lots/L1/register
      quantifier: some
    substeps:
      - id: shift
        roles: [administration]
`)
	if _, err := ParseFeragameConfigFixture(yaml); err == nil || !strings.Contains(err.Error(), "startsWhen.task") {
		t.Fatalf("startsWhen nested = %v", err)
	}
}

func TestJobJSONRejectsEmptyGroup(t *testing.T) {
	raw := []byte(`{
		"repeatable": true,
		"instanceId": "$truck.id",
		"label": "$truck.id",
		"tasks": [
			{"id": "lots", "kind": "group", "tasks": []}
		]
	}`)
	var job Job
	if err := json.Unmarshal(raw, &job); err == nil {
		t.Fatal("expected empty group rejection")
	}
}

func TestFixtureLoadsGroupNodes(t *testing.T) {
	yaml := []byte(`
id: group-fixture
org: test
steps:
  intake:
    repeatable: true
    instanceId: $truck.id
    label: $truck.id
    startLabel: Nuovo camion
    substeps:
      - id: truckArrive
        roles: [administration]
        schema:
          type: object
      - id: lots
        kind: group
        repeatable: true
        startLabel: Nuovo lotto
        closeLabel: Chiudi scarico
        substeps:
          - id: register
            roles: [administration]
            schema:
              type: object
      - id: weightCheck
        roles: [administration]
        schema:
          type: object
`)
	cfg, err := ParseFeragameConfigFixture(yaml)
	if err != nil {
		t.Fatal(err)
	}
	intakeAny, ok := cfg.Jobs.Get("intake")
	if !ok {
		t.Fatal("missing intake")
	}
	intake := intakeAny.(Job)
	if got := OrderedTaskIDs(intake.Tasks); strings.Join(got, ",") != "truckArrive,lots,weightCheck" {
		t.Fatalf("tasks = %v", got)
	}
	lots, ok := TaskByID(intake.Tasks, "lots")
	if !ok || !lots.IsGroup() {
		t.Fatalf("lots = %#v", lots)
	}
	if lots["startLabel"] != "Nuovo lotto" || lots["closeLabel"] != "Chiudi scarico" {
		t.Fatalf("group labels = %#v", lots)
	}
	if _, hasSubsteps := lots["substeps"]; hasSubsteps {
		t.Fatal("fixture must rename nested substeps → tasks")
	}
	children := lots.ChildTasks()
	if len(children) != 1 || children[0].ID() != "register" || children[0].IsGroup() {
		t.Fatalf("children = %#v", children)
	}

	ref, err := ParseTaskPath("intake/lots/L1/register")
	if err != nil {
		t.Fatal(err)
	}
	leaf, ok := LeafTaskByPath(intake.Tasks, ref)
	if !ok || leaf.ID() != "register" {
		t.Fatalf("LeafTaskByPath = %#v ok=%v", leaf, ok)
	}
}
