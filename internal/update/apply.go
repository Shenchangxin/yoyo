package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	// ErrQuitNeeded means the installer was spawned and the app should exit.
	ErrQuitNeeded = errors.New("update: quit to finish install")
	// ErrOpenedPackage means a DMG/folder was opened for the user to finish.
	ErrOpenedPackage = errors.New("update: opened package for the user")
)

// Pending names the staged payload a human asked to install. The agent loop
// has no tool that writes this file.
type Pending struct {
	Staging  string `json:"staging"`
	SHA256   string `json:"sha256,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Version  string `json:"version,omitempty"`
	Filename string `json:"filename,omitempty"`
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

func StagingName() string { return "yoyo.staging" }

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

// ApplyDownloaded installs a previously downloaded package. Desktop installers
// do not replace argv[0]; CLI archives still stage onto the running binary.
func ApplyDownloaded(exePath, dir string) error {
	p, err := ReadPending(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return ApplyStaged(exePath, filepath.Join(dir, StagingName()))
		}
		return err
	}
	path := p.Staging
	if path == "" {
		path = filepath.Join(dir, StagingName())
	}
	kind := strings.ToLower(strings.TrimSpace(p.Kind))
	if kind == "" {
		kind = kindFromName(path)
	}
	switch kind {
	case "nsis", "exe":
		dest := installDir(exePath)
		if err := applyNSIS(path, dest); err != nil {
			return err
		}
		return ErrQuitNeeded
	case "dmg":
		if err := openPath(path); err != nil {
			return err
		}
		return ErrOpenedPackage
	case "appimage":
		if err := applyAppImage(path, exePath); err != nil {
			return err
		}
		return ErrQuitNeeded
	case "zip", "tar.gz", "tgz", "cli":
		bin, err := extractCLIBinary(path, dir)
		if err != nil {
			return err
		}
		return ApplyStaged(exePath, bin)
	default:
		return ApplyStaged(exePath, path)
	}
}

func kindFromName(path string) string {
	n := strings.ToLower(path)
	switch {
	case strings.HasSuffix(n, "-setup.exe"), strings.HasSuffix(n, "installer.exe"):
		return "nsis"
	case strings.HasSuffix(n, ".dmg"):
		return "dmg"
	case strings.HasSuffix(n, ".appimage"):
		return "appimage"
	case strings.HasSuffix(n, ".zip"):
		return "zip"
	case strings.HasSuffix(n, ".tar.gz"), strings.HasSuffix(n, ".tgz"):
		return "tar.gz"
	default:
		return "bin"
	}
}

func installDir(exePath string) string {
	if exePath == "" {
		exePath, _ = os.Executable()
	}
	if exePath != "" {
		if abs, err := filepath.Abs(exePath); err == nil {
			return filepath.Dir(abs)
		}
	}
	return ""
}

func extractCLIBinary(archive, destDir string) (string, error) {
	lower := strings.ToLower(archive)
	out := filepath.Join(destDir, StagingName())
	switch {
	case strings.HasSuffix(lower, ".zip"):
		return out, unzipFirstBinary(archive, out)
	case strings.HasSuffix(lower, ".tar.gz"), strings.HasSuffix(lower, ".tgz"):
		return out, untarFirstBinary(archive, out)
	default:
		return archive, nil
	}
}

func unzipFirstBinary(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()
	var best *zip.File
	for i := range r.File {
		f := r.File[i]
		name := filepath.Base(strings.ReplaceAll(f.Name, "\\", "/"))
		if f.FileInfo().IsDir() {
			continue
		}
		if name == "yoyo" || name == "yoyo.exe" {
			best = f
			break
		}
		if best == nil {
			best = f
		}
	}
	if best == nil {
		return fmt.Errorf("update: zip has no files")
	}
	rc, err := best.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	return writeNewFile(dest, rc, 0o755)
}

func untarFirstBinary(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var found bool
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Base(hdr.Name)
		if name == "yoyo" || name == "yoyo.exe" || !found {
			if err := writeNewFile(dest, tr, 0o755); err != nil {
				return err
			}
			found = true
			if name == "yoyo" || name == "yoyo.exe" {
				return nil
			}
		}
	}
	if !found {
		return fmt.Errorf("update: tar has no files")
	}
	return nil
}

func writeNewFile(dest string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, r)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func copyFile(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	return writeNewFile(dst, in, mode)
}
