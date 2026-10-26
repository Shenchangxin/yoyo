// 本文件从用户行抽出跳转标题，并按目标用户序号取一页。转向注入不进目录。

package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// OutlineTurn 是目录里的一条操作者提示。
type OutlineTurn struct {
	// Seq 是该用户行在索引里的光标。
	Seq int64 `json:"seq"`
	// Title 是 OutlineTitle 生成的一行标题。
	Title string `json:"title"`
	// ID 来自载荷的 id，可空。
	ID string `json:"id,omitempty"`
}

// mentionTok 匹配 @file:、@folder:、@skill: 后的非空白，以及单独的 @harness。OutlineTitle 把它们换成空格。
var mentionTok = regexp.MustCompile(`@(?:file|folder|skill):\S+|@harness\b`)

// outlineTitleRunes 是标题最多保留的 rune 数，值是 42。超出后加省略号。
const outlineTitleRunes = 42

// OutlineTitle 去掉 mentionTok 匹配的提及，把空白收成单个空格。空结果返回空字符串。超过 42 个 rune 时截断并在末尾加 …。
func OutlineTitle(text string) string {
	s := mentionTok.ReplaceAllString(text, " ")
	s = strings.Join(strings.Fields(s), " ")
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) > outlineTitleRunes {
		return string(r[:outlineTitleRunes]) + "…"
	}
	return s
}

// IsSteerUser 报告这是不是转向注入的用户事件。
func IsSteerUser(ev Event) bool {
	return isSteerUser(ev)
}

// isSteerUser 在类型为 user，且 Source 为 steer，或载荷 name、kind 为 steer 时返回 true。
func isSteerUser(ev Event) bool {
	if ev.Type != TypeUser {
		return false
	}
	if ev.Source == "steer" {
		return true
	}
	name := payloadString(ev, "name")
	kind := payloadString(ev, "kind")
	return name == "steer" || kind == "steer"
}

// payloadString 按 keys 的顺序返回载荷里第一个非空白字符串。没有载荷或都空时返回空字符串。
func payloadString(ev Event, keys ...string) string {
	if ev.Payload == nil {
		return ""
	}
	for _, k := range keys {
		if s, ok := ev.Payload[k].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

// OutlineTurns 只读类型为 user 的索引行，跳过转向注入。文件不存在时返回空切片和 nil。读某一行失败或 JSON 无效就跳过该行。
func (s *Store) OutlineTurns(sessionID string) ([]OutlineTurn, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(sessionID)
	p := s.path(sessionID)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return []OutlineTurn{}, nil
		}
		return nil, err
	}
	defer f.Close()
	out := make([]OutlineTurn, 0, 16)
	buf := make([]byte, 0, 4096)
	for _, e := range idx.entries {
		if e.Type != TypeUser || e.N <= 0 {
			continue
		}
		if cap(buf) < e.N {
			buf = make([]byte, e.N)
		} else {
			buf = buf[:e.N]
		}
		n, err := f.ReadAt(buf, e.Off)
		if n == 0 {
			if err != nil {
				continue
			}
			continue
		}
		var ev Event
		if json.Unmarshal(bytes.TrimSpace(buf[:n]), &ev) != nil {
			continue
		}
		if isSteerUser(ev) {
			continue
		}
		out = append(out, OutlineTurn{
			Seq:   e.Seq,
			Title: OutlineTitle(payloadString(ev, "text")),
			ID:    payloadString(ev, "id"),
		})
	}
	return out, nil
}

// PageAroundUser 返回含 userSeq 的一页，宽度是 turns 个操作者回合。后面的回合够时，目标在窗口开头。靠近文件末尾时向前补，避免只剩一行。
// turns 小于等于 0 时用 DefaultPageTurns。userSeq 小于等于 0 时改调 PageTurns。找不到该序号时错误文案是 turn <序号> not in session。
func (s *Store) PageAroundUser(sessionID string, userSeq int64, turns int) (Page, error) {
	if turns <= 0 {
		turns = DefaultPageTurns
	}
	if userSeq <= 0 {
		return s.PageTurns(sessionID, 0, turns)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(sessionID)
	if len(idx.entries) == 0 {
		return Page{}, nil
	}
	users := make([]int, 0, 16)
	for i, e := range idx.entries {
		if e.operatorUser() {
			users = append(users, i)
		}
	}
	pos := -1
	for i, ui := range users {
		if idx.entries[ui].Seq == userSeq {
			pos = i
			break
		}
	}
	if pos < 0 {
		return Page{}, fmt.Errorf("turn %d not in session", userSeq)
	}
	start := pos
	end := start + turns
	if end > len(users) {
		end = len(users)
	}
	if end-start < turns {
		start = end - turns
		if start < 0 {
			start = 0
		}
	}
	startEntry := users[start]
	var endOff int64
	if end < len(users) {
		endOff = idx.entries[users[end]].Off
	} else {
		endOff = idx.size
	}
	off := idx.entries[startEntry].Off
	evs, err := s.readRangeLocked(sessionID, off, endOff)
	if err != nil {
		return Page{}, err
	}
	endEntry := users[end-1]
	last := idx.entries[endEntry]
	if end < len(users) {
		last = idx.entries[users[end]-1]
	} else {
		last = idx.entries[len(idx.entries)-1]
	}
	span := 0
	for i := startEntry; i < len(idx.entries); i++ {
		if idx.entries[i].Off+int64(idx.entries[i].N) > endOff {
			break
		}
		span++
	}
	if len(evs) == span {
		for i := range evs {
			evs[i].Seq = idx.entries[startEntry+i].Seq
		}
	}
	return Page{
		Events:  evs,
		HeadSeq: idx.entries[startEntry].Seq,
		TailSeq: last.Seq,
		Older:   startEntry > 0,
	}, nil
}
