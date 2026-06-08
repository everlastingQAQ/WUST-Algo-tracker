package event

import (
	"cwxu-algo/app/common/conf"
	"errors"
	"sync"

	"github.com/google/wire"
	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	Ch   *amqp.Channel
	Conn *amqp.Connection
	dsn  string
	mu   sync.Mutex
}

func NewRabbitMQ(data *conf.Server) (*RabbitMQ, func(), error) {
	rabbitMQ := &RabbitMQ{dsn: data.AmqpDsn}
	if _, err := rabbitMQ.Reconnect(); err != nil {
		return nil, func() {}, err
	}
	return rabbitMQ, rabbitMQ.Close, nil
}

var ProviderSet = wire.NewSet(NewRabbitMQ)

func (r *RabbitMQ) Channel() (*amqp.Channel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.Ch == nil {
		return nil, errors.New("rabbitmq channel is not ready")
	}
	return r.Ch, nil
}

func (r *RabbitMQ) Reconnect() (*amqp.Channel, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()
	conn, err := amqp.Dial(r.dsn)
	if err != nil {
		return nil, err
	}
	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	r.Conn = conn
	r.Ch = ch
	return ch, nil
}

func (r *RabbitMQ) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closeLocked()
}

func (r *RabbitMQ) closeLocked() {
	if r.Ch != nil {
		_ = r.Ch.Close()
		r.Ch = nil
	}
	if r.Conn != nil {
		_ = r.Conn.Close()
		r.Conn = nil
	}
}
