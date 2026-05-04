package platform

import "testing"

func TestLogin(t *testing.T) {
	gu := NewQOJ{}
	t.Log(gu.FetchSubmitLog(1, "sanenchen", true))
}
