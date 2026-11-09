// install_binary.go 拥有二进制分发的托管安装：按平台取归档，有 sha256 就校验并允许走镜像，没有 sha256 只走原始 HTTPS 地址并把实际哈希记下来；
// 解到 buddies/<id>/<版本>，统一 cmd 里的分隔符，Unix 上补可执行位，最后去掉 MOTW 或 quarantine 标记。

package buddy

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/fetch"
)

// maxBinaryBytes 是二进制归档的下载上限。
const maxBinaryBytes int64 = 2 << 30

// installBinary 下载并解开一个二进制归档。
func (m *Manager) installBinary(ctx context.Context, e Entry, t BinaryTarget, prog func(Progress), log io.Writer) (*InstallRecord, error) {
	u, err := url.Parse(t.Archive)
	if err != nil || u.Scheme != "https" {
		return nil, fmt.Errorf("binary archive must be https: %s", t.Archive)
	}
	urls := []string{t.Archive}
	if t.SHA256 != "" {
		urls = fetch.MirrorURLs(t.Archive)
	}
	name := path.Base(u.Path)
	archive := filepath.Join(m.installDir(e.ID), ".download", name)
	if err := os.MkdirAll(filepath.Dir(archive), 0o755); err != nil {
		return nil, err
	}
	defer os.RemoveAll(filepath.Dir(archive))
	fmt.Fprintf(log, "download %s\n", t.Archive)
	prog(Progress{Phase: "download"})
	res, err := fetch.Download(ctx, fetch.Spec{
		URLs: urls, SHA256: t.SHA256, MaxBytes: maxBinaryBytes, Dest: archive, Client: m.opts.HTTP,
		Progress: func(done, total int64) {
			if total > 0 {
				prog(Progress{Phase: "download", Percent: int(done * 80 / total)})
			}
		},
	})
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(log, "sha256 %s\n", res.SHA256)
	prog(Progress{Phase: "extract", Percent: 85})
	dest := filepath.Join(m.installDir(e.ID), safeName(e.Version))
	if err := fetch.Extract(archive, dest, fetch.KindOf(name), 0); err != nil {
		return nil, err
	}
	cmd := filepath.Join(dest, filepath.FromSlash(strings.TrimPrefix(strings.ReplaceAll(t.Cmd, `\`, "/"), "./")))
	if !within(dest, cmd) {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("binary cmd escapes the install dir: %s", t.Cmd)
	}
	if !fileExists(cmd) {
		_ = os.RemoveAll(dest)
		return nil, fmt.Errorf("binary cmd missing after extract: %s", t.Cmd)
	}
	if m.plat.GOOS != "windows" {
		_ = os.Chmod(cmd, 0o755)
	}
	fetch.Unblock(dest)
	ext := strings.ToLower(filepath.Ext(cmd))
	return &InstallRecord{
		Kind: "binary", Version: e.Version, Command: cmd, Args: append([]string(nil), t.Args...), Env: t.Env,
		ViaCmd: m.plat.GOOS == "windows" && (ext == ".cmd" || ext == ".bat"),
		Dir:    dest, SHA256: res.SHA256, Verified: t.SHA256 != "",
	}, nil
}
