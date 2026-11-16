package living

import (
	"context"
	"errors"
	"testing"
)

type recordingHttpPort struct {
	responses []HttpPortResponse
	errs      []error
	calls     []HttpRequest
	i         int
}

func (r *recordingHttpPort) Request(_ context.Context, req HttpRequest) (HttpPortResponse, error) {
	r.calls = append(r.calls, req)
	idx := r.i
	r.i++
	if idx < len(r.errs) && r.errs[idx] != nil {
		return HttpPortResponse{}, r.errs[idx]
	}
	if idx < len(r.responses) {
		return r.responses[idx], nil
	}
	return HttpPortResponse{URL: "recorded://" + req.Method + ":" + req.URL}, nil
}

func TestRunSubmitHTTPEffectsSuccess(t *testing.T) {
	port := &recordingHttpPort{
		responses: []HttpPortResponse{
			{URL: "blob:labels"},
			{URL: "blob:csv", Headers: map[string]string{"Content-Disposition": `attachment; filename="register.csv"`}},
		},
	}
	domain := EmptyDomain()
	putKind(&domain, "lot", EntityMap{"acme-1": {"client": "acme"}})
	putKind(&domain, "pallet", EntityMap{"acme-1.1": {"lotId": "acme-1"}})

	result := runSubmitHTTPEffects(t.Context(), port, []SubmitHttpEffect{
		{
			Method: `"POST"`, URL: `"/fake/labels"`, Label: `"Stampa etichette"`, Action: "link",
			Body:    `{ "lotId": $lot.id, "labels": [$keys($sift($domain.pallet, function($v){$v.lotId = $lot.id}))] }`,
			HasBody: true,
		},
		{
			Method: `"POST"`, URL: `"/fake/register-csv"`, Label: `"Scarica CSV registrazione"`, Action: "download",
			Body:    `{ "client": $.client, "lotId": $lot.id }`,
			HasBody: true,
		},
	}, map[string]any{"client": "acme", "palletCount": 1.0}, ExprBindings{"lot": map[string]any{"id": "acme-1"}}, domain, "acme-1", nil, "feragame")

	if len(result.Errors) != 0 {
		t.Fatalf("errors = %#v", result.Errors)
	}
	if len(port.calls) != 2 {
		t.Fatalf("calls = %#v", port.calls)
	}
	if port.calls[0].Method != "POST" || port.calls[0].URL != "/fake/labels" || port.calls[0].Org != "feragame" {
		t.Fatalf("call0 = %#v", port.calls[0])
	}
	body0, _ := port.calls[0].Body.(map[string]any)
	if body0["lotId"] != "acme-1" {
		t.Fatalf("body0 = %#v", port.calls[0].Body)
	}
	if len(result.Links) != 2 {
		t.Fatalf("links = %#v", result.Links)
	}
	if result.Links[0] != (HttpLinkItem{Label: "Stampa etichette", URL: "blob:labels", Action: HttpActionLink}) {
		t.Fatalf("link0 = %#v", result.Links[0])
	}
	if result.Links[1] != (HttpLinkItem{Label: "Scarica CSV registrazione", URL: "blob:csv", Action: HttpActionDownload, Filename: "register.csv"}) {
		t.Fatalf("link1 = %#v", result.Links[1])
	}
}

func TestRunSubmitHTTPEffectsContinuesAfterPortFailure(t *testing.T) {
	port := &recordingHttpPort{
		// Index-aligned with calls: first errs, second returns blob:ok.
		responses: []HttpPortResponse{{}, {URL: "blob:ok"}},
		errs:      []error{errors.New("printer down"), nil},
	}
	result := runSubmitHTTPEffects(t.Context(), port, []SubmitHttpEffect{
		{Method: `"POST"`, URL: `"/fake/labels"`, Label: `"Labels"`, Action: "link"},
		{Method: `"POST"`, URL: `"/fake/register-csv"`, Label: `"CSV"`, Action: "download"},
	}, map[string]any{}, nil, EmptyDomain(), "", nil, "feragame")

	if len(result.Errors) != 1 || result.Errors[0] != "printer down" {
		t.Fatalf("errors = %#v", result.Errors)
	}
	if len(result.Links) != 1 || result.Links[0].URL != "blob:ok" {
		t.Fatalf("links = %#v", result.Links)
	}
}

func TestFilenameFromContentDisposition(t *testing.T) {
	if got := filenameFromContentDisposition(`attachment; filename="reg.csv"`); got != "reg.csv" {
		t.Fatalf("plain = %q", got)
	}
	if got := filenameFromContentDisposition(`attachment; filename*=UTF-8''nord%201.csv`); got != "nord 1.csv" {
		t.Fatalf("star = %q", got)
	}
}

func TestRunSubmitHTTPEffectsUsesQrEndpointBinding(t *testing.T) {
	port := &recordingHttpPort{responses: []HttpPortResponse{{URL: "blob:labels"}}}
	domain := EmptyDomain()
	putKind(&domain, "pallet", EntityMap{
		"acme-1.1": {"lotId": "acme-1"},
		"acme-1.2": {"lotId": "acme-1"},
	})

	result := runSubmitHTTPEffects(t.Context(), port, []SubmitHttpEffect{
		{
			Method: `"POST"`, URL: `"/fake/labels"`, Label: `"Stampa etichette"`, Action: "link",
			Body: `{"labels": [$each(
				$sift($domain.pallet, function($v){$v.lotId = "acme-1"}),
				function($v, $id){{ "id": $id, "qr": $qrEndpoint & "?entityId=" & $id }}
			)]}`,
			HasBody: true,
		},
	}, map[string]any{}, ExprBindings{"qrEndpoint": "https://plant.example/app"}, domain, "truck-1", nil, "feragame")

	if len(result.Errors) != 0 {
		t.Fatalf("errors = %#v", result.Errors)
	}
	body, _ := port.calls[0].Body.(map[string]any)
	labels, _ := body["labels"].([]any)
	if len(labels) != 2 {
		t.Fatalf("labels = %#v", body)
	}
	got := map[string]string{}
	for _, raw := range labels {
		rec, _ := raw.(map[string]any)
		id, _ := rec["id"].(string)
		qr, _ := rec["qr"].(string)
		got[id] = qr
	}
	want := map[string]string{
		"acme-1.1": "https://plant.example/app?entityId=acme-1.1",
		"acme-1.2": "https://plant.example/app?entityId=acme-1.2",
	}
	if len(got) != len(want) {
		t.Fatalf("got = %#v", got)
	}
	for id, qr := range want {
		if got[id] != qr {
			t.Fatalf("label %s qr = %q, want %q (all %#v)", id, got[id], qr, got)
		}
	}
}
