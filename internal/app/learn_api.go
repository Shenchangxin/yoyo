// 学习模块的桌面接口和嵌入请求。嵌入花费记在学习库，不记进这一轮对话的用量。

package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/learn"
)

// LearnCall 处理 learn. 前缀的方法。method 决定分支。params 里的库、路径和材料 id 由操作者传入。
// 模型不能从这里改会话绑定；绑定走 learn.bind。
func (a *App) LearnCall(method string, params map[string]any) (any, error) {
	if a == nil || a.Learn == nil {
		return nil, fmt.Errorf("learn store unavailable")
	}
	s := a.Learn
	switch method {
	case "learn.today":
		return s.TodayView(time.Now())
	case "learn.libraries":
		return s.Libraries(), nil
	case "learn.library.create":
		return s.CreateLibrary(strParam(params, "name"), strParam(params, "kind"), strParam(params, "path"))
	case "learn.library.delete":
		return map[string]any{"ok": true}, s.DeleteLibrary(strParam(params, "id"))
	case "learn.library.embed":
		return map[string]any{"ok": true}, s.SetEmbedModel(strParam(params, "id"), strParam(params, "model"))
	case "learn.documents":
		return s.Documents(strParam(params, "library_id"))
	case "learn.import":
		n, err := s.Import(strParam(params, "library_id"), strParam(params, "path"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"imported": n}, nil
	case "learn.rebuild":
		return map[string]any{"ok": true}, s.StartRebuild(strParam(params, "library_id"))
	case "learn.document.remove":
		return map[string]any{"ok": true}, s.RemoveDocument(strParam(params, "library_id"), strParam(params, "document_id"))
	case "learn.search":
		hits, msg, err := s.SearchLibrary(strParam(params, "library_id"), strParam(params, "query"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"hits": hits, "message": msg}, nil
	case "learn.read":
		text, status, err := s.ReadPage(strParam(params, "library_id"), strParam(params, "document_id"), strParam(params, "source_hash"), intParam(params, "page"))
		if err != nil {
			return nil, err
		}
		page := intParam(params, "page")
		if page < 1 {
			page = 1
		}
		pages := 1
		if status == "" {
			if n := s.DocumentPages(strParam(params, "library_id"), strParam(params, "document_id")); n > 0 {
				pages = n
			}
		}
		return map[string]any{"text": text, "status": status, "pages": pages, "page": page}, nil
	case "learn.paths":
		return s.Paths(), nil
	case "learn.path.create":
		return s.CreatePath(strParam(params, "name"), strParam(params, "library_id"))
	case "learn.path":
		return s.Path(strParam(params, "id"))
	case "learn.path.outline":
		var mods []learn.OutlineModule
		raw, _ := json.Marshal(params["modules"])
		_ = json.Unmarshal(raw, &mods)
		return map[string]any{"ok": true}, s.ReplaceOutline(strParam(params, "id"), mods)
	case "learn.path.mode":
		return map[string]any{"ok": true}, s.SetMode(strParam(params, "id"), strParam(params, "mode"))
	case "learn.path.profile":
		return map[string]any{"ok": true}, s.SetProfile(strParam(params, "id"), strParam(params, "teaching"), strParam(params, "prior"), strParam(params, "level"), strParam(params, "budget"))
	case "learn.path.claim":
		return map[string]any{"ok": true}, s.ClaimMastery(strParam(params, "id"), strParam(params, "point_id"), strParam(params, "note"))
	case "learn.path.grade":
		res, err := s.GradeQuiz(strParam(params, "id"), strParam(params, "answer"))
		if err != nil {
			return nil, err
		}
		res["reveal"] = map[string]string{"answer": s.RevealGraded(strParam(params, "id"))}
		return res, nil
	case "learn.practice.due":
		return s.DueCards(time.Now()), nil
	case "learn.practice.summary":
		return s.Desk(time.Now()), nil
	case "learn.practice.grade":
		ok, err := s.GradeCard(strParam(params, "id"), strParam(params, "answer"), time.Now())
		if err != nil {
			return nil, err
		}
		return map[string]any{"correct": ok, "reveal": s.RevealCard(strParam(params, "id"))}, nil
	case "learn.practice.rate":
		if err := s.RateCard(strParam(params, "id"), strParam(params, "rating"), time.Now()); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true, "reveal": s.RevealCard(strParam(params, "id"))}, nil
	case "learn.questions":
		return s.Questions(), nil
	case "learn.questions.import":
		id, err := s.ImportQuestion(strParam(params, "stem"), strParam(params, "qtype"), strParam(params, "answer"), strParam(params, "explanation"), strSlice(params, "options"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"id": id}, nil
	case "learn.journal.get":
		return s.LoadJournal()
	case "learn.journal.set":
		return map[string]any{"ok": true}, s.SaveJournal(strParam(params, "topic"), strParam(params, "why"), strParam(params, "level"), strParam(params, "summary"), strParam(params, "next_focus"))
	case "learn.journal.add":
		return map[string]any{"ok": true}, s.AddJournal(strParam(params, "insight"))
	case "learn.books":
		return s.Books(), nil
	case "learn.book.create":
		return s.CreateBook(strParam(params, "title"), strParam(params, "library_id"))
	case "learn.book.proposal":
		return map[string]any{"ok": true}, s.SetProposal(strParam(params, "id"), strParam(params, "proposal"))
	case "learn.book.spine":
		var spine []learn.SpineItem
		raw, _ := json.Marshal(params["spine"])
		_ = json.Unmarshal(raw, &spine)
		return map[string]any{"ok": true}, s.SetSpine(strParam(params, "id"), spine)
	case "learn.book.compile":
		return s.CompileBook(strParam(params, "id"))
	case "learn.book":
		return s.Book(strParam(params, "id"), strParam(params, "reader"))
	case "learn.book.page":
		return map[string]any{"ok": true}, s.SetBookPage(strParam(params, "id"), strParam(params, "reader"), intParam(params, "page"))
	case "learn.bind":
		return a.SetSessionLearn(strParam(params, "session"), strParam(params, "library_id"), strParam(params, "path_id"), strParam(params, "material_id"))
	case "learn.materials":
		return s.Materials(), nil
	case "learn.material.open":
		return s.OpenMaterial(strParam(params, "library_id"), strParam(params, "document_id"))
	case "learn.annotate":
		return s.Annotate(strParam(params, "material_id"), strParam(params, "quote"), strParam(params, "note"), intParam(params, "page"))
	case "learn.annotations":
		return s.Annotations(strParam(params, "material_id")), nil
	case "learn.locate":
		hits, msg, err := s.Locate(strParam(params, "material_id"), strParam(params, "query"))
		if err != nil {
			return nil, err
		}
		return map[string]any{"hits": hits, "message": msg}, nil
	default:
		return nil, fmt.Errorf("unknown method %s", method)
	}
}

// SetSessionLearn 把库、路径和材料绑到会话上。session 传给 GetSession。
// libraryID、pathID、materialID 去掉空白后写入。路径上已有文库而 libraryID 为空时，用路径自己的文库。
func (a *App) SetSessionLearn(session, libraryID, pathID, materialID string) (SessionMeta, error) {
	m, err := a.GetSession(session)
	if err != nil {
		return m, err
	}
	m.LibraryID = strings.TrimSpace(libraryID)
	m.PathID = strings.TrimSpace(pathID)
	m.MaterialID = strings.TrimSpace(materialID)
	if m.LibraryID == "" && m.PathID != "" && a.Learn != nil {
		if view, err := a.Learn.Path(m.PathID); err == nil {
			m.LibraryID = view.Library
		}
	}
	if err := a.writeSession(m); err != nil {
		return m, err
	}
	return a.attachAuthMode(m), nil
}

// embedTexts 向配置的嵌入端点请求向量。model 为空时返回错误，重建仍可以只留词法索引。
// texts 按最多 16 条分批。花费不写入对话用量。
func (a *App) embedTexts(model string, texts []string) ([][]float32, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		return nil, fmt.Errorf("no embed model")
	}
	if a == nil || strings.TrimSpace(a.Config.BaseURL) == "" {
		return nil, fmt.Errorf("embeddings not configured")
	}
	key := ""
	if a.Vault != nil {
		key, _ = a.Vault.Lease("openai")
		if key == "" {
			key, _ = a.Vault.Get("api_key")
		}
	}
	if key == "" {
		return nil, fmt.Errorf("embeddings not configured")
	}
	out := make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += 16 {
		end := i + 16
		if end > len(texts) {
			end = len(texts)
		}
		part, err := a.embedBatch(model, key, texts[i:end])
		if err != nil {
			return nil, err
		}
		out = append(out, part...)
	}
	if a.Learn != nil {
		a.Learn.NoteEmbed()
	}
	return out, nil
}

func (a *App) embedBatch(model, key string, texts []string) ([][]float32, error) {
	body, _ := json.Marshal(map[string]any{"model": model, "input": texts})
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(a.Config.BaseURL, "/")+"/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("embeddings http %d", res.StatusCode)
	}
	var parsed struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Data) != len(texts) {
		return nil, fmt.Errorf("embeddings count")
	}
	out := make([][]float32, len(parsed.Data))
	for i, row := range parsed.Data {
		vec := make([]float32, len(row.Embedding))
		for j, x := range row.Embedding {
			vec[j] = float32(x)
		}
		out[i] = vec
	}
	return out, nil
}

func strParam(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	switch v := m[key].(type) {
	case string:
		return v
	default:
		return ""
	}
}

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
