// 行为：测试画布助手的能力、人设、幂等运行，以及按文件头识别 MP4。

package video

import (
	"testing"
	"time"
)

// TestCanvasAgentCapabilitiesProfileAndIdempotentRun 检查能力、人设修订号、未接桥时运行失败，以及相同幂等键不会启动第二轮。
func TestCanvasAgentCapabilitiesProfileAndIdempotentRun(t *testing.T) {
	e := testEngine(t)
	caps := e.canvasHTTP(map[string]any{"method": "GET", "path": "agent/capabilities"})
	if caps.Code != 0 {
		t.Fatal(caps.Msg)
	}
	data := caps.Data.(map[string]any)
	modes, ok := data["permissionModes"].([]string)
	if !ok || len(modes) != 3 {
		t.Fatalf("permissionModes %#v", data["permissionModes"])
	}
	profile := e.canvasHTTP(map[string]any{"method": "GET", "path": "agent/profile", "query": map[string]any{"canvasId": "c1"}})
	if profile.Code != 0 {
		t.Fatal(profile.Msg)
	}
	if profile.Data.(map[string]any)["revision"] == "" {
		t.Fatal("empty profile revision")
	}

	body := map[string]any{
		"canvasId": "c1", "prompt": "看看画布", "idempotencyKey": "k1", "permissionMode": "request_approval",
	}
	missing := e.canvasHTTP(map[string]any{"method": "POST", "path": "agent/runs", "body": body})
	run := missing.Data.(map[string]any)["run"].(map[string]any)
	if run["status"] != "failed" {
		t.Fatalf("status %v", run["status"])
	}
	if run["failureMessage"] == "" {
		t.Fatal("expected a failure when the runtime is not attached")
	}

	started := make(chan struct{}, 2)
	e.Agent = &CanvasAgentBridge{Start: func(turn CanvasAgentTurn) error {
		if turn.Prompt != "看看画布" || turn.Epoch != 1 {
			t.Errorf("turn %+v", turn)
		}
		started <- struct{}{}
		return nil
	}}
	body["idempotencyKey"] = "k2"
	first := e.canvasHTTP(map[string]any{"method": "POST", "path": "agent/runs", "body": body})
	second := e.canvasHTTP(map[string]any{"method": "POST", "path": "agent/runs", "body": body})
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("agent did not start")
	}
	select {
	case <-started:
		t.Fatal("duplicate idempotency key started a second turn")
	case <-time.After(80 * time.Millisecond):
	}
	id1 := first.Data.(map[string]any)["run"].(map[string]any)["id"]
	id2 := second.Data.(map[string]any)["run"].(map[string]any)["id"]
	if id1 == "" || id1 != id2 {
		t.Fatalf("ids %v %v", id1, id2)
	}
}

// TestPutCanvasBytesSniffsMP4 检查 ftypisom 字节在通用 MIME 下会被识别成 video/mp4。
func TestPutCanvasBytesSniffsMP4(t *testing.T) {
	e := testEngine(t)
	raw := append([]byte{0, 0, 0, 0x18}, []byte("ftypisom")...)
	raw = append(raw, []byte("rest-of-file")...)
	res, err := e.putCanvasBytes("", "application/octet-stream", "clip.mp4", raw, 0, 0, 0, "vid-sniff")
	if err != nil {
		t.Fatal(err)
	}
	if res.MimeType != "video/mp4" || res.Kind != "video" {
		t.Fatalf("mime %s kind %s", res.MimeType, res.Kind)
	}
}
