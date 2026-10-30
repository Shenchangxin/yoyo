// 把操作者在榜、对谈或街采上改的字写回工程。
// 空着的字段保持原样。结构、镜头和时间不动。

package film

import "strings"

// Cast 是三种片子上操作者能直接改的字。
// Subject 是榜的题目、对谈的话题或街上的那一句。
// Rows 是排行榜从上到下的名次，或街采按顺序揭晓的答案。
// Left、Right、Ask、Reply 只用于对谈：两个人的名字，和一问一答。
type Cast struct {
	Subject string
	Rows    []string
	Left    string
	Right   string
	Ask     string
	Reply   string
}

// ApplyCast 按结构把 Cast 写进工程，并重算待办。
// 排行榜的名次留在条目上，口播说到哪一行，那一行才会亮。
// 对谈改两个人的名字和一来一回。街采的答案按镜头顺序揭晓。
// 认不出的结构原样返回。cast 的空字段跳过。
func ApplyCast(p Project, cast Cast) Project {
	switch p.Structure {
	case "podcast", "drama":
		return castPair(p, cast)
	case "interview", "narration", "explainer":
		return castAnswers(p, cast)
	case "talking":
		return castTalking(p, cast)
	case "ranking":
		return castBoard(p, cast)
	default:
		return p
	}
}

// castBoard 把题目写成标题，把名次写进条目。
// 第一句口播是第一名，后面的名次放进下一句，这样后出现的行仍能被点亮。
// p 是排行榜工程。cast 的 Subject 和 Rows 写入；Rows 里的空项跳过。
func castBoard(p Project, cast Cast) Project {
	p = takeSubject(p, cast.Subject)
	p = dropTemplateHost(p)
	p = writeEntries(p, cast.Rows)
	ids := lineIDs(p)
	rewritten := false
	if len(ids) > 0 && clean(rowAt(cast.Rows, 0)) != "" {
		p = ReplaceLine(p, ids[0], clean(rowAt(cast.Rows, 0)))
		rewritten = true
	}
	if len(ids) > 1 {
		if rest := joinRows(cast.Rows[1:]); rest != "" {
			p = ReplaceLine(p, ids[1], rest)
			rewritten = true
		}
	}
	if rewritten {
		return p
	}
	return rehash(p)
}

// castPair 让两个人保持对看：左边的人问，右边的人答。
// 话题写进标题、产品和中间那一句。名字改在原有说话人身上，不新造第三个人。
// p 是对谈工程。cast 的 Subject、Left、Right、Ask、Reply 写入；空的跳过。
func castPair(p Project, cast Cast) Project {
	p = takeSubject(p, cast.Subject)
	names := speakerNames(p)
	if left := clean(cast.Left); left != "" && len(names) > 0 {
		p = renameSpeaker(p, names[0], left)
		p.Brief.Host = left
	}
	names = speakerNames(p)
	if right := clean(cast.Right); right != "" && len(names) > 1 {
		p = renameSpeaker(p, names[1], right)
	}
	ids := lineIDs(p)
	rewritten := false
	if len(ids) > 0 && clean(cast.Ask) != "" {
		p = ReplaceLine(p, ids[0], clean(cast.Ask))
		rewritten = true
	}
	if len(ids) > 2 && clean(cast.Subject) != "" {
		p = ReplaceLine(p, ids[1], clean(cast.Subject))
		rewritten = true
	}
	if n := len(ids); n > 1 && clean(cast.Reply) != "" {
		p = ReplaceLine(p, ids[n-1], clean(cast.Reply))
		rewritten = true
	}
	if rewritten {
		return p
	}
	return rehash(p)
}

// castTalking 让同一个人留着，三句话按顺序换上。
// p 是口播工程。cast 的 Subject、Left 和 Rows 写入；空的跳过。
func castTalking(p Project, cast Cast) Project {
	p = takeSubject(p, cast.Subject)
	p = dropTemplateHost(p)
	names := speakerNames(p)
	if left := clean(cast.Left); left != "" && len(names) > 0 {
		p = renameSpeaker(p, names[0], left)
		p.Brief.Host = left
	}
	return castAnswers(p, cast)
}

// castAnswers 把街上的那一句写成标题，把三个答案按顺序写成条目和口播。
// 先出现的答案留在条上，后面的答案要等自己的镜头才露面。
// p 是街采工程。cast 的 Subject 和 Rows 写入；Rows 里的空项跳过。
func castAnswers(p Project, cast Cast) Project {
	p = takeSubject(p, cast.Subject)
	p = dropTemplateHost(p)
	p = writeEntries(p, cast.Rows)
	ids := lineIDs(p)
	rewritten := false
	for i, id := range ids {
		text := clean(rowAt(cast.Rows, i))
		if text == "" {
			continue
		}
		p = ReplaceLine(p, id, text)
		rewritten = true
	}
	if rewritten {
		return p
	}
	return rehash(p)
}

// takeSubject 把题目写进标题、产品和开头提示。空题目不动。
// p 接收标题。subject 经 clean 后为空则原样返回。
func takeSubject(p Project, subject string) Project {
	subject = clean(subject)
	if subject == "" {
		return p
	}
	p.Title = subject
	p.Brief.Product = subject
	p.Brief.Hook = subject
	return p
}

// dropTemplateHost 清掉模板里的占位宿主，避免画面提示还写着工具自己的名字。
// 操作者后来填上的宿主不动。p 的 Brief.Host 只有等于模板占位时才清空。
func dropTemplateHost(p Project) Project {
	switch p.Brief.Host {
	case "剪辑台", "路人":
		p.Brief.Host = ""
	}
	return p
}

// writeEntries 按位置改条目文字。空项跳过，不把后面的名次往前挪。
// 条目不够时补上。p 的 Entries 按 labels 的下标改 Label。
func writeEntries(p Project, labels []string) Project {
	for i := range labels {
		label := clean(labels[i])
		if label == "" {
			continue
		}
		for len(p.Entries) <= i {
			p.Entries = append(p.Entries, Entry{ID: "e" + itoa(len(p.Entries))})
		}
		p.Entries[i].Label = label
	}
	return p
}

// renameSpeaker 把一个说话人的名字换成新的。新旧相同或有一边空着就不动。
// p 的 Words 里 Speaker 等于 from 的改成 to。
func renameSpeaker(p Project, from, to string) Project {
	from, to = clean(from), clean(to)
	if from == "" || to == "" || from == to {
		return p
	}
	for i := range p.Words {
		if p.Words[i].Speaker == from {
			p.Words[i].Speaker = to
		}
	}
	return p
}

// lineIDs 按时间顺序列出每一行的编号。同一行只出现一次。
// p 的 Words 按当前顺序读取 LineID。
func lineIDs(p Project) []string {
	var ids []string
	seen := map[string]bool{}
	for _, w := range p.Words {
		if w.LineID == "" || seen[w.LineID] {
			continue
		}
		seen[w.LineID] = true
		ids = append(ids, w.LineID)
	}
	return ids
}

// speakerNames 按第一次开口的顺序列出说话人。
// p 的 Words 按当前顺序读取 Speaker。
func speakerNames(p Project) []string {
	var names []string
	seen := map[string]bool{}
	for _, w := range p.Words {
		if w.Speaker == "" || seen[w.Speaker] {
			continue
		}
		seen[w.Speaker] = true
		names = append(names, w.Speaker)
	}
	return names
}

// rowAt 读某一行。下标越界时返回空串。
// rows 为空或 i 越界时返回空串。
func rowAt(rows []string, i int) string {
	if i < 0 || i >= len(rows) {
		return ""
	}
	return rows[i]
}

// joinRows 把非空的行用空格接成一句。
// rows 里 clean 后为空的跳过。
func joinRows(rows []string) string {
	var b strings.Builder
	for _, row := range rows {
		row = clean(row)
		if row == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(row)
	}
	return b.String()
}
