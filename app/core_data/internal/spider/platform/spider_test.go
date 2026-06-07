package platform

import (
	"os"
	"testing"
)

func TestLogin(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("set RUN_INTEGRATION_TESTS=1 to run external QOJ login test")
	}
	gu := NewQOJ{}
	t.Log(gu.FetchSubmitLog(1, "sanenchen", true))
}
