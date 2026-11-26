package org

import (
	"testing"
)

func containerRegLeafTask(t *testing.T) map[string]any {
	t.Helper()
	lp := lotProcessingJob(t)
	task, ok := TaskByID(lp.Tasks, "containerReg")
	if !ok {
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

func TestContainerRegSchemaFlatMaterialWeight(t *testing.T) {
	schema := containerRegSchema(t)
	props, _ := schema["properties"].(map[string]any)
	req, _ := schema["required"].([]any)
	if len(req) != 1 || req[0] != "material" {
		t.Fatalf("required = %#v, want [material]", req)
	}
	for _, gone := range []string{"polvereDiVetro", "plastica", "sourceLotId", "secondarySourceLotId", "weight", "incomplete"} {
		if _, has := props[gone]; has {
			t.Fatalf("schema property %s must be removed from root (weight/incomplete live under dependencies.material)", gone)
		}
	}
	if _, has := schema["if"]; has {
		t.Fatal("dualSources if/then must be removed")
	}
	mat, _ := props["material"].(map[string]any)
	if mat == nil || mat["type"] != "string" {
		t.Fatalf("material = %#v, want string", mat)
	}
	cont, _ := props["continuesContainerId"].(map[string]any)
	if cont == nil || cont["type"] != "string" {
		t.Fatalf("continuesContainerId = %#v, want string", cont)
	}
	deps, _ := schema["dependencies"].(map[string]any)
	matDep, _ := deps["material"].(map[string]any)
	if matDep == nil {
		t.Fatal("dependencies.material missing")
	}
	depReq, _ := matDep["required"].([]any)
	if len(depReq) != 1 || depReq[0] != "weight" {
		t.Fatalf("dependencies.material.required = %#v, want [weight]", depReq)
	}
	depProps, _ := matDep["properties"].(map[string]any)
	weight, _ := depProps["weight"].(map[string]any)
	if weight == nil || weight["type"] != "number" {
		t.Fatalf("dependencies.material.properties.weight = %#v, want number", weight)
	}
	inc, _ := depProps["incomplete"].(map[string]any)
	if inc == nil || inc["type"] != "boolean" {
		t.Fatalf("dependencies.material.properties.incomplete = %#v, want boolean", inc)
	}
	if inc["default"] != false {
		t.Fatalf("incomplete.default = %#v, want false", inc["default"])
	}
}

func TestContainerRegDerivedChromeAndNoEnumReads(t *testing.T) {
	reg := containerRegLeafTask(t)
	derived, _ := reg["derived"].(map[string]any)
	cont, _ := derived["continuesContainerId"].(map[string]any)
	if cont == nil {
		t.Fatal("derived.continuesContainerId missing")
	}
	if expr, _ := cont["expr"].(string); expr == "" {
		t.Fatal("derived.continuesContainerId.expr empty")
	}
	chrome, _ := reg["chrome"].(map[string]any)
	prior, _ := chrome["priorWeight"].(map[string]any)
	if prior == nil {
		t.Fatal("chrome.priorWeight missing")
	}
	if prior["hideIfNull"] != true {
		t.Fatalf("chrome.priorWeight.hideIfNull = %#v, want true", prior["hideIfNull"])
	}
	if prior["kind"] != "readonly" {
		t.Fatalf("chrome.priorWeight.kind = %#v, want readonly", prior["kind"])
	}
	if enums, _ := reg["enumReads"].(map[string]any); enums != nil {
		if _, has := enums["sourceLotId"]; has {
			t.Fatal("enumReads.sourceLotId must be removed")
		}
		if _, has := enums["secondarySourceLotId"]; has {
			t.Fatal("enumReads.secondarySourceLotId must be removed")
		}
	}
	if _, has := reg["initialData"]; has {
		t.Fatal("containerReg.initialData must be removed")
	}
}

func TestContainerRegUiSchemaHidesContinuesContainerId(t *testing.T) {
	reg := containerRegLeafTask(t)
	ui, _ := reg["uiSchema"].(map[string]any)
	if ui == nil {
		t.Fatal("containerReg uiSchema missing")
	}
	contUi, _ := ui["continuesContainerId"].(map[string]any)
	comps, _ := contUi["ui:components"].(map[string]any)
	if comps["stringField"] != "hiddenField" {
		t.Fatalf("continuesContainerId ui:components = %#v, want stringField: hiddenField", contUi)
	}
	if _, has := contUi["ui:widget"]; has {
		t.Fatal("ui:widget is RJSF-only; sjsf uses ui:components stringField: hiddenField")
	}
	opts, _ := ui["ui:options"].(map[string]any)
	order, _ := opts["order"].([]any)
	want := []any{"material", "weight", "incomplete", "continuesContainerId"}
	if len(order) != len(want) {
		t.Fatalf("order = %#v, want %#v", order, want)
	}
	for i, id := range want {
		if order[i] != id {
			t.Fatalf("order = %#v, want %#v", order, want)
		}
	}
}
