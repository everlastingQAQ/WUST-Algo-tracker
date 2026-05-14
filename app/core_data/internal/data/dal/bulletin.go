package dal

import (
	"cwxu-algo/app/core_data/internal/data"
	"cwxu-algo/app/core_data/internal/data/model"

	"gorm.io/gorm"
)

// BulletinDal 公告数据操作模块
type BulletinDal struct {
	db *gorm.DB
}

func NewBulletinDal(data *data.Data) *BulletinDal {
	return &BulletinDal{db: data.DB}
}

// Create 创建公告
func (d *BulletinDal) Create(bulletin *model.Bulletin) error {
	return d.db.Create(bulletin).Error
}

// Update 更新公告（仅更新非零字段）
func (d *BulletinDal) Update(id int64, updates map[string]interface{}) error {
	return d.db.Model(&model.Bulletin{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除公告
func (d *BulletinDal) Delete(id int64) error {
	return d.db.Delete(&model.Bulletin{}, id).Error
}

// GetById 根据ID获取公告
func (d *BulletinDal) GetById(id int64) (*model.Bulletin, error) {
	var bulletin model.Bulletin
	err := d.db.First(&bulletin, id).Error
	if err != nil {
		return nil, err
	}
	return &bulletin, nil
}

// List 分页获取公告列表（置顶优先，按创建时间倒序）
func (d *BulletinDal) List(page, pageSize int64) ([]model.Bulletin, int64, error) {
	var bulletins []model.Bulletin
	var total int64

	// 查总数
	err := d.db.Model(&model.Bulletin{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	// 分页查询：置顶优先，再按创建时间倒序
	offset := (page - 1) * pageSize
	err = d.db.Order("is_pinned DESC, created_at DESC").
		Offset(int(offset)).
		Limit(int(pageSize)).
		Find(&bulletins).Error
	if err != nil {
		return nil, 0, err
	}
	return bulletins, total, nil
}
