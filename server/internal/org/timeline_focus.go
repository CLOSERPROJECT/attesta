package org

import (
	"fmt"
	"strings"
)

type timelineFocusKind uint8

const (
	timelineFocusBoot timelineFocusKind = iota
	timelineFocusInstance
)

// TimelineFocus selects Boot (stream-only) vs Instance (stream + instance) timeline mode.
type TimelineFocus struct {
	kind       timelineFocusKind
	instanceID string
}

// TimelineBoot is the not-yet-instanced Play / boot timeline focus.
func TimelineBoot() TimelineFocus {
	return TimelineFocus{kind: timelineFocusBoot}
}

// TimelineInstance is focus on one stream instance. Blank id returns an error.
func TimelineInstance(id string) (TimelineFocus, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return TimelineFocus{}, fmt.Errorf("timeline instance id required")
	}
	return TimelineFocus{kind: timelineFocusInstance, instanceID: id}, nil
}

// ResolveTimelineFocus maps the HTTP instance query: blank ⇒ Boot; non-blank ⇒ Instance.
func ResolveTimelineFocus(instanceQuery string) (TimelineFocus, error) {
	instanceQuery = strings.TrimSpace(instanceQuery)
	if instanceQuery == "" {
		return TimelineBoot(), nil
	}
	return TimelineInstance(instanceQuery)
}

// IsBoot reports whether this focus is Boot (no instance yet).
func (f TimelineFocus) IsBoot() bool {
	return f.kind == timelineFocusBoot
}

// InstanceID returns the instance id when focus is Instance.
func (f TimelineFocus) InstanceID() (id string, ok bool) {
	if f.kind != timelineFocusInstance {
		return "", false
	}
	return f.instanceID, true
}
