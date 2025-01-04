package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParse(t *testing.T) {
	cfg, err := Parse("config.yaml")
	assert.NoError(t, err)
	assert.Equal(t, 8080, cfg.Specs.ContainerPort)
}
