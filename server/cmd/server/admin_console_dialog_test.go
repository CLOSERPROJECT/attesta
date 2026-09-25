package main

import (
	"html/template"
	"testing"
)

func TestDataAutoOpen(t *testing.T) {
	if got := dataAutoOpen(false); got != "" {
		t.Fatalf("dataAutoOpen(false) = %q, want empty", got)
	}
	if got := dataAutoOpen(true); got != template.HTMLAttr("data-auto-open") {
		t.Fatalf("dataAutoOpen(true) = %q, want data-auto-open", got)
	}
}

func TestOrgAdminViewRoleDialogAutoOpen(t *testing.T) {
	view := OrgAdminView{
		RoleError:        "name taken",
		RoleDialogAction: "edit",
		RoleDialogSlug:   "ops",
	}
	if !view.RoleDialogAutoOpen("edit", "ops") {
		t.Fatal("expected edit dialog for matching slug")
	}
	if view.RoleDialogAutoOpen("edit", "other") {
		t.Fatal("did not expect edit dialog for other slug")
	}
	if view.RoleDialogAutoOpen("create", "") {
		t.Fatal("did not expect create dialog while action is edit")
	}

	view.RoleDialogAction = "create"
	if !view.RoleDialogAutoOpen("create", "") {
		t.Fatal("expected create dialog")
	}
	if !view.RoleDialogAutoOpen("create", "ignored") {
		t.Fatal("create should open regardless of slug argument")
	}

	view.RoleError = ""
	if view.RoleDialogAutoOpen("create", "") {
		t.Fatal("did not expect dialog without role error")
	}
}

func TestOrgAdminViewInviteDialogAutoOpen(t *testing.T) {
	if (OrgAdminView{}).InviteDialogAutoOpen() {
		t.Fatal("did not expect invite dialog by default")
	}
	if !(OrgAdminView{InviteError: "bad email"}).InviteDialogAutoOpen() {
		t.Fatal("expected invite dialog for invite error")
	}
	if !(OrgAdminView{InviteLink: "/invite/x"}).InviteDialogAutoOpen() {
		t.Fatal("expected invite dialog for invite link")
	}
}

func TestPlatformAdminViewOrganizationDialogAutoOpen(t *testing.T) {
	create := PlatformAdminView{
		OrganizationError:        "slug taken",
		OrganizationDialogAction: "create",
	}
	if !create.OrganizationDialogAutoOpen("create", "") {
		t.Fatal("expected create dialog")
	}

	invite := PlatformAdminView{
		InviteError:              "bad email",
		OrganizationDialogAction: "invite",
		OrganizationDialogSlug:   "acme",
	}
	if !invite.OrganizationDialogAutoOpen("invite", "acme") {
		t.Fatal("expected invite dialog")
	}
	if invite.OrganizationDialogAutoOpen("invite", "other") {
		t.Fatal("did not expect invite dialog for other slug")
	}

	edit := PlatformAdminView{
		OrganizationError:        "bad name",
		OrganizationDialogAction: "edit",
		OrganizationDialogSlug:   "acme",
	}
	if !edit.OrganizationDialogAutoOpen("edit", "acme") {
		t.Fatal("expected edit dialog")
	}
	if edit.OrganizationDialogAutoOpen("delete", "acme") {
		t.Fatal("did not expect delete dialog while action is edit")
	}
}
