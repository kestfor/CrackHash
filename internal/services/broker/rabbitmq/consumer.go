package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/kestfor/CrackHash/internal/services/broker"
	amqp "github.com/rabbitmq/amqp091-go"
)

const reconnectDelay = 3 * time.Second

type consumer struct {
	url       string
	queueName string
	prefetch  int
	cancel    context.CancelFunc
}

func NewConsumer(url, queueName string, prefetch int) *consumer {
	return &consumer{url: url, queueName: queueName, prefetch: prefetch}
}

func (c *consumer) connect() (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(c.url)
	if err != nil {
		return nil, nil, err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, nil, err
	}

	if err := ch.Qos(c.prefetch, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return nil, nil, err
	}

	return conn, ch, nil
}

func (c *consumer) Consume(ctx context.Context, handler broker.Handler) error {
	ctx, c.cancel = context.WithCancel(ctx)

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		conn, ch, err := c.connect()
		if err != nil {
			slog.Warn("broker unavailable, retrying...", slog.Any("error", err), slog.String("queue", c.queueName))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(reconnectDelay):
				continue
			}
		}

		slog.Info("consumer connected to broker", slog.String("queue", c.queueName))
		err = c.consumeLoop(ctx, ch, handler)
		conn.Close()

		if ctx.Err() != nil {
			return ctx.Err()
		}

		slog.Warn("consumer disconnected, reconnecting...", slog.Any("error", err), slog.String("queue", c.queueName))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(reconnectDelay):
		}
	}
}

func (c *consumer) consumeLoop(ctx context.Context, ch *amqp.Channel, handler broker.Handler) error {
	deliveries, err := ch.Consume(
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
	if c.cancel != nil {
		c.cancel()
	}
	return nil
}
