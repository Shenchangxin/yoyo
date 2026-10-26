// quiet.go 让桌面应用拉起外部命令时不弹出控制台窗口。

package video

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// quiet 给命令套上 osutil.HideConsole 再返回。ffmpeg、ffprobe、npm、yt-dlp 和 edge-tts 都走这里。
func quiet(cmd *exec.Cmd) *exec.Cmd {
	osutil.HideConsole(cmd)
	return cmd
}
