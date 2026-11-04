package living

import (
	"errors"
	"testing"
	"time"
)

func TestCommitRegisterRunsOnSubmitHTTP(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg.csv"`}},
		},
	}
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.SubmitBindings = ExprBindings{"qrEndpoint": "https://plant.example/app"}

	truckID := commitTruckArrive(t, c)
	result := commitLotRegister(t, c, truckID, registerLotValue())
	loadID := firstLoadID(result.Domain)
	if result.Domain.Kind("load")[loadID]["client"] != "NO" {
		t.Fatalf("domain rolled back? %#v", result.Domain.Kind("load"))
	}
	if len(port.calls) != 2 {
		t.Fatalf("http calls = %#v", port.calls)
	}
	if port.calls[0].Method != "POST" || port.calls[0].URL != "/fake/labels" {
		t.Fatalf("call0 = %#v", port.calls[0])
	}
	if port.calls[1].URL != "/fake/register-csv" {
		t.Fatalf("call1 = %#v", port.calls[1])
	}
	body0, _ := port.calls[0].Body.(map[string]any)
	if body0["wasteCode"] != "191212" {
		t.Fatalf("wasteCode = %#v, want 191212", body0["wasteCode"])
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
	for _, raw := range labels {
		rec, _ := raw.(map[string]any)
		id, _ := rec["id"].(string)
		qr, _ := rec["qr"].(string)
		want := "https://plant.example/app?entityId=" + id
		if qr != want {
			t.Fatalf("label %#v qr = %q, want %q", rec, qr, want)
		}
	}
	if len(result.Links) != 2 {
		t.Fatalf("links = %#v", result.Links)
	}
	if result.Links[0].Label != "Stampa etichette" || result.Links[0].Action != HttpActionLink || result.Links[0].URL != "blob:labels" {
		t.Fatalf("link0 = %#v", result.Links[0])
	}
	if result.Links[1].Action != HttpActionDownload || result.Links[1].Filename != "reg.csv" {
		t.Fatalf("link1 = %#v", result.Links[1])
	}
	if len(result.HttpErrors) != 0 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}

	stored, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("load")[loadID]["client"] != "NO" {
		t.Fatalf("persisted domain = %#v", stored)
	}
}

func TestCommitRegisterOnePalletLabelsBodyIsArray(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="reg.csv"`}},
		},
	}
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
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
	if body0["wasteCode"] != "191212" {
		t.Fatalf("wasteCode = %#v, want 191212", body0["wasteCode"])
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
		errs: []error{errors.New("printer down"), errors.New("csv down")},
	}
	c, domains, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port

	truckID := commitTruckArrive(t, c)
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath:      GroupTaskPath("intake", "lots", ProvisionalGroupInstanceID, "register"),
		JobInstanceID: truckID,
		Value:         registerLotValue(),
		Mint:          spawningNewMint(),
	})
	if err != nil {
		t.Fatalf("Commit must succeed when HttpPort fails: %v", err)
	}
	loadID := firstLoadID(result.Domain)
	if result.Domain.Kind("load")[loadID]["client"] != "NO" {
		t.Fatalf("domain missing after http failure: %#v", result.Domain.Kind("load"))
	}
	if len(result.Domain.Kind("pallet")) != 2 {
		t.Fatalf("pallets = %#v", result.Domain.Kind("pallet"))
	}
	if len(result.Links) != 0 {
		t.Fatalf("links = %#v", result.Links)
	}
	if len(result.HttpErrors) != 2 {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}
	if result.HttpErrors[0] != "printer down" || result.HttpErrors[1] != "csv down" {
		t.Fatalf("httpErrors = %#v", result.HttpErrors)
	}

	stored, err := domains.Get(t.Context(), "feragame")
	if err != nil {
		t.Fatal(err)
	}
	if stored.Kind("load")[loadID]["client"] != "NO" {
		t.Fatalf("DomainStore rolled back: %#v", stored)
	}
}

func TestCommitWithoutHTTPSkipsPort(t *testing.T) {
	port := &recordingHttpPort{}
	c, _, _ := testCommitter(t, AllowAllAuthorizer{})
	c.HttpPort = port
	c.Now = func() time.Time { return time.Date(2026, 10, 29, 15, 0, 0, 0, time.UTC) }

	truckID := commitTruckArrive(t, c)
	_ = commitLotRegister(t, c, truckID, registerLotValue())
	if len(port.calls) != 2 {
		t.Fatalf("register should call http twice — calls=%d", len(port.calls))
	}

	closeLots(t, c, truckID)
	result, err := c.Commit(t.Context(), "feragame", adminOp(), CommitRequest{
		TaskPath: "intake/weightCheck", JobInstanceID: truckID,
		Value: map[string]any{"checkedTruckWeight": 1000.0, "checkedLotSum": 300.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(port.calls) != 2 {
		t.Fatalf("weightCheck must not call http — calls=%d", len(port.calls))
	}
	if len(result.Links) != 0 || len(result.HttpErrors) != 0 {
		t.Fatalf("weightCheck http fields = links %#v errors %#v", result.Links, result.HttpErrors)
	}
}
