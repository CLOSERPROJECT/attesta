package main

import (
	"net/url"
	"strings"
)

const appHomePath = "/my"

func streamPath(key string) string {
	return "/my/streams/" + strings.TrimSpace(key)
}

func publicStreamPath(key string) string {
	return "/streams/" + strings.TrimSpace(key)
}

// publicHomePath is the landing URL for a taxonomy leaf filter.
// Both slugs required; otherwise returns "/".
func publicHomePath(categorySlug, subCategorySlug string) string {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	if cat == "" || sub == "" {
		return "/"
	}
	return "/?" + url.Values{
		"category":    {cat},
		"subCategory": {sub},
	}.Encode()
}

func streamInstancePath(key, instanceID string) string {
	return streamPath(key) + "/instance/" + strings.TrimSpace(instanceID)
}

// organizationPath joins /my/organization with rest.
// rest may be "profile", "/roles", or "formata-builder?stream=x".
func organizationPath(rest string) string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "/my/organization"
	}
	rest = strings.TrimPrefix(rest, "/")
	return "/my/organization/" + rest
}
