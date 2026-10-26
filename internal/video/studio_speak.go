// 短片口播。优先走已启用的语音连接，失败再试本机 edge-tts。
// 本机有 whisper.cpp 时字幕贴着识别结果走，否则按字数估算。

package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// SpeakStudio 给已锁定的口播配音，并把音频和字幕写上工程。正文空着就用工程里的口播。还没锁定就拒绝。
// 声音空着用 alloy。语速被接口拒绝时按原速再试一次并记警告。字幕是估算的，或走了 edge-tts，也会记一条说明。配完把闸门改到画面，并尽量按音频重排镜头时长。重排失败仍返回已配好的工程。
func (e *Engine) SpeakStudio(id, text string) (StudioProject, error) {
	p, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		text = strings.TrimSpace(p.Script)
	}
	if text == "" {
		return StudioProject{}, fmt.Errorf("no text to speak")
	}
	if p.ScriptStatus != ScriptApproved {
		return p, fmt.Errorf("approve the script before Speak")
	}
	voice := strings.TrimSpace(p.Voice)
	if voice == "" {
		voice = "alloy"
	}
	raw, engine, speedIgnored, err := e.speakAudio(text, voice, p.VoiceSpeed)
	if err != nil {
		return StudioProject{}, err
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return StudioProject{}, err
	}
	p.NarrationHash = hash
	p.Storyboard.Narration = hash
	p.VoiceEngine = engine
	srt, estimated := e.studioSRT(text, hash)
	p.SubtitleEstimated = estimated
	if sh, err := e.PutBytes([]byte(srt)); err == nil {
		p.SubtitleHash = sh
		p.Storyboard.Subtitle = sh
	}
	if estimated {
		e.warnStudio(&p, GateAudio, "字幕按字数估算。本机装好 whisper.cpp 后会贴着口播走。")
	}
	if speedIgnored {
		e.warnStudio(&p, GateAudio, "这个语音接口不接受语速，已按原速配音。")
	}
	if engine == "edge" {
		e.noteStudio(&p, "speak", "本地 edge-tts · 晓晓")
	}
	p.SkipAudio = false
	p.Gate = GateVisuals
	e.noteStudio(&p, "speak", "narration ready")
	saved, err := e.saveStudio(p)
	if err != nil {
		return StudioProject{}, err
	}
	if timed, terr := e.ApplyStudioTiming(saved.ID); terr == nil {
		return timed, nil
	}
	return saved, nil
}

// speakAudio 先用已启用且有密钥的语音连接。失败或没有连接时再试 edge-tts。两边都失败时，试过云端就返回云端的错误。
func (e *Engine) speakAudio(text, voice string, speed float64) (raw []byte, engine string, speedIgnored bool, err error) {
	var cloudErr error
	if e.Conn != nil {
		if c, cerr := e.Conn.Active(connection.CapSpeech, ""); cerr == nil {
			key, kerr := e.Conn.Lease(c)
			if kerr == nil && strings.TrimSpace(key) != "" {
				model := connection.ModelFor(c, connection.CapSpeech)
				if model == "" {
					model = "gpt-4o-mini-tts"
				}
				raw, speedIgnored, err = e.ttsOpenAI(c.Endpoint, key, model, text, voice, speed)
				if err == nil {
					return raw, "cloud", speedIgnored, nil
				}
				cloudErr = err
			}
		}
	}
	raw, err = e.ttsEdge(text, speed)
	if err == nil {
		return raw, "edge", false, nil
	}
	if cloudErr != nil {
		return nil, "", false, cloudErr
	}
	return nil, "", false, fmt.Errorf("configure a speech adapter in Settings → Generation, or install edge-tts")
}

// ttsOpenAI 按 OpenAI 语音接口要音频。带了语速且对方回 400 或 422 时，去掉语速再要一次，并标明语速被忽略。
func (e *Engine) ttsOpenAI(endpoint, key, model, text, voice string, speed float64) ([]byte, bool, error) {
	raw, err := e.ttsOpenAIOnce(endpoint, key, speechPayload(model, text, voice, speed))
	if err == nil {
		return raw, false, nil
	}
	if !speechPayloadHasSpeed(speed) || !speedRejected(err) {
		return nil, false, err
	}
	raw, err = e.ttsOpenAIOnce(endpoint, key, speechPayload(model, text, voice, 0))
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

// speechPayloadHasSpeed 判断这个语速会不会写进请求。接近 1 或小于等于 0 的语速不会写。
func speechPayloadHasSpeed(speed float64) bool {
	_, ok := speechPayload("m", "t", "alloy", speed)["speed"]
	return ok
}

// speedRejected 看错误文本里有没有 speech 400 或 speech 422。空错误返回假。
func speedRejected(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "speech 400") || strings.Contains(msg, "speech 422")
}

// ttsOpenAIOnce 向语音接口发一次请求。端点空着用 api.openai.com。路径还没到 /audio/speech 就补上。超过 2 分钟或返回太短都算失败。
func (e *Engine) ttsOpenAIOnce(endpoint, key string, payload map[string]any) ([]byte, error) {
	base := strings.TrimRight(strings.TrimSpace(endpoint), "/")
	if base == "" {
		base = "https://api.openai.com/v1"
	}
	u := base
	if !strings.HasSuffix(u, "/audio/speech") {
		if strings.HasSuffix(u, "/v1") {
			u += "/audio/speech"
		} else if strings.Contains(u, "/v1/") {
			u = strings.TrimRight(u, "/") + "/audio/speech"
		} else {
			u += "/v1/audio/speech"
		}
	}
	body, _ := json.Marshal(payload)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)
	httpDo := e.HTTP
	if httpDo == nil {
		httpDo = &http.Client{Timeout: 2 * time.Minute}
	}
	res, err := httpDo.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 12<<20))
	if res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 400 {
			msg = msg[:400]
		}
		return nil, fmt.Errorf("speech %d: %s", res.StatusCode, msg)
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("speech returned empty audio")
	}
	return raw, nil
}

// ttsEdge 用本机 edge-tts，声音是 zh-CN-XiaoxiaoNeural。45 秒内没完成，或文件太短，都算失败。PATH 里没有 edge-tts 就返回查找错误。
func (e *Engine) ttsEdge(text string, speed float64) ([]byte, error) {
	bin, err := exec.LookPath("edge-tts")
	if err != nil {
		return nil, err
	}
	dir := e.Dir
	if dir == "" {
		dir = os.TempDir()
	}
	f, err := os.CreateTemp(dir, "edge-*.mp3")
	if err != nil {
		return nil, err
	}
	name := f.Name()
	f.Close()
	defer os.Remove(name)
	args := []string{"--voice", "zh-CN-XiaoxiaoNeural", "--rate", edgeRate(speed), "--text", text, "--write-media", name}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	cmd := quiet(exec.CommandContext(ctx, bin, args...))
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("edge-tts: %s", strings.TrimSpace(string(out)))
	}
	raw, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	if len(raw) < 32 {
		return nil, fmt.Errorf("edge-tts returned empty audio")
	}
	return raw, nil
}

// studioSRT 优先用 whisper.cpp 把口播对齐到识别时间。对不齐或没有识别程序时，按字数写一份估算字幕，第二个结果为真。
func (e *Engine) studioSRT(text, audioHash string) (string, bool) {
	if strings.TrimSpace(e.WHISPER) != "" && audioHash != "" {
		path, err := e.materialize(audioHash, ".mp3")
		if err == nil {
			defer os.Remove(path)
			work := e.Dir
			if work == "" {
				work = os.TempDir()
			}
			tmp, err := os.MkdirTemp(work, "whisper-*")
			if err == nil {
				defer os.RemoveAll(tmp)
				cues, _, werr := e.cutBin().Transcribe(path, tmp)
				if werr == nil {
					if aligned := alignScriptToCues(splitSpeak(text), cues); len(aligned) > 0 {
						if s := cut.EncodeSRT(aligned); strings.TrimSpace(s) != "" {
							return s, false
						}
					}
				}
			}
		}
	}
	return simpleSRT(text), true
}

// simpleSRT 按句子写 SRT。每句至少 1.6 秒，再按字数加长。没有句子就返回空串。
func simpleSRT(text string) string {
	parts := splitSpeak(text)
	if len(parts) == 0 {
		return ""
	}
	sec := 2.4
	var b strings.Builder
	t := 0.0
	for i, p := range parts {
		dur := sec + float64(utf8.RuneCountInString(p))/12
		if dur < 1.6 {
			dur = 1.6
		}
		fmt.Fprintf(&b, "%d\n%s --> %s\n%s\n\n", i+1, srtStamp(t), srtStamp(t+dur), p)
		t += dur
	}
	return b.String()
}

// splitSpeak 按句号、问号、叹号和换行把口播拆开。中英文标点都认。
func splitSpeak(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var parts []string
	var cur strings.Builder
	flush := func() {
		s := strings.TrimSpace(cur.String())
		if s != "" {
			parts = append(parts, s)
		}
		cur.Reset()
	}
	for _, r := range text {
		cur.WriteRune(r)
		if r == '。' || r == '！' || r == '？' || r == '\n' || r == '.' || r == '!' || r == '?' {
			flush()
		}
	}
	flush()
	return parts
}

// srtStamp 把秒写成 SRT 的时:分:秒,毫秒。负数当成 0。
func srtStamp(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	h := int(sec) / 3600
	m := (int(sec) % 3600) / 60
	s := int(sec) % 60
	ms := int((sec - float64(int(sec))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d,%03d", h, m, s, ms)
}
