// 掌握路径的桌面接口：建路径、大纲、档案、作答、声明和知识点详情。
// 判分要用模型时带上界面给的会话 id，这样用的是那段导师对话的模型。规则都在 internal/learn。

package app

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/learn"
)

// learnMasteryCall 处理 learn.path 开头的方法。ok 为假表示这个方法不归它管。
func (a *App) learnMasteryCall(method string, params map[string]any) (any, bool, error) {
	s := a.Learn
	id := strParam(params, "id")
	switch method {
	case "learn.paths":
		return s.Paths(), true, nil
	case "learn.path":
		out, err := s.Path(id)
		return out, true, err
	case "learn.path.create":
		req := learn.CreatePathRequest{Name: strParam(params, "name"), Goal: strParam(params, "goal"), LibraryID: strParam(params, "library_id")}
		_ = decodeParam(params, "sources", &req.Sources)
		out, err := s.CreatePath(req)
		return out, true, err
	case "learn.path.rename":
		return okMap(), true, s.RenamePath(id, strParam(params, "name"))
	case "learn.path.describe":
		return okMap(), true, s.SetPathDescription(id, strParam(params, "description"))
	case "learn.path.delete":
		return okMap(), true, s.DeletePath(id)
	case "learn.path.reset":
		return okMap(), true, s.ResetPath(id)
	case "learn.path.outline.get":
		out, err := s.Outline(id)
		return out, true, err
	case "learn.path.outline":
		var modules []learn.OutlineModule
		if err := decodeParam(params, "modules", &modules); err != nil {
			return nil, true, err
		}
		change, err := s.ReplaceOutline(id, modules, learn.OutlineOptions{Append: boolParam(params, "append")})
		if err != nil {
			return nil, true, err
		}
		view, err := s.Path(id)
		return map[string]any{"change": change, "path": view}, true, err
	case "learn.path.draft":
		out, err := s.DraftOutline(context.Background(), learn.DraftRequest{PathID: id, SessionID: strParam(params, "session"), MustCover: strParam(params, "must_cover")})
		return out, true, err
	case "learn.path.mode":
		return okMap(), true, s.SetMode(id, strParam(params, "mode"))
	case "learn.path.profile":
		return okMap(), true, s.SetProfile(id, profilePatch(params))
	case "learn.path.retention":
		return okMap(), true, s.SetRetention(id, floatParam(params, "retention"))
	case "learn.path.claim":
		on := true
		if hasParam(params, "on") {
			on = boolParam(params, "on")
		}
		return okMap(), true, s.ClaimMastery(id, strParam(params, "point_id"), on)
	case "learn.path.defer":
		on := true
		if hasParam(params, "on") {
			on = boolParam(params, "on")
		}
		return okMap(), true, s.DeferPoint(id, strParam(params, "point_id"), on)
	case "learn.path.answer":
		out, err := s.Answer(context.Background(), learn.AnswerRequest{
			PathID: id, QuestionID: strParam(params, "question_id"), SessionID: strParam(params, "session"),
			Answer: strParam(params, "answer"), GiveUp: boolParam(params, "give_up"),
		})
		return out, true, err
	case "learn.path.question":
		out, err := s.QuestionState(id, strParam(params, "question_id"))
		return out, true, err
	case "learn.path.skip":
		return okMap(), true, s.SkipQuestion(id)
	case "learn.path.repair":
		out, err := s.RepairQuestion(context.Background(), learn.RepairRequest{
			PathID: id, QuestionID: strParam(params, "question_id"), SessionID: strParam(params, "session"),
			Action: strParam(params, "action"), Expected: strParam(params, "expected"),
		})
		return out, true, err
	case "learn.path.point":
		out, err := s.PointDetail(id, strParam(params, "point_id"))
		return out, true, err
	}
	return nil, false, nil
}

// profilePatch 从 params 里取档案的修改。没出现的键保持不动。
func profilePatch(params map[string]any) learn.ProfilePatch {
	var patch learn.ProfilePatch
	pick := func(key string) *string {
		if !hasParam(params, key) {
			return nil
		}
		v := strParam(params, key)
		return &v
	}
	patch.Teaching = pick("teaching")
	patch.PriorKnowledge = pick("prior_knowledge")
	patch.TargetLevel = pick("target_level")
	patch.TimeBudget = pick("time_budget")
	patch.Preferences = pick("preferences")
	return patch
}
