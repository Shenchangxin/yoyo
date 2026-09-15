package update

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// Pending names the staged binary a human asked to install. The agent loop
// has no tool that writes this file.
type Pending struct {
	Staging string `json:"staging"`
	SHA256  string `json:"sha256,omitempty"`
}

func pendingPath(dir string) string {
	return filepath.Join(dir, "pending.json")
}

func MarkPending(dir string, p Pending) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pendingPath(dir), b, 0o644)
}

func ReadPending(dir string) (Pending, error) {
	var p Pending
	b, err := os.ReadFile(pendingPath(dir))
	if err != nil {
		return p, err
	}
	return p, json.Unmarshal(b, &p)
}

func ClearPending(dir string) error {
	err := os.Remove(pendingPath(dir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// ApplyStaged replaces exePath with the staged file. It never runs from the
// agent tool loop. On Windows the running image is renamed to .old first so
// the replacement can take the original name; the current process keeps
// serving until exit. This is not an in-place overwrite of a locked argv[0].
func ApplyStaged(exePath, stagingPath string) error {
	if exePath == "" {
		var err error
		exePath, err = os.Executable()
		if err != nil {
			return err
		}
	}
	exePath, err := filepath.Abs(exePath)
	if err != nil {
		return err
	}
	stagingPath, err = filepath.Abs(stagingPath)
	if err != nil {
		return err
	}
	if exePath == stagingPath {
		return fmt.Errorf("update: refusing to apply staging onto itself")
	}
	st, err := os.Stat(stagingPath)
	if err != nil {
		return fmt.Errorf("update: staging: %w", err)
	}
	if st.Size() == 0 {
		return fmt.Errorf("update: empty staging file")
	}
	backup := exePath + ".old"
	_ = os.Remove(backup)
	if err := os.Rename(exePath, backup); err != nil {
		return fmt.Errorf("update: cannot move current binary aside: %w", err)
	}
	if err := copyFile(stagingPath, exePath, 0o755); err != nil {
		_ = os.Rename(backup, exePath)
		return fmt.Errorf("update: copy staging: %w", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(exePath, 0o755)
	}
	return nil
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func StagingName() string { return "yoyo.staging" }
