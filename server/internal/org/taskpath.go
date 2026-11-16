package org

import (
	"fmt"
	"strings"
)

// TaskPathRef is flat jobId/taskId.
type TaskPathRef struct {
	JobID  string
	TaskID string
}

func TaskPath(jobID, taskID string) string {
	return jobID + "/" + taskID
}

func (r TaskPathRef) String() string {
	return TaskPath(r.JobID, r.TaskID)
}

func ParseTaskPath(path string) (TaskPathRef, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return TaskPathRef{}, fmt.Errorf("org: task path must be \"<jobId>/<taskId>\"")
	}
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if p == "" {
			return TaskPathRef{}, fmt.Errorf("org: task path segments must be non-empty")
		}
	}
	if len(parts) != 2 {
		return TaskPathRef{}, fmt.Errorf("org: task path must be \"<jobId>/<taskId>\"")
	}
	return TaskPathRef{JobID: parts[0], TaskID: parts[1]}, nil
}
