package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestValidateV8ConfigAcceptsModelFallback verifies that Home's strict v8
// management schema accepts the CPA model-fallback section and preserves it
// through the database and downstream projections.
func TestValidateV8ConfigAcceptsModelFallback(t *testing.T) {
	source := []byte(`model-fallback:
  enabled: true
  preserve-requested-model: false
  trigger-status-codes: [429, 503]
  no-fallback-status-codes: [400]
  rules:
    - from: primary
      to: [secondary, tertiary]
`)
	if errValidate := ValidateV8Config(source); errValidate != nil {
		t.Fatalf("ValidateV8Config() error = %v", errValidate)
	}

	var cfg Config
	if errDecode := yaml.Unmarshal(source, &cfg); errDecode != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", errDecode)
	}
	if !cfg.ModelFallback.Enabled {
		t.Fatal("model-fallback.enabled = false, want true")
	}
	if cfg.ModelFallback.PreserveRequestedModel == nil || *cfg.ModelFallback.PreserveRequestedModel {
		t.Fatal("model-fallback.preserve-requested-model was not preserved as false")
	}
	if len(cfg.ModelFallback.Rules) != 1 || cfg.ModelFallback.Rules[0].From != "primary" || len(cfg.ModelFallback.Rules[0].To) != 2 {
		t.Fatalf("model-fallback rules = %#v", cfg.ModelFallback.Rules)
	}

	migrated, _, errMigrate := NormalizeConfigLayout(source, true)
	if errMigrate != nil {
		t.Fatalf("NormalizeConfigLayout() error = %v", errMigrate)
	}
	if !strings.Contains(string(migrated), "model-fallback:") || !strings.Contains(string(migrated), "from: primary") {
		t.Fatalf("v8 migration lost model-fallback:\n%s", migrated)
	}

	var root map[string]any
	if errDecode := yaml.Unmarshal(source, &root); errDecode != nil {
		t.Fatalf("yaml.Unmarshal(root) error = %v", errDecode)
	}
	projected, errNormalize := NormalizeConfigRoot(root)
	if errNormalize != nil {
		t.Fatalf("NormalizeConfigRoot() error = %v", errNormalize)
	}
	wire, errMarshal := yaml.Marshal(projected)
	if errMarshal != nil {
		t.Fatalf("yaml.Marshal(projected) error = %v", errMarshal)
	}
	var downstream map[string]any
	if errDecode := yaml.Unmarshal(wire, &downstream); errDecode != nil {
		t.Fatalf("yaml.Unmarshal(downstream wire) error = %v", errDecode)
	}
	if _, ok := downstream["model-fallback"]; !ok {
		t.Fatalf("downstream wire lost model-fallback:\n%s", wire)
	}
}

// TestValidateV8ConfigRejectsInvalidModelFallback verifies that malformed
// fallback payloads cannot be persisted by the Management API.
func TestValidateV8ConfigRejectsInvalidModelFallback(t *testing.T) {
	for _, source := range []string{
		"model-fallback: true",
		"model-fallback: {enabled: not-a-bool}",
		"model-fallback: {rules: {from: primary}}",
		"model-fallback: {preserve-requested-model: sometimes}",
	} {
		t.Run(source, func(t *testing.T) {
			if errValidate := ValidateV8Config([]byte(source)); errValidate == nil {
				t.Fatal("invalid model-fallback accepted")
			}
		})
	}
}
