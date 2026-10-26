// vendor_body_test.go 检查抠图和配乐请求是否按各供应商的公开字段发出。

package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestCutoutVendorsSendTheirPublicBodies 确认 remove.bg 带 X-Api-Key 和 image_file_b64，PhotoRoom 带 x-api-key，火山引擎带 saliency_seg、only_mask 为 3 和 HMAC-SHA256。单段密钥会失败。同一正文和同一时间的签名两次相同。
func TestCutoutVendorsSendTheirPublicBodies(t *testing.T) {
	pngBytes := tinyPNG(t)
	var removeKey, photoKey, volcBody, volcAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/removebg"):
			removeKey = r.Header.Get("X-Api-Key")
			raw, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(raw), "image_file_b64") || !strings.Contains(string(raw), "format") {
				t.Errorf("remove.bg body missing fields")
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBytes)
		case strings.HasSuffix(r.URL.Path, "/segment"):
			photoKey = r.Header.Get("x-api-key")
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngBytes)
		case r.URL.Query().Get("Action") == "CVProcess":
			volcAuth = r.Header.Get("Authorization")
			volcBody = readString(r)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"code":10000,"data":{"binary_data_base64":["` + base64.StdEncoding.EncodeToString(pngBytes) + `"]}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	e := testEngine(t)
	e.HTTP = srv.Client()
	ctx := context.Background()
	if out, err := e.matteImage(ctx, Provider{BaseURL: srv.URL, Model: "removebg"}, "removebg", "rb-key", pngBytes); err != nil || string(out) != string(pngBytes) || removeKey != "rb-key" {
		t.Fatalf("remove.bg %v %d %s", err, len(out), removeKey)
	}
	if out, err := e.matteImage(ctx, Provider{BaseURL: srv.URL}, "photoroom", "pr-key", pngBytes); err != nil || string(out) != string(pngBytes) || photoKey != "pr-key" {
		t.Fatalf("photoroom %v %s", err, photoKey)
	}
	out, err := e.matteImage(ctx, Provider{BaseURL: srv.URL, Model: "saliency_seg"}, "volcengine-visual-matting", "AKIDtest:secret", pngBytes)
	if err != nil || string(out) != string(pngBytes) {
		t.Fatalf("volc %v %d", err, len(out))
	}
	if !strings.Contains(volcBody, "saliency_seg") || !strings.Contains(volcBody, `"only_mask":3`) || !strings.HasPrefix(volcAuth, "HMAC-SHA256 Credential=AKIDtest/") {
		t.Fatalf("body %s auth %s", volcBody, volcAuth)
	}
	if _, err := e.matteImage(ctx, Provider{BaseURL: srv.URL}, "volcengine-visual-matting", "only-one-key", pngBytes); err == nil || !strings.Contains(err.Error(), "AccessKey") {
		t.Fatal(err)
	}
	fixed := time.Date(2026, 10, 25, 8, 0, 0, 0, time.UTC)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/?Action=CVProcess&Version=2022-08-31", nil)
	signVisual(req, []byte(`{"req_key":"saliency_seg"}`), "AKIDtest", "secret", fixed)
	again, _ := http.NewRequest(http.MethodPost, srv.URL+"/?Action=CVProcess&Version=2022-08-31", nil)
	signVisual(again, []byte(`{"req_key":"saliency_seg"}`), "AKIDtest", "secret", fixed)
	if req.Header.Get("Authorization") == "" || req.Header.Get("Authorization") != again.Header.Get("Authorization") {
		t.Fatal(req.Header.Get("Authorization"))
	}
}

// TestMusicVendorsSendTheirPublicBodies 确认 MiniMax 纯器乐请求、Mureka 查询后的音频、ElevenLabs 的 xi-api-key，以及 Stability 提示词。google-lyria 保持为预设，不发出成功结果。
func TestMusicVendorsSendTheirPublicBodies(t *testing.T) {
	var miniBody, elevenKey, stablePrompt string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/music_generation"):
			miniBody = readString(r)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"data":{"audio":"6869","status":2},"base_resp":{"status_code":0,"status_msg":"success"}}`))
		case strings.HasSuffix(r.URL.Path, "/song/generate"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"task-1","status":"preparing"}`))
		case strings.Contains(r.URL.Path, "/song/query/"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"task-1","status":"succeeded","choices":[{"url":"` + srvURL(r) + `/bed.mp3"}]}`))
		case strings.HasSuffix(r.URL.Path, "/bed.mp3"):
			_, _ = w.Write([]byte("mp3"))
		case strings.HasSuffix(r.URL.Path, "/sound-generation"):
			elevenKey = r.Header.Get("xi-api-key")
			body := readString(r)
			if !strings.Contains(body, `"text"`) {
				t.Errorf("eleven body %s", body)
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("sfx"))
		case strings.Contains(r.URL.Path, "/text-to-audio"):
			raw := readString(r)
			if i := strings.Index(raw, "name=\"prompt\""); i >= 0 {
				stablePrompt = raw[i:]
			}
			w.Header().Set("Content-Type", "audio/mpeg")
			_, _ = w.Write([]byte("stab"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	e := testEngine(t)
	e.HTTP = srv.Client()
	ctx := context.Background()
	p := Provider{BaseURL: srv.URL}
	if out, err := e.scoreMusic(ctx, p, "minimax-music", "k", "soft piano", "music-3.0"); err != nil || string(out) != "hi" || !strings.Contains(miniBody, `"is_instrumental":true`) || !strings.Contains(miniBody, `"model":"music-3.0"`) {
		t.Fatalf("minimax %v %s %s", err, out, miniBody)
	}
	if out, err := e.scoreMusic(ctx, p, "mureka", "k", "warm bed", "mureka-6"); err != nil || string(out) != "mp3" {
		t.Fatalf("mureka %v %s", err, out)
	}
	if out, err := e.scoreMusic(ctx, p, "elevenlabs-sound", "xi", "room tone", "eleven_text_to_sound_v2"); err != nil || string(out) != "sfx" || elevenKey != "xi" {
		t.Fatalf("eleven %v %s %s", err, out, elevenKey)
	}
	if out, err := e.scoreMusic(ctx, p, "stability-audio", "sk", "gentle guitar", "stable-audio-2.5"); err != nil || string(out) != "stab" || !strings.Contains(stablePrompt, "gentle guitar") {
		t.Fatalf("stability %v %s", err, stablePrompt)
	}
	if _, err := e.scoreMusic(ctx, p, "google-lyria", "k", "pad", "lyria-002"); err == nil {
		t.Fatal("lyria should stay a preset")
	}
}

// tinyPNG 做一张 2×2 的 PNG。
func tinyPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.SetRGBA(0, 0, color.RGBA{10, 20, 30, 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// readString 读完请求正文并当成字符串。
func readString(r *http.Request) string {
	b, _ := io.ReadAll(r.Body)
	return string(b)
}

// srvURL 用请求的 Host 拼出 http 地址。
func srvURL(r *http.Request) string {
	return "http://" + r.Host
}
