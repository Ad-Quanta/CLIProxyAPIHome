package home

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// TestSanitizeConfigYAMLForDownstreamPreservesModelFallback guards the Home ->
// CPA payload boundary: Home stores model-fallback, so the sanitized runtime
// payload must still carry it for the CPA node to apply fallback rules.
func TestSanitizeConfigYAMLForDownstreamPreservesModelFallback(t *testing.T) {
	input := `port: 8327
model-fallback:
  enabled: true
  preserve-requested-model: true
  trigger-status-codes: [429, 500, 502, 503, 504]
  no-fallback-status-codes: [400, 401, 403, 422]
  rules:
    - from: "gpt-*"
      to: ["grok-4.7", "grok-4.6"]
api-keys:
  - test-key
`
	out, errSanitize := sanitizeConfigYAMLForDownstream([]byte(input))
	if errSanitize != nil {
		t.Fatalf("sanitizeConfigYAMLForDownstream() error = %v", errSanitize)
	}
	var root map[string]any
	if errUnmarshal := yaml.Unmarshal(out, &root); errUnmarshal != nil {
		t.Fatalf("unmarshal sanitized payload: %v", errUnmarshal)
	}
	if _, exists := root["api-keys"]; exists {
		t.Fatal("sanitized payload still contains api-keys")
	}
	raw, exists := root["model-fallback"]
	if !exists {
		t.Fatalf("sanitized payload dropped model-fallback:\n%s", out)
	}
	fb, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("model-fallback type = %T, want map", raw)
	}
	if enabled, _ := fb["enabled"].(bool); !enabled {
		t.Fatalf("model-fallback.enabled = %#v, want true", fb["enabled"])
	}
	rules, ok := fb["rules"].([]any)
	if !ok || len(rules) != 1 {
		t.Fatalf("model-fallback.rules = %#v", fb["rules"])
	}
	rule, ok := rules[0].(map[string]any)
	if !ok {
		t.Fatalf("rule type = %T", rules[0])
	}
	if from, _ := rule["from"].(string); from != "gpt-*" {
		t.Fatalf("rule.from = %#v, want gpt-*", rule["from"])
	}
	targets, ok := rule["to"].([]any)
	if !ok || len(targets) != 2 || targets[0] != "grok-4.7" || targets[1] != "grok-4.6" {
		t.Fatalf("rule.to = %#v, want [grok-4.7 grok-4.6]", rule["to"])
	}
}
