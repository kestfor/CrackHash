package app

import (
	"fmt"
	"time"

	"github.com/kestfor/CrackHash/internal/services/broker/rabbitmq"
	"github.com/kestfor/CrackHash/internal/services/manager/storage/mongodb"
	"github.com/kestfor/CrackHash/pkg/logging"
)

type HTTPConfig struct {
	Port int `yaml:"port"`
}

type HashCrackerConfig struct {
	Alphabet string `yaml:"alphabet"`
}

type Config struct {
	HTTP            *HTTPConfig           `yaml:"http"`
	Storage         *mongodb.Config       `yaml:"storage"`
	Broker          *rabbitmq.Config      `yaml:"broker"`
	HashCracker     *HashCrackerConfig    `yaml:"hash_cracker"`
	Logger          *logging.LoggerConfig `yaml:"logger"`
	RetrySendPeriod time.Duration         `yaml:"retry_send_period"`
}

func (c *Config) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf("http config is required")
	}

	if c.Broker == nil {
		return fmt.Errorf("broker config is required")
	}

	if c.Storage == nil {
		return fmt.Errorf("storage config is required")
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
