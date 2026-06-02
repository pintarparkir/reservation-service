package rabbit

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

// dialAndDeclare opens a connection, creates a channel, and declares the exchange.
// On any intermediate error it cleans up already-opened resources.
func dialAndDeclare(amqpURL, exchange string) (*amqp.Connection, *amqp.Channel, error) {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return nil, nil, fmt.Errorf("amqp dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("amqp channel: %w", err)
	}
	if err := ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, nil, fmt.Errorf("exchange declare: %w", err)
	}
	return conn, ch, nil
}

// closeConnCh safely closes a channel and connection pair.
func closeConnCh(ch *amqp.Channel, conn *amqp.Connection) {
	if ch != nil {
		_ = ch.Close()
	}
	if conn != nil {
		_ = conn.Close()
	}
}
