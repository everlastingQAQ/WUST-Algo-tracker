package event

type SpiderEvent struct {
	UserId  int64 `json:"user_id"`
	NeedAll bool  `json:"need_all"`
	JobId   int64 `json:"job_id"`
}

type SummaryEvent struct {
	UserId int64  `json:"user_id"`
	Type   string `json:"type"`
}
