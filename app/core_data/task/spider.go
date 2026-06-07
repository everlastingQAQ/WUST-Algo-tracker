package task

import (
	"cwxu-algo/app/common/event"
	"cwxu-algo/app/core_data/internal/data"
	"cwxu-algo/app/core_data/internal/data/model"
	"encoding/json"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
)

type SpiderTask struct {
	rabbitMQ *amqp.Channel
	db       *gorm.DB
}

func NewSpiderTask(rabbitMQ *event.RabbitMQ, data *data.Data) *SpiderTask {
	return &SpiderTask{
		rabbitMQ: rabbitMQ.Ch,
		db:       data.DB,
	}
}

func (t *SpiderTask) Do(userId int64, needAll bool, source string, requesterId int64, platform string) (int64, error) {
	totalPlatforms := int32(0)
	if platform != "" {
		totalPlatforms = 1
	}
	job := model.SpiderRefreshJob{
		UserID:          userId,
		RequesterID:     requesterId,
		Source:          source,
		Status:          "queued",
		NeedAll:         needAll,
		CurrentPlatform: platform,
		TotalPlatforms:  totalPlatforms,
	}
	if err := t.db.Create(&job).Error; err != nil {
		log.Errorf("SpiderTask: create job failed: %v", err)
		return 0, err
	}

	q, err := t.rabbitMQ.QueueDeclare("spider", true, false, false, false, nil)
	if err != nil {
		log.Errorf("SpiderTask: QueueDeclare failed: %v", err)
		_ = t.db.Model(&job).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}).Error
		return int64(job.ID), err
	}
	e := event.SpiderEvent{UserId: userId, NeedAll: needAll, JobId: int64(job.ID), Platform: platform}
	body, err := json.Marshal(e)
	if err != nil {
		log.Errorf("SpiderTask: json.Marshal failed: %v", err)
		_ = t.db.Model(&job).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}).Error
		return int64(job.ID), err
	}
	if err := t.rabbitMQ.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	}); err != nil {
		log.Errorf("SpiderTask: Publish failed: %v", err)
		_ = t.db.Model(&job).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}).Error
		return int64(job.ID), err
	}
	return int64(job.ID), nil
}
