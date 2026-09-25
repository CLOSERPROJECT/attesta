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
	subtitle := "Create and manage organizations"
	if active == "categories" {
		subtitle = "Manage stream discovery taxonomy"
	}
	return AdminConsoleView{
		ID:          "admin-console",
		NavLabel:    "Platform admin sections",
		Title:       "Platform admin dashboard",
		Subtitle:    subtitle,
		Breadcrumbs: view.Breadcrumbs,
		NavItems: []AdminConsoleNavItem{
			{
				Href:   adminPath("organizations"),
				Title:  "Organizations",
				Copy:   "Create and manage organizations",
				Active: active != "categories",
			},
			{
				Href:   adminPath("categories"),
				Title:  "Categories",
				Copy:   "Manage stream discovery taxonomy",
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
		ID:          "admin-console",
		NavLabel:    "Organization admin sections",
		Title:       "Organization admin dashboard",
		Subtitle:    "Manage organization settings, roles, and members",
		Breadcrumbs: view.Breadcrumbs,
		NavItems: []AdminConsoleNavItem{
			{
				Href:   organizationPath("profile"),
				Title:  "Organization profile",
				Copy:   "Update your organization name and logo",
				Active: active == "profile",
			},
			{
				Href:   organizationPath("roles"),
				Title:  "Roles",
				Copy:   "Manage the role catalog for your organization",
				Active: active == "roles",
			},
			{
				Href:   organizationPath("members"),
				Title:  "Members",
				Copy:   "Invite people and update member access",
				Active: active == "members",
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
