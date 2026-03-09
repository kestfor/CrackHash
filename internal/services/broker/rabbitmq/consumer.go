package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/kestfor/CrackHash/internal/services/broker"
	amqp "github.com/rabbitmq/amqp091-go"
)

type consumer struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	queueName string
}

func NewConsumer(conn *amqp.Connection, queueName string, prefetch int) (*consumer, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.Qos(
		prefetch, // макс. число unacked сообщений
		0,
		false,
	)

	if err != nil {
		ch.Close()
		return nil, err
	}

	return &consumer{conn: conn, ch: ch, queueName: queueName}, nil
}

func (c *consumer) Consume(ctx context.Context, handler broker.Handler) error {
	deliveries, err := c.ch.Consume(
		c.queueName,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case delivery, ok := <-deliveries:
			if !ok {
				return fmt.Errorf("delivery channel closed")
			}

			msg := broker.Message{
				RoutingKey: delivery.RoutingKey,
				Body:       delivery.Body,
				Headers:    delivery.Headers,
				Ack: func() error {
					return delivery.Ack(false)
				},
				Nack: func(requeue bool) error {
					return delivery.Nack(false, requeue)
				},
			}

			if err := handler.Handle(msg); err != nil {
				slog.Error("handler error, nacking message", slog.Any("error", err), slog.String("queue", c.queueName))
				_ = delivery.Nack(false, true)
				continue
			}
		}
	}
}

func (c *consumer) Close() error {
	return c.ch.Close()
}
