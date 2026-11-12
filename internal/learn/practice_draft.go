// 进题库之前的一道题：导入、出题、手工录入和导师存题共用的形状和校验。
// 松散的输入先变成 DraftQuestion，再由 normalizeDraft 收成能批改的样子：题型、选项字母和标准答案的写法只在这里定。
// 这里不读写库，也不调用模型。

package learn

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	maxDraftOptions = 8
	maxDraftTags    = 12
)

// DraftQuestion 是一道还没进题库的题，带标准答案。选择题的 Answer 是选项字母，多选用逗号隔开；判断题是 T 或 F；
// 填空题里用竖线隔开的每一种写法都算对。Tags 可空。
// 从 JSON 解出来时接受导入文件用的那些键名，选项可以是字符串数组，也可以写在 A 到 H 这几个键下。解出来的内容还没校验。
type DraftQuestion struct {
	Stem        string   `json:"stem"`
	Type        string   `json:"qtype"`
	Options     []Option `json:"options"`
	Answer      string   `json:"answer"`
	Explanation string   `json:"explanation"`
	Difficulty  string   `json:"difficulty"`
	Tags        []string `json:"tags,omitempty"`
}

// UnmarshalJSON 从一个 JSON 对象解出题目。data 不是对象时返回错误。键名和选项的写法见 draftFromJSON。
func (d *DraftQuestion) UnmarshalJSON(data []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf("a question must be a JSON object")
	}
	*d = draftFromJSON(obj)
	return nil
}

// draftField 把导入文件的列名或 JSON 的键认成题目的一项，不分大小写。认不出时 field 为空。
// 题干是 question、stem、题目、题干；题型是 type、qtype、题型；答案是 answer、答案、正确答案；解析是 explanation、解析；
// 难度是 difficulty、难度；标签是 tags、标签；整列选项是 options、选项。
// A 到 H 单独一个字母，或者前面带「选项」或 option，是一个选项列：field 是 option，label 是那个字母。
func draftField(name string) (field, label string) {
	key := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(name, "\ufeff")))
	switch key {
	case "question", "stem", "题目", "题干":
		return "stem", ""
	case "type", "qtype", "题型":
		return "qtype", ""
	case "answer", "答案", "正确答案":
		return "answer", ""
	case "explanation", "解析":
		return "explanation", ""
	case "difficulty", "难度":
		return "difficulty", ""
	case "tags", "标签":
		return "tags", ""
	case "options", "选项":
		return "options", ""
	}
	rest := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(key, "选项"), "option"))
	if len(rest) == 1 && rest[0] >= 'a' && rest[0] <= 'h' {
		return "option", strings.ToUpper(rest)
	}
	return "", ""
}

// draftFromJSON 把一个 JSON 对象收成还没校验的题目。键名按 draftField 认，同一项有两个键时取键名排在前面且有内容的那个。
// options 可以是字符串数组、带 label 和 body 的对象数组，或者一段用竖线或换行隔开的文字。A 到 H 的键有内容时以它们为准。
// tags 可以是数组，也可以是一段用逗号或分号隔开的文字。答案是数组时各项用竖线连起来，多选的几项和填空的几种写法都读得出。
func draftFromJSON(obj map[string]json.RawMessage) DraftQuestion {
	names := make([]string, 0, len(obj))
	for name := range obj {
		names = append(names, name)
	}
	sort.Strings(names)
	var d DraftQuestion
	lettered := map[string]string{}
	set := func(dst *string, raw json.RawMessage) {
		if *dst == "" {
			*dst = jsonText(raw)
		}
	}
	for _, name := range names {
		raw := obj[name]
		field, label := draftField(name)
		switch field {
		case "stem":
			set(&d.Stem, raw)
		case "qtype":
			set(&d.Type, raw)
		case "answer":
			set(&d.Answer, raw)
		case "explanation":
			set(&d.Explanation, raw)
		case "difficulty":
			set(&d.Difficulty, raw)
		case "tags":
			if len(d.Tags) == 0 {
				d.Tags = jsonTags(raw)
			}
		case "options":
			if len(d.Options) == 0 {
				d.Options = jsonOptions(raw)
			}
		case "option":
			if body := strings.TrimSpace(jsonText(raw)); body != "" {
				lettered[label] = body
			}
		}
	}
	if len(lettered) > 0 {
		d.Options = letteredOptions(lettered)
	}
	return d
}

// jsonText 把一个 JSON 值读成文字。字符串原样返回，数组的各项用竖线连起来，null 是空串，数字和布尔值返回它们的字面写法。
func jsonText(raw json.RawMessage) string {
	text := strings.TrimSpace(string(raw))
	if text == "" || text == "null" {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var list []json.RawMessage
	if json.Unmarshal(raw, &list) == nil {
		parts := make([]string, 0, len(list))
		for _, item := range list {
			if part := strings.TrimSpace(jsonText(item)); part != "" {
				parts = append(parts, part)
			}
		}
		return strings.Join(parts, "|")
	}
	return text
}

// jsonTags 把 JSON 里的标签读成列表。raw 是数组时逐项读，是文字时按 splitTags 拆。
func jsonTags(raw json.RawMessage) []string {
	var list []json.RawMessage
	if json.Unmarshal(raw, &list) == nil {
		out := make([]string, 0, len(list))
		for _, item := range list {
			out = append(out, jsonText(item))
		}
		return out
	}
	return splitTags(jsonText(raw))
}

// jsonOptions 把 JSON 里的选项读成列表。raw 是数组时，每一项可以是正文字符串，也可以是带 label 和 body 的对象；
// 没有 body 的对象留成空正文，校验时会被丢掉。raw 是一段文字时按 splitOptions 拆。
func jsonOptions(raw json.RawMessage) []Option {
	var list []json.RawMessage
	if json.Unmarshal(raw, &list) != nil {
		return splitOptions(jsonText(raw))
	}
	out := make([]Option, 0, len(list))
	for _, item := range list {
		if strings.HasPrefix(strings.TrimSpace(string(item)), "{") {
			var o struct {
				Label string          `json:"label"`
				Body  json.RawMessage `json:"body"`
			}
			_ = json.Unmarshal(item, &o)
			out = append(out, Option{Label: o.Label, Body: jsonText(o.Body)})
			continue
		}
		out = append(out, Option{Body: jsonText(item)})
	}
	return out
}

// letteredOptions 把按字母登记的选项正文排成列表。byLabel 的键是 A 到 H，没有内容的字母跳过，字母保持原样。
func letteredOptions(byLabel map[string]string) []Option {
	out := make([]Option, 0, len(byLabel))
	for r := 'A'; r <= 'H'; r++ {
		if body := strings.TrimSpace(byLabel[string(r)]); body != "" {
			out = append(out, Option{Label: string(r), Body: body})
		}
	}
	return out
}

var optionPrefix = regexp.MustCompile(`^\s*([A-Ha-h])\s*[.．、)）:：]\s*(\S.*)$`)

// splitOptions 把一段文字按竖线或换行拆成选项。每一段都以「A.」这样的字母开头时，字母取出来当选项字母，否则字母留空由校验按顺序补上。
func splitOptions(text string) []Option {
	parts := strings.FieldsFunc(text, func(r rune) bool { return r == '|' || r == '\n' || r == '\r' })
	out := make([]Option, 0, len(parts))
	prefixed := true
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, Option{Body: part})
		if !optionPrefix.MatchString(part) {
			prefixed = false
		}
	}
	if prefixed {
		for i := range out {
			m := optionPrefix.FindStringSubmatch(out[i].Body)
			out[i] = Option{Label: strings.ToUpper(m[1]), Body: strings.TrimSpace(m[2])}
		}
	}
	return out
}

// splitTags 把一段文字按半角和全角的逗号、分号拆成标签。
func splitTags(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool { return strings.ContainsRune(",;，；", r) })
}

// cleanTags 去掉 tags 里的空白项和重复项（不分大小写），去掉开头的井号，每个标签最多 40 个符文，最多留 12 个。
func cleanTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = clipRunes(strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(tag), "#＃")), 40)
		key := strings.ToLower(tag)
		if tag == "" || seen[key] || len(out) >= maxDraftTags {
			continue
		}
		seen[key] = true
		out = append(out, tag)
	}
	return out
}

// normalizeQType 把题型的各种写法收成六种之一。raw 为空时，hasOptions 为真取单选，否则取简答。
// 认得的写法：choice、single、single_choice、单选；multi、multiple、多选；tf、true_false、判断；fill、fill_blank、填空；
// short、short_answer、简答；open、essay、问答。不分大小写，空格和连字符当下划线，末尾的「题」字去掉。其余返回错误。
func normalizeQType(raw string, hasOptions bool) (string, error) {
	key := strings.ToLower(strings.TrimSpace(raw))
	key = strings.TrimSuffix(strings.NewReplacer(" ", "_", "-", "_").Replace(key), "题")
	switch key {
	case "":
		if hasOptions {
			return QuestionChoice, nil
		}
		return QuestionShort, nil
	case "choice", "single", "single_choice", "单选":
		return QuestionChoice, nil
	case "multi", "multiple", "多选":
		return QuestionMulti, nil
	case "tf", "true_false", "判断":
		return QuestionTrueFalse, nil
	case "fill", "fill_blank", "填空":
		return QuestionFill, nil
	case "short", "short_answer", "简答":
		return QuestionShort, nil
	case "open", "essay", "问答":
		return QuestionOpen, nil
	}
	return "", fmt.Errorf("unknown question type %q: use choice, multi, true_false, fill, short, or open", strings.TrimSpace(raw))
}

// normalizeDifficulty 把难度收成 easy、medium 或 hard。简单、容易算 easy，中等、一般算 medium，困难、难算 hard。认不出时返回空串。
func normalizeDifficulty(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "easy", "简单", "容易":
		return "easy"
	case "medium", "中等", "一般":
		return "medium"
	case "hard", "困难", "难":
		return "hard"
	}
	return ""
}

// draftOptions 去掉正文为空的选项，正文截到 1000 个符文，字母转成大写。有字母缺失或重复时，全部按顺序从 A 起重排。
func draftOptions(in []Option) []Option {
	out := make([]Option, 0, len(in))
	for _, o := range in {
		body := clipRunes(strings.TrimSpace(o.Body), 1000)
		if body == "" {
			continue
		}
		out = append(out, Option{Label: strings.ToUpper(strings.TrimSpace(o.Label)), Body: body})
	}
	seen := map[string]bool{}
	relabel := false
	for _, o := range out {
		if o.Label == "" || seen[o.Label] {
			relabel = true
		}
		seen[o.Label] = true
	}
	if relabel {
		for i := range out {
			out[i].Label = string(rune('A' + i))
		}
	}
	return out
}

// hasHan 在 s 里有汉字时为真。
func hasHan(s string) bool {
	for _, r := range s {
		if r >= 0x4E00 && r <= 0x9FFF {
			return true
		}
	}
	return false
}

// trueFalseOptions 返回判断题固定的两个选项，字母是 T 和 F。stem 里有汉字时正文用中文。
func trueFalseOptions(stem string) []Option {
	if hasHan(stem) {
		return []Option{{Label: "T", Body: "正确"}, {Label: "F", Body: "错误"}}
	}
	return []Option{{Label: "T", Body: "True"}, {Label: "F", Body: "False"}}
}

// normalizeDraft 校验一道题并收成能进题库的样子。题干不能空。
// 单选和多选要有 2 到 8 个正文各不相同的选项，答案必须指向已有的选项（字母或完全相同的正文）；选项从 A 起重排字母，答案改写成字母。
// 判断题的答案收成 T 或 F，选项换成固定的两项。填空题必须有答案。简答和开放题不留选项，答案可以为空。
// in 的题型为空时，有选项取单选，没有取简答。
func normalizeDraft(in DraftQuestion) (DraftQuestion, error) {
	out := DraftQuestion{
		Stem:        clipRunes(strings.TrimSpace(in.Stem), 8000),
		Options:     []Option{},
		Explanation: clipRunes(strings.TrimSpace(in.Explanation), 8000),
		Difficulty:  normalizeDifficulty(in.Difficulty),
		Tags:        cleanTags(in.Tags),
	}
	if out.Stem == "" {
		return DraftQuestion{}, fmt.Errorf("the question text is empty")
	}
	options := draftOptions(in.Options)
	qtype, err := normalizeQType(in.Type, len(options) > 0)
	if err != nil {
		return DraftQuestion{}, err
	}
	out.Type = qtype
	answer := clipRunes(strings.TrimSpace(in.Answer), 4000)
	switch qtype {
	case QuestionChoice, QuestionMulti:
		if len(options) < 2 || len(options) > maxDraftOptions {
			return DraftQuestion{}, fmt.Errorf("a %s question needs two to eight options", qtype)
		}
		bodies := map[string]bool{}
		for _, o := range options {
			key := strings.ToLower(o.Body)
			if bodies[key] {
				return DraftQuestion{}, fmt.Errorf("two options have the same text")
			}
			bodies[key] = true
		}
		var picked []string
		if qtype == QuestionChoice {
			label, ok := resolveChoice(options, answer)
			if !ok {
				return DraftQuestion{}, fmt.Errorf("the answer must name one option: its letter, or its exact text")
			}
			picked = []string{label}
		} else {
			labels, ok := parseLabels(options, answer)
			if !ok {
				return DraftQuestion{}, fmt.Errorf("the answer must name the correct options, such as A,C")
			}
			picked = labels
		}
		// 答案先按原来的字母认出来，再跟着选项一起换成从 A 起的新字母：中间有空选项的文件也不会指错。
		chosen := map[string]bool{}
		for _, label := range picked {
			chosen[label] = true
		}
		var labels []string
		for i := range options {
			next := string(rune('A' + i))
			if chosen[options[i].Label] {
				labels = append(labels, next)
			}
			options[i].Label = next
		}
		out.Options, out.Answer = options, strings.Join(labels, ",")
	case QuestionTrueFalse:
		value, ok := readTrueFalse(options, answer)
		if !ok {
			// 答案写的是选项字母时，看那个选项的正文是对还是错。
			if label, picked := resolveChoice(options, answer); picked {
				for _, o := range options {
					if o.Label == label {
						value, ok = parseTrueFalse(o.Body)
					}
				}
			}
		}
		if !ok {
			return DraftQuestion{}, fmt.Errorf("a true_false question needs an answer of T or F")
		}
		out.Options, out.Answer = trueFalseOptions(out.Stem), value
	case QuestionFill:
		if answer == "" {
			return DraftQuestion{}, fmt.Errorf("a fill question needs an answer")
		}
		out.Answer = answer
	default:
		out.Answer = answer
	}
	return out, nil
}

// contentKey 返回一道题内容的指纹：题干、题型、选项正文和答案连起来取 SHA-256，留前 16 个十六进制字符。
// 题干里连续的空白算一个。d 应当已经过 normalizeDraft。来源键用它去重。
func contentKey(d DraftQuestion) string {
	var b strings.Builder
	b.WriteString(strings.Join(strings.Fields(d.Stem), " "))
	b.WriteString("|")
	b.WriteString(d.Type)
	b.WriteString("|")
	for i, o := range d.Options {
		if i > 0 {
			b.WriteString("\x1f")
		}
		b.WriteString(o.Body)
	}
	b.WriteString("|")
	b.WriteString(d.Answer)
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:8])
}

// bankEntryOf 把校验过的题目变成进库用的条目。d 应当已经过 normalizeDraft。source 是来源，originKey 是去重用的来源键。
func bankEntryOf(d DraftQuestion, source, originKey string) bankEntry {
	return bankEntry{
		Stem: d.Stem, Type: d.Type, Options: d.Options, Answer: d.Answer, Explanation: d.Explanation,
		Difficulty: d.Difficulty, Tags: d.Tags, Source: source, OriginKey: originKey,
	}
}

// mergeTags 把 extra 接在 tags 后面，再按 cleanTags 去重和截断。
func mergeTags(tags, extra []string) []string {
	return cleanTags(append(append([]string(nil), tags...), extra...))
}
