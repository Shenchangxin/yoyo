// conductor_brief_test.go 锁住导演简报：花额度且账号齐全时同一次调用里出 hold 确认卡，答复写回给模型；没有账号或剪辑时不出卡。

package app

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

// briefTools 打开应用，开一条未绑定的视频会话，挂上导演工具。asks 记下每张卡的载荷，answer 是卡片的答复。
func briefTools(t *testing.T, withImage bool, answer string) (*runtime.WorkspaceTools, *[]string) {
	t.Helper()
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if a.Video == nil {
		t.Skip("video engine missing")
	}
	reg, err := connection.Open(t.TempDir(), a.Vault)
	if err != nil {
		t.Fatal(err)
	}
	if withImage {
		if _, err := reg.Upsert(connection.Connection{
			ID: "img", Name: "Kling", Vendor: "kling",
			Capabilities: []string{connection.CapImage}, Active: true,
		}, "k"); err != nil {
			t.Fatal(err)
		}
	}
	a.Video.Conn = reg
	sess, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	asks := []string{}
	tools := &runtime.WorkspaceTools{
		SessionID:  sess.ID,
		Advertised: append([]string(nil), runtime.ChatCoreTools...),
		AskUser: func(q string) (string, error) {
			asks = append(asks, q)
			return answer, nil
		},
	}
	a.attachCutTools(tools, sess.ID)
	a.attachConductorTools(tools, sess.ID)
	return tools, &asks
}

// TestConductorBriefAsksOnceWithHold 确认有图像账号的成片简报只出一张 hold 卡，答复开拍时结果写 confirmed: go。
func TestConductorBriefAsksOnceWithHold(t *testing.T) {
	tools, asks := briefTools(t, true, "go: go — 开拍\nnote: operator answered")
	res := tools.Call("video_brief", `{"intent":"film","format":"ranking","subject":"三个天天用的东西"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(*asks) != 1 {
		t.Fatalf("asks %d", len(*asks))
	}
	var form runtime.AskForm
	if err := json.Unmarshal([]byte((*asks)[0]), &form); err != nil {
		t.Fatal(err)
	}
	if !form.Hold || len(form.Questions) != 1 || !strings.Contains(form.Questions[0].Prompt, "Kling") {
		t.Fatalf("form %+v", form)
	}
	if !strings.Contains(res.Content, "confirmed: go") || !strings.Contains(res.Content, `"spend":true`) {
		t.Fatalf("result %s", res.Content)
	}
}

// TestConductorBriefEditIsNotGo 确认操作者选改一下时结果是 confirmed: edit，不是开拍。
func TestConductorBriefEditIsNotGo(t *testing.T) {
	tools, _ := briefTools(t, true, "go: edit — 改一下\nnote: operator answered")
	res := tools.Call("video_brief", `{"intent":"still","subject":"雨夜霓虹"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if !strings.Contains(res.Content, "confirmed: edit") || strings.Contains(res.Content, "confirmed: go") {
		t.Fatalf("result %s", res.Content)
	}
}

// TestConductorBriefMissingAccountDoesNotAsk 确认没有图像账号时不出卡，结果列出缺 image。
func TestConductorBriefMissingAccountDoesNotAsk(t *testing.T) {
	tools, asks := briefTools(t, false, "go: go — 开拍")
	res := tools.Call("video_brief", `{"intent":"film","subject":"早起"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(*asks) != 0 {
		t.Fatalf("must not ask without accounts: %v", *asks)
	}
	if !strings.Contains(res.Content, `"missing":["image"]`) || strings.Contains(res.Content, "confirmed:") {
		t.Fatalf("result %s", res.Content)
	}
}

// TestConductorBriefCutDoesNotAsk 确认剪辑简报不花额度，也不出卡。
func TestConductorBriefCutDoesNotAsk(t *testing.T) {
	tools, asks := briefTools(t, true, "go: go — 开拍")
	res := tools.Call("video_brief", `{"intent":"cut"}`)
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	if len(*asks) != 0 || !strings.Contains(res.Content, `"spend":false`) {
		t.Fatalf("asks %v result %s", *asks, res.Content)
	}
}
