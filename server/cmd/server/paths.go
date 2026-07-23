package main

import "strings"

const appHomePath = "/my"

func streamPath(key string) string {
	return "/my/streams/" + strings.TrimSpace(key)
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
