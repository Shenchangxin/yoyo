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
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Shenchangxin/yoyo/internal/diaglog"
	"github.com/Shenchangxin/yoyo/internal/home"
)

const (
	characterCastPublicPrefix = "/character-cast/"
	characterCastPackID       = "character-cast"
	// characterCastPackURL is the ModelScope download for the character still
	// pack. YOYO_CHARACTER_CAST_URL overrides it.
	characterCastPackURL = "https://modelscope.ai/datasets/yoyo-agent/yoyo/resolve/master/sourcetime/character-cast.zip?view=false"
	// characterCastPackSHA is the hex sha256 of that exact zip. Empty skips the
	// archive check; each JPEG is still checked against manifest.json.
	characterCastPackSHA = "bb64f1f01fe8e45f2295014b363dae733571107957c81c8ec140aae7465cfd5d"

	envCharacterCastURL     = "YOYO_CHARACTER_CAST_URL"
	envCharacterCastFile    = "YOYO_CHARACTER_CAST_FILE"
	envCharacterCastSHA     = "YOYO_CHARACTER_CAST_SHA256"
	envCharacterCastDisable = "YOYO_CHARACTER_CAST_DISABLE"

	castInstallMetaName = ".install.json"
	maxCastZip          = 64 << 20
	maxCastFile         = 8 << 20
	maxCastTotal        = 80 << 20
	maxCastManifest     = 1 << 20
	maxCastAssets       = 200
)

var (
	castIDRe   = regexp.MustCompile(`^studio-cast-[a-z0-9]+(?:-[a-z0-9]+)*$`)
	castFileRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*\.jpg$`)
	castCatRe  = regexp.MustCompile(`^[a-z][a-z0-9-]{0,32}$`)
	castFoldRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	castSHARe  = regexp.MustCompile(`^[0-9a-f]{64}$`)

	characterCastHTTPClient = newCharacterCastHTTP()
)

func newCharacterCastHTTP() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 20 * time.Minute, Jar: jar}
}

type characterCastFolder struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Position int    `json:"position"`
}

type characterCastAsset struct {
	ID       string   `json:"id"`
	File     string   `json:"file"`
	Title    string   `json:"title"`
	Note     string   `json:"note"`
	Tags     []string `json:"tags"`
	Category string   `json:"category"`
	FolderID string   `json:"folderId"`
	Kind     string   `json:"kind"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Bytes    int      `json:"bytes"`
	MIME     string   `json:"mimeType"`
	SHA256   string   `json:"sha256"`
}

type characterCastManifest struct {
	ID           string               `json:"id"`
	Version      int                  `json:"version"`
	PublicPrefix string               `json:"publicPrefix"`
	Source       string               `json:"source"`
	Folder       characterCastFolder  `json:"folder"`
	Assets       []characterCastAsset `json:"assets"`
}

type characterCastMeta struct {
	Source  string `json:"source"`
	SHA256  string `json:"sha256"`
	Version int    `json:"version"`
}

func (m *characterCastManifest) validate() error {
	if m.Version != 1 {
		return fmt.Errorf("character cast: unsupported manifest version %d", m.Version)
	}
	if m.ID == "" {
		m.ID = characterCastPackID
	}
	if m.ID != characterCastPackID {
		return fmt.Errorf("character cast: unexpected pack id %s", m.ID)
	}
	if m.PublicPrefix == "" {
		m.PublicPrefix = characterCastPublicPrefix
	}
	if m.PublicPrefix != characterCastPublicPrefix {
		return fmt.Errorf("character cast: unexpected public prefix")
	}
	if m.Source == "" {
		m.Source = "Yoyo 工作室"
	}
	if utf8.RuneCountInString(m.Source) > 40 {
		return fmt.Errorf("character cast: source too long")
	}
	if m.Folder.ID == "" {
		m.Folder.ID = "studio-folder-characters"
	}
	if !castFoldRe.MatchString(m.Folder.ID) {
		return fmt.Errorf("character cast: bad folder id")
	}
	if m.Folder.Name == "" {
		m.Folder.Name = "角色资产"
	}
	if utf8.RuneCountInString(m.Folder.Name) > 40 || m.Folder.Position < 0 || m.Folder.Position > 100 {
		return fmt.Errorf("character cast: bad folder")
	}
	if len(m.Assets) == 0 || len(m.Assets) > maxCastAssets {
		return fmt.Errorf("character cast: asset count %d", len(m.Assets))
	}
	seenID := map[string]bool{}
	seenFile := map[string]bool{}
	for i := range m.Assets {
		a := &m.Assets[i]
		a.ID = strings.TrimSpace(a.ID)
		a.File = strings.TrimSpace(a.File)
		a.Title = strings.TrimSpace(a.Title)
		a.Note = strings.TrimSpace(a.Note)
		a.Category = strings.TrimSpace(a.Category)
		a.FolderID = strings.TrimSpace(a.FolderID)
		a.Kind = strings.TrimSpace(a.Kind)
		a.MIME = strings.TrimSpace(a.MIME)
		a.SHA256 = strings.ToLower(strings.TrimSpace(a.SHA256))
		if !castIDRe.MatchString(a.ID) || seenID[a.ID] {
			return fmt.Errorf("character cast: bad asset id %s", a.ID)
		}
		seenID[a.ID] = true
		if !castFileRe.MatchString(a.File) || seenFile[a.File] {
			return fmt.Errorf("character cast: bad file %s", a.File)
		}
		seenFile[a.File] = true
		if a.Title == "" || utf8.RuneCountInString(a.Title) > 80 {
			return fmt.Errorf("character cast: bad title for %s", a.ID)
		}
		if utf8.RuneCountInString(a.Note) > 400 {
			return fmt.Errorf("character cast: note too long for %s", a.ID)
		}
		if len(a.Tags) > 16 {
			return fmt.Errorf("character cast: too many tags for %s", a.ID)
		}
		for _, tag := range a.Tags {
			if strings.TrimSpace(tag) == "" || utf8.RuneCountInString(tag) > 24 {
				return fmt.Errorf("character cast: bad tag for %s", a.ID)
			}
		}
		if a.Category == "" {
			a.Category = "character"
		}
		if !castCatRe.MatchString(a.Category) {
			return fmt.Errorf("character cast: bad category for %s", a.ID)
		}
		if a.FolderID == "" {
			a.FolderID = m.Folder.ID
		}
		if !castFoldRe.MatchString(a.FolderID) {
			return fmt.Errorf("character cast: bad folder for %s", a.ID)
		}
		if a.Kind == "" {
			a.Kind = "image"
		}
		if a.Kind != "image" {
			return fmt.Errorf("character cast: %s is not an image", a.ID)
		}
		if a.MIME == "" {
			a.MIME = "image/jpeg"
		}
		if a.MIME != "image/jpeg" {
			return fmt.Errorf("character cast: bad mime for %s", a.ID)
		}
		if a.Width < 1 || a.Height < 1 || a.Width > 8192 || a.Height > 8192 {
			return fmt.Errorf("character cast: bad dimensions for %s", a.ID)
		}
		if a.Bytes < 1 || a.Bytes > maxCastFile {
			return fmt.Errorf("character cast: bad size for %s", a.ID)
		}
		if !castSHARe.MatchString(a.SHA256) {
			return fmt.Errorf("character cast: bad sha256 for %s", a.ID)
		}
	}
	return nil
}

// CharacterCastRoot is the extracted pack directory for a Yoyo home.
func CharacterCastRoot(homeRoot string) string {
	root := strings.TrimSpace(homeRoot)
	if root == "" {
		d, err := home.Default()
		if err != nil {
			return ""
		}
		root = d
	}
	return filepath.Join(root, "video", "packs", "character-cast")
}

func (e *Engine) CharacterCastDir() string {
	if e == nil || strings.TrimSpace(e.Dir) == "" {
		return ""
	}
	return filepath.Join(e.Dir, "packs", "character-cast")
}

func (e *Engine) pinnedCastSHA() string {
	if e != nil && e.castSHA != nil {
		return strings.ToLower(strings.TrimSpace(*e.castSHA))
	}
	if v := strings.TrimSpace(os.Getenv(envCharacterCastSHA)); v != "" {
		return strings.ToLower(v)
	}
	return strings.ToLower(strings.TrimSpace(characterCastPackSHA))
}

func (e *Engine) characterCastClient() HTTPDoer {
	if e != nil && e.castHTTP != nil {
		return e.castHTTP
	}
	return characterCastHTTPClient
}

func (e *Engine) characterCastSource() (kind, value string) {
	if os.Getenv(envCharacterCastDisable) == "1" {
		return "", ""
	}
	file := ""
	if e != nil {
		file = strings.TrimSpace(e.castFile)
	}
	if file == "" {
		file = strings.TrimSpace(os.Getenv(envCharacterCastFile))
	}
	if file != "" {
		if abs, err := filepath.Abs(file); err == nil {
			file = abs
		}
		return "file", file
	}
	raw := ""
	if e != nil {
		raw = strings.TrimSpace(e.castURL)
	}
	if raw == "" {
		raw = strings.TrimSpace(os.Getenv(envCharacterCastURL))
	}
	if raw == "" && !goTestBinary() {
		raw = strings.TrimSpace(characterCastPackURL)
	}
	if raw == "" {
		return "", ""
	}
	return "url", raw
}

// StartCharacterCast downloads and imports the still pack when a URL or local
// zip is configured. It returns immediately; Open stays offline.
func (e *Engine) StartCharacterCast() {
	if e == nil {
		return
	}
	kind, value := e.characterCastSource()
	if kind == "" {
		return
	}
	ctx := e.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	go e.finishCharacterCast(ctx, kind, value)
}

func (e *Engine) finishCharacterCast(ctx context.Context, kind, value string) {
	if kind == "" || value == "" || ctx.Err() != nil {
		return
	}
	if e.characterCastCurrent(value) {
		return
	}
	unlock, ok := e.tryCastLock()
	if !ok {
		e.waitAndImportCharacterCast(ctx)
		return
	}
	defer unlock()
	if e.characterCastCurrent(value) || ctx.Err() != nil {
		return
	}
	var err error
	switch kind {
	case "file":
		err = e.installCharacterCastFile(value)
	case "url":
		err = e.downloadAndInstallCharacterCast(ctx, value)
	default:
		return
	}
	if err != nil {
		diaglog.Warn("character cast pack", "component", "video", "err", err)
		return
	}
	if err := e.importCharacterCastNow(); err != nil {
		diaglog.Warn("character cast import", "component", "video", "err", err)
		return
	}
	diaglog.Info("character cast pack ready", "component", "video")
}

func (e *Engine) waitAndImportCharacterCast(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	timer := time.NewTimer(15 * time.Minute)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			return
		case <-tick.C:
			if e.characterCastFilesOK() {
				if err := e.importCharacterCastNow(); err != nil {
					diaglog.Warn("character cast import", "component", "video", "err", err)
				}
				return
			}
		}
	}
}

func (e *Engine) tryCastLock() (func(), bool) {
	if e == nil || e.Dir == "" {
		return nil, false
	}
	dir := filepath.Join(e.Dir, "packs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, false
	}
	lockPath := filepath.Join(dir, "character-cast.lock")
	if st, err := os.Stat(lockPath); err == nil && time.Since(st.ModTime()) >= 15*time.Minute {
		_ = os.Remove(lockPath)
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, false
	}
	_, _ = f.WriteString(time.Now().UTC().Format(time.RFC3339Nano))
	_ = f.Close()
	return func() { _ = os.Remove(lockPath) }, true
}

func (e *Engine) characterCastCurrent(source string) bool {
	if !e.characterCastFilesOK() {
		return false
	}
	meta, err := readCastMeta(e.CharacterCastDir())
	if err != nil || meta.Source != source {
		return false
	}
	if want := e.pinnedCastSHA(); want != "" && !strings.EqualFold(meta.SHA256, want) {
		return false
	}
	return true
}

func (e *Engine) characterCastFilesOK() bool {
	dir := e.CharacterCastDir()
	if dir == "" {
		return false
	}
	man, err := readCharacterCastManifest(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return false
	}
	for _, a := range man.Assets {
		p := filepath.Join(dir, a.File)
		st, err := os.Lstat(p)
		if err != nil || !st.Mode().IsRegular() || (a.Bytes > 0 && st.Size() != int64(a.Bytes)) {
			return false
		}
		sum, err := fileSHA256(p)
		if err != nil || !strings.EqualFold(sum, a.SHA256) {
			return false
		}
	}
	return true
}

func (e *Engine) importCharacterCastNow() error {
	existing := map[string]bool{}
	for _, a := range loadCol[canvasAssetRec](e, colAssets) {
		if a.DeletedAt == "" {
			existing[a.ID] = true
		}
	}
	return e.importCharacterCast(Now(), existing)
}

func (e *Engine) importCharacterCast(now string, existing map[string]bool) error {
	dir := e.CharacterCastDir()
	if dir == "" {
		return nil
	}
	manPath := filepath.Join(dir, "manifest.json")
	if _, err := os.Stat(manPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	man, err := readCharacterCastManifest(manPath)
	if err != nil {
		return err
	}
	if err := e.ensureCastFolder(man, now); err != nil {
		return err
	}
	if existing == nil {
		existing = map[string]bool{}
	}
	for _, a := range man.Assets {
		if existing[a.ID] {
			continue
		}
		p := filepath.Join(dir, a.File)
		st, err := os.Lstat(p)
		if err != nil || !st.Mode().IsRegular() {
			return fmt.Errorf("character cast: missing %s", a.File)
		}
		if int64(a.Bytes) != st.Size() {
			return fmt.Errorf("character cast: size mismatch for %s", a.File)
		}
		src := characterCastPublicPrefix + a.File
		tags := a.Tags
		if tags == nil {
			tags = []string{}
		}
		e.upsertAsset(map[string]any{
			"id": a.ID, "kind": "image", "title": a.Title, "coverUrl": src, "tags": tags,
			"folderId": a.FolderID, "category": a.Category, "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": man.Source, "note": a.Note,
			"data": map[string]any{
				"dataUrl": src, "width": a.Width, "height": a.Height,
				"bytes": a.Bytes, "mimeType": a.MIME,
			},
		})
		existing[a.ID] = true
	}
	return nil
}

func (e *Engine) ensureCastFolder(man characterCastManifest, now string) error {
	if folder, err := getDoc[canvasFolderRec](e, colFolders, man.Folder.ID); err == nil && folder.DeletedAt == "" {
		return nil
	}
	rec := canvasFolderRec{
		ID: man.Folder.ID, Name: man.Folder.Name, Position: man.Folder.Position,
		CreatedAt: now, UpdatedAt: now,
	}
	return e.putDoc(colFolders, man.Folder.ID, rec)
}

func readCharacterCastManifest(path string) (characterCastManifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return characterCastManifest{}, err
	}
	if len(b) > maxCastManifest {
		return characterCastManifest{}, fmt.Errorf("character cast: manifest too large")
	}
	var man characterCastManifest
	if err := json.Unmarshal(b, &man); err != nil {
		return characterCastManifest{}, fmt.Errorf("character cast: manifest: %w", err)
	}
	if err := man.validate(); err != nil {
		return characterCastManifest{}, err
	}
	return man, nil
}

func readCastMeta(dir string) (characterCastMeta, error) {
	b, err := os.ReadFile(filepath.Join(dir, castInstallMetaName))
	if err != nil {
		return characterCastMeta{}, err
	}
	var meta characterCastMeta
	if err := json.Unmarshal(b, &meta); err != nil {
		return characterCastMeta{}, err
	}
	meta.Source = strings.TrimSpace(meta.Source)
	meta.SHA256 = strings.ToLower(strings.TrimSpace(meta.SHA256))
	if meta.Source == "" {
		return characterCastMeta{}, fmt.Errorf("character cast: install meta missing source")
	}
	return meta, nil
}

func (e *Engine) installCharacterCastFile(src string) error {
	st, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() || st.Size() < 22 || st.Size() > maxCastZip {
		return fmt.Errorf("character cast: bad archive")
	}
	if err := assertZipMagic(src); err != nil {
		return err
	}
	sum, err := fileSHA256(src)
	if err != nil {
		return err
	}
	if want := e.pinnedCastSHA(); want != "" && !strings.EqualFold(sum, want) {
		return fmt.Errorf("character cast: sha256 mismatch")
	}
	return extractCharacterCastZip(src, e.CharacterCastDir(), characterCastMeta{Source: src, SHA256: sum})
}

func (e *Engine) downloadAndInstallCharacterCast(ctx context.Context, rawURL string) error {
	urls := characterCastDownloadURLs(rawURL)
	if len(urls) == 0 {
		return fmt.Errorf("character cast: empty url")
	}
	if e == nil || e.Dir == "" {
		return fmt.Errorf("character cast: missing install directory")
	}
	dir := filepath.Join(e.Dir, "packs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "character-cast-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	_ = tmp.Close()
	defer os.Remove(tmpName)
	var last error
	for _, u := range urls {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		sum, err := e.downloadCastFile(ctx, u, tmpName)
		if err != nil {
			last = err
			continue
		}
		if want := e.pinnedCastSHA(); want != "" && !strings.EqualFold(sum, want) {
			last = fmt.Errorf("character cast: sha256 mismatch")
			continue
		}
		if err := extractCharacterCastZip(tmpName, e.CharacterCastDir(), characterCastMeta{Source: rawURL, SHA256: sum}); err != nil {
			last = err
			continue
		}
		return nil
	}
	if last == nil {
		last = fmt.Errorf("character cast: download failed")
	}
	return last
}

func (e *Engine) downloadCastFile(ctx context.Context, rawURL, dest string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Yoyo-Workstation)")
	req.Header.Set("Accept", "application/octet-stream, application/zip, */*")
	if ref := modelscopeReferer(rawURL); ref != "" {
		req.Header.Set("Referer", ref)
	}
	res, err := e.characterCastClient().Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("character cast: download %s", res.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(res.Body, maxCastZip+1))
	closeErr := f.Close()
	if copyErr != nil {
		return "", copyErr
	}
	if closeErr != nil {
		return "", closeErr
	}
	if n > maxCastZip {
		return "", fmt.Errorf("character cast: archive too large")
	}
	if err := assertZipMagic(dest); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func goTestBinary() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}

func normalizeModelScopeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return raw
	}
	if strings.EqualFold(u.Host, "modelscope.ai") {
		u.Scheme = "https"
		u.Host = "www.modelscope.ai"
		return u.String()
	}
	return raw
}

func characterCastDownloadURLs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	raw = normalizeModelScopeURL(raw)
	first := raw
	if u, err := url.Parse(raw); err == nil && strings.Contains(strings.ToLower(u.Host), "modelscope.") && strings.Contains(u.Path, "/resolve/") && u.RawQuery == "" {
		u.RawQuery = "view=false"
		first = u.String()
	}
	out := []string{first}
	if api := characterCastAPIURL(raw); api != "" && api != first {
		out = append(out, api)
	}
	return out
}

func characterCastAPIURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.Contains(strings.ToLower(u.Host), "modelscope.") {
		return ""
	}
	const marker = "/resolve/"
	i := strings.Index(u.Path, marker)
	if i < 0 {
		return ""
	}
	head := u.Path[:i]
	rest := strings.TrimPrefix(u.Path[i+len(marker):], "/")
	rev, file, ok := strings.Cut(rest, "/")
	if !ok || rev == "" || file == "" {
		return ""
	}
	const ds = "/datasets/"
	j := strings.LastIndex(head, ds)
	if j < 0 {
		return ""
	}
	repo := strings.Trim(head[j+len(ds):], "/")
	if strings.Count(repo, "/") != 1 || strings.Contains(repo, "..") {
		return ""
	}
	out, err := url.Parse(u.Scheme + "://" + u.Host + "/api/v1/datasets/" + repo + "/repo")
	if err != nil {
		return ""
	}
	q := out.Query()
	q.Set("Revision", rev)
	q.Set("FilePath", file)
	out.RawQuery = q.Encode()
	return out.String()
}

func modelscopeReferer(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" || !strings.Contains(strings.ToLower(u.Host), "modelscope.") {
		return ""
	}
	return u.Scheme + "://" + u.Host + "/"
}

func extractCharacterCastZip(zipPath, dest string, meta characterCastMeta) error {
	if dest == "" {
		return fmt.Errorf("character cast: missing install directory")
	}
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("character cast: unzip: %w", err)
	}
	defer r.Close()
	flat, err := castZipEntries(r.File)
	if err != nil {
		return err
	}
	manFile, ok := flat["manifest.json"]
	if !ok {
		return fmt.Errorf("character cast: manifest.json missing")
	}
	raw, err := readZipEntry(manFile, maxCastManifest)
	if err != nil {
		return err
	}
	var man characterCastManifest
	if err := json.Unmarshal(raw, &man); err != nil {
		return fmt.Errorf("character cast: manifest: %w", err)
	}
	if err := man.validate(); err != nil {
		return err
	}
	for _, a := range man.Assets {
		if _, ok := flat[a.File]; !ok {
			return fmt.Errorf("character cast: zip missing %s", a.File)
		}
	}
	staging := dest + ".staging"
	if err := os.RemoveAll(staging); err != nil {
		return err
	}
	if err := os.MkdirAll(staging, 0o755); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := os.WriteFile(filepath.Join(staging, "manifest.json"), raw, 0o644); err != nil {
		return err
	}
	for _, a := range man.Assets {
		rc, err := flat[a.File].Open()
		if err != nil {
			return err
		}
		err = writeVerifiedFile(filepath.Join(staging, a.File), rc, a.SHA256, a.Bytes)
		_ = rc.Close()
		if err != nil {
			return err
		}
	}
	meta.Version = man.Version
	meta.SHA256 = strings.ToLower(strings.TrimSpace(meta.SHA256))
	body, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(staging, castInstallMetaName), append(body, '\n'), 0o644); err != nil {
		return err
	}
	if err := swapDir(staging, dest); err != nil {
		return err
	}
	committed = true
	return nil
}

func castZipEntries(files []*zip.File) (map[string]*zip.File, error) {
	raw := map[string]*zip.File{}
	var total uint64
	for _, f := range files {
		if f.FileInfo().IsDir() {
			continue
		}
		if f.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("character cast: zip contains a symlink")
		}
		name := strings.TrimPrefix(zipEntryName(f.Name), "./")
		if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "..") || strings.Contains(name, ":") {
			return nil, fmt.Errorf("character cast: bad zip path")
		}
		if f.UncompressedSize64 > maxCastFile {
			return nil, fmt.Errorf("character cast: zip entry too large")
		}
		total += f.UncompressedSize64
		if total > maxCastTotal {
			return nil, fmt.Errorf("character cast: zip too large")
		}
		if _, exists := raw[name]; exists {
			return nil, fmt.Errorf("character cast: duplicate zip entry %s", name)
		}
		raw[name] = f
	}
	if _, ok := raw["manifest.json"]; ok {
		return raw, nil
	}
	prefix := ""
	for name := range raw {
		seg, rest, ok := strings.Cut(name, "/")
		if !ok || rest == "" || strings.Contains(rest, "/") {
			return nil, fmt.Errorf("character cast: manifest.json missing")
		}
		if prefix == "" {
			prefix = seg
			continue
		}
		if prefix != seg {
			return nil, fmt.Errorf("character cast: manifest.json missing")
		}
	}
	if prefix == "" {
		return nil, fmt.Errorf("character cast: manifest.json missing")
	}
	flat := map[string]*zip.File{}
	for name, f := range raw {
		flat[strings.TrimPrefix(name, prefix+"/")] = f
	}
	if _, ok := flat["manifest.json"]; !ok {
		return nil, fmt.Errorf("character cast: manifest.json missing")
	}
	return flat, nil
}

func readZipEntry(f *zip.File, limit int64) ([]byte, error) {
	if f.UncompressedSize64 > uint64(limit) {
		return nil, fmt.Errorf("character cast: entry too large")
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	b, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, fmt.Errorf("character cast: entry too large")
	}
	return b, nil
}

func writeVerifiedFile(dest string, r io.Reader, wantSHA string, wantBytes int) error {
	tmp := dest + ".partial"
	f, err := os.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), io.LimitReader(r, int64(maxCastFile)+1))
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	if n > maxCastFile {
		_ = os.Remove(tmp)
		return fmt.Errorf("character cast: file too large")
	}
	if wantBytes > 0 && int(n) != wantBytes {
		_ = os.Remove(tmp)
		return fmt.Errorf("character cast: size mismatch")
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(sum, wantSHA) {
		_ = os.Remove(tmp)
		return fmt.Errorf("character cast: sha256 mismatch for %s", filepath.Base(dest))
	}
	if err := os.Rename(tmp, dest); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func swapDir(staging, dest string) error {
	backup := dest + ".bak"
	_ = os.RemoveAll(backup)
	if _, err := os.Stat(dest); err == nil {
		if err := os.Rename(dest, backup); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(staging, dest); err != nil {
		_ = os.Rename(backup, dest)
		return err
	}
	_ = os.RemoveAll(backup)
	return nil
}

func assertZipMagic(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return fmt.Errorf("character cast: not a zip")
	}
	if magic != [4]byte{'P', 'K', 3, 4} {
		return fmt.Errorf("character cast: not a zip")
	}
	return nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func characterCastName(urlPath string) (string, bool) {
	p := path.Clean("/" + urlPath)
	if !strings.HasPrefix(p, characterCastPublicPrefix) {
		return "", false
	}
	name := strings.TrimPrefix(p, characterCastPublicPrefix)
	if !castFileRe.MatchString(name) {
		return "", false
	}
	return name, true
}

func characterCastFile(dir, name string) string {
	if dir == "" || name == "" {
		return ""
	}
	man, err := readCharacterCastManifest(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return ""
	}
	found := false
	for _, a := range man.Assets {
		if a.File == name {
			found = true
			break
		}
	}
	if !found {
		return ""
	}
	p := filepath.Join(dir, name)
	st, err := os.Lstat(p)
	if err != nil || !st.Mode().IsRegular() {
		return ""
	}
	realDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		return ""
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		return ""
	}
	rel, err := filepath.Rel(realDir, real)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return ""
	}
	return real
}

// CharacterCastHandler serves installed stills at /character-cast/<file>.
func CharacterCastHandler(dir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		name, ok := characterCastName(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		p := characterCastFile(dir, name)
		if p == "" {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(p)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		http.ServeContent(w, r, name, st.ModTime(), f)
	})
}

// CharacterCastMiddleware serves an installed still and otherwise calls next,
// so a pack that is not downloaded yet can fall through to bundled files.
func CharacterCastMiddleware(dir string, next http.Handler) http.Handler {
	if next == nil {
		next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.NotFound(w, r)
		})
	}
	if strings.TrimSpace(dir) == "" {
		return next
	}
	h := CharacterCastHandler(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}
		name, ok := characterCastName(r.URL.Path)
		if !ok || characterCastFile(dir, name) == "" {
			next.ServeHTTP(w, r)
			return
		}
		h.ServeHTTP(w, r)
	})
}
