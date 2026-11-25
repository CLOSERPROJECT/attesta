package org

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestCommitRegisterRunsOnSubmitHTTP(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
		},
	}
	c, domains, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.SubmitBindings = ExprBindings{"qrEndpoint": "https://plant.example/app"}

	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	loadID := firstLoadID(result.Domain)
	if result.Domain.Kind("lot")[loadID]["client"] != "CO" {
		t.Fatalf("domain rolled back? %#v", result.Domain.Kind("lot"))
	}
	if len(port.calls) != 1 {
		t.Fatalf("http calls = %#v", port.calls)
	}
	if port.calls[0].Method != "POST" || port.calls[0].URL != "/effects/labels" {
		t.Fatalf("call0 = %#v", port.calls[0])
	}
	body0, _ := port.calls[0].Body.(map[string]any)
	if body0["wasteCode"] != "160214" {
		t.Fatalf("wasteCode = %#v, want 160214", body0["wasteCode"])
	}
	if body0["netWeight"] != 300.0 {
		t.Fatalf("netWeight = %#v, want 300", body0["netWeight"])
	}
	if body0["palletCount"] != 2.0 {
		t.Fatalf("palletCount = %#v, want 2", body0["palletCount"])
	}
	labels, _ := body0["labels"].([]any)
	if len(labels) != 2 {
		t.Fatalf("labels body = %#v", body0)
	}
	wantIDs := []string{loadID + ".1", loadID + ".2"}
	for i, raw := range labels {
		rec, _ := raw.(map[string]any)
		id, _ := rec["id"].(string)
		if id != wantIDs[i] {
			t.Fatalf("labels[%d].id = %q, want %q (all %#v)", i, id, wantIDs[i], labels)
		}
		qr, _ := rec["qr"].(string)
		want := "https://plant.example/app?entityId=" + id
		if qr != want {
			t.Fatalf("label %#v qr = %q, want %q", rec, qr, want)
		}
	}
	if len(result.Links) != 1 {
		t.Fatalf("links = %#v", result.Links)
	}
	if result.Links[0].Label != "Stampa etichette" || result.Links[0].Action != HttpActionLink || result.Links[0].URL != "blob:labels" {
		t.Fatalf("link0 = %#v", result.Links[0])
	}
	if len(result.HttpErrors) != 0 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}

	stored, err := domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("lot")[loadID]["client"] != "CO" {
		t.Fatalf("persisted domain = %#v", stored)
	}
}

func TestCommitRegisterLabelsOrderedByProgressive(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
		},
	}
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.SubmitBindings = ExprBindings{"qrEndpoint": "https://plant.example/app"}

	truckID := commitTruckArrive(t, c)
	value := registerLotValue()
	value["palletCount"] = 12.0
	result := commitLotRegister(t, c, truckID, value)
	loadID := firstLoadID(result.Domain)
	if len(port.calls) != 1 {
		t.Fatalf("http calls = %#v", port.calls)
	}
	body0, _ := port.calls[0].Body.(map[string]any)
	labels, ok := body0["labels"].([]any)
	if !ok || len(labels) != 12 {
		t.Fatalf("labels = %#v", body0["labels"])
	}
	for i, raw := range labels {
		rec, _ := raw.(map[string]any)
		id, _ := rec["id"].(string)
		want := fmt.Sprintf("%s.%d", loadID, i+1)
		if id != want {
			t.Fatalf("labels[%d].id = %q, want %q (lexical trap: .10 before .2)", i, id, want)
		}
	}
}

func TestCommitRegisterOnePalletLabelsBodyIsArray(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
		},
	}
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.SubmitBindings = ExprBindings{"qrEndpoint": "https://plant.example/app"}

	truckID := commitTruckArrive(t, c)
	value := registerLotValue()
	value["palletCount"] = 1.0
	result := commitLotRegister(t, c, truckID, value)
	if len(result.Domain.Kind("pallet")) != 1 {
		t.Fatalf("pallets = %#v", result.Domain.Kind("pallet"))
	}
	if len(port.calls) < 1 {
		t.Fatal("expected labels http call")
	}
	body0, _ := port.calls[0].Body.(map[string]any)
	if body0["wasteCode"] != "160214" {
		t.Fatalf("wasteCode = %#v, want 160214", body0["wasteCode"])
	}
	if body0["netWeight"] != 300.0 {
		t.Fatalf("netWeight = %#v, want 300", body0["netWeight"])
	}
	if body0["palletCount"] != 1.0 {
		t.Fatalf("palletCount = %#v, want 1", body0["palletCount"])
	}
	labels, ok := body0["labels"].([]any)
	if !ok {
		t.Fatalf("labels must be []any after Config wrap, got %T %#v", body0["labels"], body0["labels"])
	}
	if len(labels) != 1 {
		t.Fatalf("labels = %#v", body0)
	}
	rec, _ := labels[0].(map[string]any)
	id, _ := rec["id"].(string)
	qr, _ := rec["qr"].(string)
	if id == "" || qr != "https://plant.example/app?entityId="+id {
		t.Fatalf("label = %#v", rec)
	}
}

func TestCommitHTTPFailureDoesNotRollbackDomain(t *testing.T) {
	port := &recordingHttpPort{
		errs: []error{errors.New("printer down")},
	}
	c, domains, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port

	truckID := commitTruckArrive(t, c)
	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath:      "intake/register",
		JobInstanceID: truckID,
		Value:         registerLotValue(),
	})
	if err != nil {
		t.Fatalf("Commit must succeed when HttpPort fails: %v", err)
	}
	loadID := firstLoadID(result.Domain)
	if result.Domain.Kind("lot")[loadID]["client"] != "CO" {
		t.Fatalf("domain missing after http failure: %#v", result.Domain.Kind("lot"))
	}
	if len(result.Domain.Kind("pallet")) != 2 {
		t.Fatalf("pallets = %#v", result.Domain.Kind("pallet"))
	}
	if len(result.Links) != 0 {
		t.Fatalf("links = %#v", result.Links)
	}
	if len(result.HttpErrors) != 1 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}
	if result.HttpErrors[0] != "printer down" {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}

	stored, err := domains.Get(t.Context(), FeragameScope())
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("lot")[loadID]["client"] != "CO" {
		t.Fatalf("DomainStore rolled back: %#v", stored)
	}
}

func TestCommitWithoutHTTPSkipsPort(t *testing.T) {
	port := &recordingHttpPort{}
	c, _, _ := testTaskCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.Now = func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) }

	truckID := commitTruckArrive(t, c)
	_ = commitLotRegister(t, c, truckID, registerLotValue())
	if len(port.calls) != 1 {
		t.Fatalf("register should call http once — calls=%d", len(port.calls))
	}

	result, err := c.Commit(t.Context(), FeragameScope(), adminOp(), CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.calls) != 1 {
		t.Fatalf("weightCheck must not call http — calls=%d", len(port.calls))
	}
	if len(result.Links) != 0 || len(result.HttpErrors) != 0 {
		t.Fatalf("weightCheck http fields = links %#v errors %#v", result.Links, result.HttpErrors)
	}
}
