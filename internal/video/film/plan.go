// 按镜头填待办，并在输入没变时留下已有产物。
// 选连接时拒绝名称或地址里带 hypihub 的那条。

package film

import "strings"

// Endpoint 是计划可以调用的一条已启用连接。
type Endpoint struct {
	ID       string
	Name     string
	Service  string
	Vendor   string
	Model    string
	BaseURL  string
	VaultKey string
	Active   bool
	HasKey   bool
}

// Plan 按镜头填待办，并刷新输入摘要。输入摘要没变且已有产物时，产物编号留下，状态写成 skipped。
// broll 要图像。排行榜每一行另要一张图。有词就要语音。还没有垫乐就要配乐。街采要抠像。代码画面不产生模型费用。
func Plan(p Project) Project {
	prev := map[string]Need{}
	for _, n := range p.Needs {
		prev[n.Key] = n
	}
	var needs []Need
	add := func(n Need) {
		if old, ok := prev[n.Key]; ok && old.InputHash == n.InputHash && old.AssetHash != "" {
			n.AssetHash = old.AssetHash
			n.Status = "skipped"
			n.Provider = old.Provider
			n.Model = old.Model
		} else {
			n.AssetHash = ""
			if n.Status == "" {
				n.Status = "planned"
			}
		}
		needs = append(needs, n)
	}
	last := len(p.Shots) - 1
	for i, shot := range p.Shots {
		text := shotText(p, shot)
		mark := stickerOn(p, shot)
		switch shot.Visual {
		case "broll":
			payload := map[string]any{
				"kind": "image", "host": p.Brief.Host, "product": p.Brief.Product,
				"text": text, "aspect": p.Brief.Aspect,
			}
			prompt := strings.TrimSpace(p.Brief.Host + " " + p.Brief.Product + " " + text)
			if i == 0 {
				payload["hook"] = p.Brief.Hook
				prompt = strings.TrimSpace(prompt + " " + p.Brief.Hook)
			}
			if i == last {
				payload["price"] = p.Brief.Price
				payload["cta"] = p.Brief.CTA
				prompt = strings.TrimSpace(prompt + " " + p.Brief.Price + " " + p.Brief.CTA)
			}
			add(Need{
				ID: "need-" + shot.ID, Key: "image:" + shot.ID, Kind: "image",
				Prompt: prompt, Cost: "按图像连接计费", InputHash: digest(payload),
			})
		default:
			payload := map[string]any{
				"kind": "code", "visual": shot.Visual, "structure": p.Structure,
				"aspect": p.Brief.Aspect, "text": text, "title": p.Title, "sticker": mark,
			}
			if shot.Visual == "handoff" {
				payload["product"] = p.Brief.Product
				payload["price"] = p.Brief.Price
				payload["cta"] = p.Brief.CTA
			}
			if p.Structure == "ranking" || shot.Visual == "reveal" || shot.Visual == "board" {
				payload["rows"] = rowLabels(p, i, last)
			}
			add(Need{
				ID: "need-" + shot.ID, Key: "code:" + shot.ID, Kind: "code",
				Prompt: text, Cost: "无模型花费", InputHash: digest(payload),
			})
		}
	}
	if p.Structure == "ranking" {
		for _, entry := range p.Entries {
			if entry.ID == "" || entry.Label == "" {
				continue
			}
			add(Need{
				ID: "need-entry-" + entry.ID, Key: "image:entry:" + entry.ID, Kind: "image",
				Prompt: entry.Label, Cost: "按图像连接计费",
				InputHash: digest(map[string]any{"kind": "entry", "label": entry.Label, "aspect": p.Brief.Aspect}),
			})
		}
	}
	if len(p.Words) > 0 {
		add(Need{
			ID: "need-speech", Key: "speech", Kind: "speech",
			Prompt: spoken(p), Cost: "按语音连接计费",
			InputHash: digest(map[string]any{"kind": "speech", "language": p.Brief.Language, "text": spoken(p)}),
		})
	}
	if strings.TrimSpace(p.Bed) == "" {
		prompt := musicPrompt(p)
		add(Need{
			ID: "need-music", Key: "music", Kind: "music",
			Prompt: prompt, Cost: "按配乐连接计费",
			InputHash: digest(map[string]any{"kind": "music", "prompt": prompt}),
		})
	}
	if p.Structure == "interview" {
		add(Need{
			ID: "need-matte", Key: "matting", Kind: "matting",
			Prompt: p.Reference.Hash, Cost: "按抠像连接计费",
			InputHash: digest(map[string]any{"kind": "matting", "hash": p.Reference.Hash, "structure": p.Structure}),
		})
	}
	p.Needs = needs
	return p
}

// Quote 先重新计划，再返回可能产生花费的步骤。代码画面的花费写成无模型花费。
func Quote(p Project) []Need {
	p = Plan(p)
	return p.Needs
}

// Swap 把简介里非空的字段写上，并重算待办。换开头时只重写第一行的词。
func Swap(p Project, patch Brief) Project {
	if clean(patch.Host) != "" {
		p.Brief.Host = clean(patch.Host)
	}
	if clean(patch.Product) != "" {
		p.Brief.Product = clean(patch.Product)
	}
	if clean(patch.Language) != "" {
		p.Brief.Language = clean(patch.Language)
	}
	if clean(patch.Aspect) != "" {
		p.Brief.Aspect = clean(patch.Aspect)
	}
	if clean(patch.Price) != "" {
		p.Brief.Price = clean(patch.Price)
	}
	if clean(patch.CTA) != "" {
		p.Brief.CTA = clean(patch.CTA)
	}
	if clean(patch.Hook) != "" {
		p.Brief.Hook = clean(patch.Hook)
		if line := openingLine(p.Words); line != "" {
			return ReplaceLine(p, line, clean(patch.Hook))
		}
	}
	return rehash(p)
}

// Fork 复制一部片子再套上简介补丁。子片记下父片编号，不带走产物和任务。父片自己的产物不动。
func Fork(p Project, patch Brief) Project {
	child := cloneProject(p)
	child.ID = ""
	child.ParentID = p.ID
	child.Tasks = nil
	child.Artifacts = nil
	return Swap(child, patch)
}

// Localize 按行号重写给出的句子，并只重排这些行的时间。没给出的行不动。
func Localize(p Project, lines map[string]string) Project {
	for _, line := range LineTexts(p) {
		text := ""
		if lines != nil {
			text = clean(lines[line.ID])
		}
		if text == "" {
			continue
		}
		p = ReplaceLine(p, line.ID, text)
	}
	return p
}

// openingLine 返回开始最早的那一行。没有词就返回空串。
func openingLine(words []Word) string {
	if len(words) == 0 {
		return ""
	}
	best := words[0]
	for _, w := range words[1:] {
		if w.Start < best.Start {
			best = w
		}
	}
	return best.LineID
}

// stickerOn 返回钉在这一镜某个词上的贴纸文字。有多条时留下后遇到的那条。
func stickerOn(p Project, shot Shot) string {
	var text string
	for _, sticker := range p.Stickers {
		for _, id := range shot.WordIDs {
			if sticker.WordID == id {
				text = sticker.Text
			}
		}
	}
	return text
}

// rowLabels 返回排到当前这一镜为止的行标签，用竖线接起来。最后一镜列出全部行。
func rowLabels(p Project, index, last int) string {
	rows := p.Entries
	n := index + 1
	if index == last {
		n = len(rows)
	}
	if n > len(rows) {
		n = len(rows)
	}
	if n < 0 {
		n = 0
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(rows[i].Label)
	}
	return b.String()
}

// rehash 清掉待办再按现在的内容重算。输入摘要没变的产物编号会抄回来。
func rehash(p Project) Project {
	old := p.Needs
	p.Needs = nil
	p = Plan(p)
	byKey := map[string]Need{}
	for _, n := range old {
		byKey[n.Key] = n
	}
	for i := range p.Needs {
		prev, ok := byKey[p.Needs[i].Key]
		if ok && prev.InputHash == p.Needs[i].InputHash && prev.AssetHash != "" {
			p.Needs[i].AssetHash = prev.AssetHash
			p.Needs[i].Status = "skipped"
		}
	}
	return p
}

// spoken 把整条词轴拼成要念的正文。
func spoken(p Project) string { return joinTokens(p.Words) }

// musicPrompt 拼一条不要人声的垫乐提示，带上产品、结构和语言。
func musicPrompt(p Project) string {
	bits := []string{"instrumental bed, no vocals"}
	if s := clean(p.Brief.Product); s != "" {
		bits = append(bits, s)
	}
	if s := clean(p.Structure); s != "" {
		bits = append(bits, s)
	}
	if s := clean(p.Brief.Language); s != "" {
		bits = append(bits, s)
	}
	return strings.Join(bits, ", ")
}

// cloneProject 复制词、镜头、贴纸、待办和条目。区域框单独拷一份。产物和任务不带上。
func cloneProject(p Project) Project {
	p.Words = append([]Word(nil), p.Words...)
	shots := make([]Shot, len(p.Shots))
	for i, shot := range p.Shots {
		shot.WordIDs = append([]string(nil), shot.WordIDs...)
		if shot.Region != nil {
			box := *shot.Region
			shot.Region = &box
		}
		shots[i] = shot
	}
	p.Shots = shots
	p.Stickers = append([]Sticker(nil), p.Stickers...)
	p.Needs = append([]Need(nil), p.Needs...)
	p.Entries = append([]Entry(nil), p.Entries...)
	p.Artifacts = nil
	p.Tasks = nil
	return p
}

// Route 为这一步挑一条已启用且有密钥的连接。名称或地址里带 hypihub 的拒绝。服务种类完全对上的优先于只是名字像。都没有就返回错误。
func Route(kind string, list []Endpoint) (Endpoint, error) {
	kind = clean(kind)
	var loose Endpoint
	foundLoose := false
	for _, ep := range list {
		if rejected(ep) || !ep.Active || !ep.HasKey {
			continue
		}
		exact, ok := classify(kind, ep)
		if !ok {
			continue
		}
		if exact {
			return ep, nil
		}
		if !foundLoose {
			loose = ep
			foundLoose = true
		}
	}
	if foundLoose {
		return loose, nil
	}
	return Endpoint{}, errNoConn(kind)
}

// rejected 判断编号、名字、地址或服务里带不带 hypihub。
func rejected(ep Endpoint) bool {
	blob := strings.ToLower(ep.ID + " " + ep.Name + " " + ep.BaseURL + " " + ep.Service)
	return strings.Contains(blob, "hypihub")
}

// classify 判断这条连接能不能做这一步。图像、视频、语音和配乐必须服务种类完全一致。抠像优先要 matting，图像连接的名字像抠像时也能用，但不是精确匹配。
func classify(kind string, ep Endpoint) (exact bool, ok bool) {
	service := strings.ToLower(clean(ep.Service))
	model := strings.ToLower(ep.Model + " " + ep.Name)
	switch kind {
	case "image":
		return true, service == "image"
	case "video":
		return true, service == "video"
	case "speech":
		return true, service == "speech"
	case "music":
		return true, service == "music"
	case "matting":
		if service == "matting" {
			return true, true
		}
		if service == "image" && (strings.Contains(model, "matting") || strings.Contains(model, "matte") || strings.Contains(model, "抠")) {
			return false, true
		}
		return false, false
	default:
		return false, false
	}
}

type noConn string

// errNoConn 表示这一步没有可用连接。
func errNoConn(kind string) error { return noConn(kind) }

// Error 说明没有已启用的连接能做这一步。
func (e noConn) Error() string { return "no enabled connection for " + string(e) }

type waitErr string

// Error 返回这一步这次做不完的原因。
func (e waitErr) Error() string { return string(e) }

// Wait 表示这一步已经选中，但这一次调用做不完。
func Wait(msg string) error { return waitErr(msg) }

// isWait 判断错误是不是 Wait 返回的。
func isWait(err error) bool {
	_, ok := err.(waitErr)
	return ok
}

// BuildEnv 负责存字节，并可以把一步交给连接。只画代码画面时两者都可以是 nil。
type BuildEnv struct {
	Providers []Endpoint
	// Put 把字节放进内容库。没有它时，产物编号用内容的摘要。
	Put func([]byte) (string, error)
	// Dispatch 把图像、视频、语音、抠像或配乐交给连接。没有它时，这些步骤记成等待。
	Dispatch func(Need) (string, error)
}

// Build 先重新计划，再跑每一步。代码画面在本机画出。其余步骤先选连接再交给 Dispatch。
// 一步失败会记下任务并继续。输入没变的产物直接跳过。词轴会另写一份一词一条的字幕。
func Build(p Project, env BuildEnv) Project {
	p = Plan(p)
	p.Tasks = nil
	var artifacts []Artifact
	for i := range p.Needs {
		n := &p.Needs[i]
		task := Task{ID: "task-" + n.Key, NeedID: n.ID}
		if n.Status == "skipped" && n.AssetHash != "" {
			task.Status = "skipped"
			p.Tasks = append(p.Tasks, task)
			artifacts = append(artifacts, Artifact{Name: n.Key, Hash: n.AssetHash, MediaType: mediaOf(n.Kind)})
			continue
		}
		switch n.Kind {
		case "code":
			png, err := RenderCode(p, shotByNeed(p, n.Key))
			if err != nil {
				n.Status = "failed"
				n.Error = err.Error()
				task.Status = "failed"
				task.Error = n.Error
				break
			}
			hash := digest(png)
			if env.Put != nil {
				if stored, err := env.Put(png); err == nil && stored != "" {
					hash = stored
				}
			}
			n.AssetHash = hash
			n.Status = "done"
			task.Status = "done"
			artifacts = append(artifacts, Artifact{Name: n.Key, Hash: hash, MediaType: "image/png"})
		default:
			ep, err := Route(n.Kind, env.Providers)
			if err != nil {
				n.Status = "waiting"
				n.Error = err.Error()
				task.Status = "waiting"
				task.Error = n.Error
				break
			}
			n.Provider = clean(ep.Vendor)
			if n.Provider == "" {
				n.Provider = ep.ID
			}
			n.Model = ep.Model
			n.VaultKey = ep.VaultKey
			if env.Dispatch == nil {
				n.Status = "waiting"
				n.Error = "connection is selected and the runner is not attached"
				task.Status = "waiting"
				task.Error = n.Error
				break
			}
			hash, err := env.Dispatch(*n)
			if err != nil {
				if isWait(err) {
					n.Status = "waiting"
					task.Status = "waiting"
				} else {
					n.Status = "failed"
					task.Status = "failed"
				}
				n.Error = err.Error()
				task.Error = n.Error
				break
			}
			n.AssetHash = hash
			n.Status = "done"
			task.Status = "done"
			artifacts = append(artifacts, Artifact{Name: n.Key, Hash: hash, MediaType: mediaOf(n.Kind)})
		}
		p.Tasks = append(p.Tasks, task)
	}
	if srt := EncodeSRT(p.Words); srt != "" {
		hash := digest(srt)
		if env.Put != nil {
			if stored, err := env.Put([]byte(srt)); err == nil && stored != "" {
				hash = stored
			}
		}
		artifacts = append(artifacts, Artifact{Name: "captions.srt", Hash: hash, MediaType: "application/x-subrip"})
	}
	p.Artifacts = artifacts
	return p
}

// shotByNeed 按 code:镜头编号 找回镜头。对不上就用第一镜。一镜都没有就返回一个代码画面。
func shotByNeed(p Project, key string) Shot {
	id := strings.TrimPrefix(key, "code:")
	for _, shot := range p.Shots {
		if shot.ID == id {
			return shot
		}
	}
	if len(p.Shots) > 0 {
		return p.Shots[0]
	}
	return Shot{Visual: "code"}
}

// mediaOf 给出这一步产物的媒体类型。语音和配乐是音频，视频是 mp4，其余是 PNG。
func mediaOf(kind string) string {
	switch kind {
	case "speech", "music":
		return "audio/mpeg"
	case "video":
		return "video/mp4"
	default:
		return "image/png"
	}
}

// EncodeSRT 一词写一条字幕，方便字幕跟着词轴走。没有词就返回空串。有说话人时写在词前面。
func EncodeSRT(words []Word) string {
	if len(words) == 0 {
		return ""
	}
	var b strings.Builder
	for i, w := range words {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(itoa(i + 1))
		b.WriteByte('\n')
		b.WriteString(stamp(w.Start))
		b.WriteString(" --> ")
		b.WriteString(stamp(w.End))
		b.WriteByte('\n')
		if w.Speaker != "" {
			b.WriteString(w.Speaker)
			b.WriteString(": ")
		}
		b.WriteString(w.Text)
		b.WriteString("\n")
	}
	return b.String()
}

// stamp 把秒写成 SRT 的时:分:秒,毫秒。负数当成 0。
func stamp(sec float64) string {
	if sec < 0 {
		sec = 0
	}
	ms := int(sec*1000 + 0.5)
	h := ms / 3600000
	ms %= 3600000
	m := ms / 60000
	ms %= 60000
	s := ms / 1000
	ms %= 1000
	return pad(h) + ":" + pad(m) + ":" + pad(s) + "," + pad3(ms)
}

// pad 把整数写成至少两位。
func pad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

// pad3 把整数写成至少三位，给毫秒用。
func pad3(n int) string {
	s := itoa(n)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}
