package org

import "strings"

// AnchorStartRoles returns Config roles allowed on the anchor Job's first Task
// (Play gate for /my Org streams).
func AnchorStartRoles(cfg OrgConfig) []string {
	if cfg.Stream == nil {
		return nil
	}
	anchor := strings.TrimSpace(cfg.Stream.Instance.Job)
	if anchor == "" {
		return nil
	}
	jobVal, ok := cfg.Jobs.Get(anchor)
	if !ok {
		return nil
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return nil
	}
	ids := OrderedTaskIDs(job.Tasks)
	if len(ids) == 0 {
		return nil
	}
	return rolesAllowedForPath(cfg, TaskPathRef{JobID: anchor, TaskID: ids[0]})
}
