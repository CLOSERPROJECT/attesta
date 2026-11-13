package living

import (
	"testing"
)

func containerRegLeafTask(t *testing.T) map[string]any {
	t.Helper()
	line := lineJob(t)
	task, ok := TaskByID(line.Tasks, "containerReg")
	if !ok || task.IsGroup() {
		t.Fatal("containerReg leaf missing")
	}
	return task
}

func containerRegSchema(t *testing.T) map[string]any {
	t.Helper()
	task := containerRegLeafTask(t)
	schema, _ := task["schema"].(map[string]any)
	if schema == nil {
		t.Fatal("containerReg schema missing")
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
	if _, has := props["secondarySourceLotId"]; has {
		t.Fatal("secondarySourceLotId form enum must be removed")
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

	reg := containerRegLeafTask(t)
	enums, _ := reg["enumReads"].(map[string]any)
	if _, has := enums["secondarySourceLotId"]; has {
		t.Fatal("enumReads.secondarySourceLotId must be removed")
	}
	if _, has := enums["sourceLotId"]; !has {
		t.Fatal("enumReads.sourceLotId missing")
	}
	if _, has := reg["initialData"]; !has {
		t.Fatal("containerReg.initialData missing")
	}
}

func TestContainerRegUiSchemaHidesContinuesContainerId(t *testing.T) {
	reg := containerRegLeafTask(t)
	ui, _ := reg["uiSchema"].(map[string]any)
	if ui == nil {
		t.Fatal("containerReg uiSchema missing")
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
