// gen_cn.go 接即梦和混元两家生图。即梦走火山引擎视觉服务的异步任务，要 AK/SK 签名；混元走腾讯云 TokenHub 的同步接口，Bearer 鉴权。

package video

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// jimengOfficial 判断即梦连接用的是火山引擎官方地址。官方地址只认 AK/SK 签名；别的地址多半是转发网关，用 Bearer。
func jimengOfficial(base string) bool {
	h := endpointHost(base)
	return h == "" || strings.HasSuffix(h, "volcengineapi.com")
}

// jimengRequest 向即梦视觉接口发一次 Action，版本 2022-08-31。
// 密钥写成 AK:SK 时按火山引擎签名；官方地址只填了一段时返回能照着做的提示；网关地址把整段密钥当 Bearer 令牌。
// code 不是 0 或 10000 时返回错误。
func (e *Engine) jimengRequest(ctx context.Context, p Provider, key, action string, body map[string]any) (map[string]any, error) {
	base := strings.TrimRight(first(strings.TrimSpace(p.BaseURL), "https://visual.volcengineapi.com"), "/")
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"?Action="+action+"&Version=2022-08-31", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	switch cred := splitCred(key, 2); {
	case cred != nil:
		signVisual(req, raw, cred[0], cred[1], time.Now())
	case jimengOfficial(base):
		return nil, fmt.Errorf("即梦要火山引擎的 AccessKey 和 SecretKey。在这个图像连接的密钥框里填「AK:SK」（英文冒号隔开）")
	default:
		req.Header.Set("Authorization", "Bearer "+key)
	}
	m, _, _, err := e.doJSON(req)
	if err != nil {
		return nil, err
	}
	if code := intAny(m["code"]); code != 0 && code != 10000 {
		return nil, fmt.Errorf("即梦返回 %d：%s", code, strAny(m, "message"))
	}
	return m, nil
}

// aspectRatioOf 从尺寸 1080x1920 或画幅 9:16 里读出宽高比。都读不出时返回 1。
func aspectRatioOf(size, aspect string) float64 {
	if w, h, ok := splitSize(size); ok {
		return float64(w) / float64(h)
	}
	var a, b float64
	if _, err := fmt.Sscanf(strings.TrimSpace(aspect), "%g:%g", &a, &b); err == nil && a > 0 && b > 0 {
		return a / b
	}
	return 1
}

// fitArea 按宽高比算出面积约为 area 的宽高，宽高都取 16 的倍数，并保证乘积不超过 area。ratio 先夹在 [lo, hi]。
func fitArea(ratio, area, lo, hi float64) (int, int) {
	ratio = math.Max(lo, math.Min(hi, ratio))
	w := int(math.Sqrt(area*ratio)) / 16 * 16
	h := int(math.Sqrt(area/ratio)) / 16 * 16
	for w*h > int(area) && h > 16 {
		h -= 16
	}
	return w, h
}

// jimengSize 按画幅给即梦挑宽高。4.0 支持 1K 到 4K，取 2K 面积；更早的版本取约 1328×1328 的面积。宽高比限制在 1:3 到 3:1。
func jimengSize(size, aspect, model string) (int, int) {
	area := 2048.0 * 2048.0
	if !strings.Contains(strings.ToLower(model), "v4") {
		area = 1328.0 * 1328.0
	}
	return fitArea(aspectRatioOf(size, aspect), area, 1.0/3, 3)
}

// publicImageURLs 只留下公网能取到的 http(s) 参考图地址。本机地址和内容库哈希对方取不到，发过去只会报错。
func publicImageURLs(refs []string, limit int) []string {
	var out []string
	for _, r := range refs {
		r = strings.TrimSpace(r)
		if !strings.HasPrefix(r, "https://") && !strings.HasPrefix(r, "http://") {
			continue
		}
		if h := endpointHost(r); h == "localhost" || strings.HasPrefix(h, "127.") || h == "wails.localhost" {
			continue
		}
		out = append(out, r)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

// jimengImage 提交即梦生图任务。模型名就是 req_key，宽高按画幅换算，强制只出一张，拿到任务编号后进入轮询。
func (e *Engine) jimengImage(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	model := first(j.Model, first(p.Model, "jimeng_t2i_v40"))
	w, h := jimengSize(params.Size, params.AspectRatio, model)
	body := map[string]any{"req_key": model, "prompt": j.Prompt, "width": w, "height": h, "force_single": true}
	if refs := publicImageURLs(params.ReferenceImages, 10); len(refs) > 0 {
		body["image_urls"] = refs
	}
	m, err := e.jimengRequest(ctx, p, key, "CVSync2AsyncSubmitTask", body)
	if err != nil {
		return err
	}
	id := strAny(pickData(m), "task_id")
	if id == "" {
		return fmt.Errorf("即梦没有返回任务编号")
	}
	j.RemoteID = id
	j.Status = "polling"
	return nil
}

// jimengPoll 查询即梦任务。done 时取 base64 或图片地址落盘；排队和生成中继续轮询；not_found、expired 等其余状态算失败。
func (e *Engine) jimengPoll(ctx context.Context, p Provider, key string, j *Job) error {
	m, err := e.jimengRequest(ctx, p, key, "CVSync2AsyncGetResult", map[string]any{
		"req_key": first(j.Model, first(p.Model, "jimeng_t2i_v40")), "task_id": j.RemoteID, "req_json": `{"return_url":true}`,
	})
	if err != nil {
		return err
	}
	data := pickData(m)
	switch st := strings.ToLower(strAny(data, "status")); st {
	case "done":
		if list, ok := data["binary_data_base64"].([]any); ok && len(list) > 0 {
			if s, _ := list[0].(string); s != "" {
				raw, err := base64.StdEncoding.DecodeString(s)
				if err != nil {
					return err
				}
				return e.finishMedia(j, raw, false)
			}
		}
		if u := firstURL(data["image_urls"]); u != "" {
			raw, err := e.DownloadURL(u)
			if err != nil {
				return err
			}
			return e.finishMedia(j, raw, false)
		}
		return fmt.Errorf("即梦任务完成了，但没有返回图片")
	case "", "in_queue", "generating":
		j.Status = "polling"
		return nil
	default:
		return fmt.Errorf("即梦任务状态是 %s，没有出图", st)
	}
}

// hunyuanSize 按画幅给混元生图 3.0 挑宽高：面积不超过 1024×1024，宽高比夹在 1:4 到 4:1，边长不小于 512。
func hunyuanSize(size, aspect string) (int, int) {
	w, h := fitArea(aspectRatioOf(size, aspect), 1024*1024, 0.25, 4)
	if w < 512 {
		w = 512
	}
	if h < 512 {
		h = 512
	}
	return w, h
}

// hunyuanImage 走腾讯云 TokenHub 的混元生图 3.0，同步返回 data[].url。
// 老的 api.hunyuan.cloud.tencent.com 只有文本模型，遇到它时提示改地址；老连接里的 hunyuan-image-v3 模型名换成 TokenHub 的 hy-image-v3。
func (e *Engine) hunyuanImage(ctx context.Context, p Provider, key string, j *Job, params JobParams) error {
	base := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	if h := endpointHost(base); h == "" || strings.Contains(h, "hunyuan.cloud.tencent.com") {
		return fmt.Errorf("混元生图已迁到腾讯云 TokenHub。把这个图像连接的地址改成 https://tokenhub.tencentmaas.com/v1，密钥换成 TokenHub 的 API Key")
	}
	if !versionTail.MatchString(base) {
		base += "/v1"
	}
	model := first(j.Model, first(p.Model, "hy-image-v3"))
	if strings.HasPrefix(model, "hunyuan-image") {
		model = "hy-image-v3"
	}
	w, h := hunyuanSize(params.Size, params.AspectRatio)
	body := map[string]any{"model": model, "prompt": j.Prompt, "size": fmt.Sprintf("%dx%d", w, h)}
	if refs := publicImageURLs(params.ReferenceImages, 3); len(refs) > 0 {
		body["images"] = refs
	}
	req, err := jsonReq(http.MethodPost, base+"/wand/hunyuan-image/v3-generation", key, body)
	if err != nil {
		return err
	}
	m, _, _, err := e.doJSON(req.WithContext(ctx))
	if err != nil {
		return err
	}
	return e.takeImageResult(j, m)
}
