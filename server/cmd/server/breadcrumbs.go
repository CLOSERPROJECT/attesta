package main

import "strings"

func buildStreamBreadcrumbs(workflowKey, workflowName string) BreadcrumbsView {
	key := strings.TrimSpace(workflowKey)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: streamCrumbLabel(workflowName, key), Href: "/w/" + key + "/", Current: true},
	}}
}

func buildProcessBreadcrumbs(workflowKey, workflowName, instanceName, processID string) BreadcrumbsView {
	key := strings.TrimSpace(workflowKey)
	id := strings.TrimSpace(processID)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: streamCrumbLabel(workflowName, key), Href: "/w/" + key + "/"},
		{Label: processInstanceCrumbLabel(instanceName, id), Href: "/w/" + key + "/process/" + id, Current: true},
	}}
}

func buildOrgAdminBreadcrumbs(activePanel string) BreadcrumbsView {
	section := strings.TrimSpace(activePanel)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Organization admin", Href: "/org-admin/profile"},
		{Label: orgAdminSectionLabel(section), Href: orgAdminSectionHref(section), Current: true},
	}}
}

func buildPlatformAdminBreadcrumbs() BreadcrumbsView {
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Platform admin", Href: "/admin/orgs", Current: true},
	}}
}

func streamCrumbLabel(workflowName, workflowKey string) string {
	if name := strings.TrimSpace(workflowName); name != "" {
		return name
	}
	return strings.TrimSpace(workflowKey)
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

func orgAdminSectionHref(activePanel string) string {
	switch strings.TrimSpace(activePanel) {
	case "roles":
		return "/org-admin/roles"
	case "members":
		return "/org-admin/members"
	default:
		return "/org-admin/profile"
	}
}
