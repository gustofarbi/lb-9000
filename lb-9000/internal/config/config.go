package config

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	RefreshRate time.Duration
	Specs       Specs
	Store       Store
}

type Specs struct {
	Namespace     string
	ServiceName   string
	Selector      string
	ContainerPort int
}

type Store struct {
	Type     string
	Addr     string
	Username string
	Password string
	DB       int
}

func Parse(path string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.AddConfigPath(".")
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading in config: %w", err)
	}

	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
