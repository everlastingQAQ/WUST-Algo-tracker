package service

import (
	"context"
	"encoding/json"

	"cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/user/internal/data/model"

	"github.com/go-kratos/kratos/v2/log"
)

type userOperationRecorder interface {
	RecordOperation(ctx context.Context, item model.OperationLog) error
}

func recordUserOperation(ctx context.Context, recorder userOperationRecorder, action, targetType string, targetID int64, detail map[string]any) {
	if recorder == nil {
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
	if err := recorder.RecordOperation(ctx, model.OperationLog{
		OperatorID:   operatorID,
		OperatorRole: operatorRole,
		Action:       action,
		TargetType:   targetType,
		TargetID:     targetID,
		Detail:       detailJSON,
	}); err != nil {
		log.Errorf("record user operation failed: action=%s target=%s:%d err=%v", action, targetType, targetID, err)
	}
}
