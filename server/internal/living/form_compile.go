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
		case "readonly":
			expr, _ := entry["expr"].(string)
			if expr == "" {
				continue
			}
			out = append(out, ChromeRule{
				ID: id, Kind: kind, Expr: expr,
				Title: title, Before: before, After: after,
			})
		case "dataReport":
			expr, _ := entry["expr"].(string)
			if strings.TrimSpace(expr) == "" {
				continue
			}
			out = append(out, ChromeRule{
				ID: id, Kind: kind, Expr: expr,
				Title: title, Before: before, After: after,
			})
		}
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
	if rule.Kind == "dataReport" && rule.Title != "" {
		return rule.Title
	}
	if title, ok := entry["title"].(string); ok && title != "" {
		return title
	}
	return id
}

// stringFieldForChrome maps known chrome kinds to sjsf stringField widgets.
// Unknown kinds return ("", false) — callers must not invent a fallback widget.
func stringFieldForChrome(kind string) (string, bool) {
	switch kind {
	case "readonly":
		return "readonlyField", true
	case "dataReport":
		return "dataReportField", true
	default:
		return "", false
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
		sf, ok := stringFieldForChrome(c.Kind)
		if !ok {
			continue
		}
		uiSchema[c.ID] = map[string]any{
			"ui:components": map[string]any{
				"stringField": sf,
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
	out := cloneMapJSON(uiSchema)
	applyDefaultPlaceholders(out, schema, skip, true)
	return out
}

func applyDefaultPlaceholders(uiNode, schemaNode map[string]any, skipRoot map[string]struct{}, atRoot bool) bool {
	props, _ := schemaNode["properties"].(map[string]any)
	if len(props) == 0 {
		return false
	}
	changed := false
	for key, raw := range props {
		if atRoot {
			if _, hit := skipRoot[key]; hit {
				continue
			}
		}
		prop, _ := raw.(map[string]any)
		if prop == nil {
			continue
		}
		fieldUi, _ := uiNode[key].(map[string]any)
		if fieldUi == nil {
			fieldUi = map[string]any{}
		} else {
			fieldUi = cloneMapJSON(fieldUi)
		}
		fieldChanged := false
		if fieldHasPlaceholder(fieldUi) {
			if expandPlaceholderToShadcn(fieldUi) {
				fieldChanged = true
			}
		} else if acceptsPlaceholder(prop) {
			writeDefaultPlaceholder(fieldUi, defaultPlaceholderText(key, prop))
			fieldChanged = true
		}
		if _, ok := prop["properties"].(map[string]any); ok {
			if applyDefaultPlaceholders(fieldUi, prop, nil, false) {
				fieldChanged = true
			}
		}
		if items, ok := prop["items"].(map[string]any); ok {
			itemsUi, _ := fieldUi["items"].(map[string]any)
			if itemsUi == nil {
				itemsUi = map[string]any{}
			} else {
				itemsUi = cloneMapJSON(itemsUi)
			}
			itemsChanged := false
			if _, ok := items["properties"].(map[string]any); ok {
				itemsChanged = applyDefaultPlaceholders(itemsUi, items, nil, false)
			} else if fieldHasPlaceholder(itemsUi) {
				itemsChanged = expandPlaceholderToShadcn(itemsUi)
			} else if acceptsPlaceholder(items) {
				writeDefaultPlaceholder(itemsUi, defaultPlaceholderText(key, items))
				itemsChanged = true
			}
			if itemsChanged {
				fieldUi["items"] = itemsUi
				fieldChanged = true
			}
		}
		if fieldChanged {
			uiNode[key] = fieldUi
			changed = true
		}
	}
	return changed
}

var shadcnPlaceholderWidgets = []string{"shadcn4Text", "shadcn4Number", "shadcn4Select"}

func writeDefaultPlaceholder(fieldUi map[string]any, text string) {
	_ = stampPlaceholder(fieldUi, text, true)
}

// Authored placeholder lives at ui:options.placeholder (or ui:placeholder);
// SPA shadcn widgets read ui:options.shadcn4*.placeholder — fill gaps only.
func expandPlaceholderToShadcn(fieldUi map[string]any) bool {
	text := authoredPlaceholderText(fieldUi)
	if text == "" {
		return false
	}
	return stampPlaceholder(fieldUi, text, false)
}

func authoredPlaceholderText(fieldUi map[string]any) string {
	if s, ok := fieldUi["ui:placeholder"].(string); ok && s != "" {
		return s
	}
	opts, _ := fieldUi["ui:options"].(map[string]any)
	if opts == nil {
		return ""
	}
	if s, ok := opts["placeholder"].(string); ok && s != "" {
		return s
	}
	for _, key := range shadcnPlaceholderWidgets {
		widget, _ := opts[key].(map[string]any)
		if s, ok := widget["placeholder"].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

// stampPlaceholder writes opts.placeholder and each shadcn4* widget placeholder.
// When force, overwrite widgets; otherwise only fill missing widget slots.
func stampPlaceholder(fieldUi map[string]any, text string, force bool) bool {
	opts, _ := fieldUi["ui:options"].(map[string]any)
	if opts == nil {
		opts = map[string]any{}
	} else {
		opts = cloneMapJSON(opts)
	}
	changed := false
	if prev, _ := opts["placeholder"].(string); prev != text {
		opts["placeholder"] = text
		changed = true
	}
	for _, widgetKey := range shadcnPlaceholderWidgets {
		prev, _ := opts[widgetKey].(map[string]any)
		if prev == nil {
			prev = map[string]any{}
		} else {
			prev = cloneMapJSON(prev)
		}
		cur, _ := prev["placeholder"].(string)
		if !force && cur != "" {
			continue
		}
		if cur == text {
			if _, ok := opts[widgetKey]; !ok {
				opts[widgetKey] = prev
				changed = true
			}
			continue
		}
		prev["placeholder"] = text
		opts[widgetKey] = prev
		changed = true
	}
	if changed {
		fieldUi["ui:options"] = opts
	}
	return changed
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
	if _, hasEnum := schema["enum"]; hasEnum {
		return fmt.Sprintf("Seleziona %s…", label)
	}
	return fmt.Sprintf("Inserisci %s…", label)
}

func compileDerivedRules(payload map[string]any, rowIDs []string) map[string]DerivedRule {
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
	if len(rowIDs) > 0 {
		return expandMapDerived(derived, rowIDs)
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
