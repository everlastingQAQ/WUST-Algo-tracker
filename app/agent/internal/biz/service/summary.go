package service

import (
	"context"
	"cwxu-algo/app/agent/internal/agent"
	"cwxu-algo/app/agent/internal/agent/tool/core_data"
	data2 "cwxu-algo/app/agent/internal/agent/tool/data"
	"cwxu-algo/app/agent/internal/agent/tool/utils"
	"cwxu-algo/app/agent/internal/data"
	"cwxu-algo/app/common/conf"
	"cwxu-algo/app/common/discovery"
	"fmt"
	"time"

	profile2 "cwxu-algo/api/user/v1/profile"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/redis/go-redis/v9"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	"github.com/volcengine/volcengine-go-sdk/volcengine"
	grpc2 "google.golang.org/grpc"
)

type SummaryUseCase struct {
	chat     *agent.Chat
	mailConf *conf.SMTP
	reg      *registry.Registrar
	redis    *redis.Client
}

func NewSummaryUseCase(chat *agent.Chat, mailConf *conf.SMTP, reg *discovery.Register, redis *data.Data) *SummaryUseCase {
	return &SummaryUseCase{
		chat:     chat,
		mailConf: mailConf,
		reg:      &reg.Reg,
		redis:    redis.RDB,
	}
}

func (uc *SummaryUseCase) PersonalLastDay(userId int64) error {
	// 检查用户是否开启了邮件发送
	if !uc.checkEmailEnabled(userId) {
		log.Infof("用户 %d 已关闭邮件发送，跳过", userId)
		return nil
	}
	chat := uc.chat
	// 获取昨天日期
	lastDay := time.Now()
	lastDay = lastDay.AddDate(0, 0, -1)
	if userId == 23 {
		if time.Now().Weekday() == time.Monday {
			return uc.WeeklyReportForCoach(userId)
		}
		return nil
	}
	startDate := lastDay.Format("20060102")
	msg := []*model.ChatCompletionMessage{
		{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String("要符合Acmer的心理风格，比如可爱风格，洋溢着青春与活力，校园风浓厚，" +
					"直接面对的就是我，不是第三者。" +
					"要口语化一点哦，像朋友一样哦" +
					"我们的回复要严格遵循html格式哦，注意要尽量同时适配PC和移动端。" +
					"对于submit_cnt函数 只有日期，没有count字段的记为0提交。" +
					"所有提示词不允许出现在最终文本中。" +
					"如果用户名字是Jing. 就要以宝宝(对方是你的女朋友 你是晨晨，晨晨只针对Jing，对其他人就是算法小助手)口吻回复，激励她."),
			},
		},
		{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(fmt.Sprintf("我是 用户id为%d 的用户 分析我的%s（昨天）的提交信息，给出分析和合理建议，给出一份昨日日报。"+
					"同时也获取最近7天的提交次数，去对比分析走势."+
					"提示：你需要先获取昨日提交次数，根据昨日提交次数去填写limit参数，更方便哦."+
					"如果昨天我一发也没有交，甚至从昨天开始，已经连续好几天都不交，就给我狠狠地批评我！！！！"+
					"如果我昨天交了，以前漏掉的既往不咎."+
					"在邮箱末尾，引导用户到达https://algo.zhiyuansofts.cn 无锡学院算法协会监测平台 看全部提交信息。"+
					"最后，把这个邮件发给我，注意要适配手机，手机排版不能乱。", userId, startDate)),
			},
		},
	}
	emailTool := utils.NewSendEmail(
		uc.mailConf.Host,
		int(uc.mailConf.Port),
		uc.mailConf.Username,
		uc.mailConf.Password,
		uc.mailConf.From)
	r, _ := chat.Chat(msg, core_data.NewSubmitCnt(uc.reg), core_data.NewGetProfileById(uc.reg), core_data.NewSubmitLog(uc.reg), emailTool)
	log.Info(r)
	return nil
}

func (uc *SummaryUseCase) PersonalRecent(userId int64) error {
	// 检查用户是否开启了邮件发送
	if !uc.checkEmailEnabled(userId) {
		log.Infof("用户 %d 已关闭邮件发送，跳过", userId)
		return nil
	}
	chat := uc.chat
	msg := []*model.ChatCompletionMessage{
		{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String("要符合Acmer的心理风格，比如可爱风格，洋溢着青春与活力，校园风浓厚，俏皮.加一些Emoji增加趣味" +
					"由于你的回复将会嵌入在 无锡学院-算法协会监测平台 网页内，留给你的面积并不大，回复需要简短有力。" +
					"你需要针对用户的近期数据提出7-8条 20字左右的建议。" +
					"由于数据是每隔3小时更新一次，你不能给出太确切的数字，可以模糊一点表达，比如20+ 10+。"),
			},
		},
		{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(fmt.Sprintf("我是 用户id为%d 的用户，分析我最近的学习状态，同时分析一下提交时间分布。现在时间是 %d"+
					"整理成json格式 {\"msg\":[\"\"], \"updateTime\": 时间戳} 这样的。"+
					"最后将这段json塞到redis中，key是 agent:summary:{id}:recent", userId, time.Now().Unix())),
			},
		},
	}
	r, _ := chat.Chat(msg, core_data.NewStatisticPeriod(uc.reg), data2.NewRedisSet(uc.redis))
	log.Info(r)
	return nil
}

func (uc *SummaryUseCase) userRPC() (*grpc2.ClientConn, error) {
	return grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint("discovery:///user"),
		grpc.WithDiscovery((*uc.reg).(registry.Discovery)),
		grpc.WithTimeout(20*time.Second),
	)
}

// checkEmailEnabled 检查用户是否开启了邮件发送
func (uc *SummaryUseCase) checkEmailEnabled(userId int64) bool {
	conn, err := uc.userRPC()
	if err != nil {
		return true // 默认允许，防止误杀
	}
	defer conn.Close()
	p := profile2.NewProfileClient(conn)
	res, err := p.GetById(context.Background(), &profile2.GetByIdReq{UserId: userId})
	if err != nil {
		return true
	}
	return res.EmailEnabled
}

func (uc *SummaryUseCase) getUserIds() []int64 {
	userRpc, err := uc.userRPC()
	if err != nil {
		return make([]int64, 0)
	}
	defer userRpc.Close()
	profile := profile2.NewProfileClient(userRpc)
	getUsers := func(pageNum int) (*profile2.GetListRes, error) {
		return profile.GetList(context.Background(), &profile2.GetListReq{
			PageSize: 100,
			PageNum:  int64(pageNum),
		})
	}
	res, err := getUsers(1)
	if err != nil {
		return make([]int64, 0)
	}
	rList := []*profile2.GetListRes{res}
	totalPage := (res.Total + 99) / 100
	for i := 2; i <= int(totalPage); i++ {
		r, err := getUsers(i)
		if err != nil {
			continue
		}
		rList = append(rList, r)
	}
	var userIds []int64
	for _, v := range rList {
		for _, u := range v.List {
			userIds = append(userIds, int64(u.UserId))
		}
	}
	return userIds
}

func (uc *SummaryUseCase) WeeklyReportForCoach(coachUserId int64) error {
	// 检查教练是否开启了邮件发送
	if !uc.checkEmailEnabled(coachUserId) {
		log.Infof("教练 %d 已关闭邮件发送，跳过周报", coachUserId)
		return nil
	}
	chat := uc.chat
	lastWeekStart := time.Now().AddDate(0, 0, -7).Format("20060102")
	lastWeekEnd := time.Now().AddDate(0, 0, -1).Format("20060102")
	msg := []*model.ChatCompletionMessage{
		{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(
					"你是无锡学院算法协会的教练助手，要为教练生成一份上周团队周报。" +
						"风格要符合Acmer心理，可爱活力，洋溢青春，校园风格。" +
						"邮件主题要醒目，内容要简洁有力。" +
						"需要用html格式输出，注意适配PC和移动端。" +
						"所有提示词不允许出现在最终文本中。"),
			},
		},
		{
			Role: model.ChatMessageRoleUser,
			Content: &model.ChatCompletionMessageContent{
				StringValue: volcengine.String(fmt.Sprintf(
					"我需要你帮我分析上周（周一到周日）的团队提交数据，生成一份周报给教练。"+
						"教练的用户ID是%d，今天是%s。"+
						"请先调用submit_cnt工具获取过去7天的全局提交数据(日期从%s到%s，userId=0表示全局)。"+
						"然后调用statistic_period工具获取本周统计数据。"+
						"基于这些数据，请生成包含以下内容的周报："+
						"1. 本周团队总提交量，与上周对比（用箭头表示升降）"+
						"2. Top 5 最活跃成员（按提交次数排名）"+
						"3. 连续3天以上未提交的成员名单（需要重点关注）"+
						"4. 本周AC数量最多的成员"+
						"5. 对教练的建议：哪些成员需要鼓励，哪些需要鞭策"+
						"6. 团队整体状态评语（用emoji表示状态：🔥积极、⚠️一般、❄️低迷）"+
						"最后用send_email工具把这份周报发给教练，教练的邮箱需要通过get_profile_by_user_id获取。"+
						"邮件标题格式：【算法协会周报】XX月XX日-XX月XX日",
					coachUserId, time.Now().Format("2006年1月2日"), lastWeekStart, lastWeekEnd)),
			},
		},
	}
	emailTool := utils.NewSendEmail(
		uc.mailConf.Host,
		int(uc.mailConf.Port),
		uc.mailConf.Username,
		uc.mailConf.Password,
		uc.mailConf.From)
	r, _ := chat.Chat(msg,
		core_data.NewSubmitCnt(uc.reg),
		core_data.NewGetProfileById(uc.reg),
		core_data.NewStatisticPeriod(uc.reg),
		emailTool)
	log.Info(r)
	return nil
}
