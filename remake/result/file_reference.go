// Copyright 2026 Hypit.AI
// Translated from @hypit/result/src/node/file-reference.ts
// (hypit e26d9142fd8a6f78bf4f3cdb800d6205f4ecca16).
// See ../LICENSE.

package result

import (
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ExternalFileAccess reads an explicit external address supplied by the Workspace.
type ExternalFileAccess interface {
	Size(uri string) (int64, error)
	Open(uri string, fileRange *BuildResultFileRange) (io.ReadCloser, error)
}

type localFiles struct{}

// LocalExternalFiles opens Workspace file: URLs.
var LocalExternalFiles ExternalFileAccess = localFiles{}

func (localFiles) Size(uri string) (int64, error) {
	info, err := os.Stat(fileURLToPath(uri))
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

func (localFiles) Open(uri string, fileRange *BuildResultFileRange) (io.ReadCloser, error) {
	if fileRange != nil {
		size, err := LocalExternalFiles.Size(uri)
		if err != nil {
			return nil, err
		}
		if !safeRangeBound(fileRange.Start) || !safeRangeBound(fileRange.EndExclusive) || fileRange.EndExclusive <= fileRange.Start || fileRange.EndExclusive > size {
			panicErr("Invalid file range for " + uri)
		}
	}
	file, err := openShared(fileURLToPath(uri))
	if err != nil {
		return nil, err
	}
	if fileRange == nil {
		return file, nil
	}
	if _, err := file.Seek(fileRange.Start, io.SeekStart); err != nil {
		file.Close()
		return nil, err
	}
	return sectionCloser{Reader: io.LimitReader(file, fileRange.EndExclusive-fileRange.Start), Closer: file}, nil
}

func safeRangeBound(value int64) bool {
	return value >= 0 && value <= maxSafeInteger
}

func fileURLToPath(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "file" {
		panicErr("The URL must be of scheme file")
	}
	escaped := strings.ToLower(parsed.EscapedPath() + parsed.RawPath)
	if strings.Contains(escaped, "%2f") || strings.Contains(escaped, "%5c") {
		panicErr("File URL path must not include encoded \\ or / characters")
	}
	if runtime.GOOS == "windows" {
		return fileURLToPathWindows(parsed)
	}
	if parsed.Host != "" {
		panicErr("File URL host must be empty on this platform")
	}
	if !strings.HasPrefix(parsed.Path, "/") {
		panicErr("File URL path must be absolute")
	}
	return parsed.Path
}

func fileURLToPathWindows(parsed *url.URL) string {
	if parsed.Host != "" {
		return `\\` + parsed.Host + filepath.FromSlash(parsed.Path)
	}
	if len(parsed.Path) < 3 || parsed.Path[0] != '/' || parsed.Path[2] != ':' {
		panicErr("File URL path must be absolute")
	}
	return filepath.FromSlash(parsed.Path[1:])
}

// FileReferenceIdentity is the explicit URI, or the owning Build and path.
func FileReferenceIdentity(owner string, file BuildResultFileRef) string {
	if file.Kind == "external-file" {
		return jsonStringify([]any{"external", file.URI})
	}
	build := owner
	if file.Build != nil {
		build = *file.Build
	}
	return jsonStringify([]any{"result", build, file.Path})
}

// OwnedFileReference makes a relative Result file reference name its owner.
func OwnedFileReference(owner string, file BuildResultFileRef) BuildResultFileRef {
	if file.Kind != "build-file" {
		return file
	}
	if file.Build == nil {
		owned := owner
		file.Build = &owned
	}
	return file
}

// CurrentFileReference reads the current size of an external file.
func CurrentFileReference(file BuildResultFileRef, access ...ExternalFileAccess) (BuildResultFileRef, error) {
	if file.Kind != "external-file" {
		return file, nil
	}
	size, err := externalFileAccess(access).Size(file.URI)
	if err != nil {
		return BuildResultFileRef{}, err
	}
	file.Size = size
	return file, nil
}

type sectionCloser struct {
	io.Reader
	io.Closer
}
