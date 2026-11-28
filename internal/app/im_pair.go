// im_pair.go 把 IM 主通道的配对码写到 home 里，桌面上生成的码，另一个 yoyo 进程收到 /绑定 时也认得。
// 码十分钟有效，对上一次就删掉。调用方都持有 imh.mu。

package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type imPairFile struct {
	Code  string    `json:"code"`
	Until time.Time `json:"until"`
}

func (a *App) imPairPath() string {
	home := a.imHomePath()
	if home == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(home), "pair.json")
}

// savePairLocked 把内存里的配对码写下来。写不成时只在这个进程里有效。
func (a *App) savePairLocked() {
	path := a.imPairPath()
	if path == "" {
		return
	}
	raw, err := json.Marshal(imPairFile{Code: a.imh.pair, Until: a.imh.pairUntil})
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	tmp := path + ".tmp"
	if os.WriteFile(tmp, raw, 0o600) == nil {
		_ = os.Rename(tmp, path)
	}
}

// loadPairLocked 读出现在有效的配对码。文件没了表示已经用掉或解除，内存里的也作废。
func (a *App) loadPairLocked() {
	path := a.imPairPath()
	if path == "" {
		return
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			a.imh.pair = ""
			a.imh.pairUntil = time.Time{}
		}
		return
	}
	var f imPairFile
	if json.Unmarshal(raw, &f) != nil {
		return
	}
	a.imh.pair, a.imh.pairUntil = f.Code, f.Until
}

func (a *App) dropPairLocked() {
	if path := a.imPairPath(); path != "" {
		_ = os.Remove(path)
	}
}
