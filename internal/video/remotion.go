package video

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func remotionCLIName() string {
	if runtime.GOOS == "windows" {
		return "remotion.cmd"
	}
	return "remotion"
}

func (e *Engine) remotionDir() string {
	if e == nil || strings.TrimSpace(e.Dir) == "" {
		return ""
	}
	return filepath.Join(e.Dir, "remotion")
}

func (e *Engine) nodePath() string {
	if e == nil {
		p, _ := exec.LookPath("node")
		return p
	}
	if e.rmNode == "-" {
		return ""
	}
	if e.rmNode != "" {
		return e.rmNode
	}
	p, _ := exec.LookPath("node")
	return p
}

func (e *Engine) npmPath() string {
	if e == nil {
		return lookNPM()
	}
	if e.rmNPM == "-" {
		return ""
	}
	if e.rmNPM != "" {
		return e.rmNPM
	}
	return lookNPM()
}

func lookNPM() string {
	if p, err := exec.LookPath("npm"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("npm.cmd"); err == nil {
			return p
		}
	}
	return ""
}

func (e *Engine) remotionBin() string {
	if e != nil && e.rmBin != "" {
		return e.rmBin
	}
	dir := ""
	if e != nil {
		dir = e.remotionDir()
	}
	if dir == "" {
		return ""
	}
	p := filepath.Join(dir, "node_modules", ".bin", remotionCLIName())
	if st, err := os.Stat(p); err == nil && !st.IsDir() {
		return p
	}
	if runtime.GOOS == "windows" {
		alt := filepath.Join(dir, "node_modules", ".bin", "remotion")
		if st, err := os.Stat(alt); err == nil && !st.IsDir() {
			return alt
		}
	}
	return ""
}

func (e *Engine) managedRemotionReady() bool {
	return e != nil && e.remotionBin() != ""
}

// RemotionStatus reports CLI detect without a video home. Prefer RemotionSnapshot
// on an Engine so managed installs under $YOYO_HOME/video/remotion are visible.
func RemotionStatus() map[string]any {
	return (&Engine{}).RemotionSnapshot()
}

func (e *Engine) RemotionSnapshot() map[string]any {
	node := ""
	npx := ""
	npm := ""
	if e != nil {
		node = e.nodePath()
		npm = e.npmPath()
	} else {
		node, _ = exec.LookPath("node")
		npm = lookNPM()
	}
	if p, err := exec.LookPath("npx"); err == nil {
		npx = p
	}
	out := map[string]any{
		"ready":    false,
		"detected": false,
		"node":     node,
		"npx":      npx,
		"npm":      npm,
		"hint":     "Remotion is optional (~540 MB). The default installer does not ship it. Install from Settings → Generation when you want title-card wrap. Needs Node.js 18+.",
		"bundled":  false,
		"source":   "",
		"dir":      "",
	}
	if e != nil {
		out["dir"] = e.remotionDir()
		out["install"] = e.RemotionInstallSnapshot()
	}
	if e.managedRemotionReady() {
		out["ready"] = true
		out["detected"] = true
		out["remotion"] = e.remotionBin()
		out["source"] = "managed"
		out["hint"] = "Remotion is ready. Shorts Assemble → Burn title uses it. FFmpeg wrap stays as fallback."
		return out
	}
	if p, err := exec.LookPath("remotion"); err == nil {
		out["detected"] = true
		out["remotion"] = p
		out["source"] = "system"
	} else if npx != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, npx, "--no-install", "remotion", "--version")
		b, err := cmd.CombinedOutput()
		if err == nil && strings.TrimSpace(string(b)) != "" {
			out["detected"] = true
			out["remotion"] = strings.TrimSpace(string(b))
			out["source"] = "npx"
		}
	}
	if node == "" {
		out["hint"] = "Install Node.js 18+ first, then install Remotion from Settings → Generation. Default talking-head wrap stays FFmpeg."
	} else if out["detected"] == true {
		out["hint"] = "A system Remotion CLI was found, but Yoyo still needs the local composition under the video home. Click Install so title wrap can run."
	}
	return out
}

func (e *Engine) RemotionInstallSnapshot() map[string]any {
	if e == nil {
		return map[string]any{"ready": false, "phase": "idle", "percent": 0, "error": ""}
	}
	e.rmMu.Lock()
	phase := e.rmPhase
	percent := e.rmPercent
	errMsg := e.rmErr
	installing := e.rmWait != nil
	e.rmMu.Unlock()
	ready := e.managedRemotionReady()
	if ready && !installing && (phase == "" || phase == "idle" || phase == "error") {
		phase = "ready"
		percent = 100
		errMsg = ""
	} else if phase == "" {
		phase = "idle"
	}
	return map[string]any{
		"ready":   ready,
		"phase":   phase,
		"percent": percent,
		"error":   errMsg,
		"dir":     e.remotionDir(),
	}
}

func (e *Engine) remotionCallResult() map[string]any {
	st := e.RemotionSnapshot()
	st["ok"] = true
	return st
}

func (e *Engine) InstallRemotion() map[string]any {
	if e.managedRemotionReady() {
		return e.remotionCallResult()
	}
	if e.nodePath() == "" {
		e.setRemotionProgress("error", 0, "Install Node.js 18+ first, then try again.")
		return e.remotionCallResult()
	}
	if e.npmPath() == "" {
		e.setRemotionProgress("error", 0, "npm was not found. Install Node.js with npm, then try again.")
		return e.remotionCallResult()
	}
	_, _ = e.beginRemotionInstall()
	return e.remotionCallResult()
}

func (e *Engine) beginRemotionInstall() (<-chan struct{}, error) {
	e.rmMu.Lock()
	defer e.rmMu.Unlock()
	if e.managedRemotionReady() {
		return nil, nil
	}
	if e.rmWait != nil {
		return e.rmWait, nil
	}
	ch := make(chan struct{})
	e.rmWait = ch
	e.rmPhase = "preparing"
	e.rmPercent = 5
	e.rmErr = ""
	go e.runRemotionInstall(ch)
	return ch, nil
}

func (e *Engine) runRemotionInstall(done chan struct{}) {
	defer func() {
		e.rmMu.Lock()
		if e.rmWait == done {
			e.rmWait = nil
		}
		e.rmMu.Unlock()
		close(done)
		e.hub("remotion", e.RemotionInstallSnapshot())
	}()
	err := e.installRemotionBundle()
	e.rmMu.Lock()
	if err != nil {
		e.rmPhase = "error"
		e.rmErr = err.Error()
	} else {
		e.rmPhase = "ready"
		e.rmErr = ""
		e.rmPercent = 100
	}
	e.rmMu.Unlock()
}

func (e *Engine) setRemotionProgress(phase string, percent int, errMsg string) {
	if e == nil {
		return
	}
	e.rmMu.Lock()
	if phase != "" {
		e.rmPhase = phase
	}
	if percent >= 0 {
		e.rmPercent = percent
	}
	if errMsg != "" || phase == "error" {
		e.rmErr = errMsg
	}
	e.rmMu.Unlock()
	e.hub("remotion", e.RemotionInstallSnapshot())
}

func (e *Engine) installRemotionBundle() error {
	e.setRemotionProgress("preparing", 8, "")
	if err := e.writeRemotionProject(); err != nil {
		return err
	}
	e.setRemotionProgress("installing", 15, "")
	if err := e.npmInstallRemotion(); err != nil {
		return err
	}
	e.setRemotionProgress("verifying", 95, "")
	if !e.managedRemotionReady() {
		return fmt.Errorf("remotion CLI missing after npm install")
	}
	return nil
}

func (e *Engine) npmInstallRemotion() error {
	dir := e.remotionDir()
	npm := e.npmPath()
	if dir == "" || npm == "" {
		return fmt.Errorf("npm required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, npm, "install", "--omit=dev", "--no-fund", "--no-audit")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "npm_config_fund=false", "ADBLOCK=1", "npm_config_progress=false")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	errCh := make(chan error, 1)
	go func() { errCh <- cmd.Run() }()
	pct := 20
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case err := <-errCh:
			if err != nil {
				return fmt.Errorf("npm install: %v: %s", err, tailRunes(buf.String(), 800))
			}
			return nil
		case <-tick.C:
			if pct < 90 {
				pct += 4
				e.setRemotionProgress("installing", pct, "")
			}
		}
	}
}

func tailRunes(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[len(r)-n:])
}

func remotionChrome() string {
	if v := strings.TrimSpace(os.Getenv("YOYO_CHROME")); v != "" {
		return v
	}
	var cands []string
	switch runtime.GOOS {
	case "windows":
		cands = []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
		}
	case "darwin":
		cands = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default:
		cands = []string{"google-chrome", "chromium", "chromium-browser", "microsoft-edge", "google-chrome-stable"}
	}
	for _, c := range cands {
		if c == "" {
			continue
		}
		if filepath.IsAbs(c) {
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				return c
			}
			continue
		}
		if p, err := exec.LookPath(c); err == nil {
			return p
		}
	}
	return ""
}

func (e *Engine) probeVideo(path string) (w, h int, seconds float64) {
	w, h = 1080, 1920
	seconds = 5
	if e == nil {
		return
	}
	if e.FFProbe == "" {
		e.refreshFFmpeg()
	}
	if e.FFProbe == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.FFProbe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height:format=duration", "-of", "json", path)
	b, err := cmd.Output()
	if err != nil || len(b) == 0 {
		return
	}
	var info struct {
		Streams []struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
		} `json:"format"`
	}
	if json.Unmarshal(b, &info) != nil {
		return
	}
	if len(info.Streams) > 0 && info.Streams[0].Width > 0 && info.Streams[0].Height > 0 {
		w, h = even(info.Streams[0].Width), even(info.Streams[0].Height)
	}
	if d, err := strconv.ParseFloat(strings.TrimSpace(info.Format.Duration), 64); err == nil && d > 0 {
		seconds = d
	}
	return
}

// RenderTitleCard runs the managed Remotion TitleCard composition onto a CAS video.
func (e *Engine) RenderTitleCard(hash, title, aspect string) (MediaRef, error) {
	if !e.managedRemotionReady() {
		return MediaRef{}, fmt.Errorf("remotion is not installed")
	}
	hash = strings.TrimSpace(hash)
	title = strings.TrimSpace(title)
	if hash == "" || title == "" {
		return MediaRef{}, fmt.Errorf("hash and title required")
	}
	raw, err := e.GetBytes(hash)
	if err != nil {
		return MediaRef{}, err
	}
	dir := e.remotionDir()
	pub := filepath.Join(dir, "public")
	outDir := filepath.Join(dir, "out")
	if err := os.MkdirAll(pub, 0o755); err != nil {
		return MediaRef{}, err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return MediaRef{}, err
	}
	inName := hash + ".mp4"
	inPath := filepath.Join(pub, inName)
	if err := os.WriteFile(inPath, raw, 0o644); err != nil {
		return MediaRef{}, err
	}
	w, h, seconds := e.probeVideo(inPath)
	if a := normalizeAspect(aspect); aspect != "" {
		w, h = aspectSize(a)
	}
	frames := int(math.Ceil(seconds * 30))
	if frames < 30 {
		frames = 30
	}
	props := map[string]any{
		"src":              inName,
		"title":            title,
		"durationInFrames": frames,
		"width":            w,
		"height":           h,
	}
	pb, err := json.Marshal(props)
	if err != nil {
		return MediaRef{}, err
	}
	propsPath := filepath.Join(dir, "props.json")
	if err := os.WriteFile(propsPath, pb, 0o644); err != nil {
		return MediaRef{}, err
	}
	outPath := filepath.Join(outDir, hash+"-title.mp4")
	args := []string{
		"render", "src/index.ts", "TitleCard", outPath,
		"--props=" + propsPath,
		"--overwrite",
		"--timeout=120000",
	}
	if chrome := remotionChrome(); chrome != "" {
		args = append(args, "--browser-executable="+chrome)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, e.remotionBin(), args...)
	cmd.Dir = dir
	b, err := cmd.CombinedOutput()
	if err != nil {
		return MediaRef{}, fmt.Errorf("remotion render: %v: %s", err, tailRunes(string(b), 1200))
	}
	out, err := os.ReadFile(outPath)
	if err != nil {
		return MediaRef{}, err
	}
	cas, err := e.PutBytes(out)
	if err != nil {
		return MediaRef{}, err
	}
	return MediaRef{
		Kind:       "video",
		CASHash:    cas,
		MIME:       "video/mp4",
		Width:      w,
		Height:     h,
		DurationMs: int(seconds * 1000),
		Source:     "studio-wrap-remotion",
	}, nil
}
