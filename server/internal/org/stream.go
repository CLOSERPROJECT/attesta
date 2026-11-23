package org

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/blues/jsonata-go"
)

// Reserved /app path segments that must not be used as stream keys.
var reservedStreamKeys = map[string]struct{}{
	"i":   {},
	"new": {},
}

var streamKeySlug = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// StreamConfig is the Org Config stream section for /my presentation and
// instance scoping (ADR-0014).
type StreamConfig struct {
	Key             string         `json:"key" yaml:"key"`
	Title           string         `json:"title" yaml:"title"`
	Description     string         `json:"description,omitempty" yaml:"description,omitempty"`
	CategorySlug    string         `json:"categorySlug,omitempty" yaml:"categorySlug,omitempty"`
	SubCategorySlug string         `json:"subCategorySlug,omitempty" yaml:"subCategorySlug,omitempty"`
	Instance        StreamInstance `json:"instance" yaml:"instance"`
	Steps           []StreamStep   `json:"steps" yaml:"steps"`
}

// streamForAnchor builds a minimal required stream section for synthetic Configs.
func streamForAnchor(key, title, job string) *StreamConfig {
	return &StreamConfig{
		Key:      key,
		Title:    title,
		Instance: StreamInstance{Job: job},
		Steps:    []StreamStep{{Job: job}},
	}
}

// StreamInstance names the anchor Job whose instances are Org stream instances.
type StreamInstance struct {
	Job string `json:"job" yaml:"job"`
}

// StreamStep is one timeline step: a Job, optionally repeated via each.
type StreamStep struct {
	Job  string      `json:"job" yaml:"job"`
	Each *StreamEach `json:"each,omitempty" yaml:"each,omitempty"`
}

// StreamEach selects Domain entities that become repeated step rows.
type StreamEach struct {
	Entity string `json:"entity" yaml:"entity"`
	Where  string `json:"where" yaml:"where"`
}

// ValidateStreamConfig checks the required stream section (§3).
func ValidateStreamConfig(cfg OrgConfig) error {
	if cfg.Stream == nil {
		return fmt.Errorf("org: stream is required")
	}
	s := cfg.Stream
	key := strings.TrimSpace(s.Key)
	if key == "" {
		return fmt.Errorf("org: stream.key is required")
	}
	if !streamKeySlug.MatchString(key) {
		return fmt.Errorf("org: stream.key %q must be a slug", key)
	}
	if _, reserved := reservedStreamKeys[key]; reserved {
		return fmt.Errorf("org: stream.key %q is reserved", key)
	}
	if strings.TrimSpace(s.Title) == "" {
		return fmt.Errorf("org: stream.title is required")
	}
	anchor := strings.TrimSpace(s.Instance.Job)
	if anchor == "" {
		return fmt.Errorf("org: stream.instance.job is required")
	}

	if len(s.Steps) == 0 {
		return fmt.Errorf("org: stream.steps is required")
	}

	anchorInSteps := false
	for i, step := range s.Steps {
		jobID := strings.TrimSpace(step.Job)
		if jobID == "" {
			return fmt.Errorf("org: stream.steps[%d].job is required", i)
		}
		jobVal, ok := cfg.Jobs.Get(jobID)
		if !ok {
			return fmt.Errorf("org: stream.steps[%d].job %q not found", i, jobID)
		}
		job, ok := AsJob(jobVal)
		if !ok {
			return fmt.Errorf("org: stream.steps[%d].job %q not found", i, jobID)
		}
		if jobID == anchor {
			anchorInSteps = true
			if !job.Repeatable {
				return fmt.Errorf("org: stream.instance.job %q must be repeatable", anchor)
			}
		}
		if step.Each != nil {
			if err := validateStreamEach(cfg, i, step.Each); err != nil {
				return err
			}
		}
	}
	if !anchorInSteps {
		return fmt.Errorf("org: stream.instance.job %q must appear in stream.steps", anchor)
	}

	anchorVal, ok := cfg.Jobs.Get(anchor)
	if !ok {
		return fmt.Errorf("org: stream.instance.job %q not found", anchor)
	}
	anchorJob, ok := AsJob(anchorVal)
	if !ok {
		return fmt.Errorf("org: stream.instance.job %q not found", anchor)
	}
	if !anchorJob.Repeatable {
		return fmt.Errorf("org: stream.instance.job %q must be repeatable", anchor)
	}
	return nil
}

func validateStreamEach(cfg OrgConfig, stepIdx int, each *StreamEach) error {
	entity := strings.TrimSpace(each.Entity)
	if entity == "" {
		return fmt.Errorf("org: stream.steps[%d].each.entity is required", stepIdx)
	}
	kinds := EntityKindsFromConfig(cfg)
	if kinds == nil {
		return fmt.Errorf("org: stream.steps[%d].each.entity %q not in domain.entities", stepIdx, entity)
	}
	if _, ok := kinds[entity]; !ok {
		return fmt.Errorf("org: stream.steps[%d].each.entity %q not in domain.entities", stepIdx, entity)
	}
	where := strings.TrimSpace(each.Where)
	if where == "" {
		return fmt.Errorf("org: stream.steps[%d].each.where is required", stepIdx)
	}
	if _, err := jsonata.Compile(where); err != nil {
		return fmt.Errorf("org: stream.steps[%d].each.where: %w", stepIdx, err)
	}
	return nil
}

func compileCheckExpr(path, expression string) error {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return fmt.Errorf("%s: empty expression", path)
	}
	if _, err := jsonata.Compile(expression); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}
