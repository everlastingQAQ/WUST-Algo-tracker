package task

import (
	"cwxu-algo/app/common/event"
	"cwxu-algo/app/core_data/internal/data"
	"cwxu-algo/app/core_data/internal/data/model"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/streadway/amqp"
	"gorm.io/gorm"
)

var ErrActiveRefreshJob = errors.New("active spider refresh job already exists")

const activeRefreshJobTTL = 2 * 60 * 60 // seconds

type SpiderTask struct {
	rabbitMQ *event.RabbitMQ
	db       *gorm.DB
}

func NewSpiderTask(rabbitMQ *event.RabbitMQ, data *data.Data) *SpiderTask {
	return &SpiderTask{
		rabbitMQ: rabbitMQ,
		db:       data.DB,
	}
}

func (t *SpiderTask) Do(userId int64, needAll bool, source string, requesterId int64, platform string) (int64, error) {
	activeJob, err := t.findActiveJob(userId, platform)
	if err != nil {
		return 0, err
	}
	if activeJob.ID > 0 {
		return int64(activeJob.ID), ErrActiveRefreshJob
	}

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
	if err := t.publish(body); err != nil {
		log.Errorf("SpiderTask: publish spider event failed: %v", err)
		_ = t.db.Model(&job).Updates(map[string]interface{}{
			"status": "failed",
			"error":  err.Error(),
		}).Error
		return int64(job.ID), err
	}
	return int64(job.ID), nil
}

func (t *SpiderTask) publish(body []byte) error {
	ch, err := t.rabbitMQ.Channel()
	if err == nil {
		if err = publishToQueue(ch, "spider", body); err == nil {
			return nil
		}
	}
	log.Warnf("SpiderTask: rabbitmq publish failed, reconnecting: %v", err)
	ch, reconnectErr := t.rabbitMQ.Reconnect()
	if reconnectErr != nil {
		if err != nil {
			return fmt.Errorf("%w; reconnect failed: %v", err, reconnectErr)
		}
		return reconnectErr
	}
	return publishToQueue(ch, "spider", body)
}

func publishToQueue(ch *amqp.Channel, queue string, body []byte) error {
	if ch == nil {
		return errors.New("rabbitmq channel is not ready")
	}
	q, err := ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return err
	}
	return ch.Publish("", q.Name, false, false, amqp.Publishing{
		ContentType: "application/json",
		Body:        body,
	})
}

func (t *SpiderTask) findActiveJob(userId int64, platform string) (model.SpiderRefreshJob, error) {
	if err := t.expireStaleActiveJobs(userId, platform); err != nil {
		return model.SpiderRefreshJob{}, err
	}
	var job model.SpiderRefreshJob
	query := t.db.
		Where("user_id = ?", userId).
		Where("status IN ?", []string{"queued", "running"})
	condition, args := ActiveRefreshConflictCondition(platform)
	query = query.Where(condition, args...)
	err := query.Order("updated_at DESC").First(&job).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.SpiderRefreshJob{}, nil
	}
	return job, err
}

func (t *SpiderTask) expireStaleActiveJobs(userId int64, platform string) error {
	condition, args := ActiveRefreshConflictCondition(platform)
	query := t.db.Model(&model.SpiderRefreshJob{}).
		Where("user_id = ?", userId).
		Where("status IN ?", []string{"queued", "running"}).
		Where("updated_at < NOW() - (? * INTERVAL '1 second')", activeRefreshJobTTL).
		Where(condition, args...)
	return query.Updates(map[string]interface{}{
		"status": "failed",
		"error":  "抓取任务长时间未更新，已自动标记为失败，可重新发起刷新",
	}).Error
}
