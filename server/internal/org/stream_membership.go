package org

import "strings"

// JobInstanceBelongsToStream reports whether a Job instance is in the filtered
// view for Org stream instance S (§3 membership): the anchor instance with id S,
// or an each-step Job whose id is selected by that step's where with $stream.id = S.
func JobInstanceBelongsToStream(cfg OrgConfig, domain Domain, jobID, jobInstanceID, streamInstanceID string) bool {
	if cfg.Stream == nil || streamInstanceID == "" || jobID == "" || jobInstanceID == "" {
		return false
	}
	if jobID == cfg.Stream.Instance.Job {
		return jobInstanceID == streamInstanceID
	}
	for _, step := range cfg.Stream.Steps {
		if step.Job != jobID || step.Each == nil {
			continue
		}
		ids, err := selectStreamEachIDs(cfg, domain, step.Each, streamInstanceID)
		if err != nil {
			return false
		}
		return containsString(ids, jobInstanceID)
	}
	return false
}

func selectStreamEachIDs(cfg OrgConfig, domain Domain, each *StreamEach, streamInstanceID string) ([]string, error) {
	stream := streamBindingFrom(cfg, streamInstanceID)
	return ResolveRead(domain, ReadDef{
		Entity: EntityType(each.Entity),
		Where:  each.Where,
		Key:    "id",
	}, "", "", cfg.Catalogs, stream)
}

func streamBindingFrom(cfg OrgConfig, streamInstanceID string) *streamBinding {
	if cfg.Stream == nil {
		return nil
	}
	key := cfg.Stream.Key
	if key == "" {
		return nil
	}
	return &streamBinding{Key: key, ID: streamInstanceID}
}

// ResolveStreamInstanceID returns the Org stream instance that owns jobInstanceID,
// or "" when unbound (Play / unknown). Anchor Job instances are themselves; each
// rows reverse-lookup via membership.
func ResolveStreamInstanceID(cfg OrgConfig, domain Domain, jobID, jobInstanceID string) string {
	if cfg.Stream == nil || jobID == "" {
		return ""
	}
	if jobID == cfg.Stream.Instance.Job {
		return jobInstanceID
	}
	if jobInstanceID == "" {
		return ""
	}
	for _, anchor := range domain.JobInstances {
		if anchor.JobID != cfg.Stream.Instance.Job {
			continue
		}
		if JobInstanceBelongsToStream(cfg, domain, jobID, jobInstanceID, anchor.ID) {
			return anchor.ID
		}
	}
	return ""
}

// EffectiveStreamInstanceID prefers ResolveStreamInstanceID; when unbound (e.g.
// provisional each-step start with empty jobInstanceID), falls back to the
// explicit stream instance from HTTP / Load.
func EffectiveStreamInstanceID(cfg OrgConfig, domain Domain, jobID, jobInstanceID, explicitStreamInstanceID string) string {
	if id := ResolveStreamInstanceID(cfg, domain, jobID, jobInstanceID); id != "" {
		return id
	}
	return strings.TrimSpace(explicitStreamInstanceID)
}

// StreamInstanceExists reports whether instanceID is an anchor Job instance for the stream.
func StreamInstanceExists(cfg OrgConfig, domain Domain, instanceID string) bool {
	if cfg.Stream == nil || instanceID == "" {
		return false
	}
	anchor := cfg.Stream.Instance.Job
	for _, inst := range domain.JobInstances {
		if inst.JobID == anchor && inst.ID == instanceID {
			return true
		}
	}
	return false
}

// IsAnchorStartTask reports whether jobID/taskID is the anchor Job's first Task (Play).
func IsAnchorStartTask(cfg OrgConfig, jobID, taskID string) bool {
	if cfg.Stream == nil || jobID == "" || taskID == "" {
		return false
	}
	if jobID != cfg.Stream.Instance.Job {
		return false
	}
	jobVal, ok := cfg.Jobs.Get(jobID)
	if !ok {
		return false
	}
	job, ok := AsJob(jobVal)
	if !ok {
		return false
	}
	ordered := OrderedTaskIDs(job.Tasks)
	return len(ordered) > 0 && ordered[0] == taskID
}
