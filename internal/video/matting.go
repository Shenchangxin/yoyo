package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type presetOnlyError struct{ msg string }

func (e presetOnlyError) Error() string { return e.msg }

func presetOnly(msg string) error { return presetOnlyError{msg} }

func (e *Engine) matteImage(ctx context.Context, p Provider, protocol, key string, raw []byte) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("没有可抠的画面")
	}
	still, err := e.stillForMatte(raw)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(protocol)) {
	case "removebg":
		return e.removeBG(ctx, p, key, still)
	case "photoroom":
		return e.photoroomCutout(ctx, p, key, still)
	case "volcengine-visual-matting":
		return e.volcengineCutout(ctx, p, key, still)
	default:
		return nil, presetOnly("抠像连接已选中，这条供应商还没有抠像请求")
	}
}

func (e *Engine) stillForMatte(raw []byte) ([]byte, error) {
	if _, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
		return raw, nil
	}
	if strings.TrimSpace(e.FFMPEG) == "" {
		return nil, fmt.Errorf("抠像需要一张图片，当前参考是视频且没有 ffmpeg")
	}
	dir, err := os.MkdirTemp(e.Dir, "film-matte-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	in := filepath.Join(dir, "in.bin")
	out := filepath.Join(dir, "frame.png")
	if err := os.WriteFile(in, raw, 0o644); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.FFMPEG, "-y", "-ss", "0.4", "-i", in, "-frames:v", "1", out)
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("抠像需要一张图片，没能从参考视频抽出一帧")
	}
	return os.ReadFile(out)
}

func (e *Engine) removeBG(ctx context.Context, p Provider, key string, raw []byte) ([]byte, error) {
	body, ctype, err := imageForm(map[string]string{
		"image_file_b64": base64.StdEncoding.EncodeToString(raw),
		"size":           "auto",
		"format":         "png",
	}, nil)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(p.BaseURL, "/v1.0", "/removebg"), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("X-Api-Key", key)
	return e.readImageBody(req)
}

func (e *Engine) photoroomCutout(ctx context.Context, p Provider, key string, raw []byte) ([]byte, error) {
	body, ctype, err := imageForm(nil, map[string][]byte{"image_file": raw})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, joinURL(p.BaseURL, "/v1", "/segment"), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", ctype)
	req.Header.Set("x-api-key", key)
	return e.readImageBody(req)
}

func (e *Engine) volcengineCutout(ctx context.Context, p Provider, key string, raw []byte) ([]byte, error) {
	ak, sk, ok := splitAKSK(key)
	if !ok {
		return nil, fmt.Errorf("火山引擎抠像的密钥要写成 AccessKey:SecretKey")
	}
	model := first(p.Model, "saliency_seg")
	payload := map[string]any{
		"req_key":            model,
		"binary_data_base64": []string{base64.StdEncoding.EncodeToString(raw)},
		"only_mask":          3,
		"refine_mask":        2,
		"rgb":                []int{-1, -1, -1},
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(strings.TrimRight(first(p.BaseURL, "https://visual.volcengineapi.com"), "/"))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("Action", "CVProcess")
	q.Set("Version", "2022-08-31")
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	signVisual(req, buf, ak, sk, time.Now())
	m, _, _, err := e.doJSON(req)
	if err != nil {
		return nil, err
	}
	if code := intAny(m["code"]); code != 0 && code != 10000 {
		msg := strAny(m, "message")
		if msg == "" {
			msg = "volcengine matting failed"
		}
		return nil, fmt.Errorf("%s", msg)
	}
	data, _ := m["data"].(map[string]any)
	list, _ := data["binary_data_base64"].([]any)
	if len(list) == 0 {
		return nil, fmt.Errorf("volcengine matting returned no image")
	}
	enc, _ := list[0].(string)
	out, err := base64.StdEncoding.DecodeString(enc)
	if err != nil || len(out) == 0 {
		return nil, fmt.Errorf("volcengine matting returned no image")
	}
	return out, nil
}

func (e *Engine) readImageBody(req *http.Request) ([]byte, error) {
	_, ctype, raw, err := e.doRaw(req, 32<<20)
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, fmt.Errorf("matting returned no image")
	}
	if strings.Contains(ctype, "json") || (len(raw) > 0 && raw[0] == '{') {
		var m map[string]any
		if json.Unmarshal(raw, &m) == nil {
			if s := strAny(m, "message", "error"); s != "" {
				return nil, fmt.Errorf("%s", s)
			}
		}
		return nil, fmt.Errorf("matting returned no image")
	}
	return raw, nil
}

func imageForm(fields map[string]string, files map[string][]byte) (*bytes.Buffer, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if err := w.WriteField(k, v); err != nil {
			return nil, "", err
		}
	}
	for name, raw := range files {
		part, err := w.CreateFormFile(name, name+".png")
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(raw); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return &buf, w.FormDataContentType(), nil
}

func splitAKSK(key string) (string, string, bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}
	for _, sep := range []string{"\n", ":"} {
		i := strings.Index(key, sep)
		if i <= 0 || i >= len(key)-len(sep) {
			continue
		}
		ak := strings.TrimSpace(key[:i])
		sk := strings.TrimSpace(key[i+len(sep):])
		if ak != "" && sk != "" {
			return ak, sk, true
		}
	}
	return "", "", false
}
