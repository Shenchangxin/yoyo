// studio_task.go 让手动配音和合成在后台跑。点下去马上返回，工程上记着正在做什么；做完清掉记号、推一条 studio_state，界面据此刷新。
// 助手调用工具时同样记号，只是在当前调用里等它做完。同一条短片一次只跑一件，和一键成片也不撞车。

package video

import (
	"fmt"
	"strings"
)

// StudioTask 是工程上正在后台跑的一件事。Kind 是 speak 或 assemble。
type StudioTask struct {
	Kind      string `json:"kind"`
	StartedAt string `json:"started_at"`
}

// studioTaskLabel 是界面和提示里对这件事的叫法。
var studioTaskLabel = map[string]string{"speak": "配音", "assemble": "合成", "bgm": "配乐"}

// studioTaskGate 是这件事失败时把问题挂在哪一闸。
func studioTaskGate(kind string) string {
	if kind == "assemble" || kind == "bgm" {
		return GateCompose
	}
	return GateAudio
}

// StartStudioBGM 在后台用配乐连接生成一段配乐。没有配乐连接时当场拒绝。
// prompt 空着按短片的语气、风格和标题写。
func (e *Engine) StartStudioBGM(id, prompt string) (StudioProject, error) {
	if _, err := e.ActiveProvider("music", ""); err != nil {
		p, _ := e.GetStudioProject(id)
		return p, fmt.Errorf("还没有配乐连接。到 设置 → 生成 里启用配乐，或把 mp3 放进 %s", e.bgmDir())
	}
	return e.StartStudioTask(id, "bgm", func() error {
		_, err := e.GenerateStudioBGM(id, prompt)
		return err
	})
}

// claimStudioTask 在工程上记下正在做的事。已经有事在跑，或一键成片正在跑时拒绝。
func (e *Engine) claimStudioTask(id, kind string) (StudioProject, error) {
	return e.mutateStudio(id, func(p *StudioProject) error {
		if p.Task != nil && p.Task.Kind != "" {
			return fmt.Errorf("这条短片正在%s，等它做完再来", studioTaskLabel[p.Task.Kind])
		}
		if filmPhaseBusy(p.AutoPhase) {
			return fmt.Errorf("一键成片正在做这条短片，等它做完再来")
		}
		p.Task = &StudioTask{Kind: kind, StartedAt: Now()}
		return nil
	})
}

// releaseStudioTask 清掉记号。run 返回错误时把原因记成这一闸的问题，界面上能看到为什么没做成。
func (e *Engine) releaseStudioTask(id, kind string, runErr error) StudioProject {
	p, _ := e.mutateStudio(id, func(p *StudioProject) error {
		owned := p.Task != nil && p.Task.Kind == kind
		if !owned && runErr == nil {
			return errUnchanged
		}
		if owned {
			p.Task = nil
		}
		if runErr != nil {
			e.warnStudio(p, studioTaskGate(kind), studioTaskLabel[kind]+"没做成："+runErr.Error())
		}
		return nil
	})
	return p
}

// StartStudioTask 把一件耗时的事放到后台跑，立即返回记了号的工程。做完或失败都会清掉记号并推一条 studio_state。
// run 是真正干活的函数；它自己在锁里写结果，这里只管记号和失败原因。中途恐慌也收成失败，记号不会挂住。
func (e *Engine) StartStudioTask(id, kind string, run func() error) (StudioProject, error) {
	p, err := e.claimStudioTask(id, kind)
	if err != nil {
		return p, err
	}
	e.emitStudio(p)
	go func() {
		var runErr error
		func() {
			defer func() {
				if r := recover(); r != nil {
					runErr = fmt.Errorf("意外中断：%v", r)
				}
			}()
			runErr = run()
		}()
		e.emitStudio(e.releaseStudioTask(id, kind, runErr))
	}()
	return p, nil
}

// runStudioTaskSync 在当前调用里做完这件事，但同样在工程上记号，界面能看到正在做，也不会和手动操作撞车。助手的工具走这里。
func (e *Engine) runStudioTaskSync(id, kind string, run func() (StudioProject, error)) (StudioProject, error) {
	if _, err := e.claimStudioTask(id, kind); err != nil {
		return StudioProject{}, err
	}
	out, err := run()
	released := e.releaseStudioTask(id, kind, err)
	e.emitStudio(released)
	if err != nil {
		return out, err
	}
	if released.ID != "" {
		return released, nil
	}
	return out, nil
}

// StartStudioSpeak 在后台给短片配音。口播没锁定、没有口播时当场拒绝，不排进去再失败。
// text 空着用工程里的口播。
func (e *Engine) StartStudioSpeak(id, text string) (StudioProject, error) {
	snap, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if snap.ScriptStatus != ScriptApproved {
		return snap, fmt.Errorf("先锁定口播再配音")
	}
	if strings.TrimSpace(firstNonEmpty(text, snap.Script)) == "" {
		return snap, fmt.Errorf("还没有口播可配")
	}
	return e.StartStudioTask(id, "speak", func() error {
		_, err := e.SpeakStudio(id, text)
		return err
	})
}

// StartStudioAssemble 在后台合成短片。没有镜头、有镜头缺画面或过期时当场拒绝。
func (e *Engine) StartStudioAssemble(id string) (StudioProject, error) {
	snap, err := e.GetStudioProject(id)
	if err != nil {
		return StudioProject{}, err
	}
	if err := assembleBlocker(snap); err != nil {
		return snap, err
	}
	return e.StartStudioTask(id, "assemble", func() error {
		_, err := e.AssembleStudio(id)
		return err
	})
}

// assembleBlocker 返回合成前就能看出来的问题：没有镜头、有镜头过期、有镜头还没有画面。没有问题返回 nil。
func assembleBlocker(p StudioProject) error {
	shots := p.Storyboard.Shots
	if len(shots) == 0 {
		return fmt.Errorf("分镜里还没有镜头。先锁定口播拆出镜头，或加一张静帧")
	}
	var empty []string
	for i, s := range shots {
		has := strings.TrimSpace(s.ImageHash) != "" || strings.TrimSpace(s.VideoHash) != ""
		if s.Stale && has {
			return fmt.Errorf("第 %d 镜的画面说明改过了，重新生成后再合成", i+1)
		}
		if !has {
			empty = append(empty, fmt.Sprint(i+1))
		}
	}
	if len(empty) > 0 {
		return fmt.Errorf("第 %s 镜还没有画面。生成、上传一张图，或删掉这一镜再合成", strings.Join(empty, "、"))
	}
	return nil
}

// emitStudio 推一条 studio_state，带上编号、正在做的事、状态和最新的问题，界面据此刷新当前短片。
func (e *Engine) emitStudio(p StudioProject) {
	if e.OnHub == nil || p.ID == "" {
		return
	}
	task := ""
	if p.Task != nil {
		task = p.Task.Kind
	}
	e.OnHub("studio_state", map[string]any{
		"id": p.ID, "task": task, "status": p.Status, "auto_phase": p.AutoPhase,
		"result_hash": p.ResultHash, "narration_hash": p.NarrationHash, "revision": p.Revision,
	})
}
