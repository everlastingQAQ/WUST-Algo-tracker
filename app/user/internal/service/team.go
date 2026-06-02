package service

import (
	"context"
	"cwxu-algo/api/user/v1/group"
	"cwxu-algo/app/common/utils/auth"
	"cwxu-algo/app/user/internal/data/model"

	"github.com/go-kratos/kratos/v2/errors"
)

type TeamCreateRequest struct {
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Describe string `json:"describe"`
}

type TeamDetailRequest struct {
	GroupId int64 `json:"groupId" form:"groupId"`
}

type TeamUpdateRequest struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Describe string `json:"describe"`
}

type TeamInviteRequest struct {
	InviteeId int64 `json:"inviteeId"`
}

type TeamRemoveMemberRequest struct {
	UserId int64 `json:"userId"`
}

type TeamRespondInviteRequest struct {
	InviteId uint `json:"inviteId"`
	Accept   bool `json:"accept"`
}

type TeamSuccessReply struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type TeamCreateReply struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	Avatar   string `json:"avatar"`
	Describe string `json:"describe"`
	OwnerId  int64  `json:"ownerId"`
}

type TeamDetailReply struct {
	Success  bool          `json:"success"`
	Id       int64         `json:"id"`
	Name     string        `json:"name"`
	Avatar   string        `json:"avatar"`
	Describe string        `json:"describe"`
	OwnerId  int64         `json:"ownerId"`
	Users    []*group.User `json:"users"`
}

type TeamInviteReply struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	InviteId uint   `json:"inviteId"`
}

type TeamInviteItem struct {
	Id          uint   `json:"id"`
	GroupId     int64  `json:"groupId"`
	GroupName   string `json:"groupName"`
	GroupAvatar string `json:"groupAvatar"`
	InviterId   int64  `json:"inviterId"`
	InviterName string `json:"inviterName"`
	Status      string `json:"status"`
	CreatedAt   int64  `json:"createdAt"`
}

type TeamInviteListReply struct {
	Success bool              `json:"success"`
	List    []*TeamInviteItem `json:"list"`
}

func groupName(m *model.Group) string {
	if m == nil || m.Name == nil {
		return ""
	}
	return *m.Name
}

func (g *GroupService) TeamDetail(ctx context.Context, req *TeamDetailRequest) (*TeamDetailReply, error) {
	if req.GroupId == 0 {
		return &TeamDetailReply{
			Success:  true,
			Id:       0,
			Name:     "无团队",
			Describe: "当前用户暂未加入团队",
			Users:    []*group.User{},
		}, nil
	}
	groupModel, users, err := g.groupUseCase.GetWithUsers(ctx, req.GroupId)
	if err != nil {
		return nil, errors.InternalServer("查询失败", err.Error())
	}
	reply := &TeamDetailReply{
		Success:  true,
		Id:       int64(groupModel.ID),
		Name:     groupName(groupModel),
		Avatar:   groupModel.Avatar,
		Describe: groupModel.Describe,
		OwnerId:  groupModel.OwnerId,
		Users:    make([]*group.User, 0, len(users)),
	}
	for _, u := range users {
		reply.Users = append(reply.Users, &group.User{
			UserId:   uint64(u.ID),
			Username: u.Username,
			Name:     u.Name,
			GroupId:  u.GroupId,
			Avatar:   u.Avatar,
		})
	}
	return reply, nil
}

func (g *GroupService) CreateTeam(ctx context.Context, req *TeamCreateRequest) (*TeamCreateReply, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}
	if req.Name == "" {
		return nil, errors.BadRequest("参数错误", "团队名称不能为空")
	}
	created, err := g.groupDal.CreateTeamForUser(ctx, int64(current.UserID), req.Name, req.Avatar, req.Describe)
	if err != nil {
		return nil, errors.BadRequest("创建失败", err.Error())
	}
	return &TeamCreateReply{
		Success:  true,
		Message:  "团队创建成功",
		Id:       int64(created.ID),
		Name:     groupName(created),
		Avatar:   created.Avatar,
		Describe: created.Describe,
		OwnerId:  created.OwnerId,
	}, nil
}

func (g *GroupService) UpdateTeam(ctx context.Context, req *TeamUpdateRequest) (*TeamSuccessReply, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}
	if req.Id == 0 {
		return nil, errors.BadRequest("参数错误", "团队ID不能为空")
	}
	if req.Name == "" {
		return nil, errors.BadRequest("参数错误", "团队名称不能为空")
	}
	if err := g.groupDal.UpdateTeam(ctx, int64(current.UserID), req.Id, req.Name, req.Avatar, req.Describe); err != nil {
		return nil, errors.BadRequest("更新失败", err.Error())
	}
	return &TeamSuccessReply{Success: true, Message: "团队资料已更新"}, nil
}

func (g *GroupService) InviteTeamMember(ctx context.Context, req *TeamInviteRequest) (*TeamInviteReply, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}
	if req.InviteeId == 0 {
		return nil, errors.BadRequest("参数错误", "请选择邀请用户")
	}
	invite, err := g.groupDal.InviteUser(ctx, int64(current.UserID), req.InviteeId)
	if err != nil {
		return nil, errors.BadRequest("邀请失败", err.Error())
	}
	return &TeamInviteReply{Success: true, Message: "邀请已发送", InviteId: invite.ID}, nil
}

func (g *GroupService) RemoveTeamMember(ctx context.Context, req *TeamRemoveMemberRequest) (*TeamSuccessReply, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}
	if req.UserId == 0 {
		return nil, errors.BadRequest("参数错误", "成员ID不能为空")
	}
	if err := g.groupDal.RemoveTeamMember(ctx, int64(current.UserID), req.UserId); err != nil {
		return nil, errors.BadRequest("移除失败", err.Error())
	}
	return &TeamSuccessReply{Success: true, Message: "成员已移出团队"}, nil
}

func (g *GroupService) ListTeamInvites(ctx context.Context) (*TeamInviteListReply, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}
	rows, err := g.groupDal.ListPendingInvites(ctx, int64(current.UserID))
	if err != nil {
		return nil, errors.InternalServer("查询失败", err.Error())
	}
	reply := &TeamInviteListReply{Success: true, List: make([]*TeamInviteItem, 0, len(rows))}
	for _, row := range rows {
		reply.List = append(reply.List, &TeamInviteItem{
			Id:          row.ID,
			GroupId:     row.GroupId,
			GroupName:   row.GroupName,
			GroupAvatar: row.GroupAvatar,
			InviterId:   row.InviterId,
			InviterName: row.InviterName,
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
		})
	}
	return reply, nil
}

func (g *GroupService) RespondTeamInvite(ctx context.Context, req *TeamRespondInviteRequest) (*TeamSuccessReply, error) {
	current := auth.GetCurrentUser(ctx)
	if current == nil || current.UserID == 0 {
		return nil, errors.Unauthorized("未登录", "请先登录")
	}
	if req.InviteId == 0 {
		return nil, errors.BadRequest("参数错误", "邀请ID不能为空")
	}
	if err := g.groupDal.RespondInvite(ctx, int64(current.UserID), req.InviteId, req.Accept); err != nil {
		return nil, errors.BadRequest("处理失败", err.Error())
	}
	if req.Accept {
		return &TeamSuccessReply{Success: true, Message: "已加入团队"}, nil
	}
	return &TeamSuccessReply{Success: true, Message: "已拒绝邀请"}, nil
}
