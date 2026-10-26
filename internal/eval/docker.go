// 有 environment/Dockerfile 且本机找得到 docker 时，用无网络容器跑验证脚本。
// 容器命令失败时 runVerifier 退回本机脚本。不准备智能体工作区，tests/ 以只读方式挂到 /grader。

package eval

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// dockerAvailable 在 PATH 里能找到 docker 时为真。
func dockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

// dockerfileOf 返回任务 environment/Dockerfile 的路径。不存在或是目录时返回空串。
func dockerfileOf(taskDir string) string {
	p := filepath.Join(taskDir, "environment", "Dockerfile")
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return ""
	}
	return p
}

// runVerifierDocker 用任务的 environment 目录构建镜像，标签为 yoyo-eval- 加小写题号，斜杠换成连字符。
// 工作区挂到 /app，任务目录的 tests 以只读挂到 /grader，网络为 none。ps1 用 pwsh，其余用 sh。退出码为 0 时返回真。
func runVerifierDocker(work string, task Task, script string) (bool, string, error) {
	ctxDir := filepath.Join(task.Dir, "environment")
	tag := "yoyo-eval-" + strings.ToLower(strings.ReplaceAll(task.ID, "/", "-"))
	build := quiet(exec.Command("docker", "build", "-q", "-t", tag, ctxDir))
	if out, err := build.CombinedOutput(); err != nil {
		return false, string(out), err
	}
	args := []string{"run", "--rm", "--network=none",
		"-v", work + ":/app",
		"-v", filepath.Join(task.Dir, "tests") + ":/grader:ro",
		"-w", "/app", tag}
	if strings.EqualFold(filepath.Ext(script), ".ps1") {
		args = append(args, "pwsh", "-File", "/grader/"+filepath.Base(script))
	} else {
		args = append(args, "sh", "/grader/"+filepath.Base(script))
	}
	cmd := quiet(exec.Command("docker", args...))
	out, err := cmd.CombinedOutput()
	return err == nil, string(out), err
}
