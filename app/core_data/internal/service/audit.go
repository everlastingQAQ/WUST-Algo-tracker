package service

import (
	"context"
	"encoding/json"

	"cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/core_data/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

func recordCoreOperation(ctx context.Context, db *gorm.DB, action, targetType string, targetID int64, detail map[string]any) {
	if db == nil {
		return
	}
	current := auth.GetCurrentUser(ctx)
	operatorID := int64(0)
	operatorRole := 0
	if current != nil {
		operatorID = int64(current.UserID)
		operatorRole = current.RoleID
	}
	detailJSON := "{}"
	if len(detail) > 0 {
		if encoded, err := json.Marshal(detail); err == nil {
			detailJSON = string(encoded)
		}
	}
	if err := db.WithContext(ctx).Create(&model.OperationLog{
		OperatorID:   operatorID,
		OperatorRole: operatorRole,
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Detail:       detailJSON,
	}).Error; err != nil {
		log.Errorf("record core operation failed: action=%s target=%s:%d err=%v", action, targetType, targetID, err)
	}
}
