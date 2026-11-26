package org

import (
	"fmt"
)

// ValidateOrgConfig runs whole-config validators used at Config load and Commit prep.
func ValidateOrgConfig(cfg OrgConfig) error {
	if err := ValidateStreamConfig(cfg); err != nil {
		return err
	}
	if err := ValidateDPPConfig(cfg); err != nil {
		return err
	}
	if err := ValidateRecapConfig(cfg); err != nil {
		return err
	}
	return nil
}

// Reserved recap column field: entity map key (TaskRecapRow.key), not a Domain field.
const taskRecapFieldID = "id"

// ValidateRecapConfig walks Jobs/Tasks and checks optional Task recap blocks
// against domain.entities.<kind>.fields (names only). Reserved field "id" is allowed.
func ValidateRecapConfig(cfg OrgConfig) error {
	fieldsByKind, err := EntityFieldsFromConfig(cfg)
	if err != nil {
		return err
	}
	kinds := EntityKindsFromConfig(cfg)

	if cfg.Jobs.Values == nil {
		return nil
	}
	for _, jobID := range cfg.Jobs.Keys {
		job, ok := AsJob(cfg.Jobs.Values[jobID])
		if !ok {
			continue
		}
		for i, t := range job.Tasks {
			prefix := fmt.Sprintf("jobs.%s.tasks[%d]", jobID, i)
			if err := validateTaskRecap(t, prefix, kinds, fieldsByKind); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTaskRecap(
	t TaskConfig,
	prefix string,
	kinds map[string]struct{},
	fieldsByKind map[string][]string,
) error {
	spec, err := taskRecap(t)
	if err != nil {
		return fmt.Errorf("%s %q: %w", prefix, t.ID(), err)
	}
	if spec == nil {
		return nil
	}
	id := t.ID()
	if spec.Entity == "" {
		return fmt.Errorf("%s %q: recap.entity is required", prefix, id)
	}
	if kinds == nil {
		return fmt.Errorf("%s %q: recap.entity %q not in domain.entities", prefix, id, spec.Entity)
	}
	if _, ok := kinds[spec.Entity]; !ok {
		return fmt.Errorf("%s %q: recap.entity %q not in domain.entities", prefix, id, spec.Entity)
	}
	allowed, ok := fieldsByKind[spec.Entity]
	if !ok {
		return fmt.Errorf("%s %q: domain.entities.%s must declare fields for recap", prefix, id, spec.Entity)
	}
	if len(spec.Columns) == 0 {
		return fmt.Errorf("%s %q: recap.columns requires at least one column", prefix, id)
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, f := range allowed {
		allowedSet[f] = struct{}{}
	}
	seen := make(map[string]struct{}, len(spec.Columns))
	for i, col := range spec.Columns {
		if _, dup := seen[col.Field]; dup {
			return fmt.Errorf("%s %q: recap.columns[%d].field %q is duplicate", prefix, id, i, col.Field)
		}
		seen[col.Field] = struct{}{}
		if col.Field == taskRecapFieldID {
			continue
		}
		if _, ok := allowedSet[col.Field]; !ok {
			return fmt.Errorf("%s %q: recap.columns[%d].field %q not in domain.entities.%s.fields", prefix, id, i, col.Field, spec.Entity)
		}
	}
	return nil
}
