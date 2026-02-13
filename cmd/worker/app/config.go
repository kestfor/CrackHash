package app

import (
	"fmt"

	"github.com/kestfor/CrackHash/internal/services/worker"
	"github.com/kestfor/CrackHash/internal/services/worker/notifier"
	"github.com/kestfor/CrackHash/internal/services/worker/registerer"
)

type HTTPServerConfig struct {
	Port int `yaml:"port"`
}

type Config struct {
	HTTP       *HTTPServerConfig                `yaml:"http"`
	Registerer *registerer.HTTPRegistererConfig `yaml:"registerer"`
	Notifier   *notifier.HTTPNotifierConfig     `yaml:"notifier"`
	Worker     *worker.Config                   `yaml:"workers"`
}

func (c *Config) Validate() error {
	if c.HTTP == nil {
		return fmt.Errorf("http config is required")
	}

	if c.Registerer == nil {
		return fmt.Errorf("registerer config is required")
	}

	if c.Notifier == nil {
		return fmt.Errorf("notifier config is required")
	}

	if c.Worker == nil {
		return fmt.Errorf("worker config is required")
	}

	return nil

}
