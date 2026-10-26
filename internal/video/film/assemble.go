// 从参考片或三个结构模板拼出工程。排行榜、播客和街采各有一套默认词和镜头。
// 不用别的产品里的样片题目。

package film

import "strings"

// FromReference 用已存的参考或一段描述建工程。标题空着写成「翻拍」，结构空着写成 free。
// 有词就用词，否则把句子均分成词。没有区域时每段先占满画面。第一个词上钉一条贴纸。建完会填待办。
// in 的 Title、Structure 经 clean 后为空时分别写成「翻拍」和 free；ID、Hash、URL、Duration 写入工程，Cues、Description、Words 用来填词，Scenes 传给 shotsFrom，Regions 为空时改用 DefaultRegions。Duration 小于等于 0 且没有字幕时，这一句的结束时间用 12。
func FromReference(in Input) Project {
	title := clean(in.Title)
	if title == "" {
		title = "翻拍"
	}
	structure := clean(in.Structure)
	if structure == "" {
		structure = "free"
	}
	p := Project{
		ID: in.ID, Title: title, Structure: structure,
		Brief:     Brief{Language: "zh", Aspect: "9:16"},
		Reference: Reference{Hash: clean(in.Hash), URL: clean(in.URL), Duration: in.Duration},
	}
	cues := append([]Cue(nil), in.Cues...)
	if len(cues) == 0 && clean(in.Description) != "" {
		end := in.Duration
		if end <= 0 {
			end = 12
		}
		cues = []Cue{{Start: 0, End: end, Text: in.Description}}
	}
	if len(in.Words) > 0 {
		p.Words = append([]Word(nil), in.Words...)
	} else {
		p.Words = WordsFromCues(cues)
	}
	p.Shots = shotsFrom(in.Scenes, p.Words, in.Duration, structure)
	boxes := in.Regions
	if len(boxes) == 0 {
		boxes = DefaultRegions(in.Duration, in.Scenes)
	}
	p.Shots = assignRegions(p.Shots, boxes)
	if len(p.Words) > 0 {
		p.Stickers = []Sticker{{ID: "sticker-0", WordID: p.Words[0].ID, Text: "这条值得看"}}
	}
	return Plan(p)
}

// FromTemplate 按排行榜、播客或街采建一部自己的工程。认不出的结构用排行榜。不用别的产品的样片。
// structure 经 clean 后为 podcast 时传给 podcast，为 interview 时传给 interview，其余传给 ranking。
// title 传给选中的模板函数。
func FromTemplate(structure, title string) Project {
	switch clean(structure) {
	case "podcast":
		return podcast(title)
	case "interview":
		return interview(title)
	default:
		return ranking(title)
	}
}

// shotsFrom 按场景切点生成镜头，并把落在这一段里的词挂上。起点尽量落在第一个词上。没有切点就做一整段。
// scenes 传给 sceneBounds。
// words 用来收集落在该段且 LineID 与段内第一个词相同的 ID；有词时起点改成第一个词的 Start。
// duration 传给 sceneBounds；一个镜头都没有且 duration 小于等于 0 时，结束时间用 12。
// structure 传给 visualFor。
func shotsFrom(scenes []float64, words []Word, duration float64, structure string) []Shot {
	bounds := sceneBounds(scenes, duration, words)
	shots := make([]Shot, 0, len(bounds))
	for i, b := range bounds {
		line := ""
		var ids []string
		for _, w := range words {
			if w.End <= b[0] || w.Start >= b[1] {
				continue
			}
			if line == "" {
				line = w.LineID
			}
			if w.LineID == line {
				ids = append(ids, w.ID)
			}
		}
		visual := visualFor(structure, i, len(bounds))
		start := b[0]
		if len(ids) > 0 {
			for _, w := range words {
				if w.ID == ids[0] {
					start = w.Start
					break
				}
			}
		}
		shots = append(shots, Shot{
			ID: "shot-" + itoa(i), LineID: line, Visual: visual, WordIDs: ids,
			Start: start, End: b[1], Beat: len(ids) > 0,
		})
	}
	if len(shots) == 0 {
		end := duration
		if end <= 0 {
			end = 12
		}
		shots = []Shot{{ID: "shot-0", Visual: visualFor(structure, 0, 1), Start: 0, End: end}}
	}
	return shots
}

// sceneBounds 把场景切点收成互不重叠的时间段。时长未知时用最后一个词的结束，再没有就用 12 秒。
// scenes 里大于 0 且小于 duration 的点作为切点。
// duration 小于等于 0 时先改用 words 里最大的 End，仍然小于等于 0 时用 12。
// words 在 duration 小于等于 0 时提供 End。
func sceneBounds(scenes []float64, duration float64, words []Word) [][2]float64 {
	if duration <= 0 {
		for _, w := range words {
			if w.End > duration {
				duration = w.End
			}
		}
	}
	if duration <= 0 {
		duration = 12
	}
	pts := []float64{0}
	for _, s := range scenes {
		if s > 0 && s < duration {
			pts = append(pts, s)
		}
	}
	var out [][2]float64
	for i := 0; i < len(pts); i++ {
		end := duration
		if i+1 < len(pts) {
			end = pts[i+1]
		}
		if end > pts[i] {
			out = append(out, [2]float64{pts[i], end})
		}
	}
	if len(out) == 0 {
		out = [][2]float64{{0, duration}}
	}
	return out
}

// visualFor 按结构给这一段一个画面种类。排行榜除最后一段是板，最后一段是 broll。播客开头是分屏。街采是揭示。其余开头是代码画面。
// structure 为 ranking、podcast、interview 时走对应分支，其余走 default。
// index 在 ranking 里等于 n 减 1 时返回 broll，在 podcast 和 default 里等于 0 时分别返回 split 和 code。
// n 在 ranking 里与 index 比较。
func visualFor(structure string, index, n int) string {
	switch structure {
	case "ranking":
		if index == n-1 {
			return "broll"
		}
		return "board"
	case "podcast":
		if index == 0 {
			return "split"
		}
		return "broll"
	case "interview":
		return "reveal"
	default:
		if index == 0 {
			return "code"
		}
		return "broll"
	}
}

// DefaultRegions 在本机检测还没给出更紧的框之前，给每一段一个占满画面的区域。
// duration 作为 sceneBounds 的时长。
// scenes 传给 sceneBounds。
func DefaultRegions(duration float64, scenes []float64) []Box {
	bounds := sceneBounds(scenes, duration, nil)
	out := make([]Box, len(bounds))
	for i := range bounds {
		out[i] = Box{X: 0, Y: 0, W: 1, H: 1, Label: "画面"}
	}
	return out
}

// assignRegions 按顺序把区域框写到镜头上。框比镜头少时，多出来的镜头不动。
// shots 按序号写入 Region，序号达到 boxes 的长度就停止。
// boxes 按同样序号拷进对应镜头的 Region。
func assignRegions(shots []Shot, boxes []Box) []Shot {
	for i := range shots {
		if i >= len(boxes) {
			break
		}
		box := boxes[i]
		shots[i].Region = &box
	}
	return shots
}

// ApplyRegions 换上检测出的区域框，并重算待办。检测可以给出脸或主体。
// boxes 传给 assignRegions。
func ApplyRegions(p Project, boxes []Box) Project {
	p.Shots = assignRegions(p.Shots, boxes)
	return rehash(p)
}

// ranking 建一部排行榜。标题空着写成「本机工具排行」。最后会填待办。
// title 经 clean 后为空时写成「本机工具排行」，再传给 base。
func ranking(title string) Project {
	if clean(title) == "" {
		title = "本机工具排行"
	}
	p := base(title, "ranking", Brief{Host: "剪辑台", Product: "创作", Language: "zh", Aspect: "9:16", Hook: "先看排行"})
	p.Entries = []Entry{{ID: "e0", Label: "剪辑台"}, {ID: "e1", Label: "创作"}, {ID: "e2", Label: "成片台"}}
	p.Words = []Word{
		{ID: "a-0", LineID: "a", Text: "剪辑台", Speaker: "旁白", Start: 0, End: 2},
		{ID: "a-1", LineID: "a", Text: "放在", Speaker: "旁白", Start: 2, End: 3},
		{ID: "a-2", LineID: "a", Text: "最上", Speaker: "旁白", Start: 3, End: 4},
		{ID: "b-0", LineID: "b", Text: "创作", Speaker: "旁白", Start: 4, End: 6},
		{ID: "b-1", LineID: "b", Text: "负责", Speaker: "旁白", Start: 6, End: 8},
		{ID: "b-2", LineID: "b", Text: "成片", Speaker: "旁白", Start: 8, End: 10},
	}
	p.Shots = []Shot{
		{ID: "shot-0", LineID: "a", Visual: "board", WordIDs: []string{"a-0", "a-1", "a-2"}, Start: 0, End: 4, Beat: true, Region: &Box{X: 0.2, Y: 0.1, W: 0.6, H: 0.3, Label: "标题"}},
		{ID: "shot-1", LineID: "b", Visual: "broll", WordIDs: []string{"b-0", "b-1", "b-2"}, Start: 4, End: 10, Beat: true, Region: &Box{X: 0, Y: 0, W: 1, H: 1, Label: "画面"}},
	}
	p.Stickers = []Sticker{{ID: "sticker-0", WordID: "a-0", Text: "这条排第一"}}
	p.Reference.Duration = 10
	return Plan(p)
}

// podcast 建一部播客分屏。标题空着写成「创作台对谈」。中间一镜交出产品。
// title 经 clean 后为空时写成「创作台对谈」，再传给 base。
func podcast(title string) Project {
	if clean(title) == "" {
		title = "创作台对谈"
	}
	p := base(title, "podcast", Brief{Host: "主播", Product: "创作台", Language: "zh", Aspect: "16:9", Hook: "先把问题说完"})
	p.Words = []Word{
		{ID: "a-0", LineID: "a", Text: "你", Speaker: "主播", Start: 0, End: 1},
		{ID: "a-1", LineID: "a", Text: "怎么", Speaker: "主播", Start: 1, End: 2},
		{ID: "a-2", LineID: "a", Text: "剪", Speaker: "主播", Start: 2, End: 3},
		{ID: "p-0", LineID: "p", Text: "交给", Speaker: "主播", Start: 3, End: 4},
		{ID: "p-1", LineID: "p", Text: "创作台", Speaker: "主播", Start: 4, End: 5},
		{ID: "b-0", LineID: "b", Text: "先", Speaker: "嘉宾", Start: 5, End: 6},
		{ID: "b-1", LineID: "b", Text: "留", Speaker: "嘉宾", Start: 6, End: 7},
		{ID: "b-2", LineID: "b", Text: "词", Speaker: "嘉宾", Start: 7, End: 8},
	}
	p.Shots = []Shot{
		{ID: "shot-0", LineID: "a", Visual: "split", WordIDs: []string{"a-0", "a-1", "a-2"}, Start: 0, End: 3, Beat: true, Region: &Box{X: 0, Y: 0.2, W: 0.5, H: 0.45, Label: "主播"}},
		{ID: "shot-1", LineID: "p", Visual: "handoff", WordIDs: []string{"p-0", "p-1"}, Start: 3, End: 5, Beat: true, Region: &Box{X: 0.2, Y: 0.35, W: 0.6, H: 0.3, Label: "产品"}},
		{ID: "shot-2", LineID: "b", Visual: "split", WordIDs: []string{"b-0", "b-1", "b-2"}, Start: 5, End: 8, Beat: true, Region: &Box{X: 0.5, Y: 0.2, W: 0.5, H: 0.45, Label: "嘉宾"}},
	}
	p.Stickers = []Sticker{{ID: "sticker-0", WordID: "b-2", Text: "词比秒重要"}}
	p.Reference.Duration = 8
	return Plan(p)
}

// interview 建一部街采揭示。标题空着写成「三条成片规矩」。每一镜都是揭示。
// title 经 clean 后为空时写成「三条成片规矩」，再传给 base。
func interview(title string) Project {
	if clean(title) == "" {
		title = "三条成片规矩"
	}
	p := base(title, "interview", Brief{Host: "路人", Product: "成片", Language: "zh", Aspect: "9:16", Hook: "先说第一条"})
	p.Words = []Word{
		{ID: "a-0", LineID: "a", Text: "先", Speaker: "路人", Start: 0, End: 1},
		{ID: "a-1", LineID: "a", Text: "看", Speaker: "路人", Start: 1, End: 2},
		{ID: "a-2", LineID: "a", Text: "完", Speaker: "路人", Start: 2, End: 3},
		{ID: "b-0", LineID: "b", Text: "再", Speaker: "路人", Start: 3, End: 4},
		{ID: "b-1", LineID: "b", Text: "换", Speaker: "路人", Start: 4, End: 5},
		{ID: "b-2", LineID: "b", Text: "一句", Speaker: "路人", Start: 5, End: 6},
		{ID: "c-0", LineID: "c", Text: "最后", Speaker: "路人", Start: 6, End: 7},
		{ID: "c-1", LineID: "c", Text: "才", Speaker: "路人", Start: 7, End: 8},
		{ID: "c-2", LineID: "c", Text: "出片", Speaker: "路人", Start: 8, End: 9},
	}
	p.Shots = []Shot{
		{ID: "shot-0", LineID: "a", Visual: "reveal", WordIDs: []string{"a-0", "a-1", "a-2"}, Start: 0, End: 3, Beat: true, Region: &Box{X: 0.3, Y: 0.15, W: 0.4, H: 0.35, Label: "脸"}},
		{ID: "shot-1", LineID: "b", Visual: "reveal", WordIDs: []string{"b-0", "b-1", "b-2"}, Start: 3, End: 6, Beat: true, Region: &Box{X: 0.3, Y: 0.15, W: 0.4, H: 0.35, Label: "脸"}},
		{ID: "shot-2", LineID: "c", Visual: "reveal", WordIDs: []string{"c-0", "c-1", "c-2"}, Start: 6, End: 9, Beat: true, Region: &Box{X: 0.3, Y: 0.15, W: 0.4, H: 0.35, Label: "脸"}},
	}
	p.Entries = []Entry{{ID: "e0", Label: "先看完"}, {ID: "e1", Label: "再换一句"}, {ID: "e2", Label: "最后出片"}}
	p.Stickers = []Sticker{{ID: "sticker-0", WordID: "a-0", Text: "第一条"}}
	p.Reference.Duration = 9
	return Plan(p)
}

// base 建一部只有标题、结构和简介的空工程。
// title 写入 Title。
// structure 写入 Structure。
// brief 写入 Brief。
func base(title, structure string, brief Brief) Project {
	return Project{Title: title, Structure: structure, Brief: brief}
}

// shotText 把这一镜挂着的词按顺序拼成一句。
// p 用 Words 按这一镜 WordIDs 的顺序拼 Text，中间加空格。
func shotText(p Project, shot Shot) string {
	var b strings.Builder
	for _, id := range shot.WordIDs {
		for _, w := range p.Words {
			if w.ID == id {
				if b.Len() > 0 {
					b.WriteString(" ")
				}
				b.WriteString(w.Text)
			}
		}
	}
	return b.String()
}
