package living

import "context"

// AuthorizeCommit carries compact attrs only (Cerbos check payload).
type AuthorizeCommit struct {
	Operator      OperatorIdentity
	Org           string
	TaskPath      string
	JobID         string
	TaskID        string
	JobInstanceID string
	RolesAllowed  []string
}

type Authorizer interface {
	AllowCommit(ctx context.Context, q AuthorizeCommit) (bool, error)
}

type AllowAllAuthorizer struct{}

func (AllowAllAuthorizer) AllowCommit(context.Context, AuthorizeCommit) (bool, error) {
	return true, nil
}

type DenyAllAuthorizer struct{}

func (DenyAllAuthorizer) AllowCommit(context.Context, AuthorizeCommit) (bool, error) {
	return false, nil
}
