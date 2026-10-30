package living

import (
	"fmt"
)

type transformMintOp struct {
	Entity EntityType
	As     string
	ID     any
	Set    map[string]any
}

type transformMintManyOp struct {
	Entity EntityType
	Count  any
	ID     any
	Set    map[string]any
}

type transformPatchEachOp struct {
	Entity EntityType
	Set    map[string]any
}

type transformPatchOp struct {
	Entity EntityType
	ID     any
	Set    map[string]any
}

type transformOp interface {
	transformOpTag()
}

func (transformMintOp) transformOpTag()      {}
func (transformMintManyOp) transformOpTag()  {}
func (transformPatchEachOp) transformOpTag() {}
func (transformPatchOp) transformOpTag()     {}

type TransformError struct {
	Kind    string
	Message string
	Op      string
}

func (e TransformError) Error() string {
	if e.Kind == "unsupported-op" {
		return "unsupported op: " + e.Op
	}
	if e.Message == "" {
		return "living: transform " + e.Kind
	}
	return "living: transform " + e.Kind + ": " + e.Message
}

type RunTransformInput struct {
	JobID         string
	TaskID        string
	JobInstanceID string
	UserID        string
	Facts         map[string]any
	Transform     any
	Allowed       map[string]struct{} // entity kinds from config domain.entities; empty = unrestricted
}

type RunTransformResult struct {
	Request  MutateRequest
	Bindings ExprBindings
}

func parseTransformOps(transform any, allowed map[string]struct{}) ([]transformOp, error) {
	rawList, ok := transform.([]any)
	if !ok {
		return nil, TransformError{Kind: "invalid-transform", Message: "transform must be an array of ops"}
	}
	ops := make([]transformOp, 0, len(rawList))
	for _, raw := range rawList {
		m, ok := raw.(map[string]any)
		if !ok {
			return nil, TransformError{Kind: "invalid-transform", Message: "each op needs a string op field"}
		}
		opName, _ := m["op"].(string)
		if opName == "" {
			return nil, TransformError{Kind: "invalid-transform", Message: "each op needs a string op field"}
		}
		switch opName {
		case "mint":
			entity, set, err := requireEntitySet(m, allowed, "mint")
			if err != nil {
				return nil, err
			}
			as, _ := m["as"].(string)
			ops = append(ops, transformMintOp{Entity: entity, As: as, ID: m["id"], Set: set})
		case "mintMany":
			entity, set, err := requireEntitySet(m, allowed, "mintMany")
			if err != nil {
				return nil, err
			}
			ops = append(ops, transformMintManyOp{Entity: entity, Count: m["count"], ID: m["id"], Set: set})
		case "patchEach":
			entity, set, err := requireEntitySet(m, allowed, "patchEach")
			if err != nil {
				return nil, err
			}
			ops = append(ops, transformPatchEachOp{Entity: entity, Set: set})
		case "patch":
			entity, set, err := requireEntitySet(m, allowed, "patch")
			if err != nil {
				return nil, err
			}
			if _, has := m["id"]; !has {
				return nil, TransformError{Kind: "invalid-transform", Message: "patch requires entity + id + set"}
			}
			ops = append(ops, transformPatchOp{Entity: entity, ID: m["id"], Set: set})
		default:
			return nil, TransformError{Kind: "unsupported-op", Op: opName}
		}
	}
	return ops, nil
}

func requireEntitySet(m map[string]any, allowed map[string]struct{}, op string) (EntityType, map[string]any, error) {
	entityRaw, _ := m["entity"].(string)
	if entityRaw == "" {
		return "", nil, TransformError{Kind: "invalid-transform", Message: op + " requires entity + set"}
	}
	if len(allowed) > 0 {
		if _, ok := allowed[entityRaw]; !ok {
			return "", nil, TransformError{
				Kind:    "invalid-transform",
				Message: fmt.Sprintf("%s entity %q not in domain.entities", op, entityRaw),
			}
		}
	}
	set, ok := m["set"].(map[string]any)
	if !ok {
		return "", nil, TransformError{Kind: "invalid-transform", Message: op + " requires entity + set"}
	}
	return EntityType(entityRaw), set, nil
}

func domainBindings(domain Domain, jobInstanceID string) ExprBindings {
	entityMaps := domain.EntityMaps()
	domainBag := make(map[string]any, len(entityMaps)+1)
	for kind, m := range entityMaps {
		domainBag[kind] = entityMapToAny(m)
	}
	subs := make([]any, len(domain.Submissions))
	for i, s := range domain.Submissions {
		subs[i] = s
	}
	domainBag["submissions"] = subs
	bindings := ExprBindings{"domain": domainBag}
	if jobInstanceID != "" {
		bindings["instance"] = map[string]any{"id": jobInstanceID}
	}
	return bindings
}

func entityMapToAny(m EntityMap) map[string]any {
	out := make(map[string]any, len(m))
	for id, facts := range m {
		out[id] = facts
	}
	return out
}

// evalScalarResult distinguishes JSONata undefined (omit) from JSON null (unset patch).
type evalScalarResult struct {
	Value   any
	Defined bool
}

func evalScalar(raw any, facts map[string]any, bindings ExprBindings) (evalScalarResult, error) {
	s, ok := raw.(string)
	if !ok {
		return evalScalarResult{Value: raw, Defined: true}, nil
	}
	// jsonata-go treats a nil $value binding as ErrUndefined; keep null for patch-unset.
	if s == "$value" {
		if v, ok := bindings["value"]; ok {
			return evalScalarResult{Value: v, Defined: true}, nil
		}
		return evalScalarResult{}, nil
	}
	value, err := evaluateExpr(s, facts, bindings)
	if err != nil {
		return evalScalarResult{}, err
	}
	if value == nil {
		// evaluateExpr maps ErrUndefined → (nil, nil); also literal null.
		// Re-check: missing path is undefined (omit). We cannot tell apart here when
		// the library collapsed both — omit only when the var path is $value.* on nil.
		return evalScalarResult{}, nil
	}
	return evalScalarResult{Value: value, Defined: true}, nil
}

func evalSet(set map[string]any, facts map[string]any, bindings ExprBindings) (map[string]any, error) {
	out := make(map[string]any, len(set))
	for key, raw := range set {
		got, err := evalScalar(raw, facts, bindings)
		if err != nil {
			return nil, TransformError{Kind: "jsonata", Message: err.Error()}
		}
		if !got.Defined {
			continue
		}
		out[key] = got.Value
	}
	return out, nil
}

func requireNonEmptyString(value any, label string) (string, error) {
	s, ok := value.(string)
	if !ok || s == "" {
		return "", TransformError{Kind: "type", Message: label + " must evaluate to a non-empty string"}
	}
	return s, nil
}

func requireCount(value any) (int, error) {
	n, ok := intFromAny(value)
	if !ok || n < 0 {
		return 0, TransformError{Kind: "type", Message: "mintMany count must evaluate to a non-negative integer"}
	}
	return n, nil
}

func applyMintLocally(working *Domain, entity EntityType, id string, set map[string]any) {
	m, err := entityMap(working, entity)
	if err != nil {
		return
	}
	(*m)[id] = cloneFacts(set)
}

func applyPatchLocally(working *Domain, entity EntityType, id string, set map[string]any) {
	m, err := entityMap(working, entity)
	if err != nil {
		return
	}
	existing, ok := (*m)[id]
	if !ok {
		return
	}
	(*m)[id] = applyFactsPatch(existing, set)
}

// RunTransform evaluates authored mint/mintMany/patch/patchEach ops into a MutateRequest.
func RunTransform(domain Domain, input RunTransformInput) (RunTransformResult, error) {
	parsed, err := parseTransformOps(input.Transform, input.Allowed)
	if err != nil {
		return RunTransformResult{}, err
	}

	working := cloneDomain(domain)
	aliases := ExprBindings{}
	ops := make([]MutateOp, 0)

	for _, op := range parsed {
		bindings := mergeBindings(domainBindings(working, input.JobInstanceID), aliases)

		switch o := op.(type) {
		case transformPatchEachOp:
			for id, cell := range input.Facts {
				cellBindings := mergeBindings(bindings, ExprBindings{"value": cell})
				set, err := evalSet(o.Set, input.Facts, cellBindings)
				if err != nil {
					return RunTransformResult{}, err
				}
				ops = append(ops, PatchOp{Entity: o.Entity, ID: id, Set: set})
				applyPatchLocally(&working, o.Entity, id, set)
			}
		case transformPatchOp:
			idGot, err := evalScalar(o.ID, input.Facts, bindings)
			if err != nil {
				return RunTransformResult{}, TransformError{Kind: "jsonata", Message: err.Error()}
			}
			id, err := requireNonEmptyString(idGot.Value, "patch id")
			if err != nil {
				return RunTransformResult{}, err
			}
			set, err := evalSet(o.Set, input.Facts, bindings)
			if err != nil {
				return RunTransformResult{}, err
			}
			ops = append(ops, PatchOp{Entity: o.Entity, ID: id, Set: set})
			applyPatchLocally(&working, o.Entity, id, set)
		case transformMintOp:
			idGot, err := evalScalar(o.ID, input.Facts, bindings)
			if err != nil {
				return RunTransformResult{}, TransformError{Kind: "jsonata", Message: err.Error()}
			}
			id, err := requireNonEmptyString(idGot.Value, "mint id")
			if err != nil {
				return RunTransformResult{}, err
			}
			set, err := evalSet(o.Set, input.Facts, bindings)
			if err != nil {
				return RunTransformResult{}, err
			}
			ops = append(ops, MintOp{Entity: o.Entity, ID: id, Set: set})
			applyMintLocally(&working, o.Entity, id, set)
			if o.As != "" {
				alias := cloneFacts(set)
				alias["id"] = id
				aliases[o.As] = alias
			}
		case transformMintManyOp:
			countGot, err := evalScalar(o.Count, input.Facts, bindings)
			if err != nil {
				return RunTransformResult{}, TransformError{Kind: "jsonata", Message: err.Error()}
			}
			count, err := requireCount(countGot.Value)
			if err != nil {
				return RunTransformResult{}, err
			}
			for index := 0; index < count; index++ {
				// Refresh $domain each iteration so count/id exprs see prior mints.
				loopBindings := mergeBindings(domainBindings(working, input.JobInstanceID), aliases)
				loopBindings["index"] = index
				idGot, err := evalScalar(o.ID, input.Facts, loopBindings)
				if err != nil {
					return RunTransformResult{}, TransformError{Kind: "jsonata", Message: err.Error()}
				}
				id, err := requireNonEmptyString(idGot.Value, "mintMany id")
				if err != nil {
					return RunTransformResult{}, err
				}
				set, err := evalSet(o.Set, input.Facts, loopBindings)
				if err != nil {
					return RunTransformResult{}, err
				}
				ops = append(ops, MintOp{Entity: o.Entity, ID: id, Set: set})
				applyMintLocally(&working, o.Entity, id, set)
			}
		default:
			return RunTransformResult{}, fmt.Errorf("living: unknown transform op %T", op)
		}
	}

	return RunTransformResult{
		Request: MutateRequest{
			JobID:         input.JobID,
			TaskID:        input.TaskID,
			JobInstanceID: input.JobInstanceID,
			Facts:         cloneFacts(input.Facts),
			Ops:           ops,
			UserID:        input.UserID,
		},
		Bindings: mergeBindings(domainBindings(working, input.JobInstanceID), aliases),
	}, nil
}

func mergeBindings(base, extra ExprBindings) ExprBindings {
	out := make(ExprBindings, len(base)+len(extra))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range extra {
		out[k] = v
	}
	return out
}

// EntityKindsFromConfig returns domain.entities keys; nil means unrestricted.
func EntityKindsFromConfig(cfg OrgConfig) map[string]struct{} {
	if cfg.Domain == nil {
		return nil
	}
	raw, ok := cfg.Domain["entities"]
	if !ok {
		return nil
	}
	entities, ok := raw.(map[string]any)
	if !ok || len(entities) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(entities))
	for kind := range entities {
		out[kind] = struct{}{}
	}
	return out
}
