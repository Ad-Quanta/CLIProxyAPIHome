package config

// ModelFallback configures cross-model retries that run only after the
// requested model fails and before any response byte is written downstream.
// Home stores and forwards this CPA option; the CPA node applies it.
type ModelFallback struct {
	// Enabled toggles the fallback chain. Default is false.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// PreserveRequestedModel rewrites response model fields back to the client
	// requested model. Nil inherits the CPA default, which is true.
	PreserveRequestedModel *bool `yaml:"preserve-requested-model,omitempty" json:"preserve-requested-model,omitempty"`

	// TriggerStatusCodes are upstream statuses that may switch models.
	// Empty means 429, 500, 502, 503, and 504.
	TriggerStatusCodes []int `yaml:"trigger-status-codes,omitempty" json:"trigger-status-codes,omitempty"`

	// NoFallbackStatusCodes never switch models. Empty means 400, 401, 403, and 422.
	NoFallbackStatusCodes []int `yaml:"no-fallback-status-codes,omitempty" json:"no-fallback-status-codes,omitempty"`

	// Rules maps a requested model, or a trailing-wildcard pattern, to ordered targets.
	Rules []ModelFallbackRule `yaml:"rules,omitempty" json:"rules,omitempty"`
}

// ModelFallbackRule is one requested-model to fallback-model mapping.
type ModelFallbackRule struct {
	From string   `yaml:"from" json:"from"`
	To   []string `yaml:"to" json:"to"`
}
