// 学习工具挂在学习频道的聊天上。库、路径、材料和课本由会话绑定决定，模型参数里的同名 id 一律不用。
// 工具始终挂着，模式在调用的那一刻检查，这样模型在同一轮里切了模式就能接着用。
// 出题是唯一会收轮的工具：题目上了卡片，这一轮就交还给学习者。

package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/capability"
	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/session"
)

// learnOut 是一个学习工具的返回。Halt 为真时这批工具之后本轮结束。
type learnOut struct {
	Value any
	Halt  bool
}

// learnMount 把工具登记到一段学习对话上，并记下名字，好一次放进开场的工具清单。
type learnMount struct {
	a       *App
	tools   *runtime.WorkspaceTools
	session string
	names   []string
}

// add 登记一个工具。readOnly 为真时按读工作区检查权限并允许并发，否则按写工作区检查并独占。
// props 是参数的 JSON Schema 属性，required 是必填的参数名。
func (m *learnMount) add(name, desc string, readOnly bool, props map[string]any, required []string, fn func(args map[string]any) (learnOut, error)) {
	if props == nil {
		props = map[string]any{}
	}
	params := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		params["required"] = required
	}
	a, sessionID := m.a, m.session
	m.tools.Extra[name] = runtime.ExtraTool{
		ReadOnly:  readOnly,
		Exclusive: !readOnly,
		JSON:      runtime.ToolJSON{Type: "function", Function: map[string]any{"name": name, "description": desc, "parameters": params}},
		Call: func(argsJSON string) runtime.ToolResult {
			level := capability.WriteWorkspace
			if readOnly {
				level = capability.ReadWorkspace
			}
			if a.Caps != nil {
				if err := a.Caps.Check(capability.Request{Level: level, Action: name, SessionID: sessionID}); err != nil {
					return runtime.ToolResult{Err: err}
				}
			}
			var args map[string]any
			_ = json.Unmarshal([]byte(argsJSON), &args)
			if args == nil {
				args = map[string]any{}
			}
			// 绑定只认会话上的。模型自己写的 id 丢掉。
			for _, key := range []string{"library_id", "path_id", "material_id", "book_id"} {
				delete(args, key)
			}
			out, err := fn(args)
			if err != nil {
				return runtime.ToolResult{Err: err}
			}
			raw, err := json.Marshal(out.Value)
			if err != nil {
				return runtime.ToolResult{Err: err}
			}
			return runtime.ToolResult{Content: string(raw), Halt: out.Halt}
		},
	}
	m.names = append(m.names, name)
}

// attachLearn 在学习频道的父会话上挂学习工具，并把它们放进开场的工具清单。
// tools 为 nil、没有学习库、不是聊天叠加层、深度不是 0，或会话不在学习频道时什么都不做。
func (a *App) attachLearn(tools *runtime.WorkspaceTools) {
	if a == nil || a.Learn == nil || tools == nil || !tools.ChatOverlay || tools.Depth != 0 {
		return
	}
	meta, err := a.GetSession(tools.SessionID)
	if err != nil || session.NormalizeChannel(meta.Channel) != session.ChannelLearn {
		return
	}
	if tools.Extra == nil {
		tools.Extra = map[string]runtime.ExtraTool{}
	}
	m := &learnMount{a: a, tools: tools, session: tools.SessionID}
	a.mountJournalTool(m)
	lib := meta.LibraryID
	if lib == "" && meta.PathID != "" {
		if view, err := a.Learn.Path(meta.PathID); err == nil {
			lib = view.LibraryID
		}
	}
	if lib != "" {
		a.mountLibraryTools(m, lib)
	}
	if meta.MaterialID != "" {
		a.mountMaterialTools(m, meta.MaterialID)
	}
	if meta.BookID != "" {
		a.mountBookTools(m, meta.BookID)
	}
	if meta.PathID != "" {
		a.mountPathTools(m, meta.PathID)
	}
	a.mountPracticeTools(m)
	// 聊天叠加层只把点了名的扩展工具放进开场清单，其余要先搜才看得见。学习工具必须一开场就在。
	tools.ExtraEnabled = uniqueStrings(tools.ExtraEnabled, m.names)
}

// mountJournalTool 挂上读写手记的工具。
func (a *App) mountJournalTool(m *learnMount) {
	m.add("learn_journal", "Read or update the learner's mission and the hand-off for the next session. action=get reads it. action=set rewrites mission fields; pass only what changed. insight appends one line the learner wants to keep.", false, map[string]any{
		"action":     map[string]any{"type": "string", "enum": []string{"get", "set"}},
		"topic":      map[string]any{"type": "string", "description": "What the learner is working toward."},
		"why":        map[string]any{"type": "string"},
		"level":      map[string]any{"type": "string"},
		"summary":    map[string]any{"type": "string", "description": "What this session covered."},
		"next_focus": map[string]any{"type": "string", "description": "Where to pick up next time."},
		"insight":    map[string]any{"type": "string"},
	}, nil, func(args map[string]any) (learnOut, error) {
		if strParam(args, "action") == "set" {
			cur, err := a.Learn.LoadJournal()
			if err != nil {
				return learnOut{}, err
			}
			keep := func(key, old string) string {
				if hasParam(args, key) {
					return strParam(args, key)
				}
				return old
			}
			if err := a.Learn.SaveJournal(keep("topic", cur.Topic), keep("why", cur.Why), keep("level", cur.Level), keep("summary", cur.Summary), keep("next_focus", cur.NextFocus)); err != nil {
				return learnOut{}, err
			}
		}
		if insight := strParam(args, "insight"); insight != "" {
			if err := a.Learn.AddJournal(insight); err != nil {
				return learnOut{}, err
			}
		}
		j, err := a.Learn.LoadJournal()
		if err != nil {
			return learnOut{}, err
		}
		j.Records = nil
		return learnOut{Value: j}, nil
	})
}

// mountPathTools 挂上掌握路径的工具。pathID 是会话绑定的路径。
func (a *App) mountPathTools(m *learnMount, pathID string) {
	s := a.Learn
	sessionID := m.session
	tools := m.tools
	// held 在占着路径租约的时候跑 fn。另一段对话正在改这条路径时返回 path busy。
	held := func(fn func() (learnOut, error)) (learnOut, error) {
		if err := s.HoldPath(pathID, sessionID); err != nil {
			return learnOut{}, err
		}
		defer s.ReleasePath(pathID, sessionID)
		return fn()
	}
	ctx := func() context.Context {
		if tools.Ctx != nil {
			return tools.Ctx
		}
		return context.Background()
	}

	m.add("learn_status", "Read the whole mastery map: modules, knowledge points with ids and status, the learner profile, due reviews, and the next step. Each learner message already carries the current next step, so call this only when you need ids or the full map.", true, nil, nil, func(map[string]any) (learnOut, error) {
		view, err := s.Path(pathID)
		if err != nil {
			return learnOut{}, err
		}
		return learnOut{Value: tutorMap(view)}, nil
	})

	m.add("learn_mode", "Switch this path among outline, study, and review. Tell the learner what you are switching to and why. The new mode applies to the next tool call in this same turn.", false, map[string]any{
		"mode":   map[string]any{"type": "string", "enum": []string{learn.ModeOutline, learn.ModeStudy, learn.ModeReview}},
		"reason": map[string]any{"type": "string"},
	}, []string{"mode"}, func(args map[string]any) (learnOut, error) {
		return held(func() (learnOut, error) {
			mode := strings.ToLower(strings.TrimSpace(strParam(args, "mode")))
			if err := s.SetMode(pathID, mode); err != nil {
				return learnOut{}, err
			}
			view, err := s.Path(pathID)
			if err != nil {
				return learnOut{}, err
			}
			return learnOut{Value: map[string]any{"mode": view.Mode, "instructions": learnModeNote(view.Mode), "next": view.Next}}, nil
		})
	})

	m.add("learn_profile", "Record what you learned about this learner on this path. Pass only the fields that changed. teaching is probe_first (question first) or teach_first (explain first). This writes; learn_status reads.", false, map[string]any{
		"prior_knowledge": map[string]any{"type": "string", "description": "What they already know."},
		"target_level":    map[string]any{"type": "string", "description": "How far they want to go; what counts as done."},
		"time_budget":     map[string]any{"type": "string"},
		"preferences":     map[string]any{"type": "string", "description": "How they like to be taught."},
		"teaching":        map[string]any{"type": "string", "enum": []string{learn.TeachProbe, learn.TeachFirst}},
	}, nil, func(args map[string]any) (learnOut, error) {
		patch := profilePatch(args)
		if patch.Teaching == nil && patch.PriorKnowledge == nil && patch.TargetLevel == nil && patch.TimeBudget == nil && patch.Preferences == nil {
			return learnOut{}, fmt.Errorf("learn_profile needs at least one field to record")
		}
		if err := s.SetProfile(pathID, patch); err != nil {
			return learnOut{}, err
		}
		view, err := s.Path(pathID)
		if err != nil {
			return learnOut{}, err
		}
		return learnOut{Value: map[string]any{"ok": true, "profile": view.Profile}}, nil
	})

	outlinePoint := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name":          map[string]any{"type": "string"},
			"type":          map[string]any{"type": "string", "enum": []string{learn.KindMemory, learn.KindProcedure, learn.KindConcept, learn.KindDesign}},
			"ref":           map[string]any{"type": "string", "description": "A short handle other points can list under prerequisites."},
			"prerequisites": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Names, refs, or ids of points this one truly depends on."},
		},
		"required": []string{"name", "type"},
	}
	m.add("learn_build", "Create or extend the outline. outline mode only. Each module has a name, a one-sentence objective, and knowledge_points. type: memory = facts to recall, procedure = a method carried out step by step, concept = an idea to explain, design = judgment and trade-offs. mode=replace rewrites the outline but keeps progress on points whose name and type are unchanged; mode=append adds to it.", false, map[string]any{
		"modules": map[string]any{"type": "array", "items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":             map[string]any{"type": "string"},
				"objective":        map[string]any{"type": "string"},
				"knowledge_points": map[string]any{"type": "array", "items": outlinePoint},
			},
			"required": []string{"name", "knowledge_points"},
		}},
		"mode": map[string]any{"type": "string", "enum": []string{"replace", "append"}},
	}, []string{"modules"}, func(args map[string]any) (learnOut, error) {
		if err := a.needMode(pathID, "learn_build", learn.ModeOutline); err != nil {
			return learnOut{}, err
		}
		modules := decodeOutline(args["modules"])
		points := 0
		for _, mod := range modules {
			points += len(mod.Points)
		}
		if points == 0 {
			return learnOut{}, fmt.Errorf(`learn_build could not read any knowledge point. Send modules as [{"name":"<module>","objective":"<one sentence>","knowledge_points":[{"name":"<point>","type":"memory|procedure|concept|design"}]}]`)
		}
		return held(func() (learnOut, error) {
			change, err := s.ReplaceOutline(pathID, modules, learn.OutlineOptions{Append: strParam(args, "mode") == "append", KeepMastered: true})
			if err != nil {
				return learnOut{}, err
			}
			view, err := s.Path(pathID)
			if err != nil {
				return learnOut{}, err
			}
			return learnOut{Value: map[string]any{
				"status": "built", "change": change, "map": tutorMap(view)["map"],
				"instruction": "Walk the learner through this outline in a few lines and ask what to change. When they agree, call learn_mode with mode=study.",
			}}, nil
		})
	})

	m.add("learn_revise", "Edit the outline in place without touching the rest. outline mode only. Mastered points cannot be removed. order must list every module id once.", false, map[string]any{
		"add": map[string]any{"type": "array", "items": map[string]any{
			"type": "object",
			"properties": map[string]any{
				"module_id":     map[string]any{"type": "string"},
				"module":        map[string]any{"type": "string", "description": "Module name; a new module is created when no module has this name."},
				"objective":     map[string]any{"type": "string"},
				"name":          map[string]any{"type": "string"},
				"type":          map[string]any{"type": "string"},
				"prerequisites": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			},
			"required": []string{"name", "type"},
		}},
		"rename": map[string]any{"type": "array", "items": map[string]any{
			"type":       "object",
			"properties": map[string]any{"id": map[string]any{"type": "string"}, "name": map[string]any{"type": "string"}, "objective": map[string]any{"type": "string"}},
			"required":   []string{"id"},
		}},
		"remove": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Knowledge point ids."},
		"order":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Module ids in the new order."},
	}, nil, func(args map[string]any) (learnOut, error) {
		if err := a.needMode(pathID, "learn_revise", learn.ModeOutline); err != nil {
			return learnOut{}, err
		}
		edit := decodeOutlineEdit(args)
		if len(edit.Add)+len(edit.Rename)+len(edit.Remove)+len(edit.Order) == 0 {
			return learnOut{}, fmt.Errorf("learn_revise needs at least one of add, rename, remove, or order")
		}
		return held(func() (learnOut, error) {
			change, err := s.ReviseOutline(pathID, edit, true)
			if err != nil {
				return learnOut{}, err
			}
			view, err := s.Path(pathID)
			if err != nil {
				return learnOut{}, err
			}
			return learnOut{Value: map[string]any{"status": "revised", "change": change, "map": tutorMap(view)["map"]}}, nil
		})
	})

	m.add("learn_quiz", "Pose one graded question on the learner's answer card and END THIS TURN. Write your lead-in in the same turn and call this last. The expected answer and explanation stay hidden until the answer is graded. study or review mode only. For choice questions pass options without letters in the question text; options are shuffled before display.", false, map[string]any{
		"point_id":         map[string]any{"type": "string", "description": "Knowledge point id from learning_state or learn_status."},
		"question":         map[string]any{"type": "string", "description": "The stem only. Do not list the options here."},
		"expected_answer":  map[string]any{"type": "string", "description": "For choice: the label of the correct option as you passed it, or its exact body. Otherwise the answer a grader should accept."},
		"question_type":    map[string]any{"type": "string", "enum": []string{learn.QuestionChoice, learn.QuestionShort, learn.QuestionOpen}},
		"options":          map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"label": map[string]any{"type": "string"}, "body": map[string]any{"type": "string"}}, "required": []string{"body"}}},
		"accepted_answers": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Other wordings that are also right."},
		"explanation":      map[string]any{"type": "string", "description": "Shown after grading. Explain why, without option letters."},
		"difficulty":       map[string]any{"type": "string", "enum": []string{"easy", "medium", "hard"}},
		"hinted":           map[string]any{"type": "boolean", "description": "True when you already gave a hint for this exact question. A hinted answer does not count toward the gate."},
	}, []string{"point_id", "question", "expected_answer"}, func(args map[string]any) (learnOut, error) {
		if err := a.needMode(pathID, "learn_quiz", learn.ModeStudy, learn.ModeReview); err != nil {
			return learnOut{}, err
		}
		pointID := strParam(args, "point_id")
		if err := a.reviewScope(pathID, pointID); err != nil {
			return learnOut{}, err
		}
		var options []learn.Option
		_ = decodeParam(args, "options", &options)
		prompt := strParam(args, "question")
		if prompt == "" {
			prompt = strParam(args, "prompt")
		}
		return held(func() (learnOut, error) {
			q, already, err := s.PoseQuiz(learn.PoseRequest{
				PathID: pathID, PointID: pointID, SessionID: sessionID, Type: strParam(args, "question_type"),
				Prompt: prompt, Expected: strParam(args, "expected_answer"), Options: options,
				Accepted: strSlice(args, "accepted_answers"), Explanation: strParam(args, "explanation"),
				Difficulty: strParam(args, "difficulty"), Hinted: boolParam(args, "hinted"),
			})
			if err != nil {
				return learnOut{}, err
			}
			// path_id 是给界面上的题卡用的：卡片靠它去读题目状态和提交作答。
			out := map[string]any{"status": "posed", "path_id": pathID, "question": q,
				"notice": "The question is on the learner's answer card and this turn ends here. Their answer arrives with the next message; when they use the card the engine grades it first."}
			if already {
				out["status"] = "already_pending"
				out["notice"] = "No new question was posed: the learner already has an open question on their card, shown here. Your parameters were ignored. This turn ends here. Use learn_skip first if that question should be dropped."
			}
			return learnOut{Value: out, Halt: true}, nil
		})
	})

	m.add("learn_grade", "Grade the pending question from what the learner just wrote in the chat. Use it only when they answered in the message instead of on the card; a card answer is already graded and appears in learning_state. study or review mode only.", false, map[string]any{
		"answer": map[string]any{"type": "string", "description": "The part of the learner's message that is their answer, copied exactly."},
	}, nil, func(args map[string]any) (learnOut, error) {
		if err := a.needMode(pathID, "learn_grade", learn.ModeStudy, learn.ModeReview); err != nil {
			return learnOut{}, err
		}
		answer := learnerWords(strParam(args, "answer"), tools.OperatorVoice)
		return held(func() (learnOut, error) {
			res, err := s.Answer(ctx(), learn.AnswerRequest{PathID: pathID, SessionID: sessionID, Answer: answer})
			if err != nil {
				return learnOut{}, err
			}
			s.MarkTold(pathID)
			out := map[string]any{
				"result": res.Result, "final": res.Final, "feedback": res.Feedback, "mastery": res.Mastery,
				"mastered": res.Mastered, "gate": res.Gate, "status": res.Status, "next": res.Next,
			}
			switch {
			case !res.Final:
				out["instruction"] = "This was NOT graded and the question is still open. Do not treat it as a wrong answer. Ask the learner to answer more clearly, or help with what they asked."
			case res.Gate == "qualitative" && !res.Mastered:
				out["instruction"] = "The card now shows the verdict. Questions do not open this point's gate: when the learner is ready, ask them to explain the idea in their own words and call learn_assess. Say something to the learner before the turn ends."
			default:
				out["instruction"] = "The card now shows the verdict and the explanation. React briefly, then continue with next. Never end the turn without saying anything."
			}
			return learnOut{Value: out}, nil
		})
	})

	m.add("learn_assess", "Check a concept or design point from the learner's own explanation in their latest message. The engine reads their message and decides; you cannot pass or fail it yourself. Ask them to explain first, and call this in the turn right after they do. study or review mode only.", false, map[string]any{
		"point_id": map[string]any{"type": "string"},
		"note":     map[string]any{"type": "string", "description": "Optional context for the grader, such as what the explanation should cover."},
	}, []string{"point_id"}, func(args map[string]any) (learnOut, error) {
		if err := a.needMode(pathID, "learn_assess", learn.ModeStudy, learn.ModeReview); err != nil {
			return learnOut{}, err
		}
		pointID := strParam(args, "point_id")
		if err := a.reviewScope(pathID, pointID); err != nil {
			return learnOut{}, err
		}
		return held(func() (learnOut, error) {
			res, err := s.Assess(ctx(), learn.AssessRequest{
				PathID: pathID, PointID: pointID, SessionID: sessionID, Explanation: tools.OperatorVoice, Note: strParam(args, "note"),
			})
			if err != nil {
				return learnOut{}, err
			}
			out := map[string]any{"result": res.Result, "final": res.Final, "feedback": res.Feedback, "gaps": res.Gaps, "mastered": res.Mastered, "status": res.Status, "next": res.Next}
			switch {
			case !res.Final:
				out["instruction"] = "Nothing was recorded: the explanation could not be judged with confidence. Ask the learner to say more, in their own words."
			case res.Mastered:
				out["instruction"] = "The gate is open. Tell the learner, then continue with next in this same reply."
			default:
				out["instruction"] = "The gate is not open yet. Address the gaps, then ask the learner to explain it again in their own words."
			}
			return learnOut{Value: out}, nil
		})
	})

	m.add("learn_skip", "Drop the pending question without grading it. No credit, no penalty. Use it when the learner wants a different question.", false, nil, nil, func(map[string]any) (learnOut, error) {
		return held(func() (learnOut, error) {
			if err := s.SkipQuestion(pathID); err != nil {
				return learnOut{}, err
			}
			return learnOut{Value: map[string]any{"ok": true, "instruction": "The question was dropped with no credit. Continue, or pose a different question with learn_quiz."}}, nil
		})
	})

	m.add("learn_repair", "Fix a question that was wrong. action=void drops the attempt so it no longer counts. action=correct sets the true expected_answer and re-grades what the learner answered. Defaults to the most recent question.", false, map[string]any{
		"action":          map[string]any{"type": "string", "enum": []string{"void", "correct"}},
		"question_id":     map[string]any{"type": "string"},
		"expected_answer": map[string]any{"type": "string"},
		"reason":          map[string]any{"type": "string"},
	}, []string{"action"}, func(args map[string]any) (learnOut, error) {
		return held(func() (learnOut, error) {
			res, err := s.RepairQuestion(ctx(), learn.RepairRequest{
				PathID: pathID, QuestionID: strParam(args, "question_id"), SessionID: sessionID,
				Action: strParam(args, "action"), Expected: strParam(args, "expected_answer"),
			})
			if err != nil {
				return learnOut{}, err
			}
			s.MarkTold(pathID)
			return learnOut{Value: map[string]any{"result": res.Result, "mastery": res.Mastery, "mastered": res.Mastered, "status": res.Status, "next": res.Next}}, nil
		})
	})

	m.add("learn_defer", "Postpone a knowledge point, or bring it back with on=false. A deferred point is not mastered; the path just moves on to other points first. Defaults to the current next point.", false, map[string]any{
		"point_id": map[string]any{"type": "string"},
		"on":       map[string]any{"type": "boolean"},
	}, nil, func(args map[string]any) (learnOut, error) {
		return held(func() (learnOut, error) {
			pointID := strParam(args, "point_id")
			if pointID == "" {
				view, err := s.Path(pathID)
				if err != nil {
					return learnOut{}, err
				}
				if view.Next != nil {
					pointID = view.Next.PointID
				}
			}
			on := true
			if hasParam(args, "on") {
				on = boolParam(args, "on")
			}
			if err := s.DeferPoint(pathID, pointID, on); err != nil {
				return learnOut{}, err
			}
			view, err := s.Path(pathID)
			if err != nil {
				return learnOut{}, err
			}
			return learnOut{Value: map[string]any{"ok": true, "next": view.Next, "instruction": "The point is deferred, not mastered. Continue with next."}}, nil
		})
	})
}

// needMode 在路径当前模式不在 allowed 里时返回一条告诉模型该怎么切的错误。tool 是被调用的工具名。
func (a *App) needMode(pathID, tool string, allowed ...string) error {
	mode, err := a.Learn.Mode(pathID)
	if err != nil {
		return err
	}
	for _, want := range allowed {
		if mode == want {
			return nil
		}
	}
	return fmt.Errorf("%s belongs to %s, and this path is in %q. Call learn_mode with mode=%q first, and tell the learner what you are switching to and why", tool, strings.Join(allowed, " or "), mode, allowed[0])
}

// reviewScope 在复习模式下拦住还没掌握的知识点。复习只回头检验已经过门的内容。
func (a *App) reviewScope(pathID, pointID string) error {
	view, err := a.Learn.Path(pathID)
	if err != nil || view.Mode != learn.ModeReview {
		return err
	}
	for _, mod := range view.Modules {
		for _, p := range mod.Points {
			if p.ID == pointID && p.Status != learn.StatusMastered {
				return fmt.Errorf("%q is not mastered yet, and this path is in review. Review re-tests what the learner has already proven. Pick a mastered point, or call learn_mode with mode=%q", p.Name, learn.ModeStudy)
			}
		}
	}
	return nil
}

// tutorMap 把路径视图收成给模型看的地图：比界面用的少几项，但带齐 id、状态和下一步。
func tutorMap(view learn.PathView) map[string]any {
	modules := make([]map[string]any, 0, len(view.Modules))
	for _, mod := range view.Modules {
		points := make([]map[string]any, 0, len(mod.Points))
		for _, p := range mod.Points {
			row := map[string]any{"id": p.ID, "name": p.Name, "type": p.Kind, "status": p.Status, "mastery": p.Mastery, "gate": p.Gate}
			if p.Source == "learner" {
				row["claimed_by_learner"] = true
			}
			if p.Blocked {
				row["blocked_by_prerequisite"] = true
			}
			if p.Deferred {
				row["deferred"] = true
			}
			if len(p.Prereqs) > 0 {
				row["prerequisites"] = p.Prereqs
			}
			points = append(points, row)
		}
		modules = append(modules, map[string]any{"id": mod.ID, "name": mod.Name, "objective": mod.Objective, "knowledge_points": points})
	}
	reviews := view.Reviews
	if len(reviews) > 5 {
		reviews = reviews[:5]
	}
	sources := make([]string, 0, len(view.Sources))
	for _, src := range view.Sources {
		sources = append(sources, src.Label)
	}
	return map[string]any{
		"path":          map[string]any{"name": view.Name, "goal": view.Goal, "mode": view.Mode},
		"profile":       view.Profile,
		"intake_needed": view.IntakeNeeded,
		"counts":        view.Counts,
		"due_reviews":   view.Due,
		"complete":      view.Complete,
		"next":          view.Next,
		"reviews":       reviews,
		"sources":       sources,
		"map":           modules,
	}
}

// decodeOutline 把模型给的 modules 解成大纲。知识点可以写在 knowledge_points 或 points 下，类型可以写成 type 或 kind。
// raw 可以是数组，也可以是装着数组的 JSON 字符串。
func decodeOutline(raw any) []learn.OutlineModule {
	var in []struct {
		Name      string `json:"name"`
		Objective string `json:"objective"`
		Points    []struct {
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			Kind          string   `json:"kind"`
			Ref           string   `json:"ref"`
			Prerequisites []string `json:"prerequisites"`
			Prereqs       []string `json:"prereqs"`
		} `json:"knowledge_points"`
		Legacy []struct {
			Name          string   `json:"name"`
			Type          string   `json:"type"`
			Kind          string   `json:"kind"`
			Ref           string   `json:"ref"`
			Prerequisites []string `json:"prerequisites"`
			Prereqs       []string `json:"prereqs"`
		} `json:"points"`
	}
	_ = decodeParam(map[string]any{"v": raw}, "v", &in)
	out := make([]learn.OutlineModule, 0, len(in))
	for _, mod := range in {
		item := learn.OutlineModule{Name: mod.Name, Objective: mod.Objective}
		points := mod.Points
		if len(points) == 0 {
			points = mod.Legacy
		}
		for _, pt := range points {
			kind := pt.Type
			if kind == "" {
				kind = pt.Kind
			}
			prereqs := pt.Prerequisites
			if prereqs == nil {
				prereqs = pt.Prereqs
			}
			if prereqs == nil {
				prereqs = []string{}
			}
			item.Points = append(item.Points, learn.OutlinePoint{Name: pt.Name, Kind: kind, Ref: pt.Ref, Prereqs: prereqs})
		}
		out = append(out, item)
	}
	return out
}

// decodeOutlineEdit 把 learn_revise 的参数解成一次大纲修改。
func decodeOutlineEdit(args map[string]any) learn.OutlineEdit {
	var edit learn.OutlineEdit
	var adds []struct {
		ModuleID      string   `json:"module_id"`
		Module        string   `json:"module"`
		Objective     string   `json:"objective"`
		Name          string   `json:"name"`
		Type          string   `json:"type"`
		Prerequisites []string `json:"prerequisites"`
	}
	_ = decodeParam(args, "add", &adds)
	for _, add := range adds {
		edit.Add = append(edit.Add, learn.OutlineAdd{ModuleID: add.ModuleID, Module: add.Module, Objective: add.Objective, Name: add.Name, Kind: add.Type, Prereqs: add.Prerequisites})
	}
	_ = decodeParam(args, "rename", &edit.Rename)
	_ = decodeParam(args, "remove", &edit.Remove)
	_ = decodeParam(args, "order", &edit.Order)
	return edit
}

// learnerWords 决定拿什么当学习者的作答。picked 是模型从学习者消息里摘出来的那一段，voice 是学习者这一轮的原话。
// picked 确实出现在原话里才用它，否则用整句原话：模型不能替学习者编一个答案。没有原话时只能用 picked。
func learnerWords(picked, voice string) string {
	picked = strings.TrimSpace(picked)
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return picked
	}
	if picked != "" && strings.Contains(squash(voice), squash(picked)) {
		return picked
	}
	return voice
}

// squash 转成小写并去掉空白，比较两段文字是否一段包含另一段时用。
func squash(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if !unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// clipLearn 把 s 截到最多 n 个符文。
func clipLearn(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}
