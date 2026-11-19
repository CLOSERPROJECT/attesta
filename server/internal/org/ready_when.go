package org

import (
	"strings"
)

// readyWhenFromTask returns the Task's optional readyWhen expression (JSONata).
func readyWhenFromTask(task map[string]any) any {
	if task == nil {
		return nil
	}
	return task["readyWhen"]
}

// readyWhenOkForCommit evaluates Task readyWhen over Domain into a compact boolean.
// Omit / null / blank ⇒ true (experiment parity). Invalid or eval failure ⇒ false (fail closed).
func readyWhenOkForCommit(domain Domain, task map[string]any, jobInstanceID string, catalogs map[string]any, stream ...*streamBinding) bool {
	var streamRef *streamBinding
	if len(stream) > 0 {
		streamRef = stream[0]
	}
	ok, err := evaluateReadyWhen(domain, readyWhenFromTask(task), jobInstanceID, catalogs, streamRef)
	if err != nil {
		return false
	}
	return ok
}

// evaluateReadyWhen runs an optional readyWhen JSONata boolean over Domain ($domain).
func evaluateReadyWhen(domain Domain, expression any, jobInstanceID string, catalogs map[string]any, stream ...*streamBinding) (bool, error) {
	if expression == nil {
		return true, nil
	}
	s, ok := expression.(string)
	if !ok {
		return false, exprError{Kind: "invalid-readyWhen", Message: "readyWhen must be a non-empty JSONata string when present"}
	}
	if strings.TrimSpace(s) == "" {
		return true, nil
	}
	var streamRef *streamBinding
	if len(stream) > 0 {
		streamRef = stream[0]
	}
	value, err := evaluateExpr(s, nil, domainBindingsWith(domain, jobInstanceID, catalogs, "", streamRef))
	if err != nil {
		return false, err
	}
	return truthy(value), nil
}
