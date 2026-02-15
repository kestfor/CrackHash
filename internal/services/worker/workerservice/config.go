package workerservice

import "time"

type Config struct {
	MaxParallel  int           `yaml:"max_parallel"`
	NotifyPeriod time.Duration `yaml:"notify_period"`
}
