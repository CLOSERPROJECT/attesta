package main

import (
	"fmt"
	"reflect"
	"strings"
)

// streamPresentationOnlyChange reports whether oldYAML and newYAML differ only in
// Stream presentation fields (name, description, categorization). Fail closed:
// parse errors or empty inputs return ok=false with an error.
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
	stripStreamPresentationFields(&oldCfg)
	stripStreamPresentationFields(&newCfg)
	return reflect.DeepEqual(oldCfg, newCfg), nil
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
