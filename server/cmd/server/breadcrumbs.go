package main

import "strings"

func buildStreamBreadcrumbs(workflowKey, workflowName string) BreadcrumbsView {
	key := strings.TrimSpace(workflowKey)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Dashboard", Href: appHomePath},
		{Label: streamCrumbLabel(workflowName, key), Href: streamPath(key), Current: true},
	}}
}

func buildProcessBreadcrumbs(workflowKey, workflowName, instanceName, processID string) BreadcrumbsView {
	key := strings.TrimSpace(workflowKey)
	id := strings.TrimSpace(processID)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Dashboard", Href: appHomePath},
		{Label: streamCrumbLabel(workflowName, key), Href: streamPath(key)},
		{Label: processInstanceCrumbLabel(instanceName, id), Href: streamInstancePath(key, id), Current: true},
	}}
}

func buildOrgAdminBreadcrumbs(activePanel string) BreadcrumbsView {
	section := strings.TrimSpace(activePanel)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Dashboard", Href: appHomePath},
		{Label: "Organization admin", Href: organizationPath("profile")},
		{Label: orgAdminSectionLabel(section), Href: orgAdminSectionHref(section), Current: true},
	}}
}

func buildPlatformAdminBreadcrumbs(activePanel string) BreadcrumbsView {
	section := strings.TrimSpace(activePanel)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Dashboard", Href: appHomePath},
		{Label: "Platform admin", Href: adminPath("")},
		{Label: platformAdminSectionLabel(section), Href: platformAdminSectionHref(section), Current: true},
	}}
}

func platformAdminSectionLabel(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "categories":
		return "Categories"
	default:
		return "Organizations"
	}
}

func platformAdminSectionHref(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "categories":
		return adminPath("categories")
	default:
		return adminPath("organizations")
	}
}

func streamCrumbLabel(workflowName, workflowKey string) string {
	if name := strings.TrimSpace(workflowName); name != "" {
		return "Stream: " + name
	}
	return "Stream: " + strings.TrimSpace(workflowKey)
}

func processInstanceCrumbLabel(instanceName, processID string) string {
	if name := strings.TrimSpace(instanceName); name != "" {
		return "Instance: " + name
	}
	return "Instance: " + strings.TrimSpace(processID)
}

func orgAdminSectionLabel(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "roles":
		return "Roles"
	case "members":
		return "Members"
	default:
		return "Profile"
	}
}

func orgAdminSectionRest(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "roles":
		return "roles"
	case "members":
		return "members"
	default:
		return "profile"
	}
}

func orgAdminSectionHref(activePanel string) string {
	return organizationPath(orgAdminSectionRest(activePanel))
}
