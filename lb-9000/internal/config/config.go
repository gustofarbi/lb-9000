package config

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

type Config struct {
	ContainerPort int    `mapstructure:"SPEC_CONTAINER_PORT"`
	Namespace     string `mapstructure:"SPEC_NAMESPACE"`
	ServiceName   string `mapstructure:"SPEC_SERVICE_NAME"`
	Selector      string `mapstructure:"SPEC_SELECTOR"`

	StoreType     string `mapstructure:"STORE_TYPE"`
	StoreAddr     string `mapstructure:"STORE_ADDR"`
	StoreUsername string `mapstructure:"STORE_USERNAME"`
	StorePassword string `mapstructure:"STORE_PASSWORD"`
	StoreDB       int    `mapstructure:"STORE_DB"`

	RefreshRate time.Duration `mapstructure:"REFRESH_RATE"`
	LockTTL     time.Duration `mapstructure:"LOCK_TTL"`
}

func Parse(path string) (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	if path != "" {
		viper.SetConfigFile(path)
	} else {
		viper.AddConfigPath(".")
	}

	// viper.SetConfigFile(".env.local")
	// _ = viper.MergeInConfig()

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading in config: %w", err)
	}

	cfg := Config{}
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	return &cfg, nil
}
