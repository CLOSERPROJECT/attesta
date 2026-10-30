package living

import (
	"fmt"
	"regexp"
	"strings"
)

var mapStarDerived = regexp.MustCompile(`^(.+)\.\*\.(.+)$`)

func readChromeRules(payload map[string]any) []ChromeRule {
	raw, _ := payload["chrome"].(map[string]any)
	if len(raw) == 0 {
		return nil
	}
	out := make([]ChromeRule, 0, len(raw))
	for _, id := range sortedStringKeys(raw) {
		entry, _ := raw[id].(map[string]any)
		if entry == nil {
			continue
		}
		kind, _ := entry["kind"].(string)
		after, _ := entry["after"].(string)
		before, _ := entry["before"].(string)
		title, _ := entry["title"].(string)
		switch kind {
		case "valuePrint", "readonly":
			expr, _ := entry["expr"].(string)
			if expr == "" {
				continue
			}
			out = append(out, ChromeRule{
				ID: id, Kind: kind, Expr: expr,
				Title: title, Before: before, After: after,
			})
		case "recap":
			rows := readChromeRecapRows(entry["rows"])
			out = append(out, ChromeRule{
				ID: id, Kind: kind, Title: title, Before: before, After: after, Rows: rows,
			})
		case "finalResults":
			out = append(out, ChromeRule{
				ID: id, Kind: kind, Title: title, Before: before, After: after,
			})
		}
	}
	return out
}

func readChromeRecapRows(raw any) []ChromeRecapRow {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]ChromeRecapRow, 0, len(arr))
	for _, entry := range arr {
		rec, _ := entry.(map[string]any)
		if rec == nil {
			continue
		}
		label, _ := rec["label"].(string)
		expr, _ := rec["expr"].(string)
		if label == "" || expr == "" {
			continue
		}
		row := ChromeRecapRow{Label: label, Expr: expr}
		if tone, ok := rec["tone"].(string); ok && tone != "" {
			row.Tone = tone
		}
		row.Bands = readChromeBands(rec["bands"])
		out = append(out, row)
	}
	return out
}

func readChromeBands(raw any) []ChromeToneBand {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]ChromeToneBand, 0, len(arr))
	for _, entry := range arr {
		rec, _ := entry.(map[string]any)
		if rec == nil {
			continue
		}
		min, ok := floatFromAny(rec["min"])
		if !ok {
			continue
		}
		tone, _ := rec["tone"].(string)
		if tone == "" {
			continue
		}
		out = append(out, ChromeToneBand{Min: min, Tone: tone})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func authoredPropertyKeys(schema map[string]any) map[string]struct{} {
	props, _ := schema["properties"].(map[string]any)
	out := make(map[string]struct{}, len(props))
	for k := range props {
		out[k] = struct{}{}
	}
	return out
}

func chromeTitle(entry map[string]any, id string, rule ChromeRule) string {
	if (rule.Kind == "recap" || rule.Kind == "finalResults") && rule.Title != "" {
		return rule.Title
	}
	if title, ok := entry["title"].(string); ok && title != "" {
		return title
	}
	return id
}

func stringFieldForChrome(kind string) string {
	switch kind {
	case "recap":
		return "recapField"
	case "readonly":
		return "readonlyField"
	case "finalResults":
		return "feragameFinalResultsField"
	default:
		return "valuePrintField"
	}
}

func placeChromeInOrder(order []string, rule ChromeRule) []string {
	for _, id := range order {
		if id == rule.ID {
			return order
		}
	}
	if rule.Before != "" {
		for i, id := range order {
			if id == rule.Before {
				next := make([]string, 0, len(order)+1)
				next = append(next, order[:i]...)
				next = append(next, rule.ID)
				next = append(next, order[i:]...)
				return next
			}
		}
	}
	if rule.After != "" {
		for i, id := range order {
			if id == rule.After {
				next := make([]string, 0, len(order)+1)
				next = append(next, order[:i+1]...)
				next = append(next, rule.ID)
				next = append(next, order[i+1:]...)
				return next
			}
		}
	}
	return append(order, rule.ID)
}

func mergeChromeIntoSchemas(
	authoredSchema map[string]any,
	authoredUi map[string]any,
	chrome []ChromeRule,
	chromeRaw map[string]any,
) (map[string]any, map[string]any) {
	schema := cloneMapJSON(authoredSchema)
	sealProps, _ := schema["properties"].(map[string]any)
	if sealProps == nil {
		sealProps = map[string]any{}
	} else {
		sealProps = cloneMapJSON(sealProps)
	}
	var sealRequired []any
	if req, ok := schema["required"].([]any); ok {
		sealRequired = append([]any(nil), req...)
	}

	for _, c := range chrome {
		entry, _ := chromeRaw[c.ID].(map[string]any)
		if entry == nil {
			entry = map[string]any{}
		}
		sealProps[c.ID] = map[string]any{
			"type":     "string",
			"title":    chromeTitle(entry, c.ID, c),
			"readOnly": true,
		}
	}
	schema["type"] = "object"
	schema["properties"] = sealProps
	schema["required"] = sealRequired

	uiSchema := cloneMapJSON(authoredUi)
	uiOpts, _ := uiSchema["ui:options"].(map[string]any)
	if uiOpts == nil {
		uiOpts = map[string]any{}
	} else {
		uiOpts = cloneMapJSON(uiOpts)
	}
	var order []string
	if rawOrder, ok := uiOpts["order"].([]any); ok {
		for _, v := range rawOrder {
			if s, ok := v.(string); ok {
				order = append(order, s)
			}
		}
	}
	if len(order) == 0 {
		order = sortedStringKeys(sealProps)
	}
	for _, c := range chrome {
		order = placeChromeInOrder(order, c)
	}
	orderAny := make([]any, len(order))
	for i, id := range order {
		orderAny[i] = id
	}
	uiOpts["order"] = orderAny
	uiSchema["ui:options"] = uiOpts

	for _, c := range chrome {
		uiSchema[c.ID] = map[string]any{
			"ui:components": map[string]any{
				"stringField": stringFieldForChrome(c.Kind),
			},
			"ui:options": map[string]any{
				"chromeId": c.ID,
			},
		}
	}
	return schema, uiSchema
}

func paintHideRootFormTitle(uiSchema map[string]any) {
	opts, _ := uiSchema["ui:options"].(map[string]any)
	if opts == nil {
		opts = map[string]any{}
	} else {
		opts = cloneMapJSON(opts)
	}
	opts["hideTitle"] = true
	uiSchema["ui:options"] = opts
}

func withDefaultPlaceholders(uiSchema, schema map[string]any, chromeFieldIDs []string) map[string]any {
	skip := make(map[string]struct{}, len(chromeFieldIDs))
	for _, id := range chromeFieldIDs {
		skip[id] = struct{}{}
	}
	props, _ := schema["properties"].(map[string]any)
	out := cloneMapJSON(uiSchema)
	for key, raw := range props {
		if _, hit := skip[key]; hit {
			continue
		}
		prop, _ := raw.(map[string]any)
		if prop == nil || !acceptsPlaceholder(prop) {
			continue
		}
		existing, _ := out[key].(map[string]any)
		if existing == nil {
			existing = map[string]any{}
		} else {
			existing = cloneMapJSON(existing)
		}
		if fieldHasPlaceholder(existing) {
			continue
		}
		text := defaultPlaceholderText(key, prop)
		opts, _ := existing["ui:options"].(map[string]any)
		if opts == nil {
			opts = map[string]any{}
		} else {
			opts = cloneMapJSON(opts)
		}
		for _, widgetKey := range []string{"shadcn4Text", "shadcn4Number", "shadcn4Select"} {
			prev, _ := opts[widgetKey].(map[string]any)
			if prev == nil {
				prev = map[string]any{}
			} else {
				prev = cloneMapJSON(prev)
			}
			prev["placeholder"] = text
			opts[widgetKey] = prev
		}
		existing["ui:options"] = opts
		out[key] = existing
	}
	return out
}

func acceptsPlaceholder(schema map[string]any) bool {
	if schema["readOnly"] == true {
		return false
	}
	switch t := schema["type"].(type) {
	case string:
		return t == "string" || t == "number" || t == "integer"
	case []any:
		for _, v := range t {
			s, _ := v.(string)
			if s == "string" || s == "number" || s == "integer" {
				return true
			}
		}
	}
	return false
}

func fieldHasPlaceholder(fieldUi map[string]any) bool {
	if _, ok := fieldUi["ui:placeholder"].(string); ok {
		return true
	}
	opts, _ := fieldUi["ui:options"].(map[string]any)
	if opts == nil {
		return false
	}
	if _, ok := opts["placeholder"].(string); ok {
		return true
	}
	for _, key := range []string{"shadcn4Text", "shadcn4Number", "shadcn4Select", "text", "number", "select"} {
		widget, _ := opts[key].(map[string]any)
		if widget == nil {
			continue
		}
		if _, ok := widget["placeholder"].(string); ok {
			return true
		}
	}
	return false
}

func defaultPlaceholderText(fieldKey string, schema map[string]any) string {
	title, _ := schema["title"].(string)
	title = strings.TrimSpace(title)
	label := title
	if label == "" {
		label = fieldKey
	}
	return fmt.Sprintf("Inserisci %s…", label)
}

func compileDerivedRules(payload map[string]any, leftoverIDs []string) map[string]DerivedRule {
	raw, _ := payload["derived"].(map[string]any)
	if len(raw) == 0 {
		return nil
	}
	derived := map[string]DerivedRule{}
	for key, entryRaw := range raw {
		entry, _ := entryRaw.(map[string]any)
		if entry == nil {
			continue
		}
		expr, _ := entry["expr"].(string)
		if expr == "" {
			continue
		}
		rule := DerivedRule{Expr: expr}
		if overridable, ok := entry["overridable"].(bool); ok {
			rule.Overridable = &overridable
		}
		derived[key] = rule
	}
	if len(derived) == 0 {
		return nil
	}
	if len(leftoverIDs) > 0 {
		return expandMapDerived(derived, leftoverIDs)
	}
	concrete := map[string]DerivedRule{}
	for key, rule := range derived {
		if mapStarDerived.MatchString(key) {
			continue
		}
		concrete[key] = rule
	}
	if len(concrete) == 0 {
		return nil
	}
	return concrete
}

func expandMapDerived(derived map[string]DerivedRule, rowIDs []string) map[string]DerivedRule {
	out := map[string]DerivedRule{}
	for key, entry := range derived {
		match := mapStarDerived.FindStringSubmatch(key)
		if match == nil {
			out[key] = entry
			continue
		}
		mapProp := match[1]
		field := match[2]
		for _, id := range rowIDs {
			path := encodeFormPath([]string{mapProp, id, field})
			out[path] = DerivedRule{
				Expr:        strings.ReplaceAll(entry.Expr, "__ROW_ID__", id),
				Overridable: entry.Overridable,
			}
		}
	}
	return out
}

func encodeFormPath(segments []string) string {
	needsPointer := false
	for _, s := range segments {
		if strings.Contains(s, ".") {
			needsPointer = true
			break
		}
	}
	if !needsPointer {
		return strings.Join(segments, ".")
	}
	parts := make([]string, len(segments))
	for i, s := range segments {
		parts[i] = strings.ReplaceAll(strings.ReplaceAll(s, "~", "~0"), "/", "~1")
	}
	return "/" + strings.Join(parts, "/")
}
