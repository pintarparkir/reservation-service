package rabbit

import (
	"context"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

type Publisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchange string
}

func NewPublisher(amqpURL, exchange string) (*Publisher, error) {
	conn, ch, err := dialAndDeclare(amqpURL, exchange)
	if err != nil {
		return nil, err
	}
	return &Publisher{conn: conn, ch: ch, exchange: exchange}, nil
}

func (p *Publisher) Publish(ctx context.Context, routingKey string, body []byte) error {
	ctx, span := otel.Tracer("rabbit").Start(ctx, "publish "+routingKey,
		trace.WithSpanKind(trace.SpanKindProducer),
	)
	defer span.End()

	headers := make(amqp.Table)
	otel.GetTextMapPropagator().Inject(ctx, &amqpHeaderCarrier{headers})

	return p.ch.PublishWithContext(ctx, p.exchange, routingKey, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Headers:      headers,
		Body:         body,
	})
}

func (p *Publisher) Close() {
	closeConnCh(p.ch, p.conn)
}
