package dal

import (
	"context"
	data2 "cwxu-algo/app/common/data"
	"cwxu-algo/app/user/internal/data"
	"cwxu-algo/app/user/internal/data/model"
	"errors"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GroupDal struct {
	db  *gorm.DB
	rdb *redis.Client
}

func NewGroupDal(data *data.Data) *GroupDal {
	return &GroupDal{db: data.DB, rdb: data.RDB}
}

func (d *GroupDal) Create(ctx context.Context, name, describe string) (int64, error) {
	group := model.Group{
		Name:     &name,
		Describe: describe,
	}
	if err := d.db.WithContext(ctx).Create(&group).Error; err != nil {
		return 0, fmt.Errorf("创建组失败: %w", err)
	}
	return int64(group.ID), nil
}

func (d *GroupDal) Delete(ctx context.Context, id int64) error {
	result := d.db.WithContext(ctx).Delete(&model.Group{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除组失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("组不存在")
	}
	if err := d.db.WithContext(ctx).Model(&model.User{}).Where("group_id = ?", id).Update("group_id", 0).Error; err != nil {
		return fmt.Errorf("重置用户组ID失败: %w", err)
	}
	return nil
}

func (d *GroupDal) Get(ctx context.Context, id int64) (*model.Group, error) {
	var group model.Group
	err := d.db.WithContext(ctx).First(&group, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("组不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询组失败: %w", err)
	}
	return &group, nil
}

func (d *GroupDal) GetWithUsers(ctx context.Context, id int64) (*model.Group, []model.User, error) {
	var group model.Group
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Preload("Users", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).First(&group, id).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("组不存在")
		}
		if err != nil {
			return fmt.Errorf("查询组失败: %w", err)
		}
		return d.ensureGroupOwner(ctx, tx, &group)
	})
	if err != nil {
		return nil, nil, err
	}
	log.Infof("group: %v", group)
	return &group, group.Users, nil
}

func (d *GroupDal) List(ctx context.Context, page, size int64) ([]model.Group, int64, error) {
	var list []model.Group
	var total int64

	if err := d.db.WithContext(ctx).Model(&model.Group{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询组总数失败: %w", err)
	}

	offset := (page - 1) * size
	if err := d.db.WithContext(ctx).
		Preload("Users", func(db *gorm.DB) *gorm.DB {
			return db.Order("id ASC")
		}).
		Order("id DESC").
		Limit(int(size)).
		Offset(int(offset)).
		Find(&list).Error; err != nil {
		return nil, 0, fmt.Errorf("查询组列表失败: %w", err)
	}

	return list, total, nil
}

func (d *GroupDal) Update(ctx context.Context, id int64, name, describe string) error {
	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if describe != "" {
		updates["describe"] = describe
	}

	result := d.db.WithContext(ctx).Model(&model.Group{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新组失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("组不存在")
	}
	return nil
}

func (d *GroupDal) clearUserCache(ctx context.Context, userId int64) {
	if d.rdb == nil || userId == 0 {
		return
	}
	_ = d.rdb.Del(ctx, fmt.Sprintf("user:%d:profile", userId)).Err()
}

func (d *GroupDal) ensureGroupOwner(ctx context.Context, tx *gorm.DB, group *model.Group) error {
	if group == nil || group.ID == 0 || group.OwnerId != 0 {
		return nil
	}
	var ownerIds []int64
	if err := tx.WithContext(ctx).Model(&model.User{}).
		Where("group_id = ?", group.ID).
		Order("id ASC").
		Limit(1).
		Pluck("id", &ownerIds).Error; err != nil {
		return fmt.Errorf("查询团队队长失败: %w", err)
	}
	if len(ownerIds) == 0 {
		return nil
	}
	group.OwnerId = ownerIds[0]
	if err := tx.WithContext(ctx).Model(&model.Group{}).
		Where("id = ?", group.ID).
		Update("owner_id", group.OwnerId).Error; err != nil {
		return fmt.Errorf("补全团队队长失败: %w", err)
	}
	return nil
}

func (d *GroupDal) CreateTeamForUser(ctx context.Context, userId int64, name, avatar, describe string) (*model.Group, error) {
	var created model.Group
	err := data2.UpdateCacheDal(ctx, d.rdb, fmt.Sprintf("user:%d:profile", userId), func() error {
		return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var user model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userId).Error; err != nil {
				return fmt.Errorf("用户不存在")
			}
			if user.GroupId != 0 {
				return fmt.Errorf("你已经加入了团队，不能重复创建或加入团队")
			}
			created = model.Group{
				Name:     &name,
				Describe: describe,
				Avatar:   avatar,
				OwnerId:  userId,
			}
			if err := tx.Create(&created).Error; err != nil {
				return fmt.Errorf("创建团队失败: %w", err)
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userId).Update("group_id", created.ID).Error; err != nil {
				return fmt.Errorf("加入团队失败: %w", err)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (d *GroupDal) UpdateTeam(ctx context.Context, userId int64, groupId int64, name, avatar, describe string) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var group model.Group
		if err := tx.First(&group, groupId).Error; errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("团队不存在")
		} else if err != nil {
			return fmt.Errorf("查询团队失败: %w", err)
		}
		var user model.User
		if err := tx.First(&user, userId).Error; err != nil {
			return fmt.Errorf("用户不存在")
		}
		if user.GroupId != groupId {
			return fmt.Errorf("只能编辑自己的团队")
		}
		if err := d.ensureGroupOwner(ctx, tx, &group); err != nil {
			return err
		}
		if group.OwnerId != userId {
			return fmt.Errorf("只有队长可以编辑团队")
		}
		updates := map[string]interface{}{
			"name":     name,
			"avatar":   avatar,
			"describe": describe,
		}
		if err := tx.Model(&model.Group{}).Where("id = ?", groupId).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新团队失败: %w", err)
		}
		return nil
	})
}

func (d *GroupDal) InviteUser(ctx context.Context, inviterId, inviteeId int64) (*model.GroupInvite, error) {
	if inviterId == inviteeId {
		return nil, fmt.Errorf("不能邀请自己")
	}
	var created model.GroupInvite
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var inviter model.User
		if err := tx.First(&inviter, inviterId).Error; err != nil {
			return fmt.Errorf("邀请人不存在")
		}
		if inviter.GroupId == 0 {
			return fmt.Errorf("请先创建或加入团队")
		}
		var group model.Group
		if err := tx.First(&group, inviter.GroupId).Error; err != nil {
			return fmt.Errorf("团队不存在")
		}
		if err := d.ensureGroupOwner(ctx, tx, &group); err != nil {
			return err
		}
		if group.OwnerId != inviterId {
			return fmt.Errorf("只有队长可以邀请成员")
		}
		var invitee model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invitee, inviteeId).Error; err != nil {
			return fmt.Errorf("被邀请用户不存在")
		}
		if invitee.GroupId != 0 {
			return fmt.Errorf("该用户已经加入了团队")
		}
		var existing model.GroupInvite
		err := tx.Where("group_id = ? AND invitee_id = ? AND status = ?", inviter.GroupId, inviteeId, "pending").First(&existing).Error
		if err == nil {
			created = existing
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查询邀请失败: %w", err)
		}
		created = model.GroupInvite{
			GroupId:   inviter.GroupId,
			InviterId: inviterId,
			InviteeId: inviteeId,
			Status:    "pending",
		}
		if err := tx.Create(&created).Error; err != nil {
			return fmt.Errorf("创建邀请失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &created, nil
}

func (d *GroupDal) RemoveTeamMember(ctx context.Context, operatorId, memberId int64) error {
	if operatorId == memberId {
		return fmt.Errorf("队长不能通过移除成员退出团队")
	}
	return data2.UpdateCacheDal(ctx, d.rdb, fmt.Sprintf("user:%d:profile", memberId), func() error {
		return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var operator model.User
			if err := tx.First(&operator, operatorId).Error; err != nil {
				return fmt.Errorf("操作用户不存在")
			}
			if operator.GroupId == 0 {
				return fmt.Errorf("请先创建或加入团队")
			}
			var group model.Group
			if err := tx.First(&group, operator.GroupId).Error; err != nil {
				return fmt.Errorf("团队不存在")
			}
			if err := d.ensureGroupOwner(ctx, tx, &group); err != nil {
				return err
			}
			if group.OwnerId != operatorId {
				return fmt.Errorf("只有队长可以编辑成员")
			}
			var member model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&member, memberId).Error; err != nil {
				return fmt.Errorf("成员不存在")
			}
			if member.GroupId != operator.GroupId {
				return fmt.Errorf("该用户不在你的团队中")
			}
			if int64(member.ID) == group.OwnerId {
				return fmt.Errorf("不能移除队长")
			}
			if err := tx.Model(&model.User{}).Where("id = ?", memberId).Update("group_id", 0).Error; err != nil {
				return fmt.Errorf("移除成员失败: %w", err)
			}
			d.clearUserCache(ctx, memberId)
			if err := tx.Model(&model.GroupInvite{}).
				Where("invitee_id = ? AND status = ?", memberId, "pending").
				Update("status", "rejected").Error; err != nil {
				return fmt.Errorf("关闭成员待处理邀请失败: %w", err)
			}
			return nil
		})
	})
}

func (d *GroupDal) LeaveTeam(ctx context.Context, userId int64) error {
	err := data2.UpdateCacheDal(ctx, d.rdb, fmt.Sprintf("user:%d:profile", userId), func() error {
		return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var user model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userId).Error; err != nil {
				return fmt.Errorf("用户不存在")
			}
			if user.GroupId == 0 {
				return fmt.Errorf("你当前没有团队")
			}
			var group model.Group
			if err := tx.First(&group, user.GroupId).Error; err != nil {
				return fmt.Errorf("团队不存在")
			}
			if err := d.ensureGroupOwner(ctx, tx, &group); err != nil {
				return err
			}
			if group.OwnerId == userId {
				return fmt.Errorf("队长不能直接退出团队，请先解散团队")
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userId).Update("group_id", 0).Error; err != nil {
				return fmt.Errorf("退出团队失败: %w", err)
			}
			if err := tx.Model(&model.GroupInvite{}).
				Where("invitee_id = ? AND status = ?", userId, "pending").
				Update("status", "rejected").Error; err != nil {
				return fmt.Errorf("关闭待处理邀请失败: %w", err)
			}
			return nil
		})
	})
	if err != nil {
		return err
	}
	d.clearUserCache(ctx, userId)
	return nil
}

func (d *GroupDal) DisbandTeam(ctx context.Context, ownerId int64) error {
	var memberIds []int64
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var owner model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&owner, ownerId).Error; err != nil {
			return fmt.Errorf("操作用户不存在")
		}
		if owner.GroupId == 0 {
			return fmt.Errorf("你当前没有团队")
		}

		var group model.Group
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&group, owner.GroupId).Error; err != nil {
			return fmt.Errorf("团队不存在")
		}
		if err := d.ensureGroupOwner(ctx, tx, &group); err != nil {
			return err
		}
		if group.OwnerId != ownerId {
			return fmt.Errorf("只有队长可以解散团队")
		}

		if err := tx.Model(&model.User{}).
			Where("group_id = ?", owner.GroupId).
			Pluck("id", &memberIds).Error; err != nil {
			return fmt.Errorf("查询团队成员失败: %w", err)
		}
		if err := tx.Model(&model.User{}).
			Where("group_id = ?", owner.GroupId).
			Update("group_id", 0).Error; err != nil {
			return fmt.Errorf("重置团队成员失败: %w", err)
		}
		if err := tx.Model(&model.GroupInvite{}).
			Where("group_id = ? AND status = ?", owner.GroupId, "pending").
			Update("status", "rejected").Error; err != nil {
			return fmt.Errorf("关闭团队邀请失败: %w", err)
		}
		if err := tx.Delete(&model.Group{}, owner.GroupId).Error; err != nil {
			return fmt.Errorf("删除团队失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, memberId := range memberIds {
		d.clearUserCache(ctx, memberId)
	}
	return nil
}

type TeamInviteView struct {
	ID          uint
	GroupId     int64
	GroupName   string
	GroupAvatar string
	InviterId   int64
	InviterName string
	Status      string
	CreatedAt   int64
}

func (d *GroupDal) ListPendingInvites(ctx context.Context, inviteeId int64) ([]TeamInviteView, error) {
	var rows []TeamInviteView
	err := d.db.WithContext(ctx).
		Table("group_invites AS gi").
		Select("gi.id, gi.group_id, COALESCE(g.name, '') AS group_name, g.avatar AS group_avatar, gi.inviter_id, u.name AS inviter_name, gi.status, EXTRACT(EPOCH FROM gi.created_at)::bigint AS created_at").
		Joins("LEFT JOIN groups AS g ON g.id = gi.group_id").
		Joins("LEFT JOIN users AS u ON u.id = gi.inviter_id").
		Where("gi.invitee_id = ? AND gi.status = ? AND gi.deleted_at IS NULL", inviteeId, "pending").
		Order("gi.id DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("查询邀请失败: %w", err)
	}
	return rows, nil
}

func (d *GroupDal) RespondInvite(ctx context.Context, userId int64, inviteId uint, accept bool) error {
	return data2.UpdateCacheDal(ctx, d.rdb, fmt.Sprintf("user:%d:profile", userId), func() error {
		return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var invite model.GroupInvite
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
				Where("id = ? AND invitee_id = ? AND status = ?", inviteId, userId, "pending").
				First(&invite).Error; errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("邀请不存在或已处理")
			} else if err != nil {
				return fmt.Errorf("查询邀请失败: %w", err)
			}
			if !accept {
				return tx.Model(&model.GroupInvite{}).Where("id = ?", inviteId).Update("status", "rejected").Error
			}
			var user model.User
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, userId).Error; err != nil {
				return fmt.Errorf("用户不存在")
			}
			if user.GroupId != 0 {
				return fmt.Errorf("你已经加入了团队，不能再接受邀请")
			}
			var group model.Group
			if err := tx.First(&group, invite.GroupId).Error; err != nil {
				return fmt.Errorf("团队不存在")
			}
			if err := tx.Model(&model.User{}).Where("id = ?", userId).Update("group_id", invite.GroupId).Error; err != nil {
				return fmt.Errorf("加入团队失败: %w", err)
			}
			if err := tx.Model(&model.GroupInvite{}).Where("id = ?", inviteId).Update("status", "accepted").Error; err != nil {
				return fmt.Errorf("更新邀请状态失败: %w", err)
			}
			if err := tx.Model(&model.GroupInvite{}).
				Where("invitee_id = ? AND status = ? AND id <> ?", userId, "pending", inviteId).
				Update("status", "rejected").Error; err != nil {
				return fmt.Errorf("关闭其他邀请失败: %w", err)
			}
			return nil
		})
	})
}
