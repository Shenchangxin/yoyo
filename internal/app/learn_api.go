// 学习模块的桌面接口入口：按方法名分给各个域，并拥有学习计划、手记和会话绑定。
// 掌握路径在 learn_api_mastery.go，文库和阅读在 learn_api_library.go，练习在 learn_api_practice.go，课本在 learn_api_books.go。
// 模型补全和嵌入在 learn_llm.go。

package app

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/trace"
)

// LearnCall 处理 learn. 前缀的方法。method 决定分支。params 里的库、路径和材料 id 由操作者传入。
// 模型不能从这里改会话绑定；绑定走 learn.bind。
func (a *App) LearnCall(method string, params map[string]any) (any, error) {
	if a == nil || a.Learn == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	for _, handle := range []func(string, map[string]any) (any, bool, error){
		a.learnCoreCall, a.learnMasteryCall, a.learnLibraryCall, a.learnPracticeCall, a.learnBooksCall,
	} {
		if out, ok, err := handle(method, params); ok {
			return out, err
		}
	}
	return nil, fmt.Errorf("unknown method %s", method)
}

// learnCoreCall 处理学习计划、手记和会话绑定。ok 为假表示这个方法不归它管。
func (a *App) learnCoreCall(method string, params map[string]any) (any, bool, error) {
	s := a.Learn
	switch method {
	case "learn.today":
		out, err := s.TodayView(time.Now())
		return out, true, err
	case "learn.journal.get":
		out, err := s.LoadJournal()
		return out, true, err
	case "learn.journal.set":
		return okMap(), true, s.SaveJournal(strParam(params, "topic"), strParam(params, "why"), strParam(params, "level"), strParam(params, "summary"), strParam(params, "next_focus"))
	case "learn.journal.add":
		return okMap(), true, s.AddJournal(strParam(params, "insight"))
	case "learn.bind":
		out, err := a.SetSessionLearn(strParam(params, "session"), LearnBinding{
			LibraryID: strParam(params, "library_id"), PathID: strParam(params, "path_id"),
			MaterialID: strParam(params, "material_id"), BookID: strParam(params, "book_id"),
		})
		return out, true, err
	}
	return nil, false, nil
}

// LearnBinding 是一段学习对话绑定的对象。四项都可空。
type LearnBinding struct {
	LibraryID  string
	PathID     string
	MaterialID string
	BookID     string
}

// SetSessionLearn 把库、路径、材料和课本绑到会话上。session 传给 GetSession。
// 各项去掉空白后写入。路径上已有文库而 LibraryID 为空时，用路径自己的文库。
func (a *App) SetSessionLearn(session string, bind LearnBinding) (SessionMeta, error) {
	m, err := a.GetSession(session)
	if err != nil {
		return m, err
	}
	m.LibraryID = strings.TrimSpace(bind.LibraryID)
	m.PathID = strings.TrimSpace(bind.PathID)
	m.MaterialID = strings.TrimSpace(bind.MaterialID)
	m.BookID = strings.TrimSpace(bind.BookID)
	if m.LibraryID == "" && m.PathID != "" && a.Learn != nil {
		if view, err := a.Learn.Path(m.PathID); err == nil {
			m.LibraryID = view.LibraryID
		}
	}
	// 课本要是以某座文库为依据，导师也要能查那座文库，好核对书里的说法。
	if m.LibraryID == "" && m.BookID != "" && a.Learn != nil {
		if view, err := a.Learn.Book(m.BookID); err == nil {
			m.LibraryID = view.Book.LibraryID
		}
	}
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

// publishLearn 把学习数据的一次变化发到 Hub。界面收到后重新读取对应的那一块。
// 载荷只有范围、对象 id 和变化种类，小到能内联进桌面通知。
func (a *App) publishLearn(c learn.Change) {
	if a == nil || a.Hub == nil {
		return
	}
	a.Hub.Publish(trace.Event{
		TS: time.Now().UTC(), Type: trace.TypeSystem, Source: "learn", ItemKind: "learn",
		Payload: map[string]any{"scope": c.Scope, "id": c.ID, "kind": c.Kind},
	})
}

// okMap 是只表示成功的返回值。
func okMap() map[string]any { return map[string]any{"ok": true} }

// strParam 从 m 里取字符串。键不存在或类型不对时返回空串。
func strParam(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// intParam 从 m 里取整数。JSON 数字是 float64，这里一并接受。其余返回 0。
func intParam(m map[string]any, key string) int {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	default:
		return 0
	}
}

// floatParam 从 m 里取小数。键不存在或类型不对时返回 0。
func floatParam(m map[string]any, key string) float64 {
	if m == nil {
		return 0
	}
	switch v := m[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	default:
		return 0
	}
}

// boolParam 从 m 里取布尔。键不存在或类型不对时返回 false。
func boolParam(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, _ := m[key].(bool)
	return v
}

// hasParam 在 m 里有 key 这一项时为真，值是什么都算。
func hasParam(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	_, ok := m[key]
	return ok
}

// strSlice 从 m 里取字符串数组，跳过不是字符串的项。
func strSlice(m map[string]any, key string) []string {
	raw, ok := m[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// decodeParam 把 m 里 key 这一项重新编码再解到 out。键不存在时不动 out。
func decodeParam(m map[string]any, key string, out any) error {
	if m == nil {
		return nil
	}
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	// 模型有时把数组和对象写成 JSON 字符串。
	if s, isString := v.(string); isString {
		if strings.TrimSpace(s) == "" {
			return nil
		}
		return json.Unmarshal([]byte(s), out)
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
