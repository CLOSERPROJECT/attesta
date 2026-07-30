package main

import (
	"net/url"
	"strings"
)

func taxonomyHasPath(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) bool {
	cat := strings.TrimSpace(categorySlug)
	sub := strings.TrimSpace(subCategorySlug)
	if cat == "" || sub == "" {
		return false
	}
	for _, category := range categories {
		if category.Slug != cat {
			continue
		}
		for _, leaf := range category.SubCategories {
			if leaf.Slug == sub {
				return true
			}
		}
		return false
	}
	return false
}

func resolvePublicHomeSelection(categories []TaxonomyCategoryNode, categorySlug, subCategorySlug string) (string, string) {
	if taxonomyHasPath(categories, categorySlug, subCategorySlug) {
		return strings.TrimSpace(categorySlug), strings.TrimSpace(subCategorySlug)
	}
	if len(categories) == 0 || len(categories[0].SubCategories) == 0 {
		return "", ""
	}
	return categories[0].Slug, categories[0].SubCategories[0].Slug
}

func publicHomeCreateStreamHref(signedIn bool) string {
	target := organizationPath("formata-builder")
	if signedIn {
		return target
	}
	return "/login?next=" + url.QueryEscape(target)
}
