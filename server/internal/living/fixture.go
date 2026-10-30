package living

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed fixtures/truck-loop-steps.yaml
var truckLoopStepsYAML []byte

// DefaultFixtureVersion is the Config version assigned when seeding the flat truck-loop fixture.
const DefaultFixtureVersion = "1"

// LoadTruckLoopFixture parses the embedded flat truck-loop YAML and renames
// steps/substeps → jobs/tasks so Load speaks living nouns only.
func LoadTruckLoopFixture() (OrgConfig, error) {
	return ParseTruckLoopFixture(truckLoopStepsYAML)
}

// ParseTruckLoopFixture parses truck-loop YAML bytes into an OrgConfig with Job/Task keys.
func ParseTruckLoopFixture(raw []byte) (OrgConfig, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return OrgConfig{}, fmt.Errorf("parse truck-loop yaml: %w", err)
	}
	doc := mappingContent(&root)
	if doc == nil {
		return OrgConfig{}, fmt.Errorf("truck-loop YAML root must be a mapping")
	}

	id, err := requiredString(doc, "id")
	if err != nil {
		return OrgConfig{}, err
	}
	org, _ := optionalString(doc, "org")

	stepsNode := child(doc, "steps")
	if stepsNode == nil {
		return OrgConfig{}, fmt.Errorf("truck-loop YAML requires steps")
	}
	stepsMap := mappingContent(stepsNode)
	if stepsMap == nil {
		return OrgConfig{}, fmt.Errorf("truck-loop YAML steps must be a mapping")
	}

	jobs := newOrderedObject()
	roleSet := map[string]struct{}{}
	var roleOrder []string

	for i := 0; i < len(stepsMap.Content); i += 2 {
		jobID := stepsMap.Content[i].Value
		jobNode := stepsMap.Content[i+1]
		job, roles, err := parseJob(jobID, jobNode)
		if err != nil {
			return OrgConfig{}, err
		}
		jobs.Set(jobID, job)
		for _, role := range roles {
			if _, ok := roleSet[role]; ok {
				continue
			}
			roleSet[role] = struct{}{}
			roleOrder = append(roleOrder, role)
		}
	}

	cfg := OrgConfig{
		ID:      id,
		Org:     org,
		Version: DefaultFixtureVersion,
		Jobs:    jobs,
		Roles:   roleOrder,
	}
	if domainNode := child(doc, "domain"); domainNode != nil {
		var domain any
		if err := domainNode.Decode(&domain); err != nil {
			return OrgConfig{}, fmt.Errorf("decode domain: %w", err)
		}
		if m, ok := domain.(map[string]any); ok {
			cfg.Domain = m
		}
	}
	return cfg, nil
}

// SeedFixtureConfigStore puts the truck-loop Org Config as the Active Config version for its org.
func SeedFixtureConfigStore(store ConfigStore) (OrgConfig, error) {
	cfg, err := LoadTruckLoopFixture()
	if err != nil {
		return OrgConfig{}, err
	}
	org := strings.TrimSpace(cfg.Org)
	if org == "" {
		return OrgConfig{}, fmt.Errorf("truck-loop fixture missing org")
	}
	ctx := context.Background()
	if err := store.Put(ctx, org, cfg.Version, cfg); err != nil {
		return OrgConfig{}, err
	}
	if err := store.SetActiveVersion(ctx, org, cfg.Version); err != nil {
		return OrgConfig{}, err
	}
	return cfg, nil
}

func parseJob(jobID string, node *yaml.Node) (Job, []string, error) {
	m := mappingContent(node)
	if m == nil {
		return Job{}, nil, fmt.Errorf("jobs.%s must be a mapping", jobID)
	}
	repeatable, err := requiredBool(m, "repeatable")
	if err != nil {
		return Job{}, nil, fmt.Errorf("jobs.%s: %w", jobID, err)
	}
	instanceID, err := requiredString(m, "instanceId")
	if err != nil {
		return Job{}, nil, fmt.Errorf("jobs.%s: %w", jobID, err)
	}
	label, err := requiredString(m, "label")
	if err != nil {
		return Job{}, nil, fmt.Errorf("jobs.%s: %w", jobID, err)
	}
	substepsNode := child(m, "substeps")
	if substepsNode == nil || substepsNode.Kind != yaml.SequenceNode || len(substepsNode.Content) == 0 {
		return Job{}, nil, fmt.Errorf("jobs.%s.tasks (from substeps) must be a non-empty list", jobID)
	}

	tasks := make([]TaskConfig, 0, len(substepsNode.Content))
	seen := make(map[string]struct{}, len(substepsNode.Content))
	var roles []string
	for i, taskNode := range substepsNode.Content {
		var payload map[string]any
		if err := taskNode.Decode(&payload); err != nil {
			return Job{}, nil, fmt.Errorf("jobs.%s.tasks[%d]: %w", jobID, i, err)
		}
		id, _ := payload["id"].(string)
		if id == "" {
			return Job{}, nil, fmt.Errorf("jobs.%s.tasks[%d]: id required", jobID, i)
		}
		if _, dup := seen[id]; dup {
			return Job{}, nil, fmt.Errorf("jobs.%s.tasks: duplicate id %q", jobID, id)
		}
		seen[id] = struct{}{}
		tasks = append(tasks, TaskConfig(payload))
		roles = append(roles, rolesFromTask(payload)...)
	}

	job := Job{
		Repeatable: repeatable,
		InstanceID: instanceID,
		Label:      label,
		Tasks:      tasks,
	}
	if startLabel, ok := optionalString(m, "startLabel"); ok {
		job.StartLabel = startLabel
	}
	if swNode := child(m, "startsWhen"); swNode != nil {
		sw, err := parseStartsWhen(jobID, swNode)
		if err != nil {
			return Job{}, nil, err
		}
		job.StartsWhen = sw
	}
	return job, roles, nil
}

func parseStartsWhen(jobID string, node *yaml.Node) (*StartsWhen, error) {
	m := mappingContent(node)
	if m == nil {
		return nil, fmt.Errorf("jobs.%s.startsWhen must be a mapping", jobID)
	}
	path, err := requiredString(m, "substep")
	if err != nil {
		return nil, fmt.Errorf("jobs.%s.startsWhen.task (from substep): %w", jobID, err)
	}
	ref, err := ParseTaskPath(path)
	if err != nil {
		return nil, fmt.Errorf("jobs.%s.startsWhen.task must be \"<jobId>/<taskId>\"", jobID)
	}
	quantifier := "some"
	if q, ok := optionalString(m, "quantifier"); ok {
		if q != "some" && q != "all" {
			return nil, fmt.Errorf("jobs.%s.startsWhen.quantifier must be \"some\" or \"all\"", jobID)
		}
		quantifier = q
	}
	return &StartsWhen{Task: StartsWhenRef{JobID: ref.JobID, TaskID: ref.TaskID}, Quantifier: quantifier}, nil
}

func rolesFromTask(payload any) []string {
	m := taskPayload(payload)
	if m == nil {
		return nil
	}
	raw, ok := m["roles"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		s, ok := item.(string)
		if !ok || s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

func rolesAllowedForTask(cfg OrgConfig, jobID, taskID string) []string {
	jobVal, ok := cfg.Jobs.Get(jobID)
	if !ok {
		return nil
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return nil
	}
	task, ok := TaskByID(job.Tasks, taskID)
	if !ok {
		return nil
	}
	return rolesFromTask(task)
}

func mappingContent(n *yaml.Node) *yaml.Node {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.DocumentNode && len(n.Content) > 0 {
		return mappingContent(n.Content[0])
	}
	if n.Kind == yaml.MappingNode {
		return n
	}
	return nil
}

func child(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil {
		return nil
	}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

func requiredString(mapping *yaml.Node, key string) (string, error) {
	n := child(mapping, key)
	if n == nil || n.Kind != yaml.ScalarNode {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}
	if n.Value == "" {
		return "", fmt.Errorf("%s must be a non-empty string", key)
	}
	return n.Value, nil
}

func optionalString(mapping *yaml.Node, key string) (string, bool) {
	n := child(mapping, key)
	if n == nil || n.Kind != yaml.ScalarNode || n.Value == "" {
		return "", false
	}
	return n.Value, true
}

func requiredBool(mapping *yaml.Node, key string) (bool, error) {
	n := child(mapping, key)
	if n == nil || n.Kind != yaml.ScalarNode {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	switch n.Value {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean", key)
	}
}
