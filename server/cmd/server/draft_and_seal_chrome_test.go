package main

import (
	"bytes"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestApplyDraftAndSealChromeParallelDrafting(t *testing.T) {
	def := feragameDraftAndSealDef()
	process := &Process{
		ID:     primitive.NewObjectID(),
		Status: processStatusActive,
		Progress: map[string]ProcessStep{
			"1.1": {State: progressStateDraft, Data: map[string]interface{}{"batches": []interface{}{}}},
			"1.2": {State: "pending"},
		},
	}
	actor := Actor{Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"}

	actions := buildSubstepViews(def, process, "feragame", actor, false, map[roleMetaKey]RoleMeta{}, nil)
	_, actions = applyDraftAndSealChrome(nil, actions, def, process, actor, "feragame")
	if len(actions) != 2 {
		t.Fatalf("actions = %d, want 2", len(actions))
	}
	for _, action := range actions {
		if action.Status != "available" {
			t.Fatalf("%s status = %q, want available", action.SubstepID, action.Status)
		}
		if action.StatusLabel != draftAndSealDraftingLabel {
			t.Fatalf("%s StatusLabel = %q, want %q", action.SubstepID, action.StatusLabel, draftAndSealDraftingLabel)
		}
		if action.Reason == "Locked by sequence" {
			t.Fatalf("%s still uses classic sequence lock copy", action.SubstepID)
		}
		if !action.SaveDraft {
			t.Fatalf("%s SaveDraft = false, want true", action.SubstepID)
		}
		if action.Hint != draftAndSealSubstepHint {
			t.Fatalf("%s hint = %q", action.SubstepID, action.Hint)
		}
		if action.Sealed {
			t.Fatalf("%s should not be sealed while drafting", action.SubstepID)
		}
	}
	batches := findSubstepView(t, actions, "1.1")
	if !strings.Contains(batches.FormValue, `"batches"`) {
		t.Fatalf("expected draft FormValue to restore batches payload, got %q", batches.FormValue)
	}
}

func TestApplyDraftAndSealChromeSealedCopy(t *testing.T) {
	def := feragameDraftAndSealDef()
	now := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	process := &Process{
		ID:     primitive.NewObjectID(),
		Status: processStatusDone,
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"batches": []interface{}{}}, DoneAt: &now},
			"1.2": {State: "done", Data: map[string]interface{}{"udc": []interface{}{}}, DoneAt: &now},
		},
	}
	actor := Actor{Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"}

	actions := buildSubstepViews(def, process, "feragame", actor, false, map[roleMetaKey]RoleMeta{}, nil)
	_, actions = applyDraftAndSealChrome(nil, actions, def, process, actor, "feragame")
	for _, action := range actions {
		if action.Status != "done" {
			t.Fatalf("%s status = %q, want done", action.SubstepID, action.Status)
		}
		if action.StatusLabel != draftAndSealSealedLabel {
			t.Fatalf("%s StatusLabel = %q, want %q", action.SubstepID, action.StatusLabel, draftAndSealSealedLabel)
		}
		if !action.Sealed {
			t.Fatalf("%s Sealed = false", action.SubstepID)
		}
		if action.SaveDraft {
			t.Fatalf("%s SaveDraft must be false after seal", action.SubstepID)
		}
		if action.Reason != draftAndSealSealedReason {
			t.Fatalf("%s reason = %q", action.SubstepID, action.Reason)
		}
	}
}

func TestBuildSubstepViewsStrategyBlindForDraftAndSealChrome(t *testing.T) {
	def := feragameDraftAndSealDef()
	process := &Process{
		ID:     primitive.NewObjectID(),
		Status: processStatusActive,
		Progress: map[string]ProcessStep{
			"1.1": {State: progressStateDraft, Data: map[string]interface{}{"batches": []interface{}{}}},
			"1.2": {State: "pending"},
		},
	}
	actions := buildSubstepViews(def, process, "feragame", Actor{Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"}, false, map[roleMetaKey]RoleMeta{}, nil)
	for _, action := range actions {
		if action.SaveDraft || action.Sealed || action.Hint != "" || action.StatusLabel != "" || action.FormValue != "" {
			t.Fatalf("%s still carries draft-and-seal chrome before apply: %#v", action.SubstepID, action)
		}
	}
}

func TestBuildStreamInstanceDetailViewDraftAndSealChrome(t *testing.T) {
	def := feragameDraftAndSealDef()
	cfg := RuntimeConfig{Workflow: def}
	process := &Process{
		ID:     primitive.NewObjectID(),
		Status: processStatusActive,
		Progress: map[string]ProcessStep{
			"1.1": {State: "pending"},
			"1.2": {State: "pending"},
		},
	}
	admin := Actor{ID: "admin", Role: "org-admin", RoleSlugs: []string{"org-admin"}, OrgSlug: "feragame"}
	view := (&Server{}).buildStreamInstanceDetailView(t.Context(), cfg, "feragame", process, admin, "", "", false)

	if len(view.Timeline) != 1 {
		t.Fatalf("timeline steps = %d, want 1", len(view.Timeline))
	}
	step := view.Timeline[0]
	if !step.DraftAndSeal || !step.CanSeal || step.Sealed {
		t.Fatalf("step chrome = %#v", step)
	}
	wantAction := "/my/streams/feragame/instance/" + process.ID.Hex() + "/step/1/seal"
	if step.SealAction != wantAction {
		t.Fatalf("SealAction = %q, want %q", step.SealAction, wantAction)
	}
	if step.SealNotice != draftAndSealDraftingNotice {
		t.Fatalf("SealNotice = %q", step.SealNotice)
	}
	if len(step.Substeps) != 2 {
		t.Fatalf("substeps = %d, want 2", len(step.Substeps))
	}
	for _, sub := range step.Substeps {
		if sub.Status != "available" {
			t.Fatalf("%s timeline status = %q, want available", sub.SubstepID, sub.Status)
		}
		if sub.Body == nil || sub.Body.StatusLabel != draftAndSealDraftingLabel {
			t.Fatalf("%s body chrome = %#v", sub.SubstepID, sub.Body)
		}
	}

	registrar := Actor{Role: "registrar", RoleSlugs: []string{"registrar"}, OrgSlug: "feragame"}
	registrarView := (&Server{}).buildStreamInstanceDetailView(t.Context(), cfg, "feragame", process, registrar, "", "", false)
	if registrarView.Timeline[0].CanSeal {
		t.Fatal("registrar must not see Seal control")
	}
	if !registrarView.Timeline[0].DraftAndSeal {
		t.Fatal("registrar still needs drafting copy")
	}
}

func TestBuildStreamInstanceDetailViewClassicHasNoSealChrome(t *testing.T) {
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
	view := (&Server{}).buildStreamInstanceDetailView(t.Context(), testRuntimeConfig(), "workflow", process, Actor{Role: "dep1", RoleSlugs: []string{"dep1"}}, "", "", false)
	if view.Timeline[0].DraftAndSeal || view.Timeline[0].CanSeal {
		t.Fatalf("classic step must not expose seal chrome, got %#v", view.Timeline[0])
	}
	second := findTimelineSubstep(t, view.Timeline, "1.2")
	if second.Status != "locked" {
		t.Fatalf("classic 1.2 status = %q, want locked", second.Status)
	}
	if second.Body == nil || second.Body.Reason != "Locked by sequence" {
		t.Fatalf("classic 1.2 reason = %#v", second.Body)
	}
	if second.Body.SaveDraft {
		t.Fatal("classic 1.2 must not use SaveDraft")
	}
}

func TestStreamTimelineTemplateRendersDraftAndSealControl(t *testing.T) {
	tmpl := parseTestTemplates(t)
	view := testStreamTimelineView()
	view.Timeline[0].DraftAndSeal = true
	view.Timeline[0].CanSeal = true
	view.Timeline[0].SealAction = "/my/streams/feragame/instance/abc/step/1/seal"
	view.Timeline[0].SealNotice = draftAndSealDraftingNotice
	view.Timeline[0].Substeps[0].Body.SaveDraft = true
	view.Timeline[0].Substeps[0].Body.Hint = draftAndSealSubstepHint
	view.Timeline[0].Substeps[0].Body.StatusLabel = draftAndSealDraftingLabel
	view.Timeline[0].Substeps[0].Body.MatchingRoles = []SubstepRoleOption{
		{Slug: "registrar", Label: "Registrar"},
		{Slug: "lead", Label: "Lead"},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "stream_timeline", view); err != nil {
		t.Fatalf("render stream_timeline: %v", err)
	}
	body := out.String()
	for _, want := range []string{
		draftAndSealDraftingNotice,
		"Seal Step",
		`id="seal-step-dialog-1"`,
		`name="confirm"`,
		`value="true"`,
		`action="/my/streams/feragame/instance/abc/step/1/seal"`,
		"Confirm seal",
		draftAndSealSubstepHint,
		"Save draft as",
		`aria-label="Status: drafting"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in draft-and-seal timeline, got: %s", want, body)
		}
	}
	if strings.Contains(body, "Complete as") {
		t.Fatalf("kind Substep must not offer Complete as, got: %s", body)
	}
}

func TestSubstepBodyTemplateClassicStillCompletes(t *testing.T) {
	tmpl := parseTestTemplates(t)
	action := SubstepBodyView{
		WorkflowKey: "workflow",
		ProcessID:   "process-1",
		SubstepID:   "1.1",
		Status:      "available",
		Mode:        SubstepBodyModeActionable,
		FormSchema:  `{"type":"object"}`,
		MatchingRoles: []SubstepRoleOption{
			{Slug: "dep1", Label: "Department 1"},
			{Slug: "dep2", Label: "Department 2"},
		},
	}
	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "substep_body", action); err != nil {
		t.Fatalf("render substep_body: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, "Complete as") {
		t.Fatalf("classic body must keep Complete as, got: %s", body)
	}
	if strings.Contains(body, "Save draft as") {
		t.Fatalf("classic body must not say Save draft as, got: %s", body)
	}
	if !strings.Contains(body, "/substep/1.1/complete") {
		t.Fatalf("classic body must keep complete save path, got: %s", body)
	}
}

func TestHandleCompleteSubstepDraftAndSealSavesDraft(t *testing.T) {
	store := NewMemoryStore()
	def := feragameDraftAndSealDef()
	process := seedFeragameProcess(t, store, def)
	fixedNow := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	server := &Server{
		store:      store,
		process:    &ProcessService{store: store, now: func() time.Time { return fixedNow }},
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{Workflow: def}, nil
		},
		now: func() time.Time { return fixedNow },
	}

	req := httptest.NewRequest(http.MethodPost, "/process/"+process.ID.Hex()+"/substep/1.1/complete", strings.NewReader(`value=%7B%22batches%22%3A%5B%5D%7D`))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	server.handleCompleteSubstep(rec, req, process.ID.Hex(), "1.1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%q", rec.Code, rec.Body.String())
	}

	reloaded, err := store.LoadProcessByID(t.Context(), process.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	reloaded.Progress = normalizeProgressKeys(reloaded.Progress)
	step := reloaded.Progress["1.1"]
	if step.State != progressStateDraft {
		t.Fatalf("state = %q, want draft", step.State)
	}
	if len(store.Notarizations()) != 0 {
		t.Fatalf("draft save must not notarize, got %d", len(store.Notarizations()))
	}
	if reloaded.Status == processStatusDone {
		t.Fatal("draft save must not complete the instance")
	}
}

func TestHandleCompleteSubstepDraftAndSealKeepsSubmittedSubstepSelected(t *testing.T) {
	store := NewMemoryStore()
	def := feragameDraftAndSealDef()
	process := seedFeragameProcess(t, store, def)
	fixedNow := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	server := &Server{
		store:   store,
		process: &ProcessService{store: store, now: func() time.Time { return fixedNow }},
		tmpl: template.Must(template.New("test").Parse(
			`{{define "process_content.html"}}SEL {{.Detail.SelectedSubstepID}}{{if .Detail.Confirmation}} TOAST {{.Detail.Confirmation}}{{end}}{{end}}`,
		)),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{Workflow: def}, nil
		},
		now: func() time.Time { return fixedNow },
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/process/"+process.ID.Hex()+"/substep/1.2/complete?substep=1.2",
		strings.NewReader(`value=%7B%22udc%22%3A%5B%5D%7D`),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "process-page-content")
	rec := httptest.NewRecorder()
	server.handleCompleteSubstep(rec, req, process.ID.Hex(), "1.2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%q", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "SEL 1.2") {
		t.Fatalf("draft save must keep submitted substep selected, got %q", body)
	}
	if !strings.Contains(body, "TOAST Draft saved.") {
		t.Fatalf("draft save must emit confirmation toast, got %q", body)
	}
}

func TestHandleSealStepRequiresConfirmAndOrgAdmin(t *testing.T) {
	store := NewMemoryStore()
	def := feragameDraftAndSealDef()
	process := seedFeragameProcess(t, store, def)
	fixedNow := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	server := &Server{
		store:      store,
		process:    &ProcessService{store: store, now: func() time.Time { return fixedNow }},
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{Workflow: def}, nil
		},
		now:         func() time.Time { return fixedNow },
		enforceAuth: true,
		identity: testIdentityForSessions(fixedNow, map[string]AccountUser{
			"admin": {
				IdentityUserID: "admin-1",
				Email:          "admin@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"org-admin"},
				Status:         "active",
			},
			"registrar": {
				IdentityUserID: "registrar-1",
				Email:          "registrar@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"registrar"},
				Status:         "active",
			},
		}),
	}

	unconfirmed := httptest.NewRequest(http.MethodPost, "/instance/"+process.ID.Hex()+"/step/1/seal", strings.NewReader(""))
	unconfirmed.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	unconfirmed.AddCookie(&http.Cookie{Name: "attesta_session", Value: "admin"})
	unconfirmedRec := httptest.NewRecorder()
	server.handleSealStep(unconfirmedRec, unconfirmed, process.ID.Hex(), "1")
	if unconfirmedRec.Code != http.StatusBadRequest {
		t.Fatalf("unconfirmed status = %d, want 400; body=%q", unconfirmedRec.Code, unconfirmedRec.Body.String())
	}

	denied := httptest.NewRequest(http.MethodPost, "/instance/"+process.ID.Hex()+"/step/1/seal", strings.NewReader("confirm=true"))
	denied.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	denied.AddCookie(&http.Cookie{Name: "attesta_session", Value: "registrar"})
	deniedRec := httptest.NewRecorder()
	server.handleSealStep(deniedRec, denied, process.ID.Hex(), "1")
	if deniedRec.Code != http.StatusForbidden {
		t.Fatalf("registrar status = %d, want 403; body=%q", deniedRec.Code, deniedRec.Body.String())
	}

	okReq := httptest.NewRequest(http.MethodPost, "/instance/"+process.ID.Hex()+"/step/1/seal", strings.NewReader("confirm=true"))
	okReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	okReq.AddCookie(&http.Cookie{Name: "attesta_session", Value: "admin"})
	okRec := httptest.NewRecorder()
	server.handleSealStep(okRec, okReq, process.ID.Hex(), "1")
	if okRec.Code != http.StatusSeeOther {
		t.Fatalf("admin confirm status = %d, want %d; body=%q", okRec.Code, http.StatusSeeOther, okRec.Body.String())
	}

	reloaded, err := store.LoadProcessByID(t.Context(), process.ID)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	reloaded.Progress = normalizeProgressKeys(reloaded.Progress)
	if reloaded.Progress["1.1"].State != "done" || reloaded.Progress["1.2"].State != "done" {
		t.Fatalf("expected sealed leaves, got %#v", reloaded.Progress)
	}
	if len(store.Notarizations()) != 2 {
		t.Fatalf("notarizations = %d, want 2", len(store.Notarizations()))
	}
}

func TestHandleSealStepRedirectsFullPageSuccessToInstance(t *testing.T) {
	store := NewMemoryStore()
	def := feragameDraftAndSealDef()
	process := seedFeragameProcess(t, store, def)
	fixedNow := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	server := &Server{
		store:      store,
		process:    &ProcessService{store: store, now: func() time.Time { return fixedNow }},
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{Workflow: def}, nil
		},
		now:         func() time.Time { return fixedNow },
		enforceAuth: true,
		identity: testIdentityForSessions(fixedNow, map[string]AccountUser{
			"admin": {
				IdentityUserID: "admin-1",
				Email:          "admin@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"org-admin"},
				Status:         "active",
			},
		}),
	}

	sealPath := "/instance/" + process.ID.Hex() + "/step/1/seal"
	req := httptest.NewRequest(http.MethodPost, sealPath, strings.NewReader("confirm=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "admin"})
	rec := httptest.NewRecorder()
	server.handleSealStep(rec, req, process.ID.Hex(), "1")

	wantLocation := streamInstancePath("workflow", process.ID.Hex())
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	gotLocation := rec.Header().Get("Location")
	if gotLocation != wantLocation {
		t.Fatalf("Location = %q, want instance path %q", gotLocation, wantLocation)
	}
	if strings.Contains(gotLocation, "/seal") {
		t.Fatalf("Location must not be the seal action URL, got %q", gotLocation)
	}
}

func TestHandleSealStepHTMXSwapsProcessContent(t *testing.T) {
	store := NewMemoryStore()
	def := feragameDraftAndSealDef()
	process := seedFeragameProcess(t, store, def)
	fixedNow := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	server := &Server{
		store:   store,
		process: &ProcessService{store: store, now: func() time.Time { return fixedNow }},
		tmpl: template.Must(template.New("test").Parse(
			`{{define "process_content.html"}}PROCESS_CONTENT {{.Detail.ProcessID}}{{end}}`,
		)),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{Workflow: def}, nil
		},
		now:         func() time.Time { return fixedNow },
		enforceAuth: true,
		identity: testIdentityForSessions(fixedNow, map[string]AccountUser{
			"admin": {
				IdentityUserID: "admin-1",
				Email:          "admin@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"org-admin"},
				Status:         "active",
			},
		}),
	}

	req := httptest.NewRequest(http.MethodPost, "/instance/"+process.ID.Hex()+"/step/1/seal", strings.NewReader("confirm=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.Header.Set("HX-Target", "process-page-content")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "admin"})
	rec := httptest.NewRecorder()
	server.handleSealStep(rec, req, process.ID.Hex(), "1")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%q", rec.Code, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Fatalf("HTMX seal must not set Location, got %q", loc)
	}
	if !strings.Contains(rec.Body.String(), "PROCESS_CONTENT "+process.ID.Hex()) {
		t.Fatalf("HTMX seal must swap process content, got %q", rec.Body.String())
	}
}


func TestHandleProcessRoutesDispatchesSeal(t *testing.T) {
	store := NewMemoryStore()
	def := feragameDraftAndSealDef()
	process := seedFeragameProcess(t, store, def)
	fixedNow := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	server := &Server{
		store:      store,
		process:    &ProcessService{store: store, now: func() time.Time { return fixedNow }},
		tmpl:       testTemplates(),
		authorizer: fakeAuthorizer{},
		sse:        newSSEHub(),
		configProvider: func() (RuntimeConfig, error) {
			return RuntimeConfig{Workflow: def}, nil
		},
		now:         func() time.Time { return fixedNow },
		enforceAuth: true,
		identity: testIdentityForSessions(fixedNow, map[string]AccountUser{
			"admin": {
				IdentityUserID: "admin-1",
				Email:          "admin@example.com",
				OrgSlug:        "feragame",
				RoleSlugs:      []string{"org-admin"},
				Status:         "active",
			},
		}),
	}

	req := httptest.NewRequest(http.MethodPost, "/instance/"+process.ID.Hex()+"/step/1/seal", strings.NewReader("confirm=true"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: "attesta_session", Value: "admin"})
	rec := httptest.NewRecorder()
	server.handleProcessRoutes(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusSeeOther, rec.Body.String())
	}
	if loc := rec.Header().Get("Location"); loc != streamInstancePath("workflow", process.ID.Hex()) {
		t.Fatalf("Location = %q, want instance path", loc)
	}
}

func findTimelineSubstep(t *testing.T, timeline []TimelineStep, substepID string) TimelineSubstep {
	t.Helper()
	for _, step := range timeline {
		for _, sub := range step.Substeps {
			if sub.SubstepID == substepID {
				return sub
			}
		}
	}
	t.Fatalf("substep %s not found", substepID)
	return TimelineSubstep{}
}
