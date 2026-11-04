package living

import (
	"fmt"
	"strings"
)

// TaskPathRef is flat jobId/taskId or nested jobId/groupId/groupInstanceId/taskId.
type TaskPathRef struct {
	JobID           string
	GroupID         string
	GroupInstanceID string
	TaskID          string
}

func (r TaskPathRef) Nested() bool {
	return r.GroupID != "" || r.GroupInstanceID != ""
}

func TaskPath(jobID, taskID string) string {
	return jobID + "/" + taskID
}

func GroupTaskPath(jobID, groupID, groupInstanceID, taskID string) string {
	return jobID + "/" + groupID + "/" + groupInstanceID + "/" + taskID
}

func (r TaskPathRef) String() string {
	if r.Nested() {
		return GroupTaskPath(r.JobID, r.GroupID, r.GroupInstanceID, r.TaskID)
	}
	return TaskPath(r.JobID, r.TaskID)
}

// ProvisionalGroupInstanceID is the nested URL sentinel when spawn-minting.
// Same SPA word as Jobs (`new`); Jobs then normalize to empty / reject `_new` on
// Commit — groups must not invert that with a `_new` product sentinel.
const ProvisionalGroupInstanceID = "new"

func ParseTaskPath(path string) (TaskPathRef, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return TaskPathRef{}, fmt.Errorf("living: task path must be \"<jobId>/<taskId>\" or \"<jobId>/<groupId>/<groupInstanceId>/<taskId>\"")
	}
	parts := strings.Split(path, "/")
	for _, p := range parts {
		if p == "" {
			return TaskPathRef{}, fmt.Errorf("living: task path segments must be non-empty")
		}
	}
	switch len(parts) {
	case 2:
		return TaskPathRef{JobID: parts[0], TaskID: parts[1]}, nil
	case 4:
		return TaskPathRef{
			JobID:           parts[0],
			GroupID:         parts[1],
			GroupInstanceID: parts[2],
			TaskID:          parts[3],
		}, nil
	default:
		return TaskPathRef{}, fmt.Errorf("living: task path must be \"<jobId>/<taskId>\" or \"<jobId>/<groupId>/<groupInstanceId>/<taskId>\"")
	}
}
