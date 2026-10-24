package video

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

// quiet keeps ffmpeg, ffprobe, npm, yt-dlp, and edge-tts from opening a
// console window when the desktop app starts them.
func quiet(cmd *exec.Cmd) *exec.Cmd {
	osutil.HideConsole(cmd)
	return cmd
}
