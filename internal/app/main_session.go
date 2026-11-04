// 主循环只有一条会话。工人不另占一条给人看的对话。
// 历史上最近的一条对话不自动变成主会话。

package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/session"
)

// MainSession 返回主循环那一条会话。
// 还没有记录、记录指着已删除、已归档或非代理车道时，新建一条空会话并记下。
func (a *App) MainSession() (SessionMeta, error) {
	if a == nil || a.Home == nil {
		return SessionMeta{}, errString("home unavailable")
	}
	if meta, ok := a.savedMain(); ok {
		return meta, nil
	}
	meta, err := a.NewSession("")
	if err != nil {
		return SessionMeta{}, err
	}
	if err := a.writeMainID(meta.ID); err != nil {
		return SessionMeta{}, err
	}
	return meta, nil
}

// AdoptMainSession 把一条已有的代理会话记成主会话。
// 桌面主循环不再为此新建一条。主会话像一条持续对话，上下文用检查点收缩，不靠换 id。
func (a *App) AdoptMainSession(id string) error {
	id = strings.TrimSpace(id)
	if a == nil || a.Home == nil || id == "" {
		return errString("session required")
	}
	meta, err := a.GetSession(id)
	if err != nil {
		return err
	}
	if meta.Archived || session.NormalizeChannel(meta.Channel) != session.ChannelAgent {
		return errString("not a main session")
	}
	return a.writeMainID(id)
}

func (a *App) savedMain() (SessionMeta, bool) {
	id := a.readMainID()
	if id == "" {
		return SessionMeta{}, false
	}
	meta, err := a.GetSession(id)
	if err != nil || meta.Archived || session.NormalizeChannel(meta.Channel) != session.ChannelAgent {
		return SessionMeta{}, false
	}
	return meta, true
}

func (a *App) mainSessionPath() string {
	return filepath.Join(a.Home.Root, "main-session")
}

func (a *App) readMainID() string {
	b, err := os.ReadFile(a.mainSessionPath())
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (a *App) writeMainID(id string) error {
	return os.WriteFile(a.mainSessionPath(), []byte(strings.TrimSpace(id)+"\n"), 0o644)
}
