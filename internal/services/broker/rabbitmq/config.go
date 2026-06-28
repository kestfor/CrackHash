package rabbitmq

type Config struct {
	URL          string `yaml:"url"`
	RequeueLimit int    `yaml:"requeue_limit" default:"3"`
}
