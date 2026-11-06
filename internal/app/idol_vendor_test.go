// 本文件锁住偶像的厂商协议：MiniMax 的十六进制音频和情绪、千问语音的音频地址、千问识别的 input_audio、千问生图的图片地址、
// OpenAI 语音的语气说明，以及按预设和地址认出厂商；还有进行中回合的长轮询。由 go test 调用。

package app

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/idol"
)

// TestMediaRoute 锁住按预设 id 和主机认出厂商，认不出的按 OpenAI 兼容。
func TestMediaRoute(t *testing.T) {
	cases := map[[2]string]string{
		{"minimax-speech", ""}:                                              routeMiniMax,
		{"", "https://api.minimax.io/v1"}:                                   routeMiniMax,
		{"qwen-asr", "https://example.com"}:                                 routeDashScope,
		{"", "https://dashscope.aliyuncs.com/compatible-mode/v1"}:           routeDashScope,
		{"", "https://ws1.cn-beijing.maas.aliyuncs.com/compatible-mode/v1"}: routeDashScope,
		{"openai-speech", "https://api.openai.com/v1"}:                      routeOpenAI,
		{"", "https://api.groq.com/openai/v1"}:                              routeOpenAI,
	}
	for in, want := range cases {
		if got := mediaRoute(in[0], in[1]); got != want {
			t.Errorf("%v: got %s want %s", in, got, want)
		}
	}
}

// TestSpeakMiniMax 锁住 MiniMax 走 /v1/t2a_v2，带上音色和由语气提示来的情绪，十六进制音频解回字节。
func TestSpeakMiniMax(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/t2a_v2" || r.Header.Get("Authorization") != "Bearer k" {
			http.Error(w, "bad", 404)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"data":{"audio":"`+hex.EncodeToString([]byte("ID3audio"))+`"},"base_resp":{"status_code":0}}`)
	}))
	defer srv.Close()
	b, ctype, err := synthesize(context.Background(), speechCall{route: routeMiniMax, base: srv.URL + "/v1", key: "k", voice: "female-shaonv", text: "你好", cue: "happily"})
	if err != nil || string(b) != "ID3audio" || ctype != "audio/mpeg" {
		t.Fatalf("%q %s %v", b, ctype, err)
	}
	vs, _ := body["voice_setting"].(map[string]any)
	if vs["voice_id"] != "female-shaonv" || vs["emotion"] != "happy" || body["text"] != "你好" {
		t.Fatalf("body %+v", body)
	}
}

// TestSpeakDashScope 锁住千问语音走 multimodal-generation，按口语语言写 language_type，再下载返回的音频地址。
func TestSpeakDashScope(t *testing.T) {
	var body map[string]any
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/services/aigc/multimodal-generation/generation":
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = io.WriteString(w, `{"output":{"audio":{"url":"`+srv.URL+`/a.wav"}}}`)
		case "/a.wav":
			_, _ = io.WriteString(w, "RIFF0000WAVEdata")
		default:
			http.Error(w, "bad", 404)
		}
	}))
	defer srv.Close()
	b, ctype, err := synthesize(context.Background(), speechCall{route: routeDashScope, base: srv.URL + "/compatible-mode/v1", key: "k", voice: "Cherry", text: "こんにちは", lang: "ja"})
	if err != nil || ctype != "audio/wav" || !strings.HasPrefix(string(b), "RIFF") {
		t.Fatalf("%q %s %v", b, ctype, err)
	}
	input, _ := body["input"].(map[string]any)
	if input["language_type"] != "Japanese" || input["voice"] != "Cherry" {
		t.Fatalf("body %+v", body)
	}
}

// TestSpeakOpenAICue 锁住 gpt-4o 语音模型把语气提示写进 instructions，其他模型不带。
func TestSpeakOpenAICue(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/audio/speech" {
			http.Error(w, "bad", 404)
			return
		}
		body = nil
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = io.WriteString(w, "ID3")
	}))
	defer srv.Close()
	if _, _, err := synthesize(context.Background(), speechCall{route: routeOpenAI, base: srv.URL, model: "gpt-4o-mini-tts", voice: "nova", text: "hi", cue: "shyly"}); err != nil {
		t.Fatal(err)
	}
	if body["instructions"] != "Speak shyly." {
		t.Fatalf("body %+v", body)
	}
	if _, _, err := synthesize(context.Background(), speechCall{route: routeOpenAI, base: srv.URL, model: "tts-1", voice: "nova", text: "hi", cue: "shyly"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := body["instructions"]; ok {
		t.Fatalf("tts-1 got instructions %+v", body)
	}
}

// TestTranscribeDashScope 锁住千问识别走兼容模式的 chat/completions，录音以 data URI 放进 input_audio。
func TestTranscribeDashScope(t *testing.T) {
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/compatible-mode/v1/chat/completions" {
			http.Error(w, "bad "+r.URL.Path, 404)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"你好呀"}}]}`)
	}))
	defer srv.Close()
	text, err := transcribe(context.Background(), transcribeCall{route: routeDashScope, base: srv.URL + "/compatible-mode/v1", key: "k", filename: "audio.webm", audio: []byte("webm")})
	if err != nil || text != "你好呀" {
		t.Fatalf("%q %v", text, err)
	}
	raw, _ := json.Marshal(body)
	if !strings.Contains(string(raw), `"type":"input_audio"`) || !strings.Contains(string(raw), "data:audio/webm;base64,") {
		t.Fatalf("body %s", raw)
	}
}

// TestTranscribeHint 锁住热词交给识别：OpenAI 兼容的写进 prompt 字段，千问识别放在第一条 system 消息里。
func TestTranscribeHint(t *testing.T) {
	prompt := ""
	oa := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseMultipartForm(1 << 20)
		prompt = r.FormValue("prompt")
		_, _ = io.WriteString(w, `{"text":"真乃你好"}`)
	}))
	defer oa.Close()
	if text, err := transcribe(context.Background(), transcribeCall{route: routeOpenAI, base: oa.URL, hint: "樱木真乃、制作人", audio: []byte("x")}); err != nil || text != "真乃你好" {
		t.Fatalf("%q %v", text, err)
	}
	if prompt != "樱木真乃、制作人" {
		t.Fatalf("prompt %q", prompt)
	}
	var body struct {
		Messages []struct {
			Role string `json:"role"`
		} `json:"messages"`
	}
	ds := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = io.WriteString(w, `{"choices":[{"message":{"content":"好"}}]}`)
	}))
	defer ds.Close()
	if _, err := transcribe(context.Background(), transcribeCall{route: routeDashScope, base: ds.URL + "/compatible-mode/v1", hint: "樱木真乃", audio: []byte("x")}); err != nil {
		t.Fatal(err)
	}
	if len(body.Messages) != 2 || body.Messages[0].Role != "system" || body.Messages[1].Role != "user" {
		t.Fatalf("messages %+v", body.Messages)
	}
}

// TestIdolTalking 锁住用户正在和她通话、或刚开过一轮时，她不先开口；挂断并过了一段时间后才行。
func TestIdolTalking(t *testing.T) {
	now := time.Now()
	if idolTalking("talk-test", now) {
		t.Fatal("idle")
	}
	noteIdolCall("talk-test", true)
	if !idolTalking("talk-test", now) {
		t.Fatal("in a call")
	}
	noteIdolCall("talk-test", false)
	noteIdolTurn("talk-test")
	if !idolTalking("talk-test", time.Now()) || idolTalking("talk-test", time.Now().Add(idolTalkingWindow+time.Second)) {
		t.Fatal("turn window")
	}
}

// TestPaintDashScope 锁住千问生图走同步 multimodal-generation，2.0 型号用 1024*1024，再下载返回的图片地址。
func TestPaintDashScope(t *testing.T) {
	var body map[string]any
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/services/aigc/multimodal-generation/generation":
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = io.WriteString(w, `{"output":{"choices":[{"message":{"content":[{"image":"`+srv.URL+`/p.png"}]}}]}}`)
		case "/p.png":
			_, _ = io.WriteString(w, "\x89PNG\r\n\x1a\nxx")
		default:
			http.Error(w, "bad", 404)
		}
	}))
	defer srv.Close()
	b, err := paint(context.Background(), paintCall{route: routeDashScope, base: srv.URL + "/compatible-mode/v1", key: "k", model: "qwen-image-2.0", prompt: "茶杯"})
	if err != nil || !strings.HasPrefix(string(b), "\x89PNG") {
		t.Fatalf("%q %v", b, err)
	}
	params, _ := body["parameters"].(map[string]any)
	if params["size"] != "1024*1024" {
		t.Fatalf("body %+v", body)
	}
}

// TestVendorTimeout 锁住厂商不回应时请求随上下文超时结束，不会一直挂着。
func TestVendorTimeout(t *testing.T) {
	block := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-block }))
	defer srv.Close()
	defer close(block)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, _, err := synthesize(ctx, speechCall{route: routeOpenAI, base: srv.URL, voice: "nova", text: "hi"}); err == nil {
		t.Fatal("want timeout")
	}
	if time.Since(start) > 3*time.Second {
		t.Fatal("did not time out")
	}
}

// TestLiveTurnWait 锁住长轮询：有新段立刻醒来只给新的那些，跑完时带上结果，没有进展时到时返回。
func TestLiveTurnWait(t *testing.T) {
	lt := &idolLiveTurn{id: "x", wake: make(chan struct{}), stop: make(chan struct{}), cancel: func() {}}
	go func() {
		time.Sleep(30 * time.Millisecond)
		lt.update(func() { lt.segs = append(lt.segs, idol.Segment{Spoken: "一"}) })
	}()
	v := lt.wait(0, time.Second)
	if len(v.Segments) != 1 || v.Done {
		t.Fatalf("%+v", v)
	}
	start := time.Now()
	if v := lt.wait(1, 50*time.Millisecond); len(v.Segments) != 0 || time.Since(start) < 40*time.Millisecond {
		t.Fatalf("timeout %+v", v)
	}
	lt.update(func() {
		lt.segs = append(lt.segs, idol.Segment{Spoken: "二"})
		lt.done = true
		lt.result = &idol.TurnResult{RecordID: "r"}
	})
	if v := lt.wait(1, time.Second); !v.Done || len(v.Segments) != 1 || v.Result.RecordID != "r" {
		t.Fatalf("%+v", v)
	}
	if len(silentWAV()) != 44+6400*2 || string(silentWAV()[:4]) != "RIFF" {
		t.Fatal("silent wav")
	}
}
