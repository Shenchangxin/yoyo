// speech_vendor_test.go 锁住按厂商配音：认路由、换声音、按厂商限制切段，以及每家接口的请求格式和返回解析。

package video

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/coder/websocket"
)

// fakeMP3 是一段假的 mp3：带 ID3 头，长度够过空音频检查。
var fakeMP3 = append([]byte("ID3"), make([]byte, 64)...)

// TestSpeechRouteByProtocolHostAndVendor 确认专门协议优先，老连接按地址认厂商，地址空着才看厂商名，网关走 OpenAI 兼容。
func TestSpeechRouteByProtocolHostAndVendor(t *testing.T) {
	cases := []struct {
		c    connection.Connection
		want string
	}{
		{connection.Connection{Protocol: "minimax-t2a", Endpoint: "https://gw.example.com"}, "minimax"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://api.minimax.io"}, "minimax"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://openspeech.bytedance.com"}, "volcengine"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://dashscope.aliyuncs.com/compatible-mode/v1"}, "dashscope"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://tts.cloud.tencent.com"}, "tencent"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://spark-api-open.xf-yun.com/v1"}, "spark"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://api.elevenlabs.io/v1"}, "elevenlabs"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://eastus.tts.speech.microsoft.com"}, "azure"},
		{connection.Connection{Vendor: "minimax"}, "minimax"},
		{connection.Connection{Vendor: "minimax", Endpoint: "https://api.302.ai/v1"}, "openai"},
		{connection.Connection{Protocol: "openai-audio", Endpoint: "https://api.openai.com"}, "openai"},
	}
	for _, c := range cases {
		if got := speechRoute(c.c); got != c.want {
			t.Errorf("%+v: got %s want %s", c.c, got, c.want)
		}
	}
}

// TestResolveSpeechVoiceMapsAcrossVendors 确认别家目录里的声音按性别换成这家的默认声音，本家的原样用，没见过的 id 原样发过去。
func TestResolveSpeechVoiceMapsAcrossVendors(t *testing.T) {
	if v := resolveSpeechVoice("minimax", "alloy"); v.ID != "presenter_female" {
		t.Fatalf("alloy on minimax: %+v", v)
	}
	if v := resolveSpeechVoice("minimax", "onyx"); v.ID != "presenter_male" {
		t.Fatalf("onyx on minimax: %+v", v)
	}
	if v := resolveSpeechVoice("minimax", "female-yujie"); v.ID != "female-yujie" || v.Name != "御姐" {
		t.Fatalf("own voice: %+v", v)
	}
	if v := resolveSpeechVoice("azure", "zh_male_wennuanahu_moon_bigtts"); v.ID != "zh-CN-YunxiNeural" {
		t.Fatalf("male volc voice on azure: %+v", v)
	}
	if v := resolveSpeechVoice("minimax", "moss_audio_clone_1"); v.ID != "moss_audio_clone_1" {
		t.Fatalf("custom voice must pass through: %+v", v)
	}
	if v := resolveSpeechVoice("tencent", ""); v.ID != "101001" {
		t.Fatalf("empty voice: %+v", v)
	}
}

// TestSpeechChunksRespectLimits 确认长口播按句切段，每段都在字数和字节上限内，拼回来一个字不少。
func TestSpeechChunksRespectLimits(t *testing.T) {
	text := strings.Repeat("今天我们聊一聊怎么把一条短片做得更好看，先从开头三秒说起。", 12)
	chunks := speechChunks(text, 140, 0)
	if len(chunks) < 3 {
		t.Fatalf("expected several chunks, got %d", len(chunks))
	}
	for _, c := range chunks {
		if n := utf8.RuneCountInString(c); n > 140 {
			t.Fatalf("chunk too long: %d", n)
		}
	}
	if strings.Join(chunks, "") != text {
		t.Fatal("chunks lost text")
	}
	long := strings.Repeat("字", 700)
	for _, c := range speechChunks(long, 280, 1000) {
		if utf8.RuneCountInString(c) > 280 || len(c) > 1000 {
			t.Fatalf("hard split exceeded limits: %d runes %d bytes", utf8.RuneCountInString(c), len(c))
		}
	}
	if got := speechChunks("短短一句。", 140, 0); len(got) != 1 {
		t.Fatalf("short text should stay whole: %v", got)
	}
}

// TestMiniMaxSpeechRequest 确认 MiniMax 走 /v1/t2a_v2、带 Bearer 和声音，十六进制音频解开，status_code 非 0 报错。
func TestMiniMaxSpeechRequest(t *testing.T) {
	e := testEngine(t)
	var body map[string]any
	fail := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/t2a_v2" || r.Header.Get("Authorization") != "Bearer mm-key" {
			t.Errorf("path %s auth %s", r.URL.Path, r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if fail {
			_, _ = w.Write([]byte(`{"base_resp":{"status_code":1004,"status_msg":"invalid api key"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"audio":"` + hex.EncodeToString(fakeMP3) + `","status":2},"base_resp":{"status_code":0,"status_msg":"success"}}`))
	}))
	defer srv.Close()
	c := connection.Connection{Protocol: "minimax-t2a", Endpoint: srv.URL + "/v1", DefaultModel: map[string]string{connection.CapSpeech: "speech-2.6-hd"}}
	out, err := e.cloudSpeech(c, "mm-key", "你好。", "onyx", 1.2)
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Audio) != string(fakeMP3) || out.Vendor != "MiniMax" || out.Voice.ID != "presenter_male" {
		t.Fatalf("out %+v", out)
	}
	vs, _ := body["voice_setting"].(map[string]any)
	if vs["voice_id"] != "presenter_male" || vs["speed"] != 1.2 || body["model"] != "speech-2.6-hd" {
		t.Fatalf("body %+v", body)
	}
	fail = true
	if _, err := e.cloudSpeech(c, "mm-key", "你好。", "", 1); err == nil || !strings.Contains(err.Error(), "invalid api key") {
		t.Fatalf("status_code error: %v", err)
	}
}

// TestVolcengineSpeechRequest 确认火山语音用 "Bearer;Token" 头，正文带 AppID 和集群，code 3000 时解开 base64；密钥格式不对时提示怎么填。
func TestVolcengineSpeechRequest(t *testing.T) {
	e := testEngine(t)
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/tts" || r.Header.Get("Authorization") != "Bearer;tok" {
			t.Errorf("path %s auth %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"code":3000,"message":"Success","data":"` + base64.StdEncoding.EncodeToString(fakeMP3) + `"}`))
	}))
	defer srv.Close()
	c := connection.Connection{Protocol: "volcengine-tts", Endpoint: srv.URL}
	out, err := e.cloudSpeech(c, "app1:tok", "你好。", "nova", 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Audio) != string(fakeMP3) {
		t.Fatal("audio not decoded")
	}
	app, _ := body["app"].(map[string]any)
	audio, _ := body["audio"].(map[string]any)
	if app["appid"] != "app1" || app["cluster"] != "volcano_tts" || audio["voice_type"] != "zh_female_shuangkuaisisi_moon_bigtts" {
		t.Fatalf("body %+v", body)
	}
	if _, err := e.cloudSpeech(c, "only-token", "你好。", "", 1); err == nil || !strings.Contains(err.Error(), "AppID:AccessToken") {
		t.Fatalf("credential hint: %v", err)
	}
}

// TestDashScopeSpeechDownloadsURL 确认千问 TTS 走多模态生成接口，按返回的地址下载音频；CosyVoice 配在通用地址上时改用千问并说明。
func TestDashScopeSpeechDownloadsURL(t *testing.T) {
	e := testEngine(t)
	var body map[string]any
	var srvURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/services/aigc/multimodal-generation/generation":
			_ = json.NewDecoder(r.Body).Decode(&body)
			_, _ = w.Write([]byte(`{"output":{"audio":{"url":"` + srvURL + `/voice.mp3","data":""}}}`))
		case "/voice.mp3":
			_, _ = w.Write(fakeMP3)
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
	}))
	defer srv.Close()
	srvURL = srv.URL
	c := connection.Connection{Protocol: "dashscope-tts", Endpoint: srv.URL, DefaultModel: map[string]string{connection.CapSpeech: "cosyvoice-v2"}}
	out, err := e.cloudSpeech(c, "sk", "你好。", "longcheng_v2", 1.3)
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Audio) != string(fakeMP3) {
		t.Fatal("audio not downloaded")
	}
	input, _ := body["input"].(map[string]any)
	if body["model"] != "qwen3-tts-flash" || input["voice"] != "Ethan" {
		t.Fatalf("cosyvoice should fall back to qwen tts with a male voice: %+v", body)
	}
	if len(out.Notes) != 2 {
		t.Fatalf("expected cosyvoice and speed notes, got %v", out.Notes)
	}
}

// TestTencentSpeechSignsAndSplits 确认腾讯云按 150 字切段、每段都签 TC3、把 Audio 解开后拼成一条。
func TestTencentSpeechSignsAndSplits(t *testing.T) {
	e := testEngine(t)
	e.FFMPEG = ""
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "TC3-HMAC-SHA256 Credential=AKIDtest/") || !strings.Contains(auth, "/tts/tc3_request") {
			t.Errorf("auth %q", auth)
		}
		if r.Header.Get("X-TC-Action") != "TextToVoice" || r.Header.Get("X-TC-Version") != "2019-08-23" {
			t.Errorf("headers %v", r.Header)
		}
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if n := utf8.RuneCountInString(body["Text"].(string)); n > 140 {
			t.Errorf("text too long for tencent: %d", n)
		}
		if body["VoiceType"] != float64(501000) {
			t.Errorf("voice %v", body["VoiceType"])
		}
		_, _ = w.Write([]byte(`{"Response":{"Audio":"` + base64.StdEncoding.EncodeToString(fakeMP3) + `","RequestId":"r"}}`))
	}))
	defer srv.Close()
	c := connection.Connection{Protocol: "tencent-tts", Endpoint: srv.URL}
	text := strings.Repeat("这是一段比较长的口播，用来确认会按句子切成好几段再分别合成。", 8)
	out, err := e.cloudSpeech(c, "AKIDtest:secret", text, "501000", 1)
	if err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) < 2 || len(out.Audio) != int(atomic.LoadInt32(&calls))*len(fakeMP3) {
		t.Fatalf("calls %d audio %d", calls, len(out.Audio))
	}
	if tencentSpeed(1.2) != 1 || tencentSpeed(0.8) != -1 || tencentSpeed(1) != 0 {
		t.Fatal("speed mapping")
	}
}

// TestSparkSpeechOverWebsocket 确认讯飞走 WebSocket：地址带鉴权参数，正文是 base64 文本，多帧音频拼起来，status 2 结束。
func TestSparkSpeechOverWebsocket(t *testing.T) {
	e := testEngine(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("authorization") == "" || q.Get("date") == "" || q.Get("host") == "" {
			t.Errorf("auth params missing: %v", q)
		}
		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.CloseNow()
		ctx := context.Background()
		_, msg, err := conn.Read(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		var req map[string]any
		_ = json.Unmarshal(msg, &req)
		data, _ := req["data"].(map[string]any)
		text, _ := base64.StdEncoding.DecodeString(data["text"].(string))
		biz, _ := req["business"].(map[string]any)
		if string(text) != "你好。" || biz["vcn"] != "aisjiuxu" {
			t.Errorf("request %v", req)
		}
		half := len(fakeMP3) / 2
		for i, part := range [][]byte{fakeMP3[:half], fakeMP3[half:]} {
			frame, _ := json.Marshal(map[string]any{"code": 0, "data": map[string]any{"audio": base64.StdEncoding.EncodeToString(part), "status": 1 + i}})
			_ = conn.Write(ctx, websocket.MessageText, frame)
		}
	}))
	defer srv.Close()
	c := connection.Connection{Protocol: "xfyun-tts", Endpoint: "ws" + strings.TrimPrefix(srv.URL, "http") + "/v2/tts"}
	out, err := e.cloudSpeech(c, "app:key:secret", "你好。", "echo", 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(out.Audio) != string(fakeMP3) {
		t.Fatalf("audio %d bytes", len(out.Audio))
	}
}

// TestElevenLabsAndAzureRequests 确认 ElevenLabs 把声音放在路径里、密钥放在 xi-api-key；Azure 发 SSML，语速写成 prosody。
func TestElevenLabsAndAzureRequests(t *testing.T) {
	e := testEngine(t)
	var ssml string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/v1/text-to-speech/"):
			if r.URL.Path != "/v1/text-to-speech/21m00Tcm4TlvDq8ikWAM" || r.Header.Get("xi-api-key") != "el" {
				t.Errorf("elevenlabs %s %s", r.URL.Path, r.Header.Get("xi-api-key"))
			}
		case r.URL.Path == "/cognitiveservices/v1":
			raw, _ := io.ReadAll(r.Body)
			ssml = string(raw)
			if r.Header.Get("Ocp-Apim-Subscription-Key") != "az" || r.Header.Get("Content-Type") != "application/ssml+xml" {
				t.Errorf("azure headers %v", r.Header)
			}
		default:
			t.Errorf("unexpected %s", r.URL.Path)
		}
		_, _ = w.Write(fakeMP3)
	}))
	defer srv.Close()
	if _, err := e.cloudSpeech(connection.Connection{Protocol: "elevenlabs-tts", Endpoint: srv.URL + "/v1"}, "el", "Hello.", "nova", 1); err != nil {
		t.Fatal(err)
	}
	az := connection.Connection{Protocol: "azure-tts", Endpoint: srv.URL, DefaultModel: map[string]string{connection.CapSpeech: "zh-CN-YunyangNeural"}}
	if _, err := e.cloudSpeech(az, "az", "你好 & 再见。", "alloy", 1.2); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ssml, `voice name="zh-CN-YunyangNeural"`) || !strings.Contains(ssml, `rate="+20%"`) || !strings.Contains(ssml, "你好 &amp; 再见") {
		t.Fatalf("ssml %s", ssml)
	}
}

// TestStudioCatalogLiveListsActiveVoices 确认没有语音连接也没有 edge-tts 时，目录里写明还没有语音。
func TestStudioCatalogLiveListsActiveVoices(t *testing.T) {
	e := testEngine(t)
	cat := e.StudioCatalogLive()
	speech, _ := cat["speech"].(map[string]any)
	if speech == nil {
		t.Fatal("speech info missing")
	}
	if !edgeTTSAvailable() && speech["route"] != "" {
		t.Fatalf("no speech should leave route empty: %v", speech)
	}
}
