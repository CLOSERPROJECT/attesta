package main

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestParseDigitalLinkPathValidAndInvalid(t *testing.T) {
	gtin, lot, serial, err := parseDigitalLinkPath("/01/09506000134352/10/LOT-001/21/SERIAL-001")
	if err != nil {
		t.Fatalf("parseDigitalLinkPath(valid): %v", err)
	}
	if gtin != "09506000134352" || lot != "LOT-001" || serial != "SERIAL-001" {
		t.Fatalf("unexpected parsed values: gtin=%q lot=%q serial=%q", gtin, lot, serial)
	}

	_, _, _, err = parseDigitalLinkPath("/01/09506000134352/10/LOT-001")
	if err == nil {
		t.Fatal("expected invalid path shape error")
	}

	_, _, _, err = parseDigitalLinkPath("/01/not-digits/10/LOT-001/21/SERIAL-001")
	if err == nil {
		t.Fatal("expected invalid gtin error")
	}
}

func TestParseDigitalLinkAttachmentPath(t *testing.T) {
	gtin, lot, serial, attachmentID, ok, err := parseDigitalLinkAttachmentPath("/01/09506000134352/10/LOT-001/21/SERIAL-001/attachment/file%201/file")
	if err != nil {
		t.Fatalf("parseDigitalLinkAttachmentPath(valid): %v", err)
	}
	if !ok {
		t.Fatal("expected attachment path")
	}
	if gtin != "09506000134352" || lot != "LOT-001" || serial != "SERIAL-001" || attachmentID != "file 1" {
		t.Fatalf("unexpected parsed values: gtin=%q lot=%q serial=%q attachmentID=%q", gtin, lot, serial, attachmentID)
	}

	_, _, _, _, ok, err = parseDigitalLinkAttachmentPath("/01/09506000134352/10/LOT-001/21/SERIAL-001")
	if ok || err != nil {
		t.Fatalf("plain DPP path returned ok=%v err=%v, want ok=false err=nil", ok, err)
	}

	_, _, _, _, ok, err = parseDigitalLinkAttachmentPath("/01/not-digits/10/LOT-001/21/SERIAL-001/attachment/file/file")
	if !ok || err == nil {
		t.Fatalf("invalid attachment path returned ok=%v err=%v, want ok=true with error", ok, err)
	}

	_, _, _, _, ok, err = parseDigitalLinkAttachmentPath("/01/09506000134352/10/LOT-001/21/SERIAL-001/attachment/%20/file")
	if !ok || err == nil {
		t.Fatalf("blank attachment id returned ok=%v err=%v, want ok=true with error", ok, err)
	}
}

func TestDigitalLinkURLPathEscapesValues(t *testing.T) {
	url := digitalLinkURL("09506000134352", "LOT 001", "SERIAL/001")
	if url != "/01/09506000134352/10/LOT%20001/21/SERIAL%2F001" {
		t.Fatalf("digitalLinkURL() = %q", url)
	}
}

func TestDPPFirstStringValueAndBuildProcessDPP(t *testing.T) {
	def := testRuntimeConfig().Workflow
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": float64(10)}},
			"1.2": {State: "done", Data: map[string]interface{}{"note": "LOT-2026"}},
			"3.2": {State: "done", Data: map[string]interface{}{"serialCode": "SER-ABC"}},
		},
	}

	if got := dppFirstStringValue(def, process, "note"); got != "LOT-2026" {
		t.Fatalf("dppFirstStringValue(note) = %q, want LOT-2026", got)
	}
	if got := dppFirstStringValue(def, process, "missing"); got != "" {
		t.Fatalf("dppFirstStringValue(missing) = %q, want empty", got)
	}

	nestedProcess := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.2": {State: "done", Data: map[string]interface{}{"b": map[string]interface{}{"note": "LOT-NESTED"}}},
		},
	}
	if got := dppFirstStringValue(def, nestedProcess, "note"); got != "LOT-NESTED" {
		t.Fatalf("dppFirstStringValue(note nested under title slug) = %q, want LOT-NESTED", got)
	}

	cfg := DPPConfig{
		Enabled:        true,
		GTIN:           "09506000134352",
		LotInputKey:    "note",
		SerialInputKey: "serialCode",
		SerialStrategy: "process_id_hex",
		LotDefault:     "defaultProduct",
	}
	now := time.Date(2026, 2, 13, 11, 0, 0, 0, time.UTC)
	dpp, err := buildProcessDPP(def, cfg, process, now)
	if err != nil {
		t.Fatalf("buildProcessDPP: %v", err)
	}
	if dpp.GTIN != cfg.GTIN || dpp.Lot != "LOT-2026" || dpp.Serial != "SER-ABC" {
		t.Fatalf("unexpected dpp: %#v", dpp)
	}

	cfg.SerialInputKey = "missing"
	dpp, err = buildProcessDPP(def, cfg, process, now)
	if err != nil {
		t.Fatalf("buildProcessDPP fallback serial: %v", err)
	}
	if dpp.Serial != process.ID.Hex() {
		t.Fatalf("serial fallback = %q, want %q", dpp.Serial, process.ID.Hex())
	}
}

func TestDPPStringValueHandlesNestedPrimitiveMap(t *testing.T) {
	value := dppStringValue(primitive.M{"lot": " LOT-9 "}, "lot")
	if value != "LOT-9" {
		t.Fatalf("dppStringValue = %q, want LOT-9", value)
	}
	if value := dppStringValue(map[string]interface{}{"lot": 42}, "lot"); value != "" {
		t.Fatalf("dppStringValue non-string = %q, want empty", value)
	}
}

func TestBuildProcessDPPErrorsAndStrategyValidation(t *testing.T) {
	def := testRuntimeConfig().Workflow
	now := time.Date(2026, 2, 13, 11, 0, 0, 0, time.UTC)
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {State: "done", Data: map[string]interface{}{"value": float64(10)}},
		},
	}

	cfg := DPPConfig{
		Enabled:        true,
		GTIN:           "09506000134352",
		LotInputKey:    "note",
		SerialInputKey: "serialCode",
		SerialStrategy: "process_id_hex",
	}

	if _, err := buildProcessDPP(def, cfg, nil, now); err == nil {
		t.Fatal("expected error for nil process")
	}

	cfg.Enabled = false
	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected error when dpp is disabled")
	}
	cfg.Enabled = true

	cfg.GTIN = ""
	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected missing gtin error")
	}
	cfg.GTIN = "09506000134352"

	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected missing lot error")
	}

	cfg.LotDefault = "LOT-DEFAULT"
	cfg.SerialStrategy = "unsupported"
	if _, err := buildProcessDPP(def, cfg, process, now); err == nil {
		t.Fatal("expected unsupported serial strategy error")
	}
}

func TestParseDigitalLinkPathUnescapeErrors(t *testing.T) {
	_, _, _, err := parseDigitalLinkPath("/01/09506000134352/10/%ZZ/21/SERIAL-001")
	if err == nil {
		t.Fatal("expected lot unescape error")
	}

	_, _, _, err = parseDigitalLinkPath("/01/09506000134352/10/LOT-001/21/%ZZ")
	if err == nil {
		t.Fatal("expected serial unescape error")
	}

	_, _, _, err = parseDigitalLinkPath("/01/09506000134352/10/ /21/SERIAL-001")
	if err == nil {
		t.Fatal("expected missing lot or serial error")
	}
}

func TestBuildDPPTraceabilityViewIncludesValuesAndFiles(t *testing.T) {
	def := testRuntimeConfig().Workflow
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State: "done",
				Data:  map[string]interface{}{"value": float64(10)},
			},
			"1.2": {
				State: "done",
				Data:  map[string]interface{}{"note": "LOT-2026"},
			},
			"1.3": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": "65f2a79b8e7f7d8f3c7c99aa",
						"filename":     "cert.pdf",
						"sha256":       "abc123",
					},
				},
			},
		},
	}

	view := buildDPPTraceabilityView(def, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, organizationNameMap(testRuntimeConfig()))
	if len(view) == 0 {
		t.Fatal("expected non-empty traceability view")
	}

	var foundValue bool
	var foundFile bool
	for _, step := range view {
		for _, sub := range step.Substeps {
			if sub.SubstepID == "1.2" {
				for _, value := range sub.Values {
					if value.Key == "note" && value.Value == "LOT-2026" {
						foundValue = true
					}
				}
			}
			if sub.SubstepID == "1.3" {
				if len(sub.Attachments) == 1 &&
					sub.Attachments[0].Filename == "cert.pdf" &&
					sub.Attachments[0].URL != "" &&
					sub.Attachments[0].PreviewKind == "document" {
					foundFile = true
				}
			}
		}
	}
	if !foundValue {
		t.Fatal("expected traceability value entry for substep 1.2")
	}
	if !foundFile {
		t.Fatal("expected inline file metadata for substep 1.3")
	}
}

func TestBuildDPPTraceabilityViewIncludesStepSummaryMetadata(t *testing.T) {
	doneAt := time.Date(2026, 3, 5, 14, 30, 0, 0, time.UTC)
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID:           "1",
				Title:            "Review materials",
				Order:            1,
				OrganizationSlug: "org-a",
				Substep: []WorkflowSub{
					{
						SubstepID: "1.1",
						Title:     "Check batch",
						Order:     1,
						Role:      "qa",
						InputKey:  "value",
						InputType: "formata",
					},
				},
			},
		},
	}
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State:  "done",
				DoneAt: &doneAt,
				DoneBy: &Actor{ID: "qa@example.com"},
				Data:   map[string]interface{}{"value": "ok"},
			},
		},
	}

	view := buildDPPTraceabilityView(def, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, map[string]string{"org-a": "Acme Org"})
	if len(view) != 1 {
		t.Fatalf("expected one traceability step, got %#v", view)
	}
	step := view[0]
	if step.OrganizationName != "Acme Org" {
		t.Fatalf("organization name = %q, want Acme Org", step.OrganizationName)
	}
	if step.CompletedAt != "2026-03-05T14:30:00Z" {
		t.Fatalf("completedAt = %q, want RFC3339", step.CompletedAt)
	}
	if step.CompletedAtHuman != "5 Mar 2026 at 14:30 UTC" {
		t.Fatalf("completedAtHuman = %q, want human-readable time", step.CompletedAtHuman)
	}
	if step.DetailsDialogID != "dpp-step-dialog-1" {
		t.Fatalf("detailsDialogID = %q, want dpp-step-dialog-1", step.DetailsDialogID)
	}
	if step.Substeps[0].DoneAtHuman != "5 Mar 2026 at 14:30 UTC" {
		t.Fatalf("substep DoneAtHuman = %q, want human-readable time", step.Substeps[0].DoneAtHuman)
	}
}

func TestBuildDPPTraceabilityViewFlattensFormataPayload(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Title:  "Step 1",
				Order:  1,
				Substep: []WorkflowSub{
					{
						SubstepID: "1.1",
						Title:     "Formata",
						Order:     1,
						Role:      "qa",
						InputKey:  "payload",
						InputType: "formata",
					},
				},
			},
		},
	}
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State: "done",
				Data: map[string]interface{}{
					"payload": map[string]interface{}{
						"details": map[string]interface{}{
							"status": "ok",
							"weight": 42.0,
						},
					},
				},
			},
		},
	}

	view := buildDPPTraceabilityView(def, process, "workflow", testRoleIndexForOrg("", map[string]RoleMeta{
		"qa": {ID: "qa", Label: "Quality", Palette: "blue"},
	}), nil, nil)
	if len(view) != 1 || len(view[0].Substeps) != 1 {
		t.Fatalf("unexpected traceability shape: %#v", view)
	}
	substep := view[0].Substeps[0]
	if substep.Role != "Quality" {
		t.Fatalf("role label = %q, want Quality", substep.Role)
	}
	if substep.Palette != "blue" {
		t.Fatalf("expected role palette blue, got %q", substep.Palette)
	}
	if len(substep.Values) != 2 {
		t.Fatalf("expected flattened formata values, got %#v", substep.Values)
	}
	if substep.Values[0].Key != "payload.details.status" || substep.Values[0].Value != "ok" {
		t.Fatalf("unexpected first flattened value: %#v", substep.Values[0])
	}
	if substep.Values[1].Key != "payload.details.weight" || substep.Values[1].Value != "42" {
		t.Fatalf("unexpected second flattened value: %#v", substep.Values[1])
	}
}

func TestBuildDPPTraceabilityViewFindsNestedAttachments(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Title:  "Step 1",
				Order:  1,
				Substep: []WorkflowSub{
					{
						SubstepID: "1.1",
						Title:     "Formata",
						Order:     1,
						Role:      "qa",
						InputKey:  "payload",
						InputType: "formata",
					},
				},
			},
		},
	}
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State: "done",
				Data: map[string]interface{}{
					"payload": map[string]interface{}{
						"docs": []interface{}{
							map[string]interface{}{
								"attachmentId": primitive.NewObjectID().Hex(),
								"filename":     "proof.pdf",
								"sha256":       "hash-proof",
							},
						},
					},
				},
			},
		},
	}

	view := buildDPPTraceabilityView(def, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, nil)
	if len(view) != 1 || len(view[0].Substeps) != 1 {
		t.Fatalf("unexpected traceability shape: %#v", view)
	}
	substep := view[0].Substeps[0]
	if len(substep.Attachments) != 1 {
		t.Fatalf("expected one nested attachment, got %#v", substep.Attachments)
	}
	if substep.Attachments[0].Filename != "proof.pdf" {
		t.Fatalf("attachment filename = %q, want proof.pdf", substep.Attachments[0].Filename)
	}
	if substep.Attachments[0].URL == "" {
		t.Fatalf("expected attachment URL, got %#v", substep.Attachments[0])
	}
}

func TestPublicDPPTraceabilityAttachmentURLs(t *testing.T) {
	traceability := []DPPTraceabilityStep{
		{
			Substeps: []DPPTraceabilitySubstep{
				{
					Attachments: []ActionAttachmentView{
						{AttachmentID: "file 1", URL: "/w/workflow/process/p1/attachment/file%201/file", PreviewKind: "document"},
						{Filename: "legacy.pdf", URL: "/w/workflow/process/p1/attachment/legacy/file"},
					},
				},
			},
		},
	}

	mapped := publicDPPTraceabilityAttachmentURLs(traceability, "/01/09506000134352/10/LOT-001/21/SERIAL-001")
	attachment := mapped[0].Substeps[0].Attachments[0]
	if attachment.URL != "/01/09506000134352/10/LOT-001/21/SERIAL-001/attachment/file%201/file" {
		t.Fatalf("public URL = %q", attachment.URL)
	}
	if attachment.PreviewURL != attachment.URL+"?inline=1#page=1&toolbar=0&navpanes=0&view=FitH" {
		t.Fatalf("preview URL = %q", attachment.PreviewURL)
	}
	if got := mapped[0].Substeps[0].Attachments[1].URL; got != "/w/workflow/process/p1/attachment/legacy/file" {
		t.Fatalf("attachment without ID URL changed to %q", got)
	}

	unchanged := publicDPPTraceabilityAttachmentURLs([]DPPTraceabilityStep{
		{
			Substeps: []DPPTraceabilitySubstep{
				{
					Attachments: []ActionAttachmentView{
						{AttachmentID: "file 1", URL: "/w/workflow/process/p1/attachment/file%201/file", PreviewKind: "document"},
					},
				},
			},
		},
	}, " ")
	if unchanged[0].Substeps[0].Attachments[0].URL != "/w/workflow/process/p1/attachment/file%201/file" {
		t.Fatalf("blank digital link changed URL to %q", unchanged[0].Substeps[0].Attachments[0].URL)
	}
}

func TestDPPProcessHasAttachment(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Substep: []WorkflowSub{
					{SubstepID: "1.1", InputType: "formata"},
				},
			},
		},
	}
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State: "done",
				Data: map[string]interface{}{
					"attachment": map[string]interface{}{
						"attachmentId": "65f2a79b8e7f7d8f3c7c99aa",
						"filename":     "cert.pdf",
					},
				},
			},
		},
	}

	if !dppProcessHasAttachment(def, process, "65f2a79b8e7f7d8f3c7c99aa") {
		t.Fatal("expected process to expose its own attachment")
	}
	if dppProcessHasAttachment(def, process, primitive.NewObjectID().Hex()) {
		t.Fatal("expected unrelated attachment to be rejected")
	}
	if dppProcessHasAttachment(def, nil, "65f2a79b8e7f7d8f3c7c99aa") {
		t.Fatal("expected nil process to be rejected")
	}
}

func TestBuildDPPTraceabilityViewRoleBadgesAndDoneRoleSelection(t *testing.T) {
	def := WorkflowDef{
		Steps: []WorkflowStep{
			{
				StepID: "1",
				Title:  "Step 1",
				Order:  1,
				Substep: []WorkflowSub{
					{
						SubstepID: "1.1",
						Title:     "Approve",
						Order:     1,
						Roles:     []string{"qa", "manager"},
						InputKey:  "value",
						InputType: "formata",
					},
					{
						SubstepID: "1.2",
						Title:     "Release",
						Order:     2,
						Role:      "qa",
						InputKey:  "value",
						InputType: "formata",
					},
					{
						SubstepID: "1.3",
						Title:     "Archive",
						Order:     3,
						Role:      "qa",
						InputKey:  "value",
						InputType: "formata",
					},
				},
			},
		},
	}
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State:  "done",
				DoneBy: &Actor{ID: primitive.NewObjectID().Hex(), Role: "manager"},
				Data:   map[string]interface{}{"value": "ok"},
			},
		},
	}
	roleMeta := testRoleIndexForOrg("", map[string]RoleMeta{
		"qa":      {ID: "qa", Label: "", Palette: "cyan"},
		"manager": {ID: "manager", Label: "Manager", Palette: "purple"},
	})

	view := buildDPPTraceabilityView(def, process, "workflow", roleMeta, nil, nil)
	if len(view) != 1 || len(view[0].Substeps) != 3 {
		t.Fatalf("unexpected traceability shape: %#v", view)
	}
	doneSub := view[0].Substeps[0]
	if doneSub.Role != "Manager" {
		t.Fatalf("done role label = %q, want Manager", doneSub.Role)
	}
	if len(doneSub.RoleBadges) != 1 || doneSub.RoleBadges[0].Label != "Manager" {
		t.Fatalf("expected selected done role badge, got %#v", doneSub.RoleBadges)
	}
	if doneSub.Palette != "purple" {
		t.Fatalf("expected selected role palette purple, got %q", doneSub.Palette)
	}
	if doneSub.Status != "done" {
		t.Fatalf("done substep status = %q, want done", doneSub.Status)
	}

	availableSub := view[0].Substeps[1]
	if availableSub.Status != "available" {
		t.Fatalf("available substep status = %q, want available", availableSub.Status)
	}
	if availableSub.Role != "qa" {
		t.Fatalf("available role label fallback = %q, want qa", availableSub.Role)
	}

	lockedSub := view[0].Substeps[2]
	if lockedSub.Status != "locked" {
		t.Fatalf("locked substep status = %q, want locked", lockedSub.Status)
	}
}

func TestBuildDPPTraceabilityViewTerminatedStreamMessages(t *testing.T) {
	def := testRuntimeConfig().Workflow
	process := &Process{
		ID: primitive.NewObjectID(),
		Progress: map[string]ProcessStep{
			"1.1": {
				State: "done",
				Data:  map[string]interface{}{"value": float64(10)},
			},
		},
		Termination: &ProcessTermination{
			Reason:    "supplier cancelled",
			SubstepID: "1.2",
		},
	}

	view := buildDPPTraceabilityView(def, process, "workflow", map[roleMetaKey]RoleMeta{}, nil, nil)
	if len(view) == 0 || len(view[0].Substeps) < 3 {
		t.Fatalf("unexpected traceability shape: %#v", view)
	}
	terminatedSub := view[0].Substeps[1]
	if terminatedSub.Status != processStatusTerminated {
		t.Fatalf("terminated substep status = %q, want %s", terminatedSub.Status, processStatusTerminated)
	}
	if terminatedSub.Reason != "Stream ended early" {
		t.Fatalf("terminated reason = %q", terminatedSub.Reason)
	}
	if terminatedSub.DetailMessage != "supplier cancelled" {
		t.Fatalf("terminated detail = %q", terminatedSub.DetailMessage)
	}

	skippedSub := view[0].Substeps[2]
	if skippedSub.Status != "skipped" {
		t.Fatalf("skipped substep status = %q, want skipped", skippedSub.Status)
	}
	if skippedSub.DetailMessage != "Step not completed because the stream was ended before this." {
		t.Fatalf("skipped detail = %q", skippedSub.DetailMessage)
	}
}

func TestDPPTraceValuesFallbackFlattensMapAndSkipsAttachmentMeta(t *testing.T) {
	sub := WorkflowSub{
		SubstepID: "1.1",
		InputKey:  "value",
		InputType: "formata",
	}
	values := dppTraceValues(sub, ProcessStep{Data: map[string]interface{}{
		"other": map[string]interface{}{
			"nested": "ok",
		},
		"attachment": map[string]interface{}{
			"attachmentId": primitive.NewObjectID().Hex(),
			"filename":     "proof.pdf",
		},
	}})
	if len(values) != 1 {
		t.Fatalf("expected one fallback flattened value, got %#v", values)
	}
	if values[0].Key != "other.nested" || values[0].Value != "ok" {
		t.Fatalf("unexpected fallback flattened value: %#v", values[0])
	}
}
