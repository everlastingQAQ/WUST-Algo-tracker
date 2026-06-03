package dal

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cwxu-algo/app/user/internal/data"
	"cwxu-algo/app/user/internal/data/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MessageDal struct {
	db *gorm.DB
}

type MessageUserView struct {
	ID       uint
	Username string
	Name     string
	Avatar   string
	Deleted  bool
}

type ConversationView struct {
	ThreadId           uint
	OtherUser          MessageUserView
	LastMessagePreview string
	LastSenderId       int64
	LastSentAt         int64
	UnreadCount        int64
}

type DirectMessageView struct {
	ID         uint
	ThreadId   uint
	SenderId   int64
	ReceiverId int64
	Content    string
	IsRead     bool
	CreatedAt  int64
}

func NewMessageDal(data *data.Data) *MessageDal {
	return &MessageDal{db: data.DB}
}

func messagePair(userId, otherId int64) (int64, int64, string) {
	if userId < otherId {
		return userId, otherId, fmt.Sprintf("%d:%d", userId, otherId)
	}
	return otherId, userId, fmt.Sprintf("%d:%d", otherId, userId)
}

func (d *MessageDal) GetUserView(ctx context.Context, userId int64, includeDeleted bool) (*MessageUserView, error) {
	var user model.User
	db := d.db.WithContext(ctx)
	if includeDeleted {
		db = db.Unscoped()
	}
	err := db.Select("id", "username", "name", "avatar", "deleted_at").First(&user, userId).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("用户不存在")
	}
	if err != nil {
		return nil, err
	}
	return &MessageUserView{
		ID:       user.ID,
		Username: user.Username,
		Name:     user.Name,
		Avatar:   user.Avatar,
		Deleted:  user.DeletedAt.Valid,
	}, nil
}

func messagePreview(content string) string {
	content = strings.TrimSpace(content)
	if len([]rune(content)) <= 80 {
		return content
	}
	return string([]rune(content)[:80]) + "..."
}

func (d *MessageDal) Send(ctx context.Context, senderId, receiverId int64, content string) (*DirectMessageView, error) {
	if _, err := d.GetUserView(ctx, receiverId, false); err != nil {
		return nil, err
	}
	userA, userB, pairKey := messagePair(senderId, receiverId)
	var created model.DirectMessage
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var thread model.DirectMessageThread
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("pair_key = ?", pairKey).First(&thread).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			thread = model.DirectMessageThread{
				PairKey:    pairKey,
				UserAId:    userA,
				UserBId:    userB,
				LastSentAt: time.Now(),
			}
			if err := tx.Create(&thread).Error; err != nil {
				return fmt.Errorf("创建会话失败: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("查询会话失败: %w", err)
		}

		created = model.DirectMessage{
			ThreadId:   thread.ID,
			SenderId:   senderId,
			ReceiverId: receiverId,
			Content:    content,
			IsRead:     false,
		}
		if err := tx.Create(&created).Error; err != nil {
			return fmt.Errorf("发送消息失败: %w", err)
		}

		updates := map[string]interface{}{
			"last_message_id":      created.ID,
			"last_message_preview": messagePreview(content),
			"last_sender_id":       senderId,
			"last_sent_at":         created.CreatedAt,
		}
		if receiverId == thread.UserAId {
			updates["unread_a"] = gorm.Expr("unread_a + 1")
		} else {
			updates["unread_b"] = gorm.Expr("unread_b + 1")
		}
		if err := tx.Model(&model.DirectMessageThread{}).Where("id = ?", thread.ID).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新会话失败: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return messageView(created), nil
}

func messageView(m model.DirectMessage) *DirectMessageView {
	return &DirectMessageView{
		ID:         m.ID,
		ThreadId:   m.ThreadId,
		SenderId:   m.SenderId,
		ReceiverId: m.ReceiverId,
		Content:    m.Content,
		IsRead:     m.IsRead,
		CreatedAt:  m.CreatedAt.Unix(),
	}
}

func (d *MessageDal) ListConversations(ctx context.Context, userId int64, page, pageSize int64) ([]*ConversationView, int64, error) {
	var total int64
	if err := d.db.WithContext(ctx).Model(&model.DirectMessageThread{}).
		Where("user_a_id = ? OR user_b_id = ?", userId, userId).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var threads []model.DirectMessageThread
	if err := d.db.WithContext(ctx).
		Where("user_a_id = ? OR user_b_id = ?", userId, userId).
		Order("last_sent_at DESC, id DESC").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize)).
		Find(&threads).Error; err != nil {
		return nil, 0, err
	}

	otherIds := make([]int64, 0, len(threads))
	for _, t := range threads {
		if t.UserAId == userId {
			otherIds = append(otherIds, t.UserBId)
		} else {
			otherIds = append(otherIds, t.UserAId)
		}
	}
	users, err := d.userViewMap(ctx, otherIds)
	if err != nil {
		return nil, 0, err
	}

	list := make([]*ConversationView, 0, len(threads))
	for _, t := range threads {
		otherId := t.UserAId
		unread := t.UnreadB
		if t.UserAId == userId {
			otherId = t.UserBId
			unread = t.UnreadA
		}
		other := fallbackUserView(otherId)
		if u, ok := users[otherId]; ok {
			other = u
		}
		list = append(list, &ConversationView{
			ThreadId:           t.ID,
			OtherUser:          other,
			LastMessagePreview: t.LastMessagePreview,
			LastSenderId:       t.LastSenderId,
			LastSentAt:         t.LastSentAt.Unix(),
			UnreadCount:        unread,
		})
	}
	return list, total, nil
}

func (d *MessageDal) userViewMap(ctx context.Context, ids []int64) (map[int64]MessageUserView, error) {
	result := make(map[int64]MessageUserView)
	if len(ids) == 0 {
		return result, nil
	}
	var users []model.User
	if err := d.db.WithContext(ctx).Unscoped().
		Select("id", "username", "name", "avatar", "deleted_at").
		Where("id IN ?", ids).
		Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		result[int64(u.ID)] = MessageUserView{
			ID:       u.ID,
			Username: u.Username,
			Name:     u.Name,
			Avatar:   u.Avatar,
			Deleted:  u.DeletedAt.Valid,
		}
	}
	return result, nil
}

func fallbackUserView(userId int64) MessageUserView {
	return MessageUserView{
		ID:       uint(userId),
		Username: "",
		Name:     "已注销用户",
		Avatar:   "",
		Deleted:  true,
	}
}

func (d *MessageDal) ListThreadMessages(ctx context.Context, userId, otherId int64, page, pageSize int64) ([]*DirectMessageView, int64, *MessageUserView, error) {
	_, _, pairKey := messagePair(userId, otherId)
	var thread model.DirectMessageThread
	if err := d.db.WithContext(ctx).Where("pair_key = ? AND (user_a_id = ? OR user_b_id = ?)", pairKey, userId, userId).First(&thread).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			other, otherErr := d.GetUserView(ctx, otherId, true)
			if otherErr != nil {
				return nil, 0, nil, otherErr
			}
			return []*DirectMessageView{}, 0, other, nil
		}
		return nil, 0, nil, err
	}

	var total int64
	if err := d.db.WithContext(ctx).Model(&model.DirectMessage{}).Where("thread_id = ?", thread.ID).Count(&total).Error; err != nil {
		return nil, 0, nil, err
	}

	var messages []model.DirectMessage
	if err := d.db.WithContext(ctx).
		Where("thread_id = ?", thread.ID).
		Order("id DESC").
		Limit(int(pageSize)).
		Offset(int((page - 1) * pageSize)).
		Find(&messages).Error; err != nil {
		return nil, 0, nil, err
	}
	sort.Slice(messages, func(i, j int) bool { return messages[i].ID < messages[j].ID })

	other, err := d.GetUserView(ctx, otherId, true)
	if err != nil {
		other = &MessageUserView{
			ID:      uint(otherId),
			Name:    "已注销用户",
			Deleted: true,
		}
	}
	result := make([]*DirectMessageView, 0, len(messages))
	for _, m := range messages {
		result = append(result, messageView(m))
	}
	return result, total, other, nil
}

func (d *MessageDal) MarkThreadRead(ctx context.Context, userId, otherId int64) error {
	_, _, pairKey := messagePair(userId, otherId)
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var thread model.DirectMessageThread
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("pair_key = ? AND (user_a_id = ? OR user_b_id = ?)", pairKey, userId, userId).
			First(&thread).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		now := time.Now()
		if err := tx.Model(&model.DirectMessage{}).
			Where("thread_id = ? AND receiver_id = ? AND is_read = ?", thread.ID, userId, false).
			Updates(map[string]interface{}{"is_read": true, "read_at": now}).Error; err != nil {
			return err
		}
		field := "unread_b"
		if thread.UserAId == userId {
			field = "unread_a"
		}
		return tx.Model(&model.DirectMessageThread{}).Where("id = ?", thread.ID).Update(field, 0).Error
	})
}

func (d *MessageDal) UnreadCount(ctx context.Context, userId int64) (int64, error) {
	var sumA, sumB int64
	if err := d.db.WithContext(ctx).Model(&model.DirectMessageThread{}).Where("user_a_id = ?", userId).Select("COALESCE(SUM(unread_a), 0)").Scan(&sumA).Error; err != nil {
		return 0, err
	}
	if err := d.db.WithContext(ctx).Model(&model.DirectMessageThread{}).Where("user_b_id = ?", userId).Select("COALESCE(SUM(unread_b), 0)").Scan(&sumB).Error; err != nil {
		return 0, err
	}
	return sumA + sumB, nil
}

func (d *MessageDal) ListActiveUserIds(ctx context.Context) ([]int64, error) {
	var ids []int64
	err := d.db.WithContext(ctx).
		Model(&model.User{}).
		Order("id ASC").
		Pluck("id", &ids).Error
	return ids, err
}
