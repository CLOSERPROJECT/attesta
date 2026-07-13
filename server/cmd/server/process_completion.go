// process_completion.go
package main

import (
	"errors"
	"time"
)

var (
	ErrProgressUpdate = errors.New("process: progress update failed")
	ErrNotarization   = errors.New("process: notarization failed")
)

type ProcessService struct {
	store Store
	now   func() time.Time
}

type CompleteSubstepCmd struct {
	Process     *Process
	WorkflowKey string
	SubstepID   string
	Substep     WorkflowSub
	Actor       Actor
	Payload     map[string]interface{}
	Config      RuntimeConfig
	Now         time.Time
}
