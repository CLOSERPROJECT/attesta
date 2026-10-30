package living

import (
	"fmt"
)

type prepResult struct {
	Facts map[string]any
}

// prepTaskSubmission filters leftover-table / containerReg values into transform facts.
func prepTaskSubmission(task map[string]any, value map[string]any, leftoverIDs []string) (prepResult, error) {
	if needsLeftoverProjection(task) {
		return prepLeftoverFacts(task, value, leftoverIDs)
	}
	if isContainerRegTask(task) {
		facts, err := flattenContainerRegFacts(value)
		if err != nil {
			return prepResult{}, err
		}
		return prepResult{Facts: containerRegFactsAsMap(facts)}, nil
	}
	return prepResult{Facts: cloneFacts(value)}, nil
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
	valueType := leftoverValueType(task, primary)

	var facts map[string]any
	switch valueType {
	case "progress":
		filled, err := filterFilledProgress(mapValue)
		if err != nil {
			return prepResult{}, err
		}
		if msg := validateProgressFacts(filled, leftoverIDs); msg != "" {
			return prepResult{}, fmt.Errorf("%s", msg)
		}
		facts = filled
	default:
		filled, err := filterFilledObjectWeights(mapValue)
		if err != nil {
			return prepResult{}, err
		}
		if msg := validateWeighFacts(filled, leftoverIDs); msg != "" {
			return prepResult{}, fmt.Errorf("%s", msg)
		}
		facts = weighFactsAsMap(filled)
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

func leftoverValueType(task map[string]any, primary string) string {
	schema, _ := task["schema"].(map[string]any)
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return "weight"
	}
	prop, _ := props[primary].(map[string]any)
	if prop == nil {
		return "weight"
	}
	add, _ := prop["additionalProperties"].(map[string]any)
	if add == nil {
		return "weight"
	}
	if additionalPropertiesIsProgress(add) {
		return "progress"
	}
	return "weight"
}

// Progress cells are string enums; clear wire may allow null via anyOf / type union.
func additionalPropertiesIsProgress(add map[string]any) bool {
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

func isContainerRegTask(task map[string]any) bool {
	schema, _ := task["schema"].(map[string]any)
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return false
	}
	if _, ok := props["sourceLoadId"]; !ok {
		return false
	}
	for _, mat := range containerRegMaterials {
		if _, ok := props[mat.Key]; !ok {
			return false
		}
	}
	return true
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
