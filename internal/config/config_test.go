package config

import (
	"testing"
)

func TestCleanURL(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://jira.mv.com.br/projects/OPS/queues/custom/5607", "https://jira.mv.com.br"},
		{"https://jira.mv.com.br/", "https://jira.mv.com.br"},
		{"http://localhost:8080/api", "http://localhost:8080"},
		{"", "https://jira.mv.com.br"},
	}

	for _, tt := range tests {
		got := CleanURL(tt.input)
		if got != tt.expected {
			t.Errorf("CleanURL(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestResolveContract(t *testing.T) {
	cfg := &Config{
		JiraContratoDefault: "INTERNO",
	}

	c1 := cfg.ResolveContract("payment-service-billing")
	if c1.Value != "INTERNO" {
		t.Errorf("expected INTERNO, got %s", c1.Value)
	}

	c2 := cfg.ResolveContract("dentalis-prod-snowflake")
	if c2.Value != "DENTALIS (SNOW FLAKE)" {
		t.Errorf("expected DENTALIS (SNOW FLAKE), got %s", c2.Value)
	}

	c3 := cfg.ResolveContract("farmacia-digital-app")
	if c3.Value != "FARMACIA DIGITAL (AWS)" {
		t.Errorf("expected FARMACIA DIGITAL (AWS), got %s", c3.Value)
	}

	c4 := cfg.ResolveContract("maida-gcp-gateway")
	if c4.Value != "MAIDA (GCP)" {
		t.Errorf("expected MAIDA (GCP), got %s", c4.Value)
	}
}
