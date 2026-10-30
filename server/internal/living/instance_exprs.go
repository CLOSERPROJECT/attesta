package living

import "fmt"

type InstanceExprError struct {
	Kind    string
	Message string
}

func (e InstanceExprError) Error() string {
	return fmt.Sprintf("living: instance %s: %s", e.Kind, e.Message)
}

// EvaluateJobInstanceExprs evaluates Job instanceId / label against facts + transform bindings.
func EvaluateJobInstanceExprs(job Job, facts map[string]any, bindings ExprBindings) (id, label string, err error) {
	idRaw, err := evaluateExpr(job.InstanceID, facts, bindings)
	if err != nil {
		return "", "", InstanceExprError{Kind: "jsonata", Message: err.Error()}
	}
	id, err = requireInstanceString(idRaw, "instanceId")
	if err != nil {
		return "", "", err
	}
	labelRaw, err := evaluateExpr(job.Label, facts, bindings)
	if err != nil {
		return "", "", InstanceExprError{Kind: "jsonata", Message: err.Error()}
	}
	label, err = requireInstanceString(labelRaw, "label")
	if err != nil {
		return "", "", err
	}
	return id, label, nil
}

func requireInstanceString(value any, name string) (string, error) {
	s, ok := value.(string)
	if !ok || s == "" {
		return "", InstanceExprError{Kind: "type", Message: name + " must evaluate to a non-empty string"}
	}
	return s, nil
}

func hasJobInstance(domain Domain, jobID, id string) bool {
	for _, inst := range domain.JobInstances {
		if inst.JobID == jobID && inst.ID == id {
			return true
		}
	}
	return false
}
