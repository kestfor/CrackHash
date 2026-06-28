package broker

import "context"

type Handler interface {
	Handle(msg Message) error
}

type Publisher interface {
	Publish(ctx context.Context, routingKey string, message []byte) error
	Close() error
}

type Consumer interface {
	Consume(ctx context.Context, handler Handler) error
	Close() error
}

type Message struct {
	Body       []byte
	RoutingKey string
	Headers    map[string]any
	Ack        func() error
	Nack       func(requeue bool) error
}
