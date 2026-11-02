package living

import (
	"fmt"
)

type prepResult struct {
	Facts map[string]any
}

// prepTaskSubmission filters leftover-table / containerReg values into transform facts.
// Chrome field ids (display-only) are stripped here — Commit law, not SPA seal.
func prepTaskSubmission(task map[string]any, value map[string]any, leftoverIDs []string) (prepResult, error) {
	value = stripChromeFields(task, value)
	if needsLeftoverProjection(task) {
		return prepLeftoverFacts(task, value, leftoverIDs)
	}
	if materials, ok := containerRegMaterials(task); ok {
		facts, err := flattenContainerRegFacts(value, materials)
		if err != nil {
			return prepResult{}, err
		}
		return prepResult{Facts: containerRegFactsAsMap(facts)}, nil
	}
	return prepResult{Facts: cloneFacts(value)}, nil
}

func stripChromeFields(task map[string]any, value map[string]any) map[string]any {
	rules := readChromeRules(task)
	if len(rules) == 0 {
		return value
	}
	out := cloneFacts(value)
	for _, rule := range rules {
		delete(out, rule.ID)
	}
	return out
}

func prepLeftoverFacts(task map[string]any, value map[string]any, leftoverIDs []string) (prepResult, error) {
	primary, completeness, ok := primaryDynamicProperty(task)
	if !ok || completeness == "" {
		return prepResult{}, fmt.Errorf("leftover-table Task missing completeness")
	}

	if completeness == "all" && len(leftoverIDs) == 0 {
		return prepResult{Facts: map[string]any{}}, nil
	}

	mapValue := leftoverMapValue(task, value, primary)
	addSchema := additionalPropertiesSchema(task, primary)

	var facts map[string]any
	if additionalPropertiesIsProgress(addSchema) {
		filled, err := filterFilledProgress(mapValue, addSchema)
		if err != nil {
			return prepResult{}, err
		}
		if msg := validateProgressFacts(filled, leftoverIDs); msg != "" {
			return prepResult{}, fmt.Errorf("%s", msg)
		}
		facts = filled
	} else {
		filled, err := filterFilledObjectRows(mapValue, addSchema)
		if err != nil {
			return prepResult{}, err
		}
		if msg := validateObjectRowFacts(filled, leftoverIDs, addSchema); msg != "" {
			return prepResult{}, fmt.Errorf("%s", msg)
		}
		facts = filled
	}

	if completeness == "all" {
		if msg := enforceAllCompletenessKeys(facts, leftoverIDs); msg != "" {
			return prepResult{}, fmt.Errorf("%s", msg)
		}
	}
	return prepResult{Facts: facts}, nil
}

func leftoverMapValue(task map[string]any, value map[string]any, primary string) map[string]any {
	if primary == "" {
		return value
	}
	nested, ok := value[primary].(map[string]any)
	if !ok || nested == nil {
		return map[string]any{}
	}
	return nested
}

// Progress cells are string enums; clear wire may allow null via anyOf / type union.
func additionalPropertiesIsProgress(add map[string]any) bool {
	if add == nil {
		return false
	}
	if schemaTypeIncludesString(add["type"]) {
		return true
	}
	anyOf, ok := add["anyOf"].([]any)
	if !ok {
		return false
	}
	for _, branch := range anyOf {
		b, ok := branch.(map[string]any)
		if !ok {
			continue
		}
		if schemaTypeIncludesString(b["type"]) {
			return true
		}
	}
	return false
}

func schemaTypeIncludesString(raw any) bool {
	switch t := raw.(type) {
	case string:
		return t == "string"
	case []any:
		for _, item := range t {
			if s, ok := item.(string); ok && s == "string" {
				return true
			}
		}
	}
	return false
}

func enforceAllCompletenessKeys(facts map[string]any, leftoverIDs []string) string {
	if len(leftoverIDs) == 0 {
		return ""
	}
	var missing []string
	for _, id := range leftoverIDs {
		if _, ok := facts[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	return "completeness all requires every leftover — missing " + joinComma(missing)
}

func joinComma(ids []string) string {
	out := ""
	for i, id := range ids {
		if i > 0 {
			out += ", "
		}
		out += id
	}
	return out
}

func taskOnSubmitTransform(task map[string]any) (any, bool) {
	onSubmit, _ := task["onSubmit"].(map[string]any)
	if onSubmit == nil {
		return nil, false
	}
	transform, ok := onSubmit["transform"]
	return transform, ok
}

func lookupTask(cfg OrgConfig, jobID, taskID string) (Job, map[string]any, error) {
	jobVal, ok := cfg.Jobs.Get(jobID)
	if !ok {
		return Job{}, nil, fmt.Errorf("unknown Job %q", jobID)
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return Job{}, nil, fmt.Errorf("invalid Job %q", jobID)
	}
	task, ok := TaskByID(job.Tasks, taskID)
	if !ok {
		return Job{}, nil, fmt.Errorf("unknown Task %q", jobID+"/"+taskID)
	}
	return job, map[string]any(task), nil
}
