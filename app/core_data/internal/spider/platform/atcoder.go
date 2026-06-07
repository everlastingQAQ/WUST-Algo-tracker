package platform

import (
	"cwxu-algo/app/core_data/internal/data/model"
	"cwxu-algo/app/core_data/internal/spider"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type NewAtCoder struct{}
type atcJson struct {
	ID            int    `json:"id"`
	EpochSecond   int64  `json:"epoch_second"` // Unix 时间戳（秒）
	ProblemID     string `json:"problem_id"`
	ContestID     string `json:"contest_id"`
	UserID        string `json:"user_id"`
	Language      string `json:"language"`
	Result        string `json:"result"`         // 如 "AC", "WA" 等
	ExecutionTime int    `json:"execution_time"` // 执行时间（毫秒）
}

var (
	atCoderAPIBaseURL = "https://atc.luckysan.top/atcoder/atcoder-api/v3/user/submissions"
	atCoderPageSize   = 500
)

func fetchAtCoderLog(client *http.Client, baseURL string, username string, fromSecond int64) ([]atcJson, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("user", username)
	q.Set("from_second", strconv.FormatInt(fromSecond, 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "WUST-ACM-Tracker/1.1 (+https://github.com/WUSTACM/WUST-Algo-tracker)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("发起http请求失败: %s", err.Error())
	}
	defer resp.Body.Close()
	// 校验状态码
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("请求响应码错误 %d, %s", resp.StatusCode, string(body))
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("解析body错误: %s", err.Error())
	}
	var atc []atcJson
	if err := json.Unmarshal(body, &atc); err != nil {
		return nil, fmt.Errorf("解析json错误：%s", err.Error())
	}
	return atc, nil
}

func atCoderRowsToSubmitLogs(userId int64, rows []atcJson) []model.SubmitLog {
	res := make([]model.SubmitLog, 0, len(rows))
	for _, v := range rows {
		res = append(res, model.SubmitLog{
			UserID:   userId,
			Platform: spider.AtCoder,
			SubmitID: strconv.Itoa(v.ID),
			Contest:  v.ContestID,
			Problem:  v.ProblemID,
			Lang:     v.Language,
			Status:   v.Result,
			Time:     time.Unix(v.EpochSecond, 0),
		})
	}
	return res
}

func (p NewAtCoder) FetchSubmitLog(userId int64, username string, needAll bool) (res []model.SubmitLog, err error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("AtCoder 用户名不能为空")
	}
	fromSecond := int64(0)
	if !needAll {
		fromSecond = time.Now().Add(-60 * time.Minute).Unix()
	}
	client := &http.Client{Timeout: 30 * time.Second}
	seen := make(map[int]struct{})
	for page := 0; ; page++ {
		atc, err := fetchAtCoderLog(client, atCoderAPIBaseURL, username, fromSecond)
		if err != nil {
			return nil, err
		}
		if len(atc) == 0 {
			break
		}
		newRows := make([]atcJson, 0, len(atc))
		lastSecond := fromSecond
		for _, row := range atc {
			if row.EpochSecond > lastSecond {
				lastSecond = row.EpochSecond
			}
			if _, ok := seen[row.ID]; ok {
				continue
			}
			seen[row.ID] = struct{}{}
			newRows = append(newRows, row)
		}
		res = append(res, atCoderRowsToSubmitLogs(userId, newRows)...)
		if !needAll || len(atc) < atCoderPageSize {
			break
		}
		if len(newRows) == 0 || lastSecond <= fromSecond {
			return nil, fmt.Errorf("AtCoder 翻页没有前进，停止以避免重复抓取: user=%s from_second=%d", username, fromSecond)
		}
		fromSecond = lastSecond
		time.Sleep(300 * time.Millisecond)
	}
	return res, nil
}
func (p NewAtCoder) Name() string {
	return spider.AtCoder
}
func init() {
	// 注册到注册中心
	spider.Register(NewAtCoder{})
}
