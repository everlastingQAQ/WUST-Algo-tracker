package service

import (
	"cwxu-algo/app/common/event"
	"encoding/json"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/streadway/amqp"
)

type Consumer struct {
	rabbitMQ *event.RabbitMQ
	spider   *SpiderUseCase
}

func NewConsumer(ch *event.RabbitMQ, spider *SpiderUseCase) *Consumer {
	return &Consumer{
		rabbitMQ: ch,
		spider:   spider,
	}
}

func (c *Consumer) Consume() {
	for {
		if err := c.consumeOnce(); err != nil {
			log.Errorf("RabbitMQ(Spider): consumer stopped: %v", err)
		}
		time.Sleep(3 * time.Second)
		if _, err := c.rabbitMQ.Reconnect(); err != nil {
			log.Errorf("RabbitMQ(Spider): reconnect failed: %v", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func (c *Consumer) consumeOnce() error {
	ch, err := c.rabbitMQ.Channel()
	if err != nil {
		return err
	}
	q, err := ch.QueueDeclare("spider", true, false, false, false, nil)
	if err != nil {
		return err
	}
	_ = ch.Qos(2, 0, false)
	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	if err != nil {
		return err
	}
	for d := range msgs {
		go func(d amqp.Delivery) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("RabbitMQ(Spider): panic recovered: %v", r)
					_ = d.Nack(false, false)
				}
			}()
			msg := event.SpiderEvent{}
			err := json.Unmarshal(d.Body, &msg)
			if err != nil {
				log.Errorf("RabbitMQ(Spider): 解析json出错 %s", err.Error())
				_ = d.Nack(false, false)
				return
			}
			err = c.spider.LoadData(msg.JobId, msg.UserId, msg.NeedAll, msg.Platform)
			if err != nil {
				log.Errorf("RabbitMQ(Spider): %v", err)
				_ = d.Nack(false, false)
				return
			}
			_ = d.Ack(false)
		}(d)
	}
	return errors.New("spider consume channel closed")
}
