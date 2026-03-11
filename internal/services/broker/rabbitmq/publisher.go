package rabbitmq

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
)

type publisher struct {
	mu   sync.Mutex
	url  string
	conn *amqp.Connection
	ch   *amqp.Channel
}

func NewPublisher(url string) (*publisher, error) {
	p := &publisher{url: url}
	if err := p.reconnect(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *publisher) reconnect() error {
	if p.ch != nil {
		p.ch.Close()
	}
	if p.conn != nil {
		p.conn.Close()
	}

	conn, err := amqp.Dial(p.url)
	if err != nil {
		return err
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return err
	}

	if err := ch.Confirm(false); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	p.conn = conn
	p.ch = ch
	return nil
}

func (p *publisher) Publish(ctx context.Context, routingKey string, message []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	dc, err := p.publish(ctx, routingKey, message)
	if err != nil {
		slog.Warn("publish failed, attempting reconnect...", slog.Any("error", err))
		if reconnErr := p.reconnect(); reconnErr != nil {
			return fmt.Errorf("publish: %w; reconnect: %v", err, reconnErr)
		}
		dc, err = p.publish(ctx, routingKey, message)
		if err != nil {
			return fmt.Errorf("publish after reconnect: %w", err)
		}
	}

	if dc.Wait() {
		return nil
	}
	return fmt.Errorf("message nacked by broker")
}

func (p *publisher) publish(ctx context.Context, routingKey string, message []byte) (*amqp.DeferredConfirmation, error) {
	return p.ch.PublishWithDeferredConfirmWithContext(
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
}

func (p *publisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.ch != nil {
		_ = p.ch.Close()
	}
	if p.conn != nil {
		return p.conn.Close()
	}
	return nil
}
