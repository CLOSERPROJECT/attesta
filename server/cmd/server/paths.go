package main

import (
	"net/url"
	"strings"
)

const appHomePath = "/my"

func onboardingPath() string {
	return "/my/onboarding"
}

func emailVerificationPath() string {
	return "/verify"
}

func emailVerificationConfirmPath() string {
	return "/verify/confirm"
}

func onboardingJoinPath() string {
	return "/my/onboarding/join"
}

func onboardingRequestOrganizationPath() string {
	return "/my/onboarding/request-organization"
}

func leaveOrganizationPath() string {
	return "/my/leave-organization"
}

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

// substepHeadingID matches templates/components/substep_shell.html
// (dots in SubstepID become dashes).
func substepHeadingID(substepID string) string {
	return "substep-" + strings.ReplaceAll(strings.TrimSpace(substepID), ".", "-") + "-heading"
}

// streamInstanceSubstepPath deep-links to a Substep: opens it via ?substep=
// and scrolls via the heading fragment used in substep_shell.
func streamInstanceSubstepPath(key, instanceID, substepID string) string {
	base := streamInstancePath(key, instanceID)
	substepID = strings.TrimSpace(substepID)
	if substepID == "" {
		return base
	}
	return base + "?substep=" + url.QueryEscape(substepID) + "#" + substepHeadingID(substepID)
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

// adminPath joins /admin with rest.
// rest may be "organizations", "/categories", or "organizations/logo/{id}".
func adminPath(rest string) string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "/admin"
	}
	rest = strings.TrimPrefix(rest, "/")
	return "/admin/" + rest
}
