package runtime

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

func hideConsole(cmd *exec.Cmd) {
	osutil.HideConsole(cmd)
}

func gitCmd(args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	hideConsole(cmd)
	return cmd
}
