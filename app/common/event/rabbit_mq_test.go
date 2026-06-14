package event

import (
	"encoding/json"
	"os"
	"testing"

	"cwxu-algo/app/common/conf"

	"github.com/streadway/amqp"
)

func TestNewRabbitMQ_ReturnsErrorForInvalidDSN(t *testing.T) {
	rabbitMQ, cleanup, err := NewRabbitMQ(&conf.Server{AmqpDsn: "amqp://invalid"})
	if err == nil {
		if cleanup != nil {
			cleanup()
		}
		t.Fatal("NewRabbitMQ() error = nil, want error")
	}
	if rabbitMQ != nil {
		t.Fatalf("NewRabbitMQ() rabbitMQ = %#v, want nil on connect error", rabbitMQ)
	}
	if cleanup == nil {
		t.Fatal("NewRabbitMQ() cleanup = nil, want no-op cleanup")
	}
	cleanup()
}

func TestRabbitMQPublishSpiderEvent(t *testing.T) {
	dsn := os.Getenv("WUST_TEST_AMQP_DSN")
	if dsn == "" {
		t.Skip("set WUST_TEST_AMQP_DSN to run RabbitMQ integration test")
	}

	conn, err := amqp.Dial(dsn)
	if err != nil {
		t.Fatalf("amqp dial: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("open channel: %v", err)
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("spider", true, false, false, false, nil)
	if err != nil {
		t.Fatalf("declare queue: %v", err)
	}
	body, err := json.Marshal(SpiderEvent{UserId: 1, NeedAll: true})
	if err != nil {
		t.Fatalf("marshal spider event: %v", err)
	}
	if err := ch.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		t.Fatalf("publish spider event: %v", err)
	}
}
