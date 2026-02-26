package main

import (
	"html/template"
	"os"
)

func testRuntimeConfig() RuntimeConfig {
	return RuntimeConfig{
		Workflow: WorkflowDef{
			Name: "Demo workflow",
			Steps: []WorkflowStep{
				{
					StepID: "1",
					Title:  "Step 1",
					Order:  1,
					Substep: []WorkflowSub{
						{SubstepID: "1.1", Title: "A", Order: 1, Role: "dep1", InputKey: "value", InputType: "number"},
						{SubstepID: "1.2", Title: "B", Order: 2, Role: "dep1", InputKey: "note", InputType: "string"},
						{SubstepID: "1.3", Title: "C", Order: 3, Role: "dep1", InputKey: "attachment", InputType: "file"},
					},
				},
				{
					StepID: "2",
					Title:  "Step 2",
					Order:  2,
					Substep: []WorkflowSub{
						{SubstepID: "2.1", Title: "D", Order: 1, Role: "dep2", InputKey: "value", InputType: "number"},
						{SubstepID: "2.2", Title: "E", Order: 2, Role: "dep2", InputKey: "note", InputType: "string"},
					},
				},
				{
					StepID: "3",
					Title:  "Step 3",
					Order:  3,
					Substep: []WorkflowSub{
						{SubstepID: "3.1", Title: "F", Order: 1, Role: "dep3", InputKey: "value", InputType: "number"},
						{SubstepID: "3.2", Title: "G", Order: 2, Role: "dep3", InputKey: "note", InputType: "string"},
					},
				},
			},
		},
		Departments: []Department{
			{ID: "dep1", Name: "Department 1", Color: "#f0f3ea", Border: "#d9e0d0"},
			{ID: "dep2", Name: "Department 2", Color: "#f0f3ea", Border: "#d9e0d0"},
			{ID: "dep3", Name: "Department 3", Color: "#f0f3ea", Border: "#d9e0d0"},
		},
		Users: []User{
			{ID: "u1", Name: "User 1", DepartmentID: "dep1"},
			{ID: "u2", Name: "User 2", DepartmentID: "dep2"},
			{ID: "u3", Name: "User 3", DepartmentID: "dep3"},
		},
	}
}

func testTemplates() *template.Template {
	return template.Must(template.New("test").Parse(`
{{define "layout.html"}}
  {{if eq .Body "home_picker_body"}}{{template "home_picker_body" .}}
  {{else if eq .Body "dashboard_body"}}{{template "dashboard_body" .}}
  {{else if eq .Body "platform_admin_body"}}{{template "platform_admin_body" .}}
  {{else if eq .Body "org_admin_body"}}{{template "org_admin_body" .}}
  {{else if eq .Body "home_body"}}{{template "home_body" .}}
  {{else if eq .Body "process_body"}}{{template "process_body" .}}
  {{else if eq .Body "dpp_body"}}{{template "dpp_body" .}}
  {{else if eq .Body "backoffice_picker_body"}}{{template "backoffice_picker_body" .}}
  {{else if eq .Body "backoffice_landing_body"}}{{template "backoffice_landing_body" .}}
  {{else if eq .Body "dept_dashboard_body"}}{{template "dept_dashboard_body" .}}
  {{else if eq .Body "dept_process_body"}}{{template "dept_process_body" .}}{{end}}
{{end}}
{{define "home_picker_body"}}HOME_PICKER {{range .Workflows}}{{.Key}}:{{.Name}}{{if .Description}}:{{.Description}}{{end}}:{{.Counts.NotStarted}}/{{.Counts.Started}}/{{.Counts.Terminated}}|{{end}}{{end}}
{{define "home_body"}}HOME {{.LatestProcessID}}{{end}}
{{define "home.html"}}{{template "layout.html" .}}{{end}}
{{define "dashboard_body"}}DASHBOARD_ME {{.UserID}} TODO {{len .TodoActions}} ACTIVE {{len .ActiveProcesses}} DONE {{len .DoneProcesses}}{{end}}
{{define "dashboard.html"}}{{template "layout.html" .}}{{end}}
{{define "dashboard_partial.html"}}{{template "dashboard_body" .}}{{end}}
{{define "platform_admin_body"}}PLATFORM_ADMIN ORGS {{len .Organizations}} {{.InviteLink}}{{if .Error}} {{.Error}}{{end}}{{end}}
{{define "platform_admin.html"}}{{template "layout.html" .}}{{end}}
{{define "org_admin_body"}}ORG_ADMIN {{.Organization.Slug}} ROLES {{len .Roles}} {{.InviteLink}}{{if .Error}} {{.Error}}{{end}}{{end}}
{{define "org_admin.html"}}{{template "layout.html" .}}{{end}}
{{define "process_body"}}PROCESS {{.ProcessID}} {{.DPPURL}}{{template "action_list.html" .ActionList}}{{template "timeline.html" .Timeline}}{{end}}
{{define "process_downloads"}}DOWNLOADS {{.ProcessID}} {{.DPPURL}}{{end}}
{{define "process.html"}}{{template "layout.html" .}}{{end}}
{{define "dpp_body"}}DPP GTIN {{.GTIN}} LOT {{.Lot}} SERIAL {{.Serial}} LINK {{.DigitalLink}} MERKLE {{.Export.Merkle.Root}}{{end}}
{{define "dpp.html"}}{{template "layout.html" .}}{{end}}
{{define "timeline.html"}}TIMELINE {{range .}}{{.StepID}} {{end}}{{end}}
{{define "backoffice_picker_body"}}BACKOFFICE_PICKER {{range .Workflows}}{{.Key}}:{{.Name}}{{if .Description}}:{{.Description}}{{end}}:{{.Counts.NotStarted}}/{{.Counts.Started}}/{{.Counts.Terminated}}|{{end}}{{end}}
{{define "backoffice_landing_body"}}BACKOFFICE{{end}}
{{define "backoffice.html"}}{{template "layout.html" .}}{{end}}
{{define "backoffice_landing.html"}}{{template "layout.html" .}}{{end}}
{{define "dept_dashboard_content"}}DASHBOARD {{.CurrentUser.Role}} TODO {{len .TodoActions}} ACTIVE {{len .ActiveProcesses}} DONE {{len .DoneProcesses}}{{end}}
{{define "dept_dashboard_body"}}{{template "dept_dashboard_content" .}}{{end}}
{{define "backoffice_department.html"}}{{template "layout.html" .}}{{end}}
{{define "backoffice_department_partial.html"}}{{template "dept_dashboard_content" .}}{{end}}
{{define "dept_process_body"}}PROCESS_PAGE {{template "action_list.html" .}}{{end}}
{{define "backoffice_process.html"}}{{template "layout.html" .}}{{end}}
{{define "action_list.html"}}ACTION_LIST {{.Error}}{{end}}
{{define "error_banner.html"}}{{if .Error}}ERROR {{.Error}}{{end}}{{end}}
`))
}

func writeTwoSubstepWorkflowConfig(t testHelperT, path, name string) {
	t.Helper()
	content := "workflow:\n" +
		"  name: \"" + name + "\"\n" +
		"  steps:\n" +
		"    - id: \"1\"\n" +
		"      title: \"Step 1\"\n" +
		"      order: 1\n" +
		"      organization: \"org1\"\n" +
		"      substeps:\n" +
		"        - id: \"1.1\"\n" +
		"          title: \"Input 1\"\n" +
		"          order: 1\n" +
		"          roles: [\"dep1\"]\n" +
		"          inputKey: \"value1\"\n" +
		"          inputType: \"string\"\n" +
		"        - id: \"1.2\"\n" +
		"          title: \"Input 2\"\n" +
		"          order: 2\n" +
		"          roles: [\"dep1\"]\n" +
		"          inputKey: \"value2\"\n" +
		"          inputType: \"string\"\n" +
		"organizations:\n" +
		"  - slug: \"org1\"\n" +
		"    name: \"Organization 1\"\n" +
		"roles:\n" +
		"  - orgSlug: \"org1\"\n" +
		"    slug: \"dep1\"\n" +
		"    name: \"Department 1\"\n" +
		"users:\n" +
		"  - id: \"u1\"\n" +
		"    name: \"User 1\"\n" +
		"    departmentId: \"dep1\"\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp config %s: %v", path, err)
	}
}

type testHelperT interface {
	Helper()
	Fatalf(format string, args ...interface{})
}
