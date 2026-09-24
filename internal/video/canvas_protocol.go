package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/video/canvas/contract"
	"github.com/Shenchangxin/yoyo/internal/video/protocol"
)

func (e *Engine) submitCanvas(ctx context.Context, j *Job) error {
	p := parseParams(j.Params)
	ch, err := e.resolveCanvasChannel(p)
	if err != nil {
		return err
	}
	adapter, err := e.resolveCanvasAdapter(ch, p.Mode)
	if err != nil {
		return e.submitCanvasFallback(ctx, j, ch, p)
	}
	req, err := e.buildGenerationRequest(*j, p, ch)
	if err != nil {
		return err
	}
	key := ""
	if ch.VaultKey != "" && e.Vault != nil {
		key, _ = e.Vault.Lease(ch.VaultKey)
	}
	spec, err := adapter.BuildCreate(ctx, protocol.RequestContext{BaseURL: ch.BaseURL, Request: req})
	if err != nil {
		return e.submitCanvasFallback(ctx, j, ch, p)
	}
	body, status, err := e.doProtocol(ctx, spec, key, ch.BaseURL)
	if err != nil {
		return err
	}
	_ = status
	created, err := adapter.ParseCreate(ctx, body)
	if err != nil {
		return err
	}
	if created.TaskID != "" {
		j.RemoteID = created.TaskID
	}
	p.RemotePayload = json.RawMessage(body)
	j.Params = marshalJSON(p)
	if created.Status == protocol.StatusSucceeded && created.Result != nil {
		return e.materializeProtocolResult(j, created.Result)
	}
	if created.Status == protocol.StatusFailed {
		return fmt.Errorf("%s", firstNonEmpty(created.Message, "provider failed"))
	}
	j.Status = "polling"
	return nil
}

func (e *Engine) pollCanvas(ctx context.Context, j *Job) error {
	p := parseParams(j.Params)
	ch, err := e.resolveCanvasChannel(p)
	if err != nil {
		return err
	}
	adapter, err := e.resolveCanvasAdapter(ch, p.Mode)
	if err != nil {
		return e.pollCanvasFallback(ctx, j, p)
	}
	req, _ := e.buildGenerationRequest(*j, p, ch)
	key := ""
	if ch.VaultKey != "" && e.Vault != nil {
		key, _ = e.Vault.Lease(ch.VaultKey)
	}
	spec, err := adapter.BuildPoll(ctx, protocol.PollContext{BaseURL: ch.BaseURL, Model: j.Model, Request: req, TaskID: j.RemoteID})
	if err != nil {
		return err
	}
	body, _, err := e.doProtocol(ctx, spec, key, ch.BaseURL)
	if err != nil {
		return err
	}
	polled, err := adapter.ParsePoll(ctx, protocol.PollContext{BaseURL: ch.BaseURL, Model: j.Model, Request: req, TaskID: j.RemoteID}, body)
	if err != nil {
		return err
	}
	switch polled.Status {
	case protocol.StatusSucceeded:
		if polled.Result == nil {
			return fmt.Errorf("completed without result")
		}
		return e.materializeProtocolResult(j, polled.Result)
	case protocol.StatusFailed, protocol.StatusCancelled:
		return fmt.Errorf("%s", firstNonEmpty(polled.Message, string(polled.Status)))
	default:
		j.Status = "polling"
		return nil
	}
}

func (e *Engine) submitCanvasFallback(ctx context.Context, j *Job, ch canvasChannel, p JobParams) error {
	mode := strings.ToLower(firstNonEmpty(p.Mode, j.Type))
	switch mode {
	case "image":
		j.Type = "image"
		j.Provider = firstNonEmpty(j.Provider, providerVendor(ch))
		j.VaultKey = firstNonEmpty(j.VaultKey, ch.VaultKey)
		return e.submitImage(ctx, j)
	case "video":
		j.Type = "video"
		j.Provider = firstNonEmpty(j.Provider, providerVendor(ch))
		j.VaultKey = firstNonEmpty(j.VaultKey, ch.VaultKey)
		return e.submitVideo(ctx, j)
	default:
		return fmt.Errorf("no protocol adapter for %s", mode)
	}
}

func (e *Engine) pollCanvasFallback(ctx context.Context, j *Job, p JobParams) error {
	mode := strings.ToLower(firstNonEmpty(p.Mode, j.Type))
	switch mode {
	case "image":
		return e.pollImage(ctx, j)
	case "video":
		return e.pollVideo(ctx, j)
	default:
		return fmt.Errorf("nothing to poll")
	}
}

func providerVendor(ch canvasChannel) string {
	id := strings.ToLower(ch.PluginID)
	switch {
	case strings.Contains(id, "gemini"):
		return "gemini"
	case strings.Contains(id, "openai"):
		return "openai"
	case strings.Contains(id, "volcengine"), strings.Contains(id, "seedream"), strings.Contains(id, "seedance"):
		return "volcengine"
	case strings.Contains(id, "minimax"), strings.Contains(id, "hailuo"):
		return "minimax"
	default:
		return ch.PluginID
	}
}

func (e *Engine) resolveCanvasChannel(p JobParams) (canvasChannel, error) {
	if p.ChannelID != "" {
		if c, err := e.getCanvasChannel(p.ChannelID); err == nil {
			return c, nil
		}
	}
	mode := p.Mode
	if mode == "tts" {
		mode = "audio"
	}
	for _, c := range e.listCanvasChannels() {
		if c.Enabled && (mode == "" || c.Capability == mode || (mode == "audio" && c.Capability == "tts")) {
			return c, nil
		}
	}
	st := mode
	if st == "audio" {
		st = "tts"
	}
	pvd, err := e.ActiveProvider(st, "")
	if err != nil {
		return canvasChannel{}, fmt.Errorf("no %s channel configured", mode)
	}
	return canvasChannel{
		ID: pvd.ID, PluginID: mapProviderPlugin(pvd.ServiceType, pvd.Provider), Name: pvd.Name,
		Capability: pvd.ServiceType, BaseURL: pvd.BaseURL, VaultKey: pvd.VaultKey, Model: pvd.Model, Models: pvd.Models, Enabled: true,
	}, nil
}

func (e *Engine) resolveCanvasAdapter(ch canvasChannel, mode string) (protocol.Adapter, error) {
	if e.Proto == nil {
		return nil, fmt.Errorf("protocol registry missing")
	}
	if ch.PluginID != "" {
		if ad, ok := e.Proto.Resolve(ch.PluginID); ok {
			return ad, nil
		}
	}
	var cap protocol.Capability
	switch mode {
	case "image":
		cap = protocol.CapabilityImage
	case "video":
		cap = protocol.CapabilityVideo
	case "audio", "tts":
		cap = protocol.CapabilityAudio
	default:
		cap = protocol.CapabilityText
	}
	list := e.Proto.List(protocol.SurfaceCanvas, cap, false)
	if len(list) == 0 {
		list = e.Proto.List("", cap, false)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("no adapter for %s", mode)
	}
	ad, ok := e.Proto.Resolve(list[0].ID)
	if !ok {
		return nil, fmt.Errorf("adapter missing")
	}
	return ad, nil
}

func (e *Engine) buildGenerationRequest(j Job, p JobParams, ch canvasChannel) (protocol.GenerationRequest, error) {
	req := protocol.GenerationRequest{
		Model:         firstNonEmpty(j.Model, ch.Model),
		Prompt:        j.Prompt,
		Duration:      p.Duration,
		AspectRatio:   p.AspectRatio,
		Resolution:    p.Resolution,
		Quality:       p.Size,
		GenerateAudio: p.GenerateAudio,
		Operation:     p.Operation,
	}
	switch p.Mode {
	case "image":
		req.Capability = protocol.CapabilityImage
	case "video":
		req.Capability = protocol.CapabilityVideo
	case "audio":
		req.Capability = protocol.CapabilityAudio
	default:
		req.Capability = protocol.CapabilityText
	}
	if len(p.GenerationSpec) > 0 {
		var spec contract.GenerationSpec
		if err := json.Unmarshal(p.GenerationSpec, &spec); err == nil {
			req.Prompt = firstNonEmpty(spec.Prompt, req.Prompt)
			if spec.Options.DurationSeconds != nil {
				req.Duration = *spec.Options.DurationSeconds
			}
			if spec.Options.Resolution != nil {
				req.Resolution = *spec.Options.Resolution
			}
			if spec.Options.GenerateAudio != nil {
				req.GenerateAudio = *spec.Options.GenerateAudio
			}
			if spec.Options.Size != nil {
				req.Quality = *spec.Options.Size
			}
			for _, b := range spec.ReferenceBindings {
				ref := protocol.MediaReference{ID: b.ResourceID, Role: b.Role, Kind: b.MediaType, Order: b.Order}
				if b.ResourceID != "" {
					if res, err := e.getCanvasResource(b.ResourceID); err == nil {
						ref.URL = res.PublicURL
						ref.MIMEType = res.MimeType
						if raw, err := e.GetBytes(res.CASHash); err == nil && len(raw) < 6<<20 {
							ref.DataURL = "data:" + res.MimeType + ";base64," + encodeB64(raw)
						}
					}
				}
				switch b.MediaType {
				case "video":
					req.Videos = append(req.Videos, ref)
				case "audio":
					req.Audios = append(req.Audios, ref)
				default:
					req.Images = append(req.Images, ref)
				}
				req.Inputs = append(req.Inputs, ref)
			}
		}
	}
	for _, h := range p.ReferenceImages {
		req.Images = append(req.Images, protocol.MediaReference{URL: h, Kind: "image", Role: "reference"})
	}
	for _, u := range p.ReferenceImageURLs {
		req.Images = append(req.Images, protocol.MediaReference{URL: u, Kind: "image", Role: "reference"})
	}
	if p.FirstFrameURL != "" {
		req.Images = append(req.Images, protocol.MediaReference{URL: p.FirstFrameURL, Kind: "image", Role: "first-frame"})
	}
	if p.LastFrameURL != "" {
		req.Images = append(req.Images, protocol.MediaReference{URL: p.LastFrameURL, Kind: "image", Role: "last-frame"})
	}
	return req, nil
}

func (e *Engine) doProtocol(ctx context.Context, spec protocol.RequestSpec, key, base string) ([]byte, int, error) {
	rawURL := spec.Path
	if spec.OriginPath || strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		// keep
	} else {
		rawURL = joinURL(base, "", spec.Path)
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	for k, vs := range spec.Query {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	u.RawQuery = q.Encode()
	var body io.Reader
	contentType := spec.ContentType
	if len(spec.Files) > 0 {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		if spec.Body != nil {
			fields := asMap(spec.Body)
			for k, v := range fields {
				_ = w.WriteField(k, fmt.Sprint(v))
			}
		}
		for _, f := range spec.Files {
			raw, err := e.mediaRefBytes(f.Reference)
			if err != nil {
				return nil, 0, err
			}
			part, err := w.CreateFormFile(f.Name, firstNonEmpty(f.Filename, f.Name))
			if err != nil {
				return nil, 0, err
			}
			_, _ = part.Write(raw)
		}
		_ = w.Close()
		body = &buf
		contentType = w.FormDataContentType()
	} else if spec.Body != nil {
		b, _ := json.Marshal(spec.Body)
		body = bytes.NewReader(b)
		if contentType == "" {
			contentType = "application/json"
		}
	}
	req, err := http.NewRequestWithContext(ctx, firstNonEmpty(spec.Method, http.MethodPost), u.String(), body)
	if err != nil {
		return nil, 0, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	if key != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("x-goog-api-key", key)
	}
	res, err := e.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return nil, res.StatusCode, err
	}
	if res.StatusCode >= 400 {
		return raw, res.StatusCode, fmt.Errorf("%s", strings.TrimSpace(string(raw)))
	}
	return raw, res.StatusCode, nil
}

func (e *Engine) mediaRefBytes(ref protocol.MediaReference) ([]byte, error) {
	if strings.HasPrefix(ref.DataURL, "data:") {
		return decodeDataURL(ref.DataURL)
	}
	if ref.ID != "" {
		if res, err := e.getCanvasResource(ref.ID); err == nil {
			return e.GetBytes(res.CASHash)
		}
	}
	if ref.URL != "" {
		return e.DownloadURL(ref.URL)
	}
	return nil, fmt.Errorf("empty media reference")
}

func (e *Engine) materializeProtocolResult(j *Job, result *protocol.Result) error {
	var raw []byte
	var video bool
	var mime string
	pick := func(list []protocol.MediaReference, isVideo bool) bool {
		if len(list) == 0 {
			return false
		}
		ref := list[0]
		b, err := e.mediaRefBytes(ref)
		if err != nil {
			return false
		}
		raw = b
		video = isVideo
		mime = ref.MIMEType
		return true
	}
	switch {
	case pick(result.Videos, true):
	case pick(result.Images, false):
	case pick(result.Audios, false):
		mime = firstNonEmpty(mime, "audio/mpeg")
	case result.Text != "":
		p := parseParams(j.Params)
		p.TextDraft = result.Text
		p.ResultState = "READY"
		j.Params = marshalJSON(p)
		j.Status = "succeeded"
		j.CompletedAt = Now()
		return nil
	default:
		return fmt.Errorf("empty provider result")
	}
	_ = mime
	return e.finishMedia(j, raw, video)
}

func (e *Engine) RecoverCanvasMedia(id string) (Job, error) {
	j, err := e.GetJob(id)
	if err != nil {
		return j, err
	}
	p := parseParams(j.Params)
	if p.RecoverAttempts >= 3 {
		return j, fmt.Errorf("media recovery exhausted")
	}
	p.RecoverAttempts++
	j.Params = marshalJSON(p)
	if j.ResultHash != "" {
		if _, err := e.GetBytes(j.ResultHash); err != nil {
			j.Status = "failed"
			j.Error = err.Error()
			_ = e.saveJob(j)
			return j, err
		}
		kind := firstNonEmpty(p.Mode, "image")
		res, err := e.putCanvasResourceFromHash(j.ResultHash, kind, "", 0, 0, 0, 0)
		if err != nil {
			j.Status = "failed"
			j.Error = err.Error()
			_ = e.saveJob(j)
			return j, err
		}
		p.ResourceID = res.ID
		p.ResultState = "READY"
		p.MediaStage = "completed"
		j.Params = marshalJSON(p)
		j.Status = "succeeded"
		j.Error = ""
		_ = e.saveJob(j)
		return j, nil
	}
	j.Status = "queued"
	j.Error = ""
	_ = e.saveJob(j)
	return j, nil
}
