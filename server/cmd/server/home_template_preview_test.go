package main

import (
	"bytes"
	"strings"
	"testing"
)

func testHomeProcessGroups(items ...StreamInstanceCard) []ProcessStatusGroup {
	byStatus := map[string][]StreamInstanceCard{}
	for _, item := range items {
		byStatus[item.Status] = append(byStatus[item.Status], item)
	}
	groups := make([]ProcessStatusGroup, 0, len(homeProcessStatuses()))
	for _, status := range homeProcessStatuses() {
		processes := byStatus[status]
		if status == "all" {
			processes = append([]StreamInstanceCard(nil), items...)
		}
		totalPages := 1
		pageLinks := []PaginationLink{{Page: 1, URL: "/w/workflow/#stream-section-" + status, IsCurrent: true}}
		groups = append(groups, ProcessStatusGroup{
			Status:      status,
			Label:       processStatusLabel(status),
			PanelID:     "stream-section-" + status,
			TotalCount:  len(processes),
			Sort:        "time_desc",
			SortParam:   homeProcessStatusSortParam(status),
			CurrentPage: 1,
			TotalPages:  totalPages,
			PageNumbers: []int{1},
			PageLinks:   pageLinks,
			PreviousURL: "/w/workflow/#stream-section-" + status,
			NextURL:     "/w/workflow/#stream-section-" + status,
			Processes:   processes,
		})
	}
	return groups
}

func TestHomeTemplateRendersSidebarAndReadOnlyPreview(t *testing.T) {
	tmpl := parseTestTemplates(t)

	process := StreamInstanceCard{ID: "process-1", Name: "Pilot batch", Status: "active", DetailHref: "/w/workflow/process/process-1", Percent: 25, DoneSubsteps: 1, TotalSubsteps: 4, CreatedAt: "1 Mar 2026 at 10:00 UTC"}
	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		WorkflowDescription: "Previewable workflow",
		Processes:           []StreamInstanceCard{process},
		ProcessGroups:       testHomeProcessGroups(process),
		Preview: StreamInstanceDetailView{
			HideStatus: true,
			Timeline: []TimelineStep{
				{
					Summary: StepSummaryView{
						StepID: "1",
						Title:  "Collect",
					},
					Expanded: true,
					Substeps: []TimelineSubstep{
						{
							SubstepID: "1.1",
							Title:     "Record input",
							Status:    "available",
							Selected:  true,
							Body: &SubstepBodyView{
								WorkflowKey:   "workflow",
								SubstepID:     "1.1",
								Title:         "Record input",
								Status:        "available",
								FormSchema:    `{"type":"object","properties":{"value":{"type":"string"}}}`,
								ReadOnly:      true,
								Reason:        "Preview only. Start an instance to submit data.",
								MatchingRoles: []SubstepRoleOption{{Slug: "dep1", Label: "Department 1"}},
							},
						},
					},
				},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, marker := range []string{
		`data-home-nav="all"`,
		`data-home-nav="available"`,
		`data-home-nav="active"`,
		`data-home-nav="done"`,
		`data-home-nav="terminated"`,
		`data-home-nav="preview"`,
		`id="stream-preview-dialog"`,
		`class="dialog-card"`,
		`class="stream-preview-body"`,
		`id="new-instance-dialog"`,
		`name="name"`,
		`Pilot batch`,
		`data-home-root`,
		`data-formata-disabled="true"`,
		`Preview only. Start an instance to submit data.`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected marker %q in output, got: %s", marker, body)
		}
	}
	if !strings.Contains(compactBody, `aria-labelledby="stream-status-filter-label"`) ||
		!strings.Contains(compactBody, `Filter by status`) {
		t.Fatalf("expected stream status filter label, got: %s", body)
	}
	for _, marker := range []string{
		`id="stream-section-all" data-home-panel="all"`,
		`id="stream-section-available" data-home-panel="available" hidden`,
		`id="stream-section-active" data-home-panel="active" hidden`,
		`id="stream-section-done" data-home-panel="done" hidden`,
		`id="stream-section-terminated" data-home-panel="terminated" hidden`,
		`panels.forEach((panel)`,
		`panel.hidden = currentName !== panelName`,
	} {
		if !strings.Contains(compactBody, marker) {
			t.Fatalf("expected single visible stream status marker %q, got: %s", marker, body)
		}
	}
	if strings.Contains(body, `scrollIntoView`) {
		t.Fatalf("did not expect stream status nav to scroll between panels, got: %s", body)
	}

	if strings.Contains(body, `/process//substep/1.1/complete`) {
		t.Fatalf("did not expect preview to render a submit form action, got: %s", body)
	}
	if strings.Contains(body, `<span class="status">available</span>`) {
		t.Fatalf("did not expect preview to render substep status, got: %s", body)
	}
	if strings.Contains(body, `class="substep substep-available"`) {
		t.Fatalf("did not expect preview to render status-colored substep class, got: %s", body)
	}
}

func TestHomeTemplateRendersEmptyStatusSections(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		WorkflowDescription: "Previewable workflow",
		ProcessGroups:       testHomeProcessGroups(),
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, marker := range []string{
		`No instances`,
		`No available instances`,
		`No active instances`,
		`No completed instances`,
		`No terminated instances`,
	} {
		if !strings.Contains(compactBody, marker) {
			t.Fatalf("expected empty status marker %q, got: %s", marker, body)
		}
	}
}

func TestHomeTemplateRendersStatusPagination(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		Sort:         "status",
		StatusFilter: "active",
		ProcessGroups: []ProcessStatusGroup{
			{
				Status:      "all",
				Label:       "All",
				PanelID:     "stream-section-all",
				TotalCount:  0,
				Sort:        "time_desc",
				SortParam:   "all_sort",
				CurrentPage: 1,
				TotalPages:  1,
				PageNumbers: []int{1},
				PageLinks:   []PaginationLink{{Page: 1, URL: "/w/workflow/#stream-section-all", IsCurrent: true}},
				PreviousURL: "/w/workflow/#stream-section-all",
				NextURL:     "/w/workflow/#stream-section-all",
				Processes:   nil,
			},
			{
				Status:      "available",
				Label:       "Available",
				PanelID:     "stream-section-available",
				TotalCount:  0,
				Sort:        "time_desc",
				SortParam:   "available_sort",
				CurrentPage: 1,
				TotalPages:  1,
				PageNumbers: []int{1},
				PageLinks:   []PaginationLink{{Page: 1, URL: "/w/workflow/#stream-section-available", IsCurrent: true}},
				PreviousURL: "/w/workflow/#stream-section-available",
				NextURL:     "/w/workflow/#stream-section-available",
				Processes:   nil,
			},
			{
				Status:          "active",
				Label:           "Active",
				PanelID:         "stream-section-active",
				TotalCount:      11,
				Sort:            "status",
				SortParam:       "active_sort",
				CurrentPage:     2,
				TotalPages:      3,
				PageNumbers:     []int{1, 2, 3},
				PageLinks:       []PaginationLink{{Page: 1, URL: "/w/workflow/?active_sort=status#stream-section-active"}, {Page: 2, URL: "/w/workflow/?active_page=2&active_sort=status#stream-section-active", IsCurrent: true}, {Page: 3, URL: "/w/workflow/?active_page=3&active_sort=status#stream-section-active"}},
				HasPreviousPage: true,
				HasNextPage:     true,
				PreviousURL:     "/w/workflow/?active_sort=status#stream-section-active",
				NextURL:         "/w/workflow/?active_page=3&active_sort=status#stream-section-active",
				Processes:       []StreamInstanceCard{{ID: "process-13", Status: "active", DetailHref: "/w/workflow/process/process-13", Percent: 25, DoneSubsteps: 1, TotalSubsteps: 4, CreatedAt: "1 Mar 2026 at 10:00 UTC"}},
			},
		},
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	if !strings.Contains(compactBody, `Active stream instances pagination`) {
		t.Fatalf("expected active stream instances pagination nav, got: %s", body)
	}
	if !strings.Contains(compactBody, `/w/workflow/?active_page=3&amp;active_sort=status#stream-section-active`) {
		t.Fatalf("expected pagination to preserve sort and active page, got: %s", body)
	}
	if !strings.Contains(compactBody, `name="active_sort"`) {
		t.Fatalf("expected active group sort select, got: %s", body)
	}
	if !strings.Contains(compactBody, `class="stream-status-sort-control"`) ||
		!strings.Contains(compactBody, `stream-status-rail-label`) ||
		!strings.Contains(compactBody, `Sort by`) {
		t.Fatalf("expected stream-status-sort-control with Sort by label, got: %s", body)
	}
	stickyStart := strings.Index(compactBody, `class="panel panel-sticky"`)
	stickyEnd := strings.Index(compactBody, `class="rail-layout-main home-workflow-panel-main"`)
	if stickyStart == -1 || stickyEnd == -1 || stickyStart >= stickyEnd {
		t.Fatal("expected sticky sidebar before rail-layout-main")
	}
	stickyBlock := compactBody[stickyStart:stickyEnd]
	navIdx := strings.Index(stickyBlock, `aria-labelledby="stream-status-filter-label"`)
	sortIdx := strings.Index(stickyBlock, `name="active_sort"`)
	if navIdx == -1 || sortIdx == -1 || !(navIdx < sortIdx) {
		t.Fatalf("expected status sort control below stream-status-filter-nav inside sticky panel, got:\n%s", stickyBlock)
	}
	sectionStart := strings.Index(compactBody, `id="stream-section-active"`)
	sectionEnd := strings.Index(compactBody, `Active stream instances pagination`)
	if sectionStart == -1 || sectionEnd == -1 || sectionStart >= sectionEnd {
		t.Fatal("expected active status section")
	}
	if strings.Contains(compactBody[sectionStart:sectionEnd], `select name="active_sort"`) {
		t.Fatalf("status sort must not remain in stream status section, got:\n%s", compactBody[sectionStart:sectionEnd])
	}
	if !strings.Contains(compactBody, `#stream-section-active`) {
		t.Fatalf("expected pagination links to target active section, got: %s", body)
	}
}

func TestHomeTemplateHighlightsProcessWhenItIsUsersTurn(t *testing.T) {
	tmpl := parseTestTemplates(t)

	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		ProcessGroups: testHomeProcessGroups(StreamInstanceCard{
			ID:            "process-1",
			Status:        "available",
			DetailHref:    "/w/workflow/process/process-1",
			Percent:       25,
			DoneSubsteps:  1,
			TotalSubsteps: 4,
			CreatedAt:     "1 Mar 2026 at 10:00 UTC",
		}),
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	if !strings.Contains(body, `class="stream-instance-card stream-instance-card-available"`) {
		t.Fatalf("expected available process class, got: %s", body)
	}
	if !strings.Contains(compactBody, `class="status-tag status-tag-compact" data-stream-status="available"`) ||
		!strings.Contains(compactBody, `available`) {
		t.Fatalf("expected available status tag, got: %s", body)
	}
}
