package app

import (
	"fmt"
	"time"

	"github.com/kestfor/CrackHash/internal/services/broker/rabbitmq"
	"github.com/kestfor/CrackHash/internal/services/worker/workerservice"
	"github.com/kestfor/CrackHash/pkg/logging"
)

type HTTPServerConfig struct {
	Port int `yaml:"port"`
}

type Config struct {
	HTTP   *HTTPServerConfig     `yaml:"http"`
	Broker *rabbitmq.Config      `yaml:"broker"`
	Worker *workerservice.Config `yaml:"workers"`
	Logger *logging.LoggerConfig `yaml:"logger"`
}

func (c *Config) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf("http config is required")
	}

	if c.Broker == nil {
		return fmt.Errorf("broker config is required")
	}

	if c.Worker == nil {
		return fmt.Errorf("worker config is required")
	}

	if c.Worker.NotifyPeriod == 0 {
		c.Worker.NotifyPeriod = 5 * time.Second
	}

	if c.Logger == nil {
		return fmt.Errorf("logger config is required")
	}

	return nil

}
