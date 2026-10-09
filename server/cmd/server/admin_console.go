package main

import (
	"net/http"
	"strings"
)

func htmxTargetID(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Header.Get("HX-Target"))
}

func wantsAdminConsolePartial(r *http.Request) bool {
	if !isHTMXRequest(r) {
		return false
	}
	target := htmxTargetID(r)
	return target == "" || target == "admin-console"
}

func platformAdminConsole(view PlatformAdminView) AdminConsoleView {
	active := strings.TrimSpace(view.ActivePanel)
	return AdminConsoleView{
		ID:         "admin-console",
		NavLabel:   "Platform admin sections",
		NavHeading: "Platform settings",
		NavItems: []AdminConsoleNavItem{
			{
				Href:         adminPath("organizations"),
				Title:        "Organizations",
				Icon:         "icon-building-grid",
				Active:       active == "organizations" || active == "",
				HasAttention: view.HasOrgCreationAttention,
			},
			{
				Href:   adminPath("streams"),
				Title:  "Streams",
				Icon:   "icon-layers-2",
				Active: active == "streams",
			},
			{
				Href:   adminPath("categories"),
				Title:  "Categories",
				Icon:   "icon-layout-grid",
				Active: active == "categories",
			},
		},
		MainTemplate: "platform_admin_main",
		MainData:     view,
	}
}

func orgAdminConsole(view OrgAdminView) AdminConsoleView {
	active := strings.TrimSpace(view.ActivePanel)
	if active == "" {
		active = "profile"
	}
	return AdminConsoleView{
		ID:         "admin-console",
		NavLabel:   "Organization admin sections",
		NavHeading: "Organization settings",
		NavItems: []AdminConsoleNavItem{
			{
				Href:   organizationPath("profile"),
				Title:  "Profile",
				Icon:   "icon-building-grid",
				Active: active == "profile",
			},
			{
				Href:   organizationPath("roles"),
				Title:  "Roles",
				Icon:   "icon-settings",
				Active: active == "roles",
			},
			{
				Href:         organizationPath("members"),
				Title:        "Members",
				Icon:         "icon-users-group",
				Active:       active == "members",
				HasAttention: view.HasJoinRequestAttention,
			},
		},
		MainTemplate: "org_admin_main",
		MainData:     view,
	}
}

// RoleDialogAutoOpen reports whether a role dialog should reopen after a failed POST.
func (v OrgAdminView) RoleDialogAutoOpen(action, slug string) bool {
	if strings.TrimSpace(v.RoleError) == "" {
		return false
	}
	if strings.TrimSpace(v.RoleDialogAction) != action {
		return false
	}
	if action == "create" {
		return true
	}
	return slug != "" && strings.TrimSpace(v.RoleDialogSlug) == slug
}

// InviteDialogAutoOpen reports whether the add-user dialog should reopen.
func (v OrgAdminView) InviteDialogAutoOpen() bool {
	return strings.TrimSpace(v.InviteError) != "" || strings.TrimSpace(v.InviteLink) != ""
}

// OrganizationDialogAutoOpen reports whether an org dialog should reopen after a failed POST.
func (v PlatformAdminView) OrganizationDialogAutoOpen(action, slug string) bool {
	if strings.TrimSpace(v.OrganizationDialogAction) != action {
		return false
	}
	switch action {
	case "create":
		return strings.TrimSpace(v.OrganizationError) != ""
	case "invite":
		return strings.TrimSpace(v.InviteError) != "" &&
			slug != "" &&
			strings.TrimSpace(v.OrganizationDialogSlug) == slug
	case "edit", "delete":
		return strings.TrimSpace(v.OrganizationError) != "" &&
			slug != "" &&
			strings.TrimSpace(v.OrganizationDialogSlug) == slug
	default:
		return false
	}
}
