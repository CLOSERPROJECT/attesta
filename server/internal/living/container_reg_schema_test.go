package living

import (
	"regexp"
	"testing"
)

func containerRegSchema(t *testing.T) map[string]any {
	t.Helper()
	line := lineJob(t)
	task, ok := TaskByID(line.Tasks, "containerReg")
	if !ok {
		t.Fatal("containerReg missing")
	}
	schema, _ := task["schema"].(map[string]any)
	if schema == nil {
		t.Fatal("containerReg schema missing")
	}
	return schema
}

func TestContainerRegSchemaRejectsBlankDualSecondary(t *testing.T) {
	schema := containerRegSchema(t)
	props, _ := schema["properties"].(map[string]any)
	sec, _ := props["secondarySourceLoadId"].(map[string]any)
	if sec == nil {
		t.Fatal("secondarySourceLoadId missing")
	}

	minLen := 0
	switch n := sec["minLength"].(type) {
	case int:
		minLen = n
	case int64:
		minLen = int(n)
	case float64:
		minLen = int(n)
	}
	if minLen < 1 {
		t.Fatalf("secondarySourceLoadId minLength = %#v, want >= 1", sec["minLength"])
	}

	pat, _ := sec["pattern"].(string)
	if pat == "" {
		t.Fatal("secondarySourceLoadId pattern missing (trim-aware non-empty)")
	}
	re, err := regexp.Compile(pat)
	if err != nil {
		t.Fatalf("pattern %q: %v", pat, err)
	}
	for _, blank := range []string{"", " ", "   ", "\t"} {
		if len(blank) >= minLen && re.MatchString(blank) {
			t.Fatalf("pattern %q must reject %q", pat, blank)
		}
	}
	if !re.MatchString("load-b") {
		t.Fatalf("pattern %q must accept non-blank id", pat)
	}

	then, _ := schema["then"].(map[string]any)
	req, _ := then["required"].([]any)
	found := false
	for _, r := range req {
		if r == "secondarySourceLoadId" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("dual if/then must require secondarySourceLoadId; then = %#v", then)
	}
}
