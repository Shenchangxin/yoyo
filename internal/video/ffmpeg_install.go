package video

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	ffmpegMaxZip    = 256 << 20
	ffmpegUserAgent = "Mozilla/5.0 (compatible; Yoyo-Workstation)"
)

var (
	ffmpegEndpoint = "https://www.modelscope.cn"
	ffmpegRepo     = "NanoModel/ffmpeg"
	ffmpegRevision = "master"
	ffmpegHTTP     = newFFmpegHTTP()
	ffmpegVerify   = verifyFFmpegBin
)

func newFFmpegHTTP() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 25 * time.Minute, Jar: jar}
}

type ffmpegRemoteFile struct {
	Path   string
	Sha256 string
	Size   int64
}

func (e *Engine) ffmpegBinDir() string {
	if e == nil || e.Dir == "" {
		return ""
	}
	return filepath.Join(e.Dir, "bin")
}

func (e *Engine) ffmpegPlatform() (string, string) {
	goos, goarch := runtime.GOOS, runtime.GOARCH
	if e != nil && e.ffmpegOS != "" {
		goos = e.ffmpegOS
	}
	if e != nil && e.ffmpegArch != "" {
		goarch = e.ffmpegArch
	}
	return goos, goarch
}

func (e *Engine) ffmpegClient() HTTPDoer {
	if e != nil && e.ffmpegHTTP != nil {
		return e.ffmpegHTTP
	}
	return ffmpegHTTP
}

func (e *Engine) refreshFFmpeg() {
	if e == nil {
		return
	}
	e.FFMPEG, e.FFProbe = LookFFmpegIn([]string{e.ffmpegBinDir()})
}

func ffmpegRepoSpec() (endpoint, repo, rev string) {
	endpoint = strings.TrimRight(ffmpegEndpoint, "/")
	repo = ffmpegRepo
	rev = ffmpegRevision
	if v := strings.TrimSpace(os.Getenv("YOYO_FFMPEG_ENDPOINT")); v != "" {
		endpoint = strings.TrimRight(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("YOYO_FFMPEG_REPO")); v != "" {
		repo = strings.Trim(v, "/")
	}
	if v := strings.TrimSpace(os.Getenv("YOYO_FFMPEG_REVISION")); v != "" {
		rev = v
	}
	return endpoint, repo, rev
}

func ffmpegArchiveName(goos, goarch string) (string, error) {
	osName := ""
	switch goos {
	case "windows":
		osName = "windows"
	case "linux":
		osName = "linux"
	case "darwin":
		osName = "macos"
	default:
		return "", fmt.Errorf("ffmpeg: unsupported os %s", goos)
	}
	arch := ""
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	default:
		return "", fmt.Errorf("ffmpeg: unsupported arch %s", goarch)
	}
	return "ffmpeg-" + osName + "-" + arch + ".zip", nil
}

func matchFFmpegArchive(name, goos, goarch string) bool {
	s := strings.ToLower(path.Base(name))
	if !strings.HasSuffix(s, ".zip") || !strings.Contains(s, "ffmpeg") {
		return false
	}
	osOK := false
	switch goos {
	case "windows":
		osOK = strings.Contains(s, "windows") || strings.Contains(s, "win")
	case "linux":
		osOK = strings.Contains(s, "linux")
	case "darwin":
		osOK = strings.Contains(s, "macos") || strings.Contains(s, "darwin") || strings.Contains(s, "osx")
	}
	if !osOK {
		return false
	}
	switch goarch {
	case "amd64":
		return strings.Contains(s, "x86_64") || strings.Contains(s, "amd64") || hasX64(s)
	case "arm64":
		return strings.Contains(s, "aarch64") || strings.Contains(s, "arm64")
	default:
		return false
	}
}

func hasX64(s string) bool {
	return strings.Contains(s, "x64") && !strings.Contains(s, "x86_64")
}

func pickFFmpegArchive(names []string, goos, goarch string) (string, error) {
	want, err := ffmpegArchiveName(goos, goarch)
	if err != nil {
		return "", err
	}
	for _, n := range names {
		if strings.EqualFold(path.Base(n), want) {
			return n, nil
		}
	}
	var hits []string
	for _, n := range names {
		if matchFFmpegArchive(n, goos, goarch) {
			hits = append(hits, n)
		}
	}
	if len(hits) == 0 {
		return want, nil
	}
	best := hits[0]
	for _, h := range hits[1:] {
		if len(path.Base(h)) < len(path.Base(best)) {
			best = h
		}
	}
	return best, nil
}

func ffmpegResolveURL(endpoint, repo, rev, file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	return strings.TrimRight(endpoint, "/") + "/datasets/" + repo + "/resolve/" + rev + "/" + file
}

func ffmpegAPIFileURL(endpoint, repo, rev, file string) string {
	u, err := url.Parse(strings.TrimRight(endpoint, "/") + "/api/v1/datasets/" + repo + "/repo")
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("Revision", rev)
	q.Set("FilePath", file)
	u.RawQuery = q.Encode()
	return u.String()
}

func ffmpegTreeURL(endpoint, repo, rev string) string {
	u, err := url.Parse(strings.TrimRight(endpoint, "/") + "/api/v1/datasets/" + repo + "/repo/tree")
	if err != nil {
		return ""
	}
	q := u.Query()
	q.Set("Revision", rev)
	q.Set("Recursive", "true")
	u.RawQuery = q.Encode()
	return u.String()
}

func parseFFmpegRepoTree(raw []byte) []ffmpegRemoteFile {
	var wrap struct {
		Code int `json:"Code"`
		Data struct {
			Files []struct {
				Name   string `json:"Name"`
				Path   string `json:"Path"`
				Type   string `json:"Type"`
				Size   int64  `json:"Size"`
				Sha256 string `json:"Sha256"`
			} `json:"Files"`
		} `json:"Data"`
	}
	if json.Unmarshal(raw, &wrap) != nil {
		return nil
	}
	var out []ffmpegRemoteFile
	for _, f := range wrap.Data.Files {
		p := strings.TrimSpace(f.Path)
		if p == "" {
			p = strings.TrimSpace(f.Name)
		}
		if p == "" || strings.EqualFold(f.Type, "tree") {
			continue
		}
		out = append(out, ffmpegRemoteFile{Path: p, Sha256: strings.TrimSpace(f.Sha256), Size: f.Size})
	}
	return out
}

func (e *Engine) listFFmpegRepo(ctx context.Context) ([]ffmpegRemoteFile, error) {
	endpoint, repo, rev := ffmpegRepoSpec()
	rawURL := ffmpegTreeURL(endpoint, repo, rev)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ffmpegUserAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://www.modelscope.cn/")
	res, err := e.ffmpegClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 300 {
		return nil, fmt.Errorf("ffmpeg: list %s", res.Status)
	}
	files := parseFFmpegRepoTree(body)
	if len(files) == 0 {
		return nil, fmt.Errorf("ffmpeg: empty repo listing")
	}
	return files, nil
}

func (e *Engine) ffmpegDo(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", ffmpegUserAgent)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Referer", "https://www.modelscope.cn/")
	return e.ffmpegClient().Do(req)
}

func (e *Engine) EnsureFFmpeg(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	e.refreshFFmpeg()
	if e.FFmpegOK() {
		return nil
	}
	wait, err := e.beginFFmpegInstall()
	if err != nil {
		return err
	}
	if wait == nil {
		e.refreshFFmpeg()
		if e.FFmpegOK() {
			return nil
		}
		return fmt.Errorf("ffmpeg is not available")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-wait:
	}
	e.refreshFFmpeg()
	if e.FFmpegOK() {
		return nil
	}
	e.ffMu.Lock()
	msg := strings.TrimSpace(e.ffErr)
	e.ffMu.Unlock()
	if msg == "" {
		msg = "ffmpeg install failed"
	}
	return fmt.Errorf("%s", msg)
}

func (e *Engine) InstallFFmpeg() map[string]any {
	e.refreshFFmpeg()
	if !e.FFmpegOK() {
		_, _ = e.beginFFmpegInstall()
	}
	return e.ffmpegCallResult()
}

func (e *Engine) ffmpegCallResult() map[string]any {
	return map[string]any{
		"ok":             true,
		"ffmpeg":         e.FFmpegOK(),
		"ffmpeg_bin":     e.FFMPEG,
		"ffprobe_bin":    e.FFProbe,
		"ffmpeg_install": e.FFmpegInstallSnapshot(),
	}
}

func (e *Engine) beginFFmpegInstall() (<-chan struct{}, error) {
	goos, goarch := e.ffmpegPlatform()
	if _, err := ffmpegArchiveName(goos, goarch); err != nil {
		e.ffMu.Lock()
		e.ffPhase = "error"
		e.ffErr = err.Error()
		e.ffMu.Unlock()
		return nil, err
	}
	e.ffMu.Lock()
	defer e.ffMu.Unlock()
	if e.FFMPEG != "" {
		return nil, nil
	}
	if e.ffWait != nil {
		return e.ffWait, nil
	}
	ch := make(chan struct{})
	e.ffWait = ch
	e.ffPhase = "resolving"
	e.ffPercent = 0
	e.ffBytes = 0
	e.ffTotal = 0
	e.ffErr = ""
	go e.runFFmpegInstall(ch)
	return ch, nil
}

func (e *Engine) runFFmpegInstall(done chan struct{}) {
	defer func() {
		e.ffMu.Lock()
		if e.ffWait == done {
			e.ffWait = nil
		}
		e.ffMu.Unlock()
		close(done)
		e.hub("ffmpeg", e.FFmpegInstallSnapshot())
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Minute)
	defer cancel()
	err := e.installFFmpegBundle(ctx)
	e.ffMu.Lock()
	if err != nil {
		e.ffPhase = "error"
		e.ffErr = err.Error()
	} else {
		e.ffPhase = "ready"
		e.ffErr = ""
		e.ffPercent = 100
	}
	e.ffMu.Unlock()
}

func (e *Engine) setFFmpegProgress(phase string, percent int, bytes, total int64, archive string) {
	e.ffMu.Lock()
	if phase != "" {
		e.ffPhase = phase
	}
	if percent >= 0 {
		e.ffPercent = percent
	}
	e.ffBytes = bytes
	if total > 0 {
		e.ffTotal = total
	}
	if archive != "" {
		e.ffArchive = archive
	}
	e.ffMu.Unlock()
}

func (e *Engine) installFFmpegBundle(ctx context.Context) error {
	goos, goarch := e.ffmpegPlatform()
	want, err := ffmpegArchiveName(goos, goarch)
	if err != nil {
		return err
	}
	e.setFFmpegProgress("resolving", 0, 0, 0, want)
	var sha string
	archive := want
	if files, listErr := e.listFFmpegRepo(ctx); listErr == nil {
		names := make([]string, 0, len(files))
		byBase := map[string]ffmpegRemoteFile{}
		for _, f := range files {
			names = append(names, f.Path)
			byBase[strings.ToLower(path.Base(f.Path))] = f
		}
		if picked, pickErr := pickFFmpegArchive(names, goos, goarch); pickErr == nil && picked != "" {
			archive = picked
		}
		if meta, ok := byBase[strings.ToLower(path.Base(archive))]; ok {
			// Tiny SHA values usually belong to Git LFS pointer blobs, not the zip.
			if meta.Size == 0 || meta.Size >= 1<<20 {
				sha = meta.Sha256
			}
			if meta.Size > 0 {
				e.setFFmpegProgress("resolving", 0, 0, meta.Size, archive)
			}
		}
	}
	e.setFFmpegProgress("downloading", 0, 0, 0, archive)
	binDir := e.ffmpegBinDir()
	if binDir == "" {
		return fmt.Errorf("ffmpeg: missing install directory")
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return err
	}
	zipPath := filepath.Join(binDir, "ffmpeg-download.zip")
	defer os.Remove(zipPath)
	if err := e.downloadFFmpegZip(ctx, archive, zipPath, sha); err != nil {
		return err
	}
	e.setFFmpegProgress("extracting", 92, 0, 0, archive)
	ffmpegPath, probePath, err := extractFFmpegZip(zipPath, binDir)
	if err != nil {
		return err
	}
	e.setFFmpegProgress("verifying", 96, 0, 0, archive)
	if err := ffmpegVerify(ffmpegPath); err != nil {
		return err
	}
	e.FFMPEG = ffmpegPath
	e.FFProbe = probePath
	e.refreshFFmpeg()
	if e.FFMPEG == "" {
		e.FFMPEG = ffmpegPath
	}
	if e.FFProbe == "" {
		e.FFProbe = probePath
	}
	return nil
}

func (e *Engine) downloadFFmpegZip(ctx context.Context, archive, dest, wantSHA string) error {
	endpoint, repo, rev := ffmpegRepoSpec()
	urls := []string{
		ffmpegResolveURL(endpoint, repo, rev, archive),
		ffmpegAPIFileURL(endpoint, repo, rev, archive),
	}
	var last error
	for _, rawURL := range urls {
		if rawURL == "" {
			continue
		}
		err := e.downloadFFmpegURL(ctx, rawURL, dest, wantSHA)
		if err == nil {
			return nil
		}
		last = err
	}
	if last == nil {
		last = fmt.Errorf("ffmpeg: download failed")
	}
	return last
}

func (e *Engine) downloadFFmpegURL(ctx context.Context, rawURL, dest, wantSHA string) error {
	res, err := e.ffmpegDo(ctx, rawURL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("ffmpeg: download %s", res.Status)
	}
	total := res.ContentLength
	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	ok := false
	closed := false
	defer func() {
		if !closed {
			f.Close()
		}
		if !ok {
			_ = os.Remove(tmp)
		}
	}()
	h := sha256.New()
	pr := &ffmpegProgressReader{r: res.Body, total: total, on: func(n, tot int64) {
		pct := 0
		if tot > 0 {
			pct = int(n * 90 / tot)
			if pct > 90 {
				pct = 90
			}
		}
		e.setFFmpegProgress("downloading", pct, n, tot, "")
	}}
	n, err := io.Copy(io.MultiWriter(f, h), io.LimitReader(pr, ffmpegMaxZip+1))
	if err != nil {
		return err
	}
	if n > ffmpegMaxZip {
		return fmt.Errorf("ffmpeg: archive larger than %d bytes", ffmpegMaxZip)
	}
	if err := f.Close(); err != nil {
		return err
	}
	closed = true
	if wantSHA != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, wantSHA) {
			return fmt.Errorf("ffmpeg: sha256 mismatch")
		}
	}
	if err := replaceFile(tmp, dest); err != nil {
		return err
	}
	ok = true
	return nil
}

type ffmpegProgressReader struct {
	r     io.Reader
	n     int64
	total int64
	last  int
	on    func(n, total int64)
}

func (p *ffmpegProgressReader) Read(b []byte) (int, error) {
	n, err := p.r.Read(b)
	p.n += int64(n)
	if p.on != nil {
		pct := 0
		if p.total > 0 {
			pct = int(p.n * 90 / p.total)
		}
		if pct != p.last || err != nil {
			p.last = pct
			p.on(p.n, p.total)
		}
	}
	return n, err
}

func zipEntryName(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}

func extractFFmpegZip(zipPath, destDir string) (ffmpegPath, probePath string, err error) {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return "", "", fmt.Errorf("ffmpeg: unzip: %w", err)
	}
	defer r.Close()
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", "", err
	}
	var ffName, probeName, ffDir string
	ffScore, probeScore := -1, -1
	for _, f := range r.File {
		if f.FileInfo().IsDir() || f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		name := zipEntryName(f.Name)
		kind, score := scoreZipBinary(name)
		switch kind {
		case "ffmpeg":
			if score > ffScore {
				ffScore = score
				ffName = name
				ffDir = path.Dir(name)
			}
		case "ffprobe":
			if score > probeScore {
				probeScore = score
				probeName = name
			}
		}
	}
	if ffName == "" {
		return "", "", fmt.Errorf("ffmpeg: archive has no ffmpeg binary")
	}
	for _, f := range r.File {
		if f.FileInfo().IsDir() || f.Mode()&os.ModeSymlink != 0 {
			continue
		}
		name := zipEntryName(f.Name)
		base := path.Base(name)
		if base == "" || base == "." || base == ".." {
			continue
		}
		kind, _ := scoreZipBinary(name)
		sameDir := path.Dir(name) == ffDir
		ext := strings.ToLower(path.Ext(base))
		if kind == "" && !(sameDir && ext == ".dll") {
			continue
		}
		if f.UncompressedSize64 > ffmpegMaxZip {
			return "", "", fmt.Errorf("ffmpeg: zip entry too large")
		}
		dest := filepath.Join(destDir, base)
		if err := writeZipFile(f, dest); err != nil {
			return "", "", err
		}
		if runtime.GOOS != "windows" {
			_ = os.Chmod(dest, 0o755)
		}
		switch kind {
		case "ffmpeg":
			if name == ffName {
				ffmpegPath = dest
			}
		case "ffprobe":
			if name == probeName {
				probePath = dest
			}
		}
	}
	if ffmpegPath == "" {
		ffmpegPath = lookExtractedBinary(destDir, "ffmpeg")
	}
	if probePath == "" {
		probePath = lookExtractedBinary(destDir, "ffprobe")
	}
	if ffmpegPath == "" {
		return "", "", fmt.Errorf("ffmpeg: failed to extract ffmpeg")
	}
	return ffmpegPath, probePath, nil
}

func lookExtractedBinary(dir, stem string) string {
	cands := []string{stem, stem + ".exe"}
	if runtime.GOOS == "windows" {
		cands = []string{stem + ".exe", stem}
	}
	for _, name := range cands {
		p := filepath.Join(dir, name)
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return ""
}

func scoreZipBinary(name string) (kind string, score int) {
	name = zipEntryName(name)
	base := strings.ToLower(path.Base(name))
	switch base {
	case "ffmpeg", "ffmpeg.exe":
		kind = "ffmpeg"
	case "ffprobe", "ffprobe.exe":
		kind = "ffprobe"
	default:
		return "", 0
	}
	score = 50 - strings.Count(path.Clean(name), "/")
	lower := strings.ToLower(name)
	if strings.Contains(lower, "/bin/") || strings.HasPrefix(lower, "bin/") {
		score += 10
	}
	return kind, score
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
	_, copyErr := io.Copy(out, io.LimitReader(rc, ffmpegMaxZip+1))
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if err := replaceFile(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func replaceFile(tmp, dest string) error {
	_ = os.Remove(dest)
	if err := os.Rename(tmp, dest); err == nil {
		return nil
	}
	in, err := os.Open(tmp)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(dest)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(dest)
		return closeErr
	}
	_ = os.Remove(tmp)
	return nil
}

func verifyFFmpegBin(bin string) error {
	if strings.TrimSpace(bin) == "" {
		return fmt.Errorf("ffmpeg: extracted binary missing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bin, "-hide_banner", "-version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("ffmpeg verify: %s", msg)
	}
	if !strings.Contains(strings.ToLower(string(out)), "ffmpeg") {
		return fmt.Errorf("ffmpeg verify: unexpected output")
	}
	return nil
}

func (e *Engine) ffmpegSource() string {
	if e == nil || e.FFMPEG == "" {
		return ""
	}
	binDir := e.ffmpegBinDir()
	if binDir == "" {
		return "system"
	}
	rel, err := filepath.Rel(binDir, e.FFMPEG)
	if err == nil && rel != "" && !strings.HasPrefix(rel, "..") {
		return "managed"
	}
	return "system"
}

func (e *Engine) FFmpegInstallSnapshot() map[string]any {
	goos, goarch := e.ffmpegPlatform()
	e.ffMu.Lock()
	phase := e.ffPhase
	percent := e.ffPercent
	bytes := e.ffBytes
	total := e.ffTotal
	archive := e.ffArchive
	errMsg := e.ffErr
	installing := e.ffWait != nil
	e.ffMu.Unlock()
	if e.FFMPEG != "" && !installing && (phase == "" || phase == "idle" || phase == "error") {
		phase = "ready"
		percent = 100
		errMsg = ""
	} else if phase == "" {
		phase = "idle"
	}
	return map[string]any{
		"ready":    e.FFMPEG != "",
		"phase":    phase,
		"percent":  percent,
		"bytes":    bytes,
		"total":    total,
		"archive":  archive,
		"error":    errMsg,
		"source":   e.ffmpegSource(),
		"platform": goos + "/" + goarch,
		"bin":      e.FFMPEG,
		"probe":    e.FFProbe,
	}
}
