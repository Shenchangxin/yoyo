package eval

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func dockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func dockerfileOf(taskDir string) string {
	p := filepath.Join(taskDir, "environment", "Dockerfile")
	st, err := os.Stat(p)
	if err != nil || st.IsDir() {
		return ""
	}
	return p
}

func runVerifierDocker(work string, task Task, script string) (bool, string, error) {
	ctxDir := filepath.Join(task.Dir, "environment")
	tag := "yoyo-eval-" + strings.ToLower(strings.ReplaceAll(task.ID, "/", "-"))
	build := exec.Command("docker", "build", "-q", "-t", tag, ctxDir)
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
	cmd := exec.Command("docker", args...)
	out, err := cmd.CombinedOutput()
	return err == nil, string(out), err
}
