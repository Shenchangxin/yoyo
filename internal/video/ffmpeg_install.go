// ffmpeg_install.go 从 ModelScope 数据集下载 ffmpeg 压缩包，解压到视频目录的 bin，并核对 -version。

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

// 下载压缩包时使用这些上限和请求头。
const (
	// ffmpegMaxZip 是压缩包和解压条目的上限，256 MiB。
	ffmpegMaxZip    = 256 << 20
	// ffmpegUserAgent 是下载请求的 User-Agent。
	ffmpegUserAgent = "Mozilla/5.0 (compatible; Yoyo-Workstation)"
)

// 默认仓库是 modelscope.cn 上的 NanoModel/ffmpeg，修订为 master。
var (
	// ffmpegEndpoint 是默认站点。
	ffmpegEndpoint = "https://www.modelscope.cn"
	// ffmpegRepo 是默认数据集。
	ffmpegRepo     = "NanoModel/ffmpeg"
	// ffmpegRevision 是默认修订。
	ffmpegRevision = "master"
	// ffmpegHTTP 是带 cookie 的客户端，超时 25 分钟。
	ffmpegHTTP     = newFFmpegHTTP()
	// ffmpegVerify 指向校验解压结果的函数，测试可以换掉。
	ffmpegVerify   = verifyFFmpegBin
)

// newFFmpegHTTP 返回带 cookie jar、超时 25 分钟的客户端。
func newFFmpegHTTP() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 25 * time.Minute, Jar: jar}
}

// ffmpegRemoteFile 是仓库清单里的一个文件，含路径、SHA256 和大小。
type ffmpegRemoteFile struct {
	Path   string
	Sha256 string
	Size   int64
}

// ffmpegBinDir 返回视频目录下的 bin。目录为空时返回空。
func (e *Engine) ffmpegBinDir() string {
	if e == nil || e.Dir == "" {
		return ""
	}
	return filepath.Join(e.Dir, "bin")
}

// ffmpegPlatform 返回系统名和架构。引擎上若指定了 ffmpegOS 或 ffmpegArch 就用指定值。
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

// ffmpegClient 返回引擎上的客户端，没有时用默认的 ffmpegHTTP。
func (e *Engine) ffmpegClient() HTTPDoer {
	if e != nil && e.ffmpegHTTP != nil {
		return e.ffmpegHTTP
	}
	return ffmpegHTTP
}

// refreshFFmpeg 在 bin 目录里重新查找 ffmpeg 和 ffprobe。
func (e *Engine) refreshFFmpeg() {
	if e == nil {
		return
	}
	e.FFMPEG, e.FFProbe = LookFFmpegIn([]string{e.ffmpegBinDir()})
}

// ffmpegRepoSpec 返回站点、仓库和修订。YOYO_FFMPEG_ENDPOINT、YOYO_FFMPEG_REPO、YOYO_FFMPEG_REVISION 非空时覆盖默认值。
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

// ffmpegArchiveName 把系统收成 windows、linux、macos，把架构收成 x86_64、aarch64，拼出 ffmpeg-<系统>-<架构>.zip。其他系统或架构返回错误。
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

// matchFFmpegArchive 要求名字含 ffmpeg、以 .zip 结尾，并匹配系统和架构。amd64 还认 x86_64、amd64 和单独的 x64。
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

// hasX64 在名字含 x64 且不含 x86_64 时为真。
func hasX64(s string) bool {
	return strings.Contains(s, "x64") && !strings.Contains(s, "x86_64")
}

// pickFFmpegArchive 先找与标准名完全相同的文件。没有时在匹配项里取文件名最短的。一个匹配都没有时仍返回标准名。
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

// ffmpegResolveURL 拼出 /datasets/<仓库>/resolve/<修订>/<文件>。
func ffmpegResolveURL(endpoint, repo, rev, file string) string {
	file = strings.TrimPrefix(strings.ReplaceAll(file, "\\", "/"), "/")
	return strings.TrimRight(endpoint, "/") + "/datasets/" + repo + "/resolve/" + rev + "/" + file
}

// ffmpegAPIFileURL 拼出 /api/v1/datasets/<仓库>/repo，查询参数是 Revision 和 FilePath。
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

// ffmpegTreeURL 拼出仓库树接口，Recursive 为 true。
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

// parseFFmpegRepoTree 解析清单。跳过空路径和 Type 为 tree 的项。Path 空时用 Name。
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

// listFFmpegRepo 用 GET 拉仓库树，正文最多读 2 MiB。状态码大于等于 300 或清单为空时失败。
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

// ffmpegDo 用 ffmpeg 的 User-Agent、Accept 和 Referer 发起 GET。
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

// EnsureFFmpeg 已可用时直接返回。否则等待安装。上下文取消时返回该错误。安装结束仍不可用时返回 ffErr，空则是 ffmpeg install failed。
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

// InstallFFmpeg 在还不可用时开始安装，并立刻返回当前结果。
func (e *Engine) InstallFFmpeg() map[string]any {
	e.refreshFFmpeg()
	if !e.FFmpegOK() {
		_, _ = e.beginFFmpegInstall()
	}
	return e.ffmpegCallResult()
}

// ffmpegCallResult 返回 ok、是否可用、两条命令路径和安装快照。
func (e *Engine) ffmpegCallResult() map[string]any {
	return map[string]any{
		"ok":             true,
		"ffmpeg":         e.FFmpegOK(),
		"ffmpeg_bin":     e.FFMPEG,
		"ffprobe_bin":    e.FFProbe,
		"ffmpeg_install": e.FFmpegInstallSnapshot(),
	}
}

// beginFFmpegInstall 先检查系统是否有对应压缩包名。已经有 ffmpeg 时返回空通道。正在安装时返回原通道。否则把阶段设为 resolving 并在后台安装。
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

// runFFmpegInstall 用 25 分钟超时安装。成功时阶段为 ready、100，失败时为 error。结束后送出 ffmpeg 快照。
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

// setFFmpegProgress 更新阶段、百分比、已下载字节、总字节和压缩包名。百分比小于 0 时不改百分比。总字节和压缩包名只在非空时覆盖。
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

// installFFmpegBundle 列出仓库并挑选压缩包。体积为 0 或不少于 1 MiB 时才带上清单里的 SHA256。下载后解压到 bin，校验 ffmpeg -version，再刷新路径。
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
			// 体积为 0 或不少于 1 MiB 时才采用清单里的 SHA256。更小的非空文件通常是 Git LFS 指针，不是压缩包。
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

// downloadFFmpegZip 先试 resolve 地址，再试 API 文件地址。都失败时返回最后一次错误。
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

// downloadFFmpegURL 把响应写到临时文件，同时计算 SHA256。超过 256 MiB 失败。wantSHA 非空时忽略大小写比较，不符返回 sha256 mismatch。进度百分比不超过 90。状态码大于等于 300 失败。
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

// ffmpegProgressReader 在读取时按已读字节回调进度。
type ffmpegProgressReader struct {
	r     io.Reader
	n     int64
	total int64
	last  int
	on    func(n, total int64)
}

// Read 累计已读字节。百分比变化或读到错误时回调。
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

// zipEntryName 把压缩包条目名里的反斜杠换成斜杠。
func zipEntryName(name string) string {
	return strings.ReplaceAll(name, "\\", "/")
}

// extractFFmpegZip 跳过目录和符号链接，按分数选出 ffmpeg 和 ffprobe，并解压它们以及同目录的 dll。单条目超过 256 MiB 失败。非 Windows 上把解压出的文件设为 0755。
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

// lookExtractedBinary 在目录里找 stem 或 stem.exe。Windows 先找 exe。
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

// scoreZipBinary 只认 ffmpeg、ffmpeg.exe、ffprobe、ffprobe.exe。基础分是 50 减去路径层数，路径含 /bin/ 再加 10。
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

// writeZipFile 把条目写到临时文件再替换目标，读取上限是 256 MiB 加 1 字节。
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

// replaceFile 先删掉目标再改名。改名失败时复制过去，并删掉临时文件。
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

// verifyFFmpegBin 用 -hide_banner -version 跑 20 秒。输出里不含 ffmpeg 时失败。
func verifyFFmpegBin(bin string) error {
	if strings.TrimSpace(bin) == "" {
		return fmt.Errorf("ffmpeg: extracted binary missing")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := quiet(exec.CommandContext(ctx, bin, "-hide_banner", "-version"))
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

// ffmpegSource 在命令位于 bin 目录内时返回 managed，否则返回 system。还没有命令时返回空。
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

// FFmpegInstallSnapshot 返回阶段、进度、字节数、压缩包、错误、来源和平台。已经有命令且不在安装中时，空闲或错误阶段改报 ready、100。
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
