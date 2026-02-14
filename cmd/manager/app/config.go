package app

import (
	"fmt"
	"time"

	"github.com/kestfor/CrackHash/internal/services/manager/healthchecker"
	"github.com/kestfor/CrackHash/pkg/logging"
)

type HTTPConfig struct {
	Port int `yaml:"port"`
}

type HashCrackerConfig struct {
	Alphabet string `yaml:"alphabet"`
}

type HealthCheckConfig struct {
	Period   time.Duration `yaml:"period"`
	MaxTries int           `yaml:"max_tries"`
}

type Config struct {
	HTTP        *HTTPConfig                            `yaml:"http"`
	Healthcheck *healthchecker.HTTPHealthCheckerConfig `yaml:"healthcheck"`
	HashCracker *HashCrackerConfig                     `yaml:"hash_cracker"`
	Logger      *logging.LoggerConfig                  `yaml:"logger"`
}

func (c *Config) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf("http config is required")
	}

	if c.Healthcheck == nil {
		return fmt.Errorf("healthcheck config is required")
	}

	if c.HashCracker == nil {
		return fmt.Errorf("hash_cracker config is required")
	}

	if c.HashCracker.Alphabet == "" {
		return fmt.Errorf("hash_cracker alphabet is required")
	}

	if c.Logger == nil {
		return fmt.Errorf("logger config is required")
	}

	return nil
}
