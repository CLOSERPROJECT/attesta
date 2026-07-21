package main

import (
	"bytes"
	"strings"
	"testing"
)

func testHomeFilterOptions(items ...StreamInstanceCard) []ProcessStatusGroup {
	return buildHomeFilterOptions(items)
}

func testHomeActiveProcessGroups(items []StreamInstanceCard, statusFilter, sortKey string, page int) []ProcessStatusGroup {
	if sortKey == "" {
		sortKey = "time_desc"
	}
	if statusFilter == "" {
		statusFilter = "all"
	}
	return []ProcessStatusGroup{buildHomeActiveProcessGroup("/w/workflow", items, statusFilter, sortKey, page)}
}

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
		panelURL := "/w/workflow/"
		if status != "all" {
			panelURL = "/w/workflow/?filter=" + status
		}
		pageLinks := []PaginationLink{{Page: 1, URL: panelURL, IsCurrent: true}}
		var sortFields []QueryInput
		if status != "all" {
			sortFields = []QueryInput{{Name: "filter", Value: status}}
		}
		navAriaLabel, navTitle, heading, emptyMessage, paginationAriaLabel := homeProcessStatusCopy(status)
		groups = append(groups, ProcessStatusGroup{
			Status:              status,
			Label:               processStatusLabel(status),
			NavAriaLabel:        navAriaLabel,
			NavTitle:            navTitle,
			Heading:             heading,
			EmptyMessage:        emptyMessage,
			PaginationAriaLabel: paginationAriaLabel,
			PanelID:             "stream-section-" + status,
			TotalCount:          len(processes),
			Sort:                "time_desc",
			SortFields:          sortFields,
			CurrentPage:         1,
			TotalPages:          totalPages,
			PageNumbers:         []int{1},
			PageLinks:           pageLinks,
			PreviousURL:         panelURL,
			NextURL:             panelURL,
			Processes:           processes,
		})
	}
	return groups
}

func TestHomeTemplateRendersSidebarAndReadOnlyPreview(t *testing.T) {
	tmpl := parseTestTemplates(t)

	process := StreamInstanceCard{ID: "process-1", Name: "Pilot batch", Status: "active", DetailHref: "/w/workflow/process/process-1", Percent: 25, DoneSubsteps: 1, TotalSubsteps: 4, CreatedAt: "1 Mar 2026 at 10:00 UTC"}
	items := []StreamInstanceCard{process}
	view := HomeView{
		PageBase: PageBase{
			WorkflowKey:  "workflow",
			WorkflowPath: "/w/workflow",
			WorkflowName: "Demo workflow",
		},
		WorkflowDescription: "Previewable workflow",
		StatusFilter:        "all",
		Sort:                "time_desc",
		FilterOptions:       testHomeFilterOptions(items...),
		ProcessGroups:       testHomeActiveProcessGroups(items, "all", "time_desc", 1),
		Preview: StreamInstanceDetailView{
			HideStatus: true,
			Timeline: []TimelineStep{
				{
					Summary: StepSummaryView{
						StepID: "1",
						Title:  "Collect",
					},
					Substeps: []TimelineSubstep{
						{
							SubstepID: "1.1",
							Title:     "Record input",
							Status:    "available",
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
		`id="stream-dashboard-results"`,
		`hx-get="/w/workflow/"`,
		`filter=available`,
		`filter=active`,
		`filter=done`,
		`filter=terminated`,
		`id="stream-preview-dialog"`,
		`class="dialog-card"`,
		`class="stream-preview-body"`,
		`id="new-instance-dialog"`,
		`name="name"`,
		`Pilot batch`,
		`data-formata-disabled="true"`,
		`Preview only. Start an instance to submit data.`,
	} {
		if !strings.Contains(body, marker) {
			t.Fatalf("expected marker %q in output, got: %s", marker, body)
		}
	}
	if !strings.Contains(compactBody, `aria-labelledby="stream-status-filter-label"`) ||
		!strings.Contains(compactBody, `class="stream-status-filter-select"`) ||
		!strings.Contains(compactBody, `Filter by status`) {
		t.Fatalf("expected stream status filter label, got: %s", body)
	}
	for _, marker := range []string{
		`id="stream-section-all"`,
		`class="stream-status-filter-option is-active"`,
		`hx-target="#stream-dashboard-results"`,
	} {
		if !strings.Contains(compactBody, marker) {
			t.Fatalf("expected stream dashboard HTMX marker %q, got: %s", marker, body)
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
		StatusFilter:        "all",
		Sort:                "time_desc",
		FilterOptions:       testHomeFilterOptions(),
		ProcessGroups:       testHomeActiveProcessGroups(nil, "all", "time_desc", 1),
	}

	var out bytes.Buffer
	if err := tmpl.ExecuteTemplate(&out, "home_body", view); err != nil {
		t.Fatalf("render home template: %v", err)
	}
	body := out.String()
	compactBody := strings.Join(strings.Fields(body), " ")

	for _, marker := range []string{
		`No instances`,
	} {
		if !strings.Contains(compactBody, marker) {
			t.Fatalf("expected empty status marker %q, got: %s", marker, body)
		}
	}
	if !strings.Contains(compactBody, `filter=available`) ||
		!strings.Contains(compactBody, `filter=done`) {
		t.Fatalf("expected filter nav links for all statuses, got: %s", body)
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
		FilterOptions: buildHomeFilterOptions([]StreamInstanceCard{
			{ID: "process-13", Status: "active"},
		}),
		ProcessGroups: []ProcessStatusGroup{
			{
				Status:              "active",
				Label:               "Active",
				NavAriaLabel:        "Active streams",
				NavTitle:            "Streams waiting for someone else input",
				Heading:             "Active stream instances",
				EmptyMessage:        "No active instances",
				PaginationAriaLabel: "Active stream instances pagination",
				PanelID:             "stream-section-active",
				TotalCount:          11,
				Sort:                "status",
				SortFields:          []QueryInput{{Name: "filter", Value: "active"}},
				CurrentPage:         2,
				TotalPages:          3,
				PageNumbers:         []int{1, 2, 3},
				PageLinks:           []PaginationLink{{Page: 1, URL: "/w/workflow/?filter=active&sort=status"}, {Page: 2, URL: "/w/workflow/?filter=active&page=2&sort=status", IsCurrent: true}, {Page: 3, URL: "/w/workflow/?filter=active&page=3&sort=status"}},
				HasPreviousPage:     true,
				HasNextPage:         true,
				PreviousURL:         "/w/workflow/?filter=active&sort=status",
				NextURL:             "/w/workflow/?filter=active&page=3&sort=status",
				Processes:           []StreamInstanceCard{{ID: "process-13", Status: "active", DetailHref: "/w/workflow/process/process-13", Percent: 25, DoneSubsteps: 1, TotalSubsteps: 4, CreatedAt: "1 Mar 2026 at 10:00 UTC"}},
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
	if !strings.Contains(compactBody, `/w/workflow/?filter=active&amp;page=3&amp;sort=status`) {
		t.Fatalf("expected pagination to preserve filter, sort and page, got: %s", body)
	}
	if !strings.Contains(compactBody, `name="sort"`) {
		t.Fatalf("expected global sort select, got: %s", body)
	}
	if !strings.Contains(compactBody, `name="filter" value="active"`) {
		t.Fatalf("expected sort form to preserve filter, got: %s", body)
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
	sortIdx := strings.Index(stickyBlock, `name="sort"`)
	if navIdx == -1 || sortIdx == -1 || !(navIdx < sortIdx) {
		t.Fatalf("expected status sort control below stream-status-filter-nav inside sticky panel, got:\n%s", stickyBlock)
	}
	sectionStart := strings.Index(compactBody, `id="stream-section-active"`)
	sectionEnd := strings.Index(compactBody, `Active stream instances pagination`)
	if sectionStart == -1 || sectionEnd == -1 || sectionStart >= sectionEnd {
		t.Fatal("expected active status section")
	}
	if strings.Contains(compactBody[sectionStart:sectionEnd], `select name="sort"`) {
		t.Fatalf("status sort must not remain in stream status section, got:\n%s", compactBody[sectionStart:sectionEnd])
	}
	if strings.Contains(compactBody, `#stream-section-`) {
		t.Fatalf("did not expect hash anchors on pagination links, got: %s", body)
	}
	if !strings.Contains(compactBody, `hx-get="/w/workflow/?filter=active&amp;page=3&amp;sort=status"`) {
		t.Fatalf("expected HTMX pagination link, got: %s", body)
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
		StatusFilter: "all",
		FilterOptions: testHomeFilterOptions(StreamInstanceCard{
			ID:            "process-1",
			Status:        "available",
			DetailHref:    "/w/workflow/process/process-1",
			Percent:       25,
			DoneSubsteps:  1,
			TotalSubsteps: 4,
			CreatedAt:     "1 Mar 2026 at 10:00 UTC",
		}),
		ProcessGroups: testHomeActiveProcessGroups([]StreamInstanceCard{
			{
				ID:            "process-1",
				Status:        "available",
				DetailHref:    "/w/workflow/process/process-1",
				Percent:       25,
				DoneSubsteps:  1,
				TotalSubsteps: 4,
				CreatedAt:     "1 Mar 2026 at 10:00 UTC",
			},
		}, "all", "time_desc", 1),
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
