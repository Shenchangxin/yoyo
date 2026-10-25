package film

import "strings"

// Endpoint is one enabled Yoyo connection the planner may call.
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

// Plan fills needs from the shots and refreshes input hashes.
// Asset hashes stay when the input hash is unchanged.
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

// Quote lists what a build may spend. Code steps are free of model charges.
func Quote(p Project) []Need {
	p = Plan(p)
	return p.Needs
}

// Swap applies the non-empty brief fields and rehashes needs.
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

// Fork copies a film and applies a brief patch. The parent keeps its assets.
func Fork(p Project, patch Brief) Project {
	child := cloneProject(p)
	child.ID = ""
	child.ParentID = p.ID
	child.Tasks = nil
	child.Artifacts = nil
	return Swap(child, patch)
}

// Localize rewrites the named lines and reflows only those lines.
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

func spoken(p Project) string { return joinTokens(p.Words) }

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

// Route picks an enabled connection for a need and refuses a hosted gateway of another product.
// An exact service match wins over a name that only looks related.
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

func rejected(ep Endpoint) bool {
	blob := strings.ToLower(ep.ID + " " + ep.Name + " " + ep.BaseURL + " " + ep.Service)
	return strings.Contains(blob, "hypihub")
}

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

func errNoConn(kind string) error { return noConn(kind) }

func (e noConn) Error() string { return "no enabled connection for " + string(e) }

type waitErr string

func (e waitErr) Error() string { return string(e) }

// Wait marks a selected step that cannot finish in this call.
func Wait(msg string) error { return waitErr(msg) }

func isWait(err error) bool {
	_, ok := err.(waitErr)
	return ok
}

// BuildEnv stores bytes and may call a connection. Both may be nil in a code-only build.
type BuildEnv struct {
	Providers []Endpoint
	Put       func([]byte) (string, error)
	Dispatch  func(Need) (string, error)
}

// Build runs planned needs. A failed generation records a task and continues.
// Unchanged input hashes keep their asset hashes.
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

// EncodeSRT writes one cue per word so captions can follow the word axis.
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

func pad(n int) string {
	if n < 10 {
		return "0" + itoa(n)
	}
	return itoa(n)
}

func pad3(n int) string {
	s := itoa(n)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}
