package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestStreamIsStartableViaFirstSubstepRole(t *testing.T) {
	cfg := twoOrgDiscoveryConfig()
	user := &AccountUser{OrgSlug: "org1", RoleSlugs: []string{"dep1"}, Status: "active"}
	if !streamIsStartable(user, cfg) {
		t.Fatal("org1 member with first-substep role dep1 should be startable")
	}
}

func TestStreamIsStartableViaOrgAdminOnFirstStepOrg(t *testing.T) {
	cfg := twoOrgDiscoveryConfig()
	user := &AccountUser{OrgSlug: "org1", RoleSlugs: []string{"org-admin"}, Status: "active"}
	if !streamIsStartable(user, cfg) {
		t.Fatal("org1 org-admin should be startable without first-substep role")
	}
}

func TestStreamIsStartableFalseWhenOrgOnlyLaterStep(t *testing.T) {
	cfg := twoOrgDiscoveryConfig()
	user := &AccountUser{OrgSlug: "org2", RoleSlugs: []string{"dep2"}, Status: "active"}
	if streamIsStartable(user, cfg) {
		t.Fatal("org2 owns only later step; must not be startable")
	}
}

func TestStreamIsStartableFalseWrongRole(t *testing.T) {
	cfg := twoOrgDiscoveryConfig()
	user := &AccountUser{OrgSlug: "org1", RoleSlugs: []string{"dep2"}, Status: "active"}
	if streamIsStartable(user, cfg) {
		t.Fatal("org1 with wrong role must not be startable")
	}
}

func TestStreamIsStartableFalseForPANilUnaffiliated(t *testing.T) {
	cfg := twoOrgDiscoveryConfig()
	if streamIsStartable(nil, cfg) {
		t.Fatal("nil user must not be startable")
	}
	if streamIsStartable(&AccountUser{IsPlatformAdmin: true, OrgSlug: "org1", RoleSlugs: []string{"dep1"}}, cfg) {
		t.Fatal("platform admin must not be startable on affiliated stack")
	}
	if streamIsStartable(&AccountUser{RoleSlugs: []string{"dep1"}, Status: "active"}, cfg) {
		t.Fatal("unaffiliated user must not be startable")
	}
}

func TestStreamIsStartableOrgAdminEmptyFirstSubsteps(t *testing.T) {
	cfg := RuntimeConfig{
		Workflow: WorkflowDef{
			Name: "No substeps",
			Steps: []WorkflowStep{
				{StepID: "1", Order: 1, OrganizationSlug: "org1"},
			},
		},
	}
	admin := &AccountUser{OrgSlug: "org1", RoleSlugs: []string{"org-admin"}}
	if !streamIsStartable(admin, cfg) {
		t.Fatal("org-admin owning first-step org remains startable when first step has no substeps")
	}
	member := &AccountUser{OrgSlug: "org1", RoleSlugs: []string{"dep1"}}
	if streamIsStartable(member, cfg) {
		t.Fatal("role member must not be startable when first step has no substeps")
	}
}

func TestStreamIsStartableFalseEmptySteps(t *testing.T) {
	cfg := RuntimeConfig{Workflow: WorkflowDef{Name: "Empty"}}
	if streamIsStartable(&AccountUser{OrgSlug: "org1", RoleSlugs: []string{"org-admin"}}, cfg) {
		t.Fatal("empty steps must not be startable")
	}
}

func TestListDiscoveryStreamsPAAndUnaffiliatedEmpty(t *testing.T) {
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))
	server := &Server{configDir: tempDir}

	pa, err := server.listDiscoveryStreams(context.Background(), &AccountUser{
		IsPlatformAdmin: true, OrgSlug: "org1", RoleSlugs: []string{"dep1"},
	})
	if err != nil || len(pa) != 0 {
		t.Fatalf("PA discovery = %+v err=%v", pa, err)
	}

	unaffiliated, err := server.listDiscoveryStreams(context.Background(), &AccountUser{
		IdentityUserID: "orphan", RoleSlugs: []string{"dep1"}, Status: "active",
	})
	if err != nil || len(unaffiliated) != 0 {
		t.Fatalf("unaffiliated discovery = %+v err=%v", unaffiliated, err)
	}
}

func TestListDiscoveryStreamsParticipatingOnly(t *testing.T) {
	tempDir := t.TempDir()
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "workflow.yaml"))
	writeOtherOrgOnlyWorkflowConfig(t, filepath.Join(tempDir, "outsider.yaml"))
	server := &Server{configDir: tempDir}

	items, err := server.listDiscoveryStreams(context.Background(), &AccountUser{
		OrgSlug: "org1", RoleSlugs: []string{"dep1"}, Status: "active",
	})
	if err != nil {
		t.Fatalf("listDiscoveryStreams: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("participating-only discovery = %+v, want 1 item", items)
	}
	if items[0].WorkflowKey != "workflow" {
		t.Fatalf("WorkflowKey = %q, want workflow", items[0].WorkflowKey)
	}
}

func TestListDiscoveryStreamsOrderingStartableBeforeRest(t *testing.T) {
	tempDir := t.TempDir()
	// alpha: org2 owns first step → not startable for org1, but org1 participates later.
	writeOrg2FirstThenOrg1WorkflowConfig(t, filepath.Join(tempDir, "alpha.yaml"))
	// zeta: org1 owns first step → startable for org1.
	writeTwoOrgWorkflowConfig(t, filepath.Join(tempDir, "zeta.yaml"))
	server := &Server{configDir: tempDir}

	items, err := server.listDiscoveryStreams(context.Background(), &AccountUser{
		OrgSlug: "org1", RoleSlugs: []string{"dep1"}, Status: "active",
	})
	if err != nil {
		t.Fatalf("listDiscoveryStreams: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("discovery = %+v, want 2 items", items)
	}
	if !items[0].Startable || items[0].WorkflowKey != "zeta" {
		t.Fatalf("first item = %+v, want startable zeta", items[0])
	}
	if items[1].Startable || items[1].WorkflowKey != "alpha" {
		t.Fatalf("second item = %+v, want non-startable alpha", items[1])
	}
	if items[0].StartAction != streamPath("zeta")+"/instance/start" {
		t.Fatalf("StartAction = %q", items[0].StartAction)
	}
	if items[1].StartAction != "" {
		t.Fatalf("non-startable StartAction = %q, want empty", items[1].StartAction)
	}
	if items[0].Href != streamPath("zeta")+"/" {
		t.Fatalf("Href = %q", items[0].Href)
	}
	if items[0].WorkflowName != "Two-org workflow" {
		t.Fatalf("WorkflowName = %q", items[0].WorkflowName)
	}
}

func twoOrgDiscoveryConfig() RuntimeConfig {
	return RuntimeConfig{
		Workflow: WorkflowDef{
			Name:        "Two-org workflow",
			Description: "Cross-org stream",
			Steps: []WorkflowStep{
				{
					StepID: "1", Order: 1, OrganizationSlug: "org1",
					Substep: []WorkflowSub{{SubstepID: "1.1", Order: 1, Roles: []string{"dep1"}}},
				},
				{
					StepID: "2", Order: 2, OrganizationSlug: "org2",
					Substep: []WorkflowSub{{SubstepID: "2.1", Order: 1, Roles: []string{"dep2"}}},
				},
			},
		},
		Organizations: []WorkflowOrganization{
			{Slug: "org1", Name: "Organization 1"},
			{Slug: "org2", Name: "Organization 2"},
		},
		Roles: []WorkflowRole{
			{OrgSlug: "org1", Slug: "dep1", Name: "Department 1"},
			{OrgSlug: "org2", Slug: "dep2", Name: "Department 2"},
		},
	}
}

func writeOtherOrgOnlyWorkflowConfig(t *testing.T, path string) {
	t.Helper()
	content := `workflow:
  name: "Outsider only"
  steps:
    - id: "1"
      title: "Step 1"
      order: 1
      organization: "org3"
      substeps:
        - id: "1.1"
          title: "Input"
          order: 1
          roles: ["dep3"]
          inputKey: "value"
          inputType: "formata"
          schema:
            type: object
organizations:
  - slug: "org3"
    name: "Organization 3"
roles:
  - orgSlug: "org3"
    slug: "dep3"
    name: "Department 3"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write outsider config: %v", err)
	}
}

func writeOrg2FirstThenOrg1WorkflowConfig(t *testing.T, path string) {
	t.Helper()
	content := `workflow:
  name: "Org2-first workflow"
  steps:
    - id: "1"
      title: "Step 1"
      order: 1
      organization: "org2"
      substeps:
        - id: "1.1"
          title: "Org 2 input"
          order: 1
          roles: ["dep2"]
          inputKey: "value"
          inputType: "formata"
          schema:
            type: object
    - id: "2"
      title: "Step 2"
      order: 2
      organization: "org1"
      substeps:
        - id: "2.1"
          title: "Org 1 input"
          order: 1
          roles: ["dep1"]
          inputKey: "value"
          inputType: "formata"
          schema:
            type: object
organizations:
  - slug: "org1"
    name: "Organization 1"
  - slug: "org2"
    name: "Organization 2"
roles:
  - orgSlug: "org1"
    slug: "dep1"
    name: "Department 1"
  - orgSlug: "org2"
    slug: "dep2"
    name: "Department 2"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write org2-first config: %v", err)
	}
}
