package org

import (
	"fmt"
	"sort"
	"strings"
)

const (
	taskRecapModeEntity     = "entity"
	taskRecapModeSubmission = "submission"
)

// TaskRecapTable is the server-projected overview table for one Submission address.
type TaskRecapTable struct {
	Mode    string         `json:"mode"` // "entity" | "submission"
	Columns []TaskRecapCol `json:"columns"`
	Rows    []TaskRecapRow `json:"rows"`
}

// TaskRecapCol is one table column.
type TaskRecapCol struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}

// TaskRecapRow is one overview row (entity id or submission timestamp as Key).
type TaskRecapRow struct {
	Key          string         `json:"key"`
	Cells        map[string]any `json:"cells"`
	SubmissionID string         `json:"submissionId"` // Submission.Timestamp of the row's latest touch
	Timestamp    string         `json:"timestamp"`
	UserID       string         `json:"userId"`
	UserName     string         `json:"userName"`
	Links        []HttpLinkItem `json:"links,omitempty"`
}

// ProjectTaskRecap builds the overview table for addr from Active Config + Domain.
// Entity mode when the Task authors recap:; otherwise submissions mode.
// Caller attaches LinkStore links; empty matched submissions yield an empty table
// (LoadTaskRecap maps that to ErrNotFound).
func ProjectTaskRecap(cfg OrgConfig, domain Domain, addr SubmissionAddress) (TaskRecapTable, error) {
	addr = addr.normalized()
	if !addr.valid() {
		return TaskRecapTable{}, fmt.Errorf("org: submission address required")
	}

	jobVal, ok := cfg.Jobs.Get(addr.JobID)
	if !ok {
		return TaskRecapTable{}, fmt.Errorf("org: unknown job %q", addr.JobID)
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return TaskRecapTable{}, fmt.Errorf("org: job %q is not a Job", addr.JobID)
	}
	leaf, ok := TaskByID(job.Tasks, addr.TaskID)
	if !ok {
		return TaskRecapTable{}, fmt.Errorf("org: unknown task %q", addr.TaskID)
	}

	matched := submissionsMatching(domain.Submissions, addr)
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Timestamp < matched[j].Timestamp
	})

	spec, err := taskRecap(leaf)
	if err != nil {
		return TaskRecapTable{}, fmt.Errorf("org: task %q: %w", addr.TaskID, err)
	}
	if spec != nil {
		return projectEntityRecap(*spec, domain, matched), nil
	}
	return projectSubmissionRecap(leaf, matched), nil
}

func projectEntityRecap(spec RecapSpec, domain Domain, matched []Submission) TaskRecapTable {
	cols := make([]TaskRecapCol, 0, len(spec.Columns))
	for _, c := range spec.Columns {
		title := c.Title
		if title == "" {
			title = c.Field
		}
		cols = append(cols, TaskRecapCol{Key: c.Field, Title: title})
	}

	type agg struct {
		latest    Submission
		seenTouch bool
	}
	byID := map[string]*agg{}
	order := make([]string, 0)

	for _, sub := range matched {
		for _, touch := range sub.EntitiesTouched {
			if touch.Entity != spec.Entity {
				continue
			}
			id := strings.TrimSpace(touch.ID)
			if id == "" {
				continue
			}
			a, ok := byID[id]
			if !ok {
				a = &agg{}
				byID[id] = a
				order = append(order, id)
			}
			if !a.seenTouch || sub.Timestamp >= a.latest.Timestamp {
				a.latest = sub
				a.seenTouch = true
			}
		}
	}

	kind := domain.Kind(spec.Entity)
	rows := make([]TaskRecapRow, 0, len(order))
	for _, id := range order {
		facts, ok := kind[id]
		if !ok {
			continue
		}
		a := byID[id]
		cells := make(map[string]any, len(spec.Columns))
		for _, c := range spec.Columns {
			if v, has := facts[c.Field]; has {
				cells[c.Field] = v
			}
		}
		rows = append(rows, TaskRecapRow{
			Key:          id,
			Cells:        cells,
			SubmissionID: a.latest.Timestamp,
			Timestamp:    a.latest.Timestamp,
			UserID:       a.latest.UserID,
			UserName:     submissionUserName(a.latest),
		})
	}

	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Timestamp != rows[j].Timestamp {
			return rows[i].Timestamp < rows[j].Timestamp
		}
		return rows[i].Key < rows[j].Key
	})

	return TaskRecapTable{Mode: taskRecapModeEntity, Columns: cols, Rows: rows}
}

func projectSubmissionRecap(leaf TaskConfig, matched []Submission) TaskRecapTable {
	payload := taskPayload(leaf)
	var schema map[string]any
	if payload != nil {
		schema, _ = payload["schema"].(map[string]any)
	}
	titles := schemaPropertyTitles(schema)

	keyOrder := make([]string, 0)
	seen := map[string]struct{}{}
	for _, sub := range matched {
		keys := make([]string, 0, len(sub.Facts))
		for k := range sub.Facts {
			if k == "" {
				continue
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			keyOrder = append(keyOrder, k)
		}
	}

	cols := make([]TaskRecapCol, 0, len(keyOrder))
	for _, k := range keyOrder {
		title := k
		if titles != nil {
			if t, ok := titles[k]; ok && t != "" {
				title = t
			}
		}
		cols = append(cols, TaskRecapCol{Key: k, Title: title})
	}

	rows := make([]TaskRecapRow, 0, len(matched))
	for _, sub := range matched {
		rows = append(rows, TaskRecapRow{
			Key:          sub.Timestamp,
			Cells:        cloneFacts(sub.Facts),
			SubmissionID: sub.Timestamp,
			Timestamp:    sub.Timestamp,
			UserID:       sub.UserID,
			UserName:     submissionUserName(sub),
		})
	}

	return TaskRecapTable{Mode: taskRecapModeSubmission, Columns: cols, Rows: rows}
}

func submissionUserName(sub Submission) string {
	name := strings.TrimSpace(sub.UserName)
	if name == "" {
		return strings.TrimSpace(sub.UserID)
	}
	return name
}
