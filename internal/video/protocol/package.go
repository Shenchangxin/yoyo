// package.go 检查上传的插件 zip，只解析清单和文件，不执行其中的代码。

package protocol

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
)

// 包格式名和各类体积上限。
const (
	// PluginPackageFormat 是包格式名 yingce.plugin/package-v1。
	PluginPackageFormat = "yingce.plugin/package-v1"
	// PluginPackageMaxBytes 是整个包的上限，48 MiB。
	PluginPackageMaxBytes = 48 << 20
	// PluginManifestMaxBytes 是 manifest.json 的上限，512 KiB。
	PluginManifestMaxBytes = 512 << 10
	// pluginPackageMaxFiles 是包内文件数上限，256。
	pluginPackageMaxFiles = 256
	// pluginPackageMaxEntry 是单个文件解压后的上限，16 MiB。
	pluginPackageMaxEntry = 16 << 20
)

// PluginPackage 是上传插件的传输信封。清单仍是唯一的能力合同。文件是可选的运行产物，由 manifest.entry 指向。
type PluginPackage struct {
	Manifest    Manifest
	ManifestRaw []byte
	Files       map[string][]byte
}

// ParsePluginPackage 检查 zip 容器并返回清单和文件。这个函数不执行上传的代码。空包、超过 PluginPackageMaxBytes、文件数不在 1 到 256、符号链接、单文件超过 16 MiB、缺少 manifest.json 或清单不合法都会失败。
// data 长度为 0 或大于 PluginPackageMaxBytes 时失败，否则作为 zip 读入。
func ParsePluginPackage(data []byte) (PluginPackage, error) {
	if len(data) == 0 || len(data) > PluginPackageMaxBytes {
		return PluginPackage{}, fmt.Errorf("plugin package must be between 1 and %d bytes", PluginPackageMaxBytes)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return PluginPackage{}, fmt.Errorf("decode plugin package: %w", err)
	}
	if len(reader.File) == 0 || len(reader.File) > pluginPackageMaxFiles {
		return PluginPackage{}, fmt.Errorf("plugin package must contain between 1 and %d files", pluginPackageMaxFiles)
	}
	files := make(map[string][]byte, len(reader.File))
	var manifestRaw []byte
	for _, file := range reader.File {
		name, err := validatePluginPackagePath(file.Name)
		if err != nil {
			return PluginPackage{}, err
		}
		if _, exists := files[name]; exists {
			return PluginPackage{}, fmt.Errorf("duplicate plugin package file %q", name)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return PluginPackage{}, fmt.Errorf("plugin package cannot contain symlink %q", name)
		}
		if file.UncompressedSize64 > pluginPackageMaxEntry {
			return PluginPackage{}, fmt.Errorf("plugin package file %q exceeds %d bytes", name, pluginPackageMaxEntry)
		}
		stream, err := file.Open()
		if err != nil {
			return PluginPackage{}, fmt.Errorf("open plugin package file %q: %w", name, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(stream, pluginPackageMaxEntry+1))
		closeErr := stream.Close()
		if readErr != nil {
			return PluginPackage{}, fmt.Errorf("read plugin package file %q: %w", name, readErr)
		}
		if closeErr != nil {
			return PluginPackage{}, fmt.Errorf("close plugin package file %q: %w", name, closeErr)
		}
		if len(content) > pluginPackageMaxEntry {
			return PluginPackage{}, fmt.Errorf("plugin package file %q exceeds %d bytes", name, pluginPackageMaxEntry)
		}
		files[name] = content
		if name == "manifest.json" {
			manifestRaw = content
		}
	}
	if len(manifestRaw) == 0 {
		return PluginPackage{}, fmt.Errorf("plugin package must contain manifest.json")
	}
	if len(manifestRaw) > PluginManifestMaxBytes {
		return PluginPackage{}, fmt.Errorf("manifest.json exceeds %d bytes", PluginManifestMaxBytes)
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return PluginPackage{}, fmt.Errorf("decode plugin manifest: %w", err)
	}
	if err := ValidateManifest(manifest); err != nil {
		return PluginPackage{}, err
	}
	if err := validatePluginPackageRuntime(manifest, files); err != nil {
		return PluginPackage{}, err
	}
	return PluginPackage{Manifest: manifest, ManifestRaw: manifestRaw, Files: files}, nil
}

// validatePluginPackagePath 拒绝空路径、反斜杠、绝对路径和 ..。只允许 manifest.json、README.md、LICENSE，以及 web/、backend/、assets/、docs/ 下面的相对路径。
// name 去掉首尾空白；空串、含反斜杠、以 / 开头、绝对路径、Clean 后变化、以 ../ 开头或等于 .. 都失败。
func validatePluginPackagePath(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsRune(name, '\\') || strings.HasPrefix(name, "/") || path.IsAbs(name) || path.Clean(name) != name || strings.HasPrefix(name, "../") || name == ".." {
		return "", fmt.Errorf("invalid plugin package path %q", name)
	}
	if name == "manifest.json" || strings.HasPrefix(name, "web/") || strings.HasPrefix(name, "backend/") || strings.HasPrefix(name, "assets/") || strings.HasPrefix(name, "docs/") || name == "README.md" || name == "LICENSE" {
		return name, nil
	}
	return "", fmt.Errorf("plugin package path %q is outside the allowed package roots", name)
}

// validatePluginPackageRuntime 在后端是 rpc 时要求包里有可执行产物，是 wasm 时要求条目以 .wasm 结尾且文件在包内。有网页文件或 runtime.web 时，manifest.entry 必须落在 web/ 里，并且 runtime.web 等于 sandbox 或 worker。
// manifest 的 Runtime.Backend 为 rpc 或 wasm 时检查 BackendEntry；Entry 为空时再看 Runtime.Web。
// files 在 rpc 时传给 HasAnyPaymentRPCBackend，在 wasm 和网页入口时按路径查找，Entry 为空时还看有没有 web/ 前缀。
func validatePluginPackageRuntime(manifest Manifest, files map[string][]byte) error {
	backend := strings.TrimSpace(manifest.Runtime.Backend)
	if backend == "rpc" || backend == "wasm" {
		entry := strings.TrimSpace(manifest.Runtime.BackendEntry)
		if !strings.HasPrefix(entry, "backend/") || path.Clean(entry) != entry {
			return fmt.Errorf("manifest.runtime.backendEntry must point inside backend/")
		}
		if backend == "rpc" {
			if !HasAnyPaymentRPCBackend(entry, files) {
				return fmt.Errorf("manifest runtime backend entry %q is missing a host-executable artifact", entry)
			}
		} else if _, ok := files[entry]; !ok {
			return fmt.Errorf("manifest runtime backend entry %q is missing from package", entry)
		}
		if backend == "wasm" && !strings.HasSuffix(strings.ToLower(entry), ".wasm") {
			return fmt.Errorf("wasm backend entry must have .wasm extension")
		}
	}
	if strings.TrimSpace(manifest.Entry) == "" {
		if strings.TrimSpace(manifest.Runtime.Web) != "" {
			return fmt.Errorf("manifest.runtime.web requires manifest.entry")
		}
		for name := range files {
			if strings.HasPrefix(name, "web/") {
				return fmt.Errorf("plugin package web files require manifest.entry")
			}
		}
		return nil
	}
	entry := strings.TrimSpace(manifest.Entry)
	if !strings.HasPrefix(entry, "web/") || path.Clean(entry) != entry {
		return fmt.Errorf("plugin manifest entry must point inside web/")
	}
	if _, ok := files[entry]; !ok {
		return fmt.Errorf("plugin manifest entry %q is missing from package", entry)
	}
	if manifest.Runtime.Web != "sandbox" && manifest.Runtime.Web != "worker" {
		return fmt.Errorf("uploaded web plugins require runtime.web sandbox or worker")
	}
	return nil
}
