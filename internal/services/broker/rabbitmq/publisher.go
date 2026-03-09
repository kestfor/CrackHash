package rabbitmq

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type publisher struct {
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(conn *amqp.Connection) (*publisher, error) {
	ch, err := conn.Channel()
	if err != nil {
		return nil, err
	}

	if err := ch.Confirm(false); err != nil {
		return nil, err
	}

	return &publisher{
		conn: conn,
		ch:   ch,
	}, nil

}

func (p *publisher) Publish(ctx context.Context, routingKey string, message []byte) error {
	dc, err := p.ch.PublishWithDeferredConfirmWithContext(
		ctx,
		"",
		routingKey,
		true,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:         message,
			DeliveryMode: amqp.Persistent,
			MessageId:    uuid.New().String(),
			Timestamp:    time.Now(),
		},
	)
	if err != nil {
		return fmt.Errorf("publish: %w", err)
	}

	if dc.Wait() {
		return nil
	}
	return fmt.Errorf("message nacked by broker")
}

func (p *publisher) Close() error {
	return p.ch.Close()
}
