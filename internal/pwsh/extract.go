package pwsh

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func extractZip(zipPath, destDir string) (string, error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", fmt.Errorf("pwsh: unzip: %w", err)
	}
	defer r.Close()
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	root := filepath.Clean(destDir)
	var total uint64
	for _, f := range r.File {
		name := zipEntryName(f.Name)
		if name == "" || strings.HasSuffix(name, "/") {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		rel, ok := safeRel(name)
		if !ok {
			continue
		}
		total += f.UncompressedSize64
		if total > maxUncompressedBytes {
			return "", fmt.Errorf("pwsh: uncompressed archive larger than %d bytes", maxUncompressedBytes)
		}
		if f.UncompressedSize64 > maxZipBytes {
			return "", fmt.Errorf("pwsh: zip entry too large")
		}
		dest := filepath.Join(root, rel)
		if !within(root, dest) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return "", err
		}
		if err := writeZipFile(f, dest); err != nil {
			return "", err
		}
	}
	bin := lookDir(root)
	if bin == "" {
		return "", fmt.Errorf("pwsh: archive has no pwsh.exe")
	}
	return bin, nil
}

func zipEntryName(name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, "\\", "/"))
}

func safeRel(name string) (string, bool) {
	name = zipEntryName(name)
	if name == "" || name == "." {
		return "", false
	}
	parts := strings.Split(name, "/")
	var keep []string
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			return "", false
		}
		keep = append(keep, p)
	}
	if len(keep) == 0 {
		return "", false
	}
	return filepath.Join(keep...), true
}

func within(root, dest string) bool {
	rel, err := filepath.Rel(root, dest)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != "" && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)
}

func writeZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	tmp := dest + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, io.LimitReader(rc, maxZipBytes+1))
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
