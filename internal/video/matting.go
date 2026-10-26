// matting.go 把静帧或视频中的一帧交给抠图供应商。

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

// presetOnlyError 表示连接已选中，但这条供应商还没有对应请求。
type presetOnlyError struct{ msg string }

// Error 返回构造时的说明。
func (e presetOnlyError) Error() string { return e.msg }

// presetOnly 用这段说明构造 presetOnlyError。
// msg 写入 presetOnlyError。
func presetOnly(msg string) error { return presetOnlyError{msg} }

// matteImage 先得到一张静帧。removebg、photoroom、volcengine-visual-matting 会发请求。其他协议返回 presetOnly，说明抠像连接已选中但还没有抠像请求。空画面返回「没有可抠的画面」。
// ctx 传给 removeBG、photoroomCutout 或 volcengineCutout。
// protocol 去掉空白并转成小写后与 removebg、photoroom、volcengine-visual-matting 比较，其他值返回 presetOnly。
// key 传给这三个抠像请求。
// raw 长度为 0 时返回没有可抠的画面，否则传给 stillForMatte。
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

// stillForMatte 能解码成图片时原样返回。否则用 ffmpeg 在 0.4 秒处抽一帧 PNG，超时 20 秒。没有 ffmpeg 或抽帧失败时返回说明。
// raw 能解码成图片时原样返回，否则写进临时文件再抽帧。
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

// removeBG 向基址加 /v1.0/removebg 发送 image_file_b64，size 为 auto，format 为 png。鉴权头是 X-Api-Key。
// ctx 传给 NewRequestWithContext。
// key 写入 X-Api-Key。
// raw 编成 base64 后作为 image_file_b64。
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

// photoroomCutout 向基址加 /v1/segment 上传表单文件 image_file。鉴权头是 x-api-key。
// ctx 传给 NewRequestWithContext。
// raw 作为表单文件 image_file。
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

// volcengineCutout 把密钥按 AccessKey 和 SecretKey 拆开，向视觉接口发 CVProcess，版本 2022-08-31。模型空着时用 saliency_seg，基址空着时用 https://visual.volcengineapi.com。请求经 signVisual 签名。code 为 0 或 10000 时解码 data.binary_data_base64 的第一项。
// ctx 传给 NewRequestWithContext。
// key 传给 splitAKSK。
// raw 编成 base64 后写入 binary_data_base64。
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

// readImageBody 最多读 32 MiB。空正文，或 JSON 里没有可用图片时返回 matting returned no image。JSON 里有 message 或 error 时返回那段文字。其他正文原样返回。
// req 传给 doRaw。
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

// imageForm 把字段和文件打成 multipart。文件部分的文件名是字段名加 .png。
// fields 逐项写成表单字段。
// files 按字段名加 .png 作为文件名写入。
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

// splitAKSK 用换行或冒号把密钥拆成两段非空文本。拆不开时返回假。
// key 去掉空白后为空则返回假，否则按换行或冒号拆成两段非空文本。
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
