package main

import "strings"

func buildStreamBreadcrumbs(workflowKey, workflowName string) BreadcrumbsView {
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: streamCrumbLabel(workflowName, workflowKey), Href: ""},
	}}
}

func buildProcessBreadcrumbs(workflowKey, workflowName, instanceName, processID string) BreadcrumbsView {
	key := strings.TrimSpace(workflowKey)
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: streamCrumbLabel(workflowName, key), Href: "/w/" + key + "/"},
		{Label: processInstanceCrumbLabel(instanceName, processID), Href: ""},
	}}
}

func buildOrgAdminBreadcrumbs(activePanel string) BreadcrumbsView {
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Organization admin", Href: "/org-admin/profile"},
		{Label: orgAdminSectionLabel(activePanel), Href: ""},
	}}
}

func buildPlatformAdminBreadcrumbs() BreadcrumbsView {
	return BreadcrumbsView{Items: []BreadcrumbItem{
		{Label: "Streams", Href: "/"},
		{Label: "Platform admin", Href: ""},
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
