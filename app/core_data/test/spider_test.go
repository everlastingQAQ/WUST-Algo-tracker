package test

import (
	"cwxu-algo/app/core_data/internal/spider"
	_ "cwxu-algo/app/core_data/internal/spider/platform"
	"os"
	"testing"
)

func TestSpider(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run external spider integration test")
	}
	pms := []string{spider.CodeForces}
	for _, pm := range pms {
		t.Run(pm, func(t *testing.T) {
			if p, ok := spider.Get(pm); ok {
				if slf, ok := p.(spider.SubmitContestFetcher); ok {
					r, err := slf.FetchContestLog(1, "wanli_", true)
					if err != nil {
						t.Errorf("测试出错 %s", err.Error())
					}
					for _, v := range r {
						t.Log(v)
					}
				}
			} else {
				t.Errorf("没有找到%s提供器", pm)
			}
		})
	}
}
