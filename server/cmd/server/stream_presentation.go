package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

// streamPresentationOnlyChange reports whether oldYAML and newYAML differ only in
// Stream presentation fields (name, description, categorization). Fail closed:
// parse errors or empty inputs return ok=false with an error.
//
// Formata App.build() rewrites organizations/roles from the live catalog on every
// save (display names + encounter order). Those fields are not executable identity
// (slugs are), so they are normalized away before compare.
func streamPresentationOnlyChange(oldYAML, newYAML string) (bool, error) {
	oldYAML = strings.TrimSpace(oldYAML)
	newYAML = strings.TrimSpace(newYAML)
	if oldYAML == "" || newYAML == "" {
		return false, fmt.Errorf("stream yaml is required for presentation compare")
	}
	oldCfg, err := parseRuntimeConfigData("existing-stream", []byte(oldYAML))
	if err != nil {
		return false, err
	}
	newCfg, err := parseRuntimeConfigData("updated-stream", []byte(newYAML))
	if err != nil {
		return false, err
	}
	normalizeStreamCompareConfig(&oldCfg)
	normalizeStreamCompareConfig(&newCfg)
	return reflect.DeepEqual(oldCfg, newCfg), nil
}

func normalizeStreamCompareConfig(cfg *RuntimeConfig) {
	if cfg == nil {
		return
	}
	stripStreamPresentationFields(cfg)
	normalizeOrgRoleIdentity(cfg)
}

func stripStreamPresentationFields(cfg *RuntimeConfig) {
	if cfg == nil {
		return
	}
	cfg.Workflow.Name = ""
	cfg.Workflow.Description = ""
	cfg.Workflow.CategorySlug = ""
	cfg.Workflow.SubCategorySlug = ""
}

func normalizeOrgRoleIdentity(cfg *RuntimeConfig) {
	if cfg == nil {
		return
	}
	for i := range cfg.Organizations {
		cfg.Organizations[i].Name = ""
	}
	sort.Slice(cfg.Organizations, func(i, j int) bool {
		return cfg.Organizations[i].Slug < cfg.Organizations[j].Slug
	})
	for i := range cfg.Roles {
		cfg.Roles[i].Name = ""
	}
	sort.Slice(cfg.Roles, func(i, j int) bool {
		if cfg.Roles[i].OrgSlug != cfg.Roles[j].OrgSlug {
			return cfg.Roles[i].OrgSlug < cfg.Roles[j].OrgSlug
		}
		return cfg.Roles[i].Slug < cfg.Roles[j].Slug
	})
}
