package cut

import (
	"os/exec"

	"github.com/Shenchangxin/yoyo/internal/osutil"
)

func quiet(cmd *exec.Cmd) *exec.Cmd {
	osutil.HideConsole(cmd)
	return cmd
}
