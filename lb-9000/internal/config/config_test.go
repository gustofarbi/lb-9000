package config

import "testing"

func TestParse(t *testing.T) {
	cfg, err := Parse("config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Specs.ContainerPort != 8080 {
		t.Errorf("expected 8080, got %d", cfg.Specs.ContainerPort)
	}
}
