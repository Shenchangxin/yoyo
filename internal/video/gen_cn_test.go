// gen_cn_test.go 锁住即梦和混元生图：即梦按 AK/SK 签名提交并轮询出图，混元走 TokenHub 同步出图，尺寸符合两家的限制，老地址给出改法。

package video

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestJimengSubmitSignsAndPolls 确认即梦提交时签了火山 HMAC-SHA256、宽高按画幅换算并强制单图；轮询拿到 base64 后落盘。
func TestJimengSubmitSignsAndPolls(t *testing.T) {
	e := testEngine(t)
	png := []byte("\x89PNG\r\n\x1a\nfake-image-bytes-for-test")
	var submit map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "HMAC-SHA256 Credential=AKLTtest/") || !strings.Contains(auth, "/cn-north-1/cv/request") || r.Header.Get("X-Date") == "" {
			t.Errorf("auth %q", auth)
		}
		switch r.URL.Query().Get("Action") {
		case "CVSync2AsyncSubmitTask":
			_ = json.NewDecoder(r.Body).Decode(&submit)
			_, _ = w.Write([]byte(`{"code":10000,"data":{"task_id":"task-1"},"message":"Success"}`))
		case "CVSync2AsyncGetResult":
			_, _ = w.Write([]byte(`{"code":10000,"data":{"status":"done","binary_data_base64":["` + base64.StdEncoding.EncodeToString(png) + `"]}}`))
		default:
			t.Errorf("action %s", r.URL.RawQuery)
		}
	}))
	defer srv.Close()
	p := Provider{BaseURL: srv.URL, Protocol: "volcengine-jimeng-image", Model: "jimeng_t2i_v40"}
	j := &Job{ID: "j1", Type: "image", Prompt: "雨夜街角", Model: "jimeng_t2i_v40"}
	ctx := context.Background()
	if err := e.jimengImage(ctx, p, "AKLTtest:secret", j, JobParams{Size: "1080x1920"}); err != nil {
		t.Fatal(err)
	}
	if j.RemoteID != "task-1" || j.Status != "polling" {
		t.Fatalf("job %+v", j)
	}
	w, h := submit["width"].(float64), submit["height"].(float64)
	if submit["force_single"] != true || submit["req_key"] != "jimeng_t2i_v40" || w >= h || w*h > 4096*4096 || w*h < 1024*1024 {
		t.Fatalf("submit %+v", submit)
	}
	if err := e.jimengPoll(ctx, p, "AKLTtest:secret", j); err != nil {
		t.Fatal(err)
	}
	if j.ResultHash == "" {
		t.Fatalf("image not stored: %+v", j)
	}
}

// TestJimengOfficialNeedsAKSK 确认官方地址只填一段密钥时直接说明要填 AK:SK，不去发一个注定被拒的请求。
func TestJimengOfficialNeedsAKSK(t *testing.T) {
	e := testEngine(t)
	p := Provider{BaseURL: "https://visual.volcengineapi.com", Protocol: "volcengine-jimeng-image"}
	err := e.jimengImage(context.Background(), p, "single-key", &Job{Prompt: "x"}, JobParams{})
	if err == nil || !strings.Contains(err.Error(), "AK:SK") {
		t.Fatalf("err %v", err)
	}
}

// TestHunyuanImageUsesTokenHub 确认混元走 TokenHub 的 v3-generation，老模型名换成 hy-image-v3，尺寸面积不超过 1024×1024，按返回地址下载落盘。
func TestHunyuanImageUsesTokenHub(t *testing.T) {
	e := testEngine(t)
	var body map[string]any
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/wand/hunyuan-image/v3-generation":
			if r.Header.Get("Authorization") != "Bearer th-key" {
				t.Errorf("auth %q", r.Header.Get("Authorization"))
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = w.Write([]byte(`{"data":[{"url":"` + srvURL + `/out.png"}],"request_id":"r"}`))
		case "/out.png":
			_, _ = w.Write([]byte("\x89PNG\r\n\x1a\nhunyuan-image"))
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL
	p := Provider{BaseURL: srv.URL, Protocol: "hunyuan-image", Model: "hunyuan-image-v3"}
	j := &Job{ID: "j2", Type: "image", Prompt: "海边日落"}
	if err := e.hunyuanImage(context.Background(), p, "th-key", j, JobParams{Size: "1920x1080"}); err != nil {
		t.Fatal(err)
	}
	if body["model"] != "hy-image-v3" || j.ResultHash == "" {
		t.Fatalf("body %+v job %+v", body, j)
	}
	w, h, ok := splitSize(body["size"].(string))
	if !ok || w*h > 1024*1024 || w < 512 || h < 512 || w <= h {
		t.Fatalf("size %v", body["size"])
	}
}

// TestHunyuanLegacyEndpointExplains 确认还填着老混元地址时提示迁到 TokenHub。
func TestHunyuanLegacyEndpointExplains(t *testing.T) {
	e := testEngine(t)
	p := Provider{BaseURL: "https://api.hunyuan.cloud.tencent.com/v1", Protocol: "hunyuan-image"}
	err := e.hunyuanImage(context.Background(), p, "k", &Job{Prompt: "x"}, JobParams{})
	if err == nil || !strings.Contains(err.Error(), "TokenHub") {
		t.Fatalf("err %v", err)
	}
}

// TestImageRouteKnowsChineseVendors 确认即梦和混元的协议和厂商名都能认出来。
func TestImageRouteKnowsChineseVendors(t *testing.T) {
	cases := map[string]Provider{
		"jimeng":  {Protocol: "volcengine-jimeng-image"},
		"hunyuan": {Provider: "tencent"},
	}
	for want, p := range cases {
		if got := imageRoute(p); got != want {
			t.Errorf("%+v: got %s want %s", p, got, want)
		}
	}
}
