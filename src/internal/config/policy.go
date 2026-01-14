package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Policy struct {
	Version  string `yaml:"version"`
	Defaults struct {
		Timeouts struct {
			RequestMS int `yaml:"request_ms"`
		} `yaml:"timeouts"`
		Retries struct {
			MaxAttempts int `yaml:"max_attempts"`
		} `yaml:"retries"`
		Fallback struct {
			MaxProviderHops int `yaml:"max_provider_hops"`
		} `yaml:"fallback"`
	} `yaml:"defaults"`
	Providers map[string]struct {
		Capabilities  []string `yaml:"capabilities"`
		PriorityWeight int     `yaml:"priority_weight"`
	} `yaml:"providers"`
	RoutingRules []struct {
		Name string `yaml:"name"`
		When struct {
			Method string `yaml:"method"`
		} `yaml:"when"`
		Providers []string `yaml:"providers"`
	} `yaml:"routing_rules"`
}

func LoadPolicy(path string) (Policy, error) {
	b, err := os.ReadFile(path)
	if err != nil { return Policy{}, err }
	var p Policy
	if err := yaml.Unmarshal(b, &p); err != nil { return Policy{}, err }
	return p, nil
}
