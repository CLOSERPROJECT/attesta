package living

import (
	"errors"
	"fmt"
	"strings"

	"github.com/blues/jsonata-go"
)

// ExprBindings are JSONata variables ($name). `$` is the evaluation input separately.
type ExprBindings map[string]any

type exprError struct {
	Kind    string
	Message string
}

func (e exprError) Error() string {
	return fmt.Sprintf("living: expr %s: %s", e.Kind, e.Message)
}

// evaluateExpr runs a JSONata expression. ErrUndefined → (nil, nil) so callers can omit.
func evaluateExpr(expression string, input any, bindings ExprBindings) (any, error) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return nil, exprError{Kind: "compile", Message: "empty expression"}
	}
	expr, err := jsonata.Compile(expression)
	if err != nil {
		return nil, exprError{Kind: "compile", Message: err.Error()}
	}
	if len(bindings) > 0 {
		vars := make(map[string]any, len(bindings))
		for k, v := range bindings {
			vars[k] = v
		}
		if err := expr.RegisterVars(vars); err != nil {
			return nil, exprError{Kind: "compile", Message: err.Error()}
		}
	}
	value, err := expr.Eval(input)
	if err != nil {
		if errors.Is(err, jsonata.ErrUndefined) {
			return nil, nil
		}
		return nil, exprError{Kind: "evaluate", Message: err.Error()}
	}
	return value, nil
}
