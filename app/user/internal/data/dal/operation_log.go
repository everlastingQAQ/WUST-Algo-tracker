package dal

import (
	"context"
	"cwxu-algo/app/user/internal/data/model"
)

func (d *ProfileDal) RecordOperation(ctx context.Context, item model.OperationLog) error {
	return d.db.WithContext(ctx).Create(&item).Error
}

func (d *GroupDal) RecordOperation(ctx context.Context, item model.OperationLog) error {
	return d.db.WithContext(ctx).Create(&item).Error
}
