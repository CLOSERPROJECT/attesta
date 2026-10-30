package living

import (
	"fmt"
	"strings"
)

// TaskPathRef is flat jobId/taskId (no nested paths yet).
type TaskPathRef struct {
	JobID  string
	TaskID string
}

func TaskPath(jobID, taskID string) string {
	return jobID + "/" + taskID
}

func ParseTaskPath(path string) (TaskPathRef, error) {
	path = strings.TrimSpace(path)
	slash := strings.IndexByte(path, '/')
	if slash <= 0 || slash == len(path)-1 {
		return TaskPathRef{}, fmt.Errorf("living: task path must be \"<jobId>/<taskId>\"")
	}
	if strings.IndexByte(path[slash+1:], '/') >= 0 {
		return TaskPathRef{}, fmt.Errorf("living: nested task paths are not supported yet")
	}
	return TaskPathRef{JobID: path[:slash], TaskID: path[slash+1:]}, nil
}
