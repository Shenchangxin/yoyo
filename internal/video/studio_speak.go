package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

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
	if e.Conn == nil {
		return StudioProject{}, fmt.Errorf("configure a speech adapter in Settings → Generation")
	}
	c, err := e.Conn.Active(connection.CapSpeech, "")
	if err != nil {
		return StudioProject{}, fmt.Errorf("configure a speech adapter in Settings → Generation")
	}
	key, err := e.Conn.Lease(c)
	if err != nil || strings.TrimSpace(key) == "" {
		return StudioProject{}, fmt.Errorf("speech adapter has no API key")
	}
	model := connection.ModelFor(c, connection.CapSpeech)
	if model == "" {
		model = "gpt-4o-mini-tts"
	}
	raw, err := e.ttsOpenAI(c.Endpoint, key, model, text)
	if err != nil {
		return StudioProject{}, err
	}
	hash, err := e.PutBytes(raw)
	if err != nil {
		return StudioProject{}, err
	}
	p.NarrationHash = hash
	p.Storyboard.Narration = hash
	srt := simpleSRT(text)
	if sh, err := e.PutBytes([]byte(srt)); err == nil {
		p.SubtitleHash = sh
		p.Storyboard.Subtitle = sh
	}
	p.UpdatedAt = Now()
	if err := e.putDoc(colStudioProjects, p.ID, p); err != nil {
		return StudioProject{}, err
	}
	return p, nil
}

func (e *Engine) ttsOpenAI(endpoint, key, model, text string) ([]byte, error) {
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
	body, _ := json.Marshal(map[string]any{"model": model, "input": text, "voice": "alloy", "format": "mp3"})
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
