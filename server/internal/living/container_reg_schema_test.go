package living

import (
	"testing"
)

func containerRegRegisterTask(t *testing.T) map[string]any {
	t.Helper()
	line := lineJob(t)
	group, ok := TaskByID(line.Tasks, "containerReg")
	if !ok || !group.IsGroup() {
		t.Fatal("containerReg group missing")
	}
	task, ok := TaskByID(group.ChildTasks(), "register")
	if !ok {
		t.Fatal("containerReg.register missing")
	}
	return task
}

func containerRegSchema(t *testing.T) map[string]any {
	t.Helper()
	task := containerRegRegisterTask(t)
	schema, _ := task["schema"].(map[string]any)
	if schema == nil {
		t.Fatal("containerReg.register schema missing")
	}
	return schema
}

func TestContainerRegSchemaExposesIncompleteDefaultFalse(t *testing.T) {
	schema := containerRegSchema(t)
	props, _ := schema["properties"].(map[string]any)
	arr, _ := props["polvereDiVetro"].(map[string]any)
	items, _ := arr["items"].(map[string]any)
	itemProps, _ := items["properties"].(map[string]any)
	inc, _ := itemProps["incomplete"].(map[string]any)
	if inc == nil {
		t.Fatal("material row incomplete missing")
	}
	if inc["type"] != "boolean" {
		t.Fatalf("incomplete.type = %#v, want boolean", inc["type"])
	}
	if inc["default"] != false {
		t.Fatalf("incomplete.default = %#v, want false", inc["default"])
	}
}

func TestContainerRegSchemaDropsDualSourcesAndExposesContinuesContainerId(t *testing.T) {
	schema := containerRegSchema(t)
	props, _ := schema["properties"].(map[string]any)
	if _, has := props["secondarySourceLoadId"]; has {
		t.Fatal("secondarySourceLoadId form enum must be removed")
	}
	if _, has := schema["if"]; has {
		t.Fatal("dualSources if/then must be removed")
	}
	arr, _ := props["polvereDiVetro"].(map[string]any)
	items, _ := arr["items"].(map[string]any)
	itemProps, _ := items["properties"].(map[string]any)
	if _, has := itemProps["dualSources"]; has {
		t.Fatal("dualSources row checkbox must be removed")
	}
	cont, _ := itemProps["continuesContainerId"].(map[string]any)
	if cont == nil || cont["type"] != "string" {
		t.Fatalf("continuesContainerId = %#v, want string", cont)
	}

	reg := containerRegRegisterTask(t)
	enums, _ := reg["enumReads"].(map[string]any)
	if _, has := enums["secondarySourceLoadId"]; has {
		t.Fatal("enumReads.secondarySourceLoadId must be removed")
	}
	if _, has := enums["sourceLoadId"]; !has {
		t.Fatal("enumReads.sourceLoadId missing")
	}
	if _, has := reg["initialData"]; !has {
		t.Fatal("register.initialData missing")
	}
}

func TestContainerRegUiSchemaHidesContinuesContainerId(t *testing.T) {
	reg := containerRegRegisterTask(t)
	ui, _ := reg["uiSchema"].(map[string]any)
	if ui == nil {
		t.Fatal("containerReg.register uiSchema missing")
	}
	mat, _ := ui["polvereDiVetro"].(map[string]any)
	items, _ := mat["items"].(map[string]any)
	contUi, _ := items["continuesContainerId"].(map[string]any)
	comps, _ := contUi["ui:components"].(map[string]any)
	if comps["stringField"] != "hiddenField" {
		t.Fatalf("continuesContainerId ui:components = %#v, want stringField: hiddenField", contUi)
	}
	if _, has := contUi["ui:widget"]; has {
		t.Fatal("ui:widget is RJSF-only; sjsf uses ui:components stringField: hiddenField")
	}
	opts, _ := contUi["ui:options"].(map[string]any)
	if opts["presenceMark"] != true {
		t.Fatalf("continuesContainerId ui:options.presenceMark = %#v, want true", opts["presenceMark"])
	}
	if opts["title"] != "Continua" {
		t.Fatalf("continuesContainerId ui:options.title = %#v, want Continua", opts["title"])
	}
	itemOpts, _ := items["ui:options"].(map[string]any)
	order, _ := itemOpts["order"].([]any)
	if len(order) < 1 || order[0] != "continuesContainerId" {
		t.Fatalf("items order = %#v, want continuesContainerId first for presence mark hug", order)
	}
}
