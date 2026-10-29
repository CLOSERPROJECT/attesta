package living

import "context"

// AuthorizeCommit is the lean allow/deny query (Cerbos later).
type AuthorizeCommit struct {
	Operator      OperatorIdentity
	TaskPath      string
	JobID         string
	TaskID        string
	JobInstanceID string
}

// Authorizer gates Commit; product path will be Cerbos.
type Authorizer interface {
	AllowCommit(ctx context.Context, q AuthorizeCommit) (bool, error)
}

// AllowAllAuthorizer is the pre-Cerbos / test stub that always allows.
type AllowAllAuthorizer struct{}

func (AllowAllAuthorizer) AllowCommit(context.Context, AuthorizeCommit) (bool, error) {
	return true, nil
}

// DenyAllAuthorizer is a test stub that always denies.
type DenyAllAuthorizer struct{}

func (DenyAllAuthorizer) AllowCommit(context.Context, AuthorizeCommit) (bool, error) {
	return false, nil
}
