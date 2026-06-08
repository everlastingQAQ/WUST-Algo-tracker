package task

import (
	"cwxu-algo/app/common/event"
	"encoding/json"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
)

type SummaryTask struct {
	rabbitMQ *event.RabbitMQ
}

func NewSummaryTask(rabbitMQ *event.RabbitMQ) *SummaryTask {
	return &SummaryTask{
		rabbitMQ: rabbitMQ,
	}
}

func (t *SummaryTask) Do(userId int64, typ string) {
	e := event.SummaryEvent{UserId: userId, Type: typ}
	body, err := json.Marshal(e)
	if err != nil {
		log.Errorf("SummaryTask: json.Marshal failed: %v", err)
		return
	}
	if err := t.publish(body); err != nil {
		log.Errorf("SummaryTask: publish summary event failed: %v", err)
	}
}

func (t *SummaryTask) publish(body []byte) error {
	ch, err := t.rabbitMQ.Channel()
	if err == nil {
		if err = publishToQueue(ch, "summary", body); err == nil {
			return nil
		}
	}
	log.Warnf("SummaryTask: rabbitmq publish failed, reconnecting: %v", err)
	ch, reconnectErr := t.rabbitMQ.Reconnect()
	if reconnectErr != nil {
		if err != nil {
			return fmt.Errorf("%w; reconnect failed: %v", err, reconnectErr)
		}
		return reconnectErr
	}
	return publishToQueue(ch, "summary", body)
}
