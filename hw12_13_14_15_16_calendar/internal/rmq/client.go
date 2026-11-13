package rmq

import (
	"context"
	"errors"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const prefetchCount = 10

var (
	ErrNotConnected     = errors.New("rmq: client is not connected")
	ErrDeliveriesClosed = errors.New("rmq: deliveries channel closed")
	ErrNotConfirmed     = errors.New("rmq: message was not confirmed by broker")
)

type HandlerFunc = func(ctx context.Context, body []byte) error

type Config struct {
	URL        string
	Exchange   string
	Queue      string
	RoutingKey string
}

type Client struct {
	cfg  Config
	conn *amqp.Connection
	ch   *amqp.Channel
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Connect(_ context.Context) error {
	conn, err := amqp.Dial(c.cfg.URL)
	if err != nil {
		return fmt.Errorf("rmq: dial: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("rmq: open channel: %w", err)
	}

	c.conn, c.ch = conn, ch

	if err := c.ch.Confirm(false); err != nil {
		_ = c.Close()
		return fmt.Errorf("rmq: enable publisher confirms: %w", err)
	}

	if err := c.declareTopology(); err != nil {
		_ = c.Close()
		return err
	}
	return nil
}

func (c *Client) declareTopology() error {
	err := c.ch.ExchangeDeclare(
		c.cfg.Exchange,
		amqp.ExchangeDirect,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("rmq: declare exchange %q: %w", c.cfg.Exchange, err)
	}

	_, err = c.ch.QueueDeclare(
		c.cfg.Queue,
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("rmq: declare queue %q: %w", c.cfg.Queue, err)
	}

	err = c.ch.QueueBind(
		c.cfg.Queue,
		c.cfg.RoutingKey,
		c.cfg.Exchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("rmq: bind queue %q to exchange %q: %w", c.cfg.Queue, c.cfg.Exchange, err)
	}
	return nil
}

func (c *Client) Close() error {
	var errs []error
	if c.ch != nil {
		if err := c.ch.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			errs = append(errs, fmt.Errorf("rmq: close channel: %w", err))
		}
	}
	if c.conn != nil {
		if err := c.conn.Close(); err != nil && !errors.Is(err, amqp.ErrClosed) {
			errs = append(errs, fmt.Errorf("rmq: close connection: %w", err))
		}
	}
	c.ch, c.conn = nil, nil
	return errors.Join(errs...)
}

func (c *Client) Publish(ctx context.Context, body []byte) error {
	if c.ch == nil {
		return ErrNotConnected
	}

	confirm, err := c.ch.PublishWithDeferredConfirmWithContext(ctx,
		c.cfg.Exchange,
		c.cfg.RoutingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         body,
		},
	)
	if err != nil {
		return fmt.Errorf("rmq: publish: %w", err)
	}

	acked, err := confirm.WaitContext(ctx)
	if err != nil {
		return fmt.Errorf("rmq: wait for confirm: %w", err)
	}
	if !acked {
		return ErrNotConfirmed
	}
	return nil
}

func (c *Client) Consume(ctx context.Context, handle HandlerFunc) error {
	if c.ch == nil {
		return ErrNotConnected
	}

	if err := c.ch.Qos(prefetchCount, 0, false); err != nil {
		return fmt.Errorf("rmq: set qos: %w", err)
	}

	deliveries, err := c.ch.Consume(
		c.cfg.Queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("rmq: consume from %q: %w", c.cfg.Queue, err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case d, ok := <-deliveries:
			if !ok {
				return ErrDeliveriesClosed
			}
			if err := settle(d, handle(ctx, d.Body)); err != nil {
				return err
			}
		}
	}
}

func settle(d amqp.Delivery, handleErr error) error {
	if handleErr == nil {
		if err := d.Ack(false); err != nil {
			return fmt.Errorf("rmq: ack: %w", err)
		}
		return nil
	}

	if err := d.Nack(false, false); err != nil {
		return fmt.Errorf("rmq: nack: %w", err)
	}
	return nil
}
