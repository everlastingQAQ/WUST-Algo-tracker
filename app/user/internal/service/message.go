package service

import (
	"context"
	"strings"

	"cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/user/internal/data/dal"

	"github.com/go-kratos/kratos/v2/errors"
)

type MessageService struct {
	messageDal *dal.MessageDal
	profileDal *dal.ProfileDal
}

type MessageUserReply struct {
	UserId   int64  `json:"userId"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Deleted  bool   `json:"deleted"`
}

type ConversationItem struct {
	ThreadId           uint             `json:"threadId"`
	OtherUser          MessageUserReply `json:"otherUser"`
	LastMessagePreview string           `json:"lastMessagePreview"`
	LastSenderId       int64            `json:"lastSenderId"`
	LastSentAt         int64            `json:"lastSentAt"`
	UnreadCount        int64            `json:"unreadCount"`
}

type ConversationListRequest struct {
	Page     int64 `json:"page" form:"page"`
	PageSize int64 `json:"pageSize" form:"pageSize"`
}

type ConversationListReply struct {
	Success     bool                `json:"success"`
	List        []*ConversationItem `json:"list"`
	Total       int64               `json:"total"`
	Page        int64               `json:"page"`
	PageSize    int64               `json:"pageSize"`
	UnreadCount int64               `json:"unreadCount"`
}

type ThreadMessagesRequest struct {
	UserId   int64 `json:"userId" form:"userId"`
	Page     int64 `json:"page" form:"page"`
	PageSize int64 `json:"pageSize" form:"pageSize"`
}

type DirectMessageItem struct {
	Id         uint   `json:"id"`
	ThreadId   uint   `json:"threadId"`
	SenderId   int64  `json:"senderId"`
	ReceiverId int64  `json:"receiverId"`
	Content    string `json:"content"`
	IsRead     bool   `json:"isRead"`
	CreatedAt  int64  `json:"createdAt"`
}

type ThreadMessagesReply struct {
	Success   bool                 `json:"success"`
	OtherUser MessageUserReply     `json:"otherUser"`
	List      []*DirectMessageItem `json:"list"`
	Total     int64                `json:"total"`
	Page      int64                `json:"page"`
	PageSize  int64                `json:"pageSize"`
}

type SendMessageRequest struct {
	ReceiverId int64  `json:"receiverId"`
	Content    string `json:"content"`
}

type SendMessageReply struct {
	Success bool              `json:"success"`
	Message string            `json:"message"`
	Data    DirectMessageItem `json:"data"`
}

type MarkMessageReadRequest struct {
	UserId int64 `json:"userId"`
}

type MessageSuccessReply struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type UnreadCountReply struct {
	Success     bool  `json:"success"`
	UnreadCount int64 `json:"unreadCount"`
}

type BroadcastMessageRequest struct {
	Content string `json:"content"`
}

type BroadcastMessageReply struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Count   int64  `json:"count"`
}

func NewMessageService(messageDal *dal.MessageDal, profileDal *dal.ProfileDal) *MessageService {
	return &MessageService{messageDal: messageDal, profileDal: profileDal}
}

func currentUserId(ctx context.Context) (int64, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return 0, errors.Unauthorized("未登录", "请先登录")
	}
	return int64(current.UserID), nil
}

func normalizePage(page, pageSize int64) (int64, int64) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 50 {
		pageSize = 50
	}
	return page, pageSize
}

func normalizeMessageContent(content string) (string, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return "", errors.BadRequest("参数错误", "消息不能为空")
	}
	if len([]rune(content)) > 1000 {
		return "", errors.BadRequest("参数错误", "消息不能超过1000字")
	}
	return content, nil
}

func replyUser(v dal.MessageUserView) MessageUserReply {
	name := v.Name
	if v.Deleted || name == "" {
		name = "已注销用户"
	}
	return MessageUserReply{
		UserId:   int64(v.ID),
		Username: v.Username,
		Name:     name,
		Avatar:   v.Avatar,
		Deleted:  v.Deleted,
	}
}

func replyMessage(v *dal.DirectMessageView) DirectMessageItem {
	return DirectMessageItem{
		Id:         v.ID,
		ThreadId:   v.ThreadId,
		SenderId:   v.SenderId,
		ReceiverId: v.ReceiverId,
		Content:    v.Content,
		IsRead:     v.IsRead,
		CreatedAt:  v.CreatedAt,
	}
}

func (s *MessageService) Conversations(ctx context.Context, req *ConversationListRequest) (*ConversationListReply, error) {
	userId, err := currentUserId(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	rows, total, err := s.messageDal.ListConversations(ctx, userId, page, pageSize)
	if err != nil {
		return nil, errors.InternalServer("查询失败", err.Error())
	}
	unread, err := s.messageDal.UnreadCount(ctx, userId)
	if err != nil {
		return nil, errors.InternalServer("查询失败", err.Error())
	}
	reply := &ConversationListReply{
		Success:     true,
		List:        make([]*ConversationItem, 0, len(rows)),
		Total:       total,
		Page:        page,
		PageSize:    pageSize,
		UnreadCount: unread,
	}
	for _, row := range rows {
		reply.List = append(reply.List, &ConversationItem{
			ThreadId:           row.ThreadId,
			OtherUser:          replyUser(row.OtherUser),
			LastMessagePreview: row.LastMessagePreview,
			LastSenderId:       row.LastSenderId,
			LastSentAt:         row.LastSentAt,
			UnreadCount:        row.UnreadCount,
		})
	}
	return reply, nil
}

func (s *MessageService) Thread(ctx context.Context, req *ThreadMessagesRequest) (*ThreadMessagesReply, error) {
	userId, err := currentUserId(ctx)
	if err != nil {
		return nil, err
	}
	if req.UserId == 0 {
		return nil, errors.BadRequest("参数错误", "请选择用户")
	}
	if req.UserId == userId {
		return nil, errors.BadRequest("参数错误", "不能和自己私信")
	}
	page, pageSize := normalizePage(req.Page, req.PageSize)
	rows, total, other, err := s.messageDal.ListThreadMessages(ctx, userId, req.UserId, page, pageSize)
	if err != nil {
		return nil, errors.BadRequest("查询失败", err.Error())
	}
	reply := &ThreadMessagesReply{
		Success:   true,
		OtherUser: replyUser(*other),
		List:      make([]*DirectMessageItem, 0, len(rows)),
		Total:     total,
		Page:      page,
		PageSize:  pageSize,
	}
	for _, row := range rows {
		item := replyMessage(row)
		reply.List = append(reply.List, &item)
	}
	return reply, nil
}

func (s *MessageService) Send(ctx context.Context, req *SendMessageRequest) (*SendMessageReply, error) {
	userId, err := currentUserId(ctx)
	if err != nil {
		return nil, err
	}
	if req.ReceiverId == 0 {
		return nil, errors.BadRequest("参数错误", "请选择接收用户")
	}
	if req.ReceiverId == userId {
		return nil, errors.BadRequest("参数错误", "不能给自己发私信")
	}
	content, contentErr := normalizeMessageContent(req.Content)
	if contentErr != nil {
		return nil, contentErr
	}
	created, err := s.messageDal.Send(ctx, userId, req.ReceiverId, content)
	if err != nil {
		return nil, errors.BadRequest("发送失败", err.Error())
	}
	item := replyMessage(created)
	return &SendMessageReply{
		Success: true,
		Message: "发送成功",
		Data:    item,
	}, nil
}

func (s *MessageService) MarkRead(ctx context.Context, req *MarkMessageReadRequest) (*MessageSuccessReply, error) {
	userId, err := currentUserId(ctx)
	if err != nil {
		return nil, err
	}
	if req.UserId == 0 {
		return nil, errors.BadRequest("参数错误", "请选择用户")
	}
	if req.UserId == userId {
		return nil, errors.BadRequest("参数错误", "不能和自己私信")
	}
	if err := s.messageDal.MarkThreadRead(ctx, userId, req.UserId); err != nil {
		return nil, errors.InternalServer("标记失败", err.Error())
	}
	return &MessageSuccessReply{Success: true, Message: "已读"}, nil
}

func (s *MessageService) UnreadCount(ctx context.Context) (*UnreadCountReply, error) {
	userId, err := currentUserId(ctx)
	if err != nil {
		return nil, err
	}
	unread, err := s.messageDal.UnreadCount(ctx, userId)
	if err != nil {
		return nil, errors.InternalServer("查询失败", err.Error())
	}
	return &UnreadCountReply{Success: true, UnreadCount: unread}, nil
}

func (s *MessageService) Broadcast(ctx context.Context, req *BroadcastMessageRequest) (*BroadcastMessageReply, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}
	sender, err := s.profileDal.GetById(ctx, int64(current.UserID))
	if err != nil {
		return nil, errors.Forbidden("权限不足", "用户不存在或已被禁用")
	}
	if !canBroadcastMessage(sender.RoleID) {
		return nil, errors.Forbidden("权限不足", "仅管理员或教练可以群发消息")
	}
	content, contentErr := normalizeMessageContent(req.Content)
	if contentErr != nil {
		return nil, contentErr
	}
	ids, err := s.messageDal.ListActiveUserIds(ctx)
	if err != nil {
		return nil, errors.InternalServer("查询失败", err.Error())
	}
	var sent int64
	senderId := int64(current.UserID)
	for _, receiverId := range ids {
		if receiverId == senderId {
			continue
		}
		if _, err := s.messageDal.Send(ctx, senderId, receiverId, content); err != nil {
			return nil, errors.InternalServer("群发失败", err.Error())
		}
		sent++
	}
	recordUserOperation(ctx, s.profileDal, "message.broadcast", "message", 0, map[string]any{
		"sent": sent,
	})
	return &BroadcastMessageReply{Success: true, Message: "群发成功", Count: sent}, nil
}
