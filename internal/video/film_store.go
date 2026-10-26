// 翻拍工程的存取。打开参考、换简介、构建，以及把画面交给已有的剪辑台。
// 下载、词级标注和区域检测用本机程序，没有就跳过，工程仍会保存。

package video

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/connection"
	"github.com/Shenchangxin/yoyo/internal/video/film"
)

// lookFilmTools 查找 yt-dlp、词级标注、区域程序和浏览器，并写到引擎上。
func (e *Engine) lookFilmTools() {
	e.YTDLP = lookBin("YOYO_YTDLP", "yt-dlp")
	if e.YTDLP == "" {
		e.YTDLP = lookNamed(e, "yt-dlp")
	}
	e.StampBin = lookBin("YOYO_STAMP_BIN", "whisperx")
	e.RegionBin = lookBin("YOYO_REGION_BIN", "yoyo-region")
	e.BrowserBin = lookBrowser()
}

// lookNamed 在视频目录和用户主目录的 bin 里找这个程序。Windows 上会补 .exe。
// name 在 Windows 上若不以 .exe 结尾就补上，再在目录里查找。
func lookNamed(e *Engine, name string) string {
	file := name
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(file), ".exe") {
		file += ".exe"
	}
	var dirs []string
	if e != nil && e.Dir != "" {
		dirs = append(dirs, filepath.Join(e.Dir, "bin"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".yoyo", "video", "bin"))
	}
	for _, dir := range dirs {
		path := filepath.Join(dir, file)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	return ""
}

// lookBin 先看环境变量，再在 PATH 里找。都没有就返回空串。
// envKey 去掉空白后非空则返回该环境变量。
// name 传给 LookPath。
func lookBin(envKey, name string) string {
	if v := strings.TrimSpace(os.Getenv(envKey)); v != "" {
		return v
	}
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return ""
}

// lookBrowser 先看 YOYO_BROWSER，再在 PATH 里找 Chrome 或 Edge。Windows 上还会看 Program Files。
func lookBrowser() string {
	if v := strings.TrimSpace(os.Getenv("YOYO_BROWSER")); v != "" {
		return v
	}
	for _, name := range []string{"chrome", "msedge", "google-chrome", "chromium"} {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	if runtime.GOOS != "windows" {
		return ""
	}
	cands := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Microsoft", "Edge", "Application", "msedge.exe"),
		filepath.Join(os.Getenv("ProgramFiles"), "Microsoft", "Edge", "Application", "msedge.exe"),
	}
	for _, path := range cands {
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			return path
		}
	}
	return ""
}

// ListFilms 列出已保存的翻拍工程。没有时返回空切片。
func (e *Engine) ListFilms() []film.Project {
	out := loadCol[film.Project](e, colFilms)
	if out == nil {
		return []film.Project{}
	}
	return out
}

// GetFilm 按编号读取一部翻拍。没有时返回 film not found。
// id 去掉空白后传给 getDoc。
func (e *Engine) GetFilm(id string) (film.Project, error) {
	p, err := getDoc[film.Project](e, colFilms, strings.TrimSpace(id))
	if err != nil {
		return film.Project{}, fmt.Errorf("film not found")
	}
	return p, nil
}

// saveFilm 写回工程。编号空着就新建一个。
// p 的 ID 去掉空白后为空时写成 NewID，再 putDoc。
func (e *Engine) saveFilm(p film.Project) (film.Project, error) {
	if strings.TrimSpace(p.ID) == "" {
		p.ID = NewID()
	}
	if err := e.putDoc(colFilms, p.ID, p); err != nil {
		return film.Project{}, err
	}
	return p, nil
}

// OpenFilm 把已有的参考收成可再跑的工程。PATH 或本机 bin 里有 yt-dlp 时才下载地址。场景切分需要 ffmpeg。
// 词轴优先用词级标注程序，否则在有 whisper.cpp 时把句子再切开。下载不了就把地址留在工程里。有画面但还没有词就在备注里说明。
func (e *Engine) OpenFilm(in film.Input) (film.Project, error) {
	in.Hash = strings.TrimSpace(in.Hash)
	in.URL = strings.TrimSpace(in.URL)
	in = e.fetchFilmURL(in)
	in = e.readFilmMedia(in)
	if strings.TrimSpace(in.Structure) == "" && len(in.Cues) == 0 && len(in.Words) == 0 && strings.TrimSpace(in.Description) == "" {
		in.Structure = "free"
	}
	p := film.FromReference(in)
	if in.URL != "" && in.Hash == "" && strings.TrimSpace(e.YTDLP) == "" {
		p.Note = "还没有 yt-dlp，参考地址留在工程里，没有下载。"
	} else if in.Hash != "" && len(p.Words) == 0 {
		p.Note = "还没有词轴。本机有词级标注或 whisper.cpp 时会写成词。"
	}
	return e.saveFilm(p)
}

// fetchFilmURL 在还没有内容编号、且有 yt-dlp 时下载参考。失败就保持原样，地址仍在。
// in 在 Hash 非空、URL 为空或没有 yt-dlp 时原样返回，否则用 URL 下载并把内容编号写入 Hash。
func (e *Engine) fetchFilmURL(in film.Input) film.Input {
	if in.Hash != "" || in.URL == "" || strings.TrimSpace(e.YTDLP) == "" {
		return in
	}
	dir, err := os.MkdirTemp(e.Dir, "film-dl-*")
	if err != nil {
		return in
	}
	defer os.RemoveAll(dir)
	out := filepath.Join(dir, "ref.mp4")
	args := film.DownloadCommand(e.YTDLP, in.URL, out)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Run(); err != nil {
		return in
	}
	raw, err := os.ReadFile(out)
	if err != nil || len(raw) == 0 {
		return in
	}
	hash, err := e.PutBytes(raw)
	if err != nil || hash == "" {
		return in
	}
	in.Hash = hash
	return in
}

// readFilmMedia 读时长、场景、词和区域。没有 ffmpeg 就只尽量读时长。词级标注没有结果时才用 whisper.cpp 的句子。
// in 的 Hash 为空时原样返回，否则传给 probeCutDuration 和 materialize。
func (e *Engine) readFilmMedia(in film.Input) film.Input {
	if in.Hash == "" {
		return in
	}
	if d := e.probeCutDuration(in.Hash); d > 0 {
		in.Duration = d
	}
	e.refreshFFmpeg()
	if !e.FFmpegOK() {
		return in
	}
	src, err := e.materialize(in.Hash, ".mp4")
	if err != nil {
		return in
	}
	defer os.Remove(src)
	if bd, err := e.cutBin().Analyze(src); err == nil {
		in.Scenes = bd.Scenes
		if bd.Duration > 0 {
			in.Duration = bd.Duration
		}
	}
	if words := e.stampFilmWords(src); len(words) > 0 {
		in.Words = words
	}
	if len(in.Words) == 0 && strings.TrimSpace(e.WHISPER) != "" {
		if cues, _, err := e.cutBin().Transcribe(src, filepath.Join(e.Dir, "film-asr")); err == nil {
			for _, cue := range cues {
				in.Cues = append(in.Cues, film.Cue{Start: cue.Start, End: cue.End, Text: cue.Text})
			}
		}
	}
	if boxes := e.detectFilmRegions(src); len(boxes) > 0 {
		in.Regions = boxes
	}
	return in
}

// stampFilmWords 调用词级标注程序，读它写出的 JSON。程序没有、运行失败或解不开就返回 nil。
// src 传给 StampCommand，并用它的文件名查找 JSON。
func (e *Engine) stampFilmWords(src string) []film.Word {
	if strings.TrimSpace(e.StampBin) == "" {
		return nil
	}
	dir, err := os.MkdirTemp(e.Dir, "film-stamp-*")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(dir)
	args := film.StampCommand(e.StampBin, src, dir)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Run(); err != nil {
		return nil
	}
	stem := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	raw, err := os.ReadFile(filepath.Join(dir, stem+".json"))
	if err != nil {
		matches, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		if len(matches) == 0 {
			return nil
		}
		raw, err = os.ReadFile(matches[0])
		if err != nil {
			return nil
		}
	}
	words, err := film.WordsFromStamps(raw)
	if err != nil {
		return nil
	}
	return words
}

// detectFilmRegions 优先跑区域程序。没有结果就用 ffmpeg 的 cropdetect。
// src 在有区域程序时传给 runRegionBin，没有结果再传给 cropFilmRegion。
func (e *Engine) detectFilmRegions(src string) []film.Box {
	if strings.TrimSpace(e.RegionBin) != "" {
		if boxes := e.runRegionBin(src); len(boxes) > 0 {
			return boxes
		}
	}
	return e.cropFilmRegion(src)
}

// runRegionBin 调用区域程序并解析它写出的 JSON。失败就返回 nil。
// src 传给 RegionCommand，并加上 .regions.json 作为输出路径。
func (e *Engine) runRegionBin(src string) []film.Box {
	out := src + ".regions.json"
	defer os.Remove(out)
	args := film.RegionCommand(e.RegionBin, src, out)
	cmd := exec.Command(args[0], args[1:]...)
	if err := cmd.Run(); err != nil {
		return nil
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		return nil
	}
	boxes, err := film.ParseRegions(raw)
	if err != nil {
		return nil
	}
	return boxes
}

// cropFilmRegion 用 ffprobe 读尺寸，再用 ffmpeg cropdetect 收一个区域。没有 ffmpeg 或 ffprobe，或解析失败，就返回 nil。最多等 12 秒。
// src 作为 ffprobe 和 ffmpeg 的输入。
func (e *Engine) cropFilmRegion(src string) []film.Box {
	if strings.TrimSpace(e.FFMPEG) == "" || strings.TrimSpace(e.FFProbe) == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	probe, err := exec.CommandContext(ctx, e.FFProbe, "-v", "error", "-select_streams", "v:0", "-show_entries", "stream=width,height", "-of", "csv=p=0", src).Output()
	if err != nil {
		return nil
	}
	parts := strings.Split(strings.TrimSpace(string(probe)), ",")
	if len(parts) < 2 {
		return nil
	}
	width, werr := strconv.Atoi(strings.TrimSpace(parts[0]))
	height, herr := strconv.Atoi(strings.TrimSpace(parts[1]))
	if werr != nil || herr != nil {
		return nil
	}
	combined, _ := exec.CommandContext(ctx, e.FFMPEG, "-i", src, "-vf", "cropdetect=24:16:0", "-f", "null", "-").CombinedOutput()
	last := ""
	for _, line := range strings.Split(string(combined), "\n") {
		if i := strings.Index(line, "crop="); i >= 0 {
			last = line[i:]
		}
	}
	box, ok := film.ParseCrop(last, width, height)
	if !ok {
		return nil
	}
	return []film.Box{box}
}

// DescribeFilm 按描述或三个结构模板建工程，不需要参考文件。结构空着就用排行榜。排行榜、播客和街采在描述是空的时用模板。
// structure 去掉空白后为空时写成 ranking，并与 ranking、podcast、interview 比较。
// title 传给 FromTemplate 或写入 Input 的 Title。
// description 去掉空白后为空且结构是这三个时走模板，否则写入 Input 的 Description。
func (e *Engine) DescribeFilm(structure, title, description string) (film.Project, error) {
	structure = strings.TrimSpace(structure)
	if structure == "" {
		structure = "ranking"
	}
	if structure == "ranking" || structure == "podcast" || structure == "interview" {
		if strings.TrimSpace(description) == "" {
			p := film.FromTemplate(structure, title)
			return e.saveFilm(p)
		}
	}
	p := film.FromReference(film.Input{Title: title, Structure: structure, Description: description})
	return e.saveFilm(p)
}

// PlanFilm 按当前镜头重算待办和输入摘要，并写回。输入没变的产物编号会留下。
// id 传给 GetFilm。
func (e *Engine) PlanFilm(id string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p = film.Plan(p)
	return e.saveFilm(p)
}

// SwapFilm 改简介里非空的字段，并重算待办。输入没变的产物留下。换了语言就试着用对话连接逐行重写。没有对话连接时只记下语言，备注说明还没改写。
// id 传给 GetFilm。
// patch 传给 film.Swap；Language 去掉空白后非空时再翻译。
func (e *Engine) SwapFilm(id string, patch film.Brief) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p = film.Swap(p, patch)
	if strings.TrimSpace(patch.Language) != "" {
		rewritten, err := e.translateFilm(p)
		if err != nil {
			p.Note = "语言已记下。接上对话连接后可以按这个语言重写每一行。"
		} else {
			p = rewritten
			p.Note = ""
		}
	}
	return e.saveFilm(p)
}

// ForkFilm 另存一份并套上简介补丁。原片的产物不动。新片没有自己的产物和任务。
// id 传给 GetFilm。
// patch 传给 film.Fork。
func (e *Engine) ForkFilm(id string, patch film.Brief) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	return e.saveFilm(film.Fork(p, patch))
}

// SetFilmBed 把操作者已经存好的垫乐编号写上工程。不在这里生成音乐。
// id 传给 GetFilm。
// hash 去掉空白后写入 Bed。
func (e *Engine) SetFilmBed(id, hash string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p.Bed = strings.TrimSpace(hash)
	return e.saveFilm(p)
}

// translateFilm 用已启用的对话连接按目标语言重写每一行，并只重排这些行的时间。返回的行数对不上就拒绝，原片不改。没有行时原样返回。
// p 没有行时原样返回，否则用 Brief.Language 和行文本调用 chatComplete，行数一致时传给 Localize。
func (e *Engine) translateFilm(p film.Project) (film.Project, error) {
	lines := film.LineTexts(p)
	if len(lines) == 0 {
		return p, nil
	}
	c, key, err := e.chatForFilm()
	if err != nil {
		return p, err
	}
	model := connection.ModelFor(c, connection.CapChat)
	if model == "" {
		model = "gpt-4.1-mini"
	}
	var body strings.Builder
	for _, line := range lines {
		body.WriteString(line.Text)
		body.WriteByte('\n')
	}
	raw, err := e.chatComplete(c.Endpoint, key, model,
		"你把口播改写成指定语言。保持行数和顺序，保留每行的口气。只输出改写后的正文，一行一句，不要编号，不要解释。",
		"语言："+p.Brief.Language+"\n"+body.String(),
	)
	if err != nil {
		return p, err
	}
	var parts []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts = append(parts, line)
	}
	if len(parts) != len(lines) {
		return p, fmt.Errorf("localize returned %d lines, want %d", len(parts), len(lines))
	}
	mapped := map[string]string{}
	for i, line := range lines {
		mapped[line.ID] = parts[i]
	}
	return film.Localize(p, mapped), nil
}

// chatForFilm 找一条已启用、有密钥、且名称或地址里不带 hypihub 的对话连接。没有就返回错误。
func (e *Engine) chatForFilm() (connection.Connection, string, error) {
	if e.Conn == nil {
		return connection.Connection{}, "", fmt.Errorf("no enabled connection for localize")
	}
	list, err := e.Conn.List(connection.CapChat)
	if err != nil {
		return connection.Connection{}, "", fmt.Errorf("no enabled connection for localize")
	}
	for _, c := range list {
		if !c.Active || !c.HasKey {
			continue
		}
		blob := strings.ToLower(c.ID + " " + c.Name + " " + c.Endpoint + " " + c.Vendor)
		if strings.Contains(blob, "hypihub") {
			continue
		}
		key, err := e.Conn.Lease(c)
		if err != nil || strings.TrimSpace(key) == "" {
			continue
		}
		return c, key, nil
	}
	return connection.Connection{}, "", fmt.Errorf("no enabled connection for localize")
}

// RewriteFilmLine 替换一行词，并只重排这一行的时间。不用这一行的镜头和产物保持原样。
// id 传给 GetFilm。
// lineID 传给 ReplaceLine。
// text 传给 ReplaceLine。
func (e *Engine) RewriteFilmLine(id, lineID, text string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	return e.saveFilm(film.ReplaceLine(p, lineID, text))
}

// BuildFilm 按计划构建。代码画面在本机画出，图、语音、抠像和配乐交给已启用的连接。
// 本机有浏览器时再存一张页面截图。然后尝试合成成片。生成的配乐如果还没写上垫乐，会把音乐产物当成垫乐。
// id 传给 GetFilm。
func (e *Engine) BuildFilm(id string) (film.Project, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return film.Project{}, err
	}
	p = film.Build(p, film.BuildEnv{
		Providers: e.filmEndpoints(),
		Put: func(b []byte) (string, error) {
			return e.PutBytes(b)
		},
		Dispatch: e.dispatchFilmNeed,
	})
	if strings.TrimSpace(p.Bed) == "" {
		for _, a := range p.Artifacts {
			if a.Name == "music" && a.Hash != "" {
				p.Bed = a.Hash
				break
			}
		}
	}
	if e.BrowserBin != "" {
		p = e.captureFilmBrowser(p)
	}
	p = e.assembleFilm(p)
	return e.saveFilm(p)
}

// filmEndpoints 从图像、视频、语音、抠像和配乐菜单收集已启用的连接。
func (e *Engine) filmEndpoints() []film.Endpoint {
	var out []film.Endpoint
	for _, kind := range []string{"image", "video", "speech", "matting", "music"} {
		for _, row := range e.ProviderMenu(kind) {
			id := strAnyMap(row, "id")
			ep := film.Endpoint{
				ID: id, Name: strAnyMap(row, "name"), Service: kind, Vendor: strAnyMap(row, "vendor"),
				Model: firstModel(row["models"]), BaseURL: strAnyMap(row, "endpoint", "base_url"),
				Active: true, HasKey: true,
			}
			if e.Conn != nil && id != "" {
				if c, err := e.Conn.Get(id); err == nil {
					ep.BaseURL = c.Endpoint
					ep.VaultKey = c.VaultKey
					if ep.Vendor == "" {
						ep.Vendor = c.Vendor
					}
				}
			}
			out = append(out, ep)
		}
	}
	return out
}

// firstModel 取模型列表里的第一个。不是列表就返回空串。
// v 是字符串切片或任意切片时返回第一项，否则返回空串。
func firstModel(v any) string {
	switch t := v.(type) {
	case []string:
		if len(t) > 0 {
			return t[0]
		}
	case []any:
		if len(t) > 0 {
			return fmt.Sprint(t[0])
		}
	}
	return ""
}

// dispatchFilmNeed 按种类提交图像、视频或语音。语音走短片的配音。还在远端跑时返回等待，不把它当成失败。抠像和配乐另走各自的接口。
// n 的 Kind 为 matting 或 music 时分别传给 dispatchFilmMatte 和 dispatchFilmMusic；image、video 写入任务后提交，speech 用 Prompt、alloy 和 1 调用 speakAudio。
func (e *Engine) dispatchFilmNeed(n film.Need) (string, error) {
	ctx := context.Background()
	switch n.Kind {
	case "matting":
		return e.dispatchFilmMatte(ctx, n)
	case "music":
		return e.dispatchFilmMusic(ctx, n)
	}
	j := &Job{
		Provider: n.Provider,
		VaultKey: n.VaultKey,
		Model:    n.Model,
		Prompt:   n.Prompt,
		Params:   `{"prompt":` + jsonString(n.Prompt) + `}`,
	}
	var err error
	switch n.Kind {
	case "image":
		err = e.submitImage(ctx, j)
	case "video":
		err = e.submitVideo(ctx, j)
	case "speech":
		raw, _, _, err2 := e.speakAudio(n.Prompt, "alloy", 1)
		if err2 != nil {
			return "", err2
		}
		return e.PutBytes(raw)
	default:
		return "", fmt.Errorf("unknown need %s", n.Kind)
	}
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(j.ResultHash) != "" {
		return j.ResultHash, nil
	}
	if strings.TrimSpace(j.RemoteID) != "" {
		return "", film.Wait("生成还在进行：" + j.RemoteID)
	}
	return "", fmt.Errorf("need %s produced no asset", n.ID)
}

// dispatchFilmMatte 用抠像连接处理参考画面。供应商还只有预设、没有请求体时返回等待。没有可抠的画面也返回等待。
// ctx 传给 matteImage。
// n 的 Provider、VaultKey、Model、Prompt 写入任务，Prompt 去掉空白后传给 matteSource。
func (e *Engine) dispatchFilmMatte(ctx context.Context, n film.Need) (string, error) {
	j := &Job{Provider: n.Provider, VaultKey: n.VaultKey, Model: n.Model, Prompt: n.Prompt}
	p, err := e.providerForFilm(*j, "matting")
	if err != nil {
		return "", err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return "", err
	}
	raw, err := e.matteSource(strings.TrimSpace(n.Prompt))
	if err != nil {
		return "", err
	}
	out, err := e.matteImage(ctx, p, e.protocolOf(p, "matting"), key, raw)
	if err != nil {
		var preset presetOnlyError
		if errors.As(err, &preset) {
			return "", film.Wait(preset.Error())
		}
		return "", err
	}
	return e.PutBytes(out)
}

// dispatchFilmMusic 用配乐连接生成垫乐。供应商还只有预设、没有请求体时返回等待。
// ctx 传给 scoreMusic。
// n 的 Provider、VaultKey、Model、Prompt 写入任务，Prompt 和 Model 传给 scoreMusic。
func (e *Engine) dispatchFilmMusic(ctx context.Context, n film.Need) (string, error) {
	j := &Job{Provider: n.Provider, VaultKey: n.VaultKey, Model: n.Model, Prompt: n.Prompt}
	p, err := e.providerForFilm(*j, "music")
	if err != nil {
		return "", err
	}
	key, err := e.leaseProvider(p)
	if err != nil {
		return "", err
	}
	out, err := e.scoreMusic(ctx, p, e.protocolOf(p, "music"), key, n.Prompt, n.Model)
	if err != nil {
		var preset presetOnlyError
		if errors.As(err, &preset) {
			return "", film.Wait(preset.Error())
		}
		return "", err
	}
	return e.PutBytes(out)
}

// providerForFilm 先按任务找提供商。找不到且有保险库键时，再按键找一条已启用的连接。
// j 传给 providerForJob；VaultKey 非空时再按它匹配已启用连接。
// kind 传给 providerForJob，找到连接时再传给 providerFromConn。
func (e *Engine) providerForFilm(j Job, kind string) (Provider, error) {
	p, err := e.providerForJob(j, kind)
	if err == nil {
		return p, nil
	}
	if j.VaultKey != "" && e.Conn != nil {
		list, lerr := e.Conn.List("")
		if lerr == nil {
			for _, c := range list {
				if c.VaultKey == j.VaultKey && c.Active {
					return providerFromConn(c, kind, nil), nil
				}
			}
		}
	}
	return Provider{}, err
}

// protocolOf 优先用连接上写明的协议，否则按提供商和能力推断。
// cap 传给 InferProtocol。
func (e *Engine) protocolOf(p Provider, cap string) string {
	if e.Conn != nil && p.ID != "" {
		if c, err := e.Conn.Get(p.ID); err == nil && strings.TrimSpace(c.Protocol) != "" {
			return c.Protocol
		}
	}
	return connection.InferProtocol(p.Provider, cap)
}

// matteSource 读出要抠的画面。编号空着或内容库里没有就返回等待。
// hash 为空，或 GetBytes 失败、内容为空时返回等待，否则传给 GetBytes。
func (e *Engine) matteSource(hash string) ([]byte, error) {
	if hash == "" {
		return nil, film.Wait("没有可抠的画面")
	}
	raw, err := e.GetBytes(hash)
	if err != nil || len(raw) == 0 {
		return nil, film.Wait("没有可抠的画面")
	}
	return raw, nil
}

// jsonString 把字符串编成 JSON 字面量，给任务参数用。
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// captureFilmBrowser 用本机浏览器把代码画面截成 PNG 存进产物。broll 跳过。写文件或截图失败就跳过这一镜，不中断其余镜头。
func (e *Engine) captureFilmBrowser(p film.Project) film.Project {
	for _, shot := range p.Shots {
		if shot.Visual == "broll" {
			continue
		}
		html := film.RenderHTML(p, shot)
		dir, err := os.MkdirTemp(e.Dir, "film-html-*")
		if err != nil {
			return p
		}
		page := filepath.Join(dir, "shot.html")
		png := filepath.Join(dir, "shot.png")
		if err := os.WriteFile(page, []byte(html), 0o644); err != nil {
			os.RemoveAll(dir)
			continue
		}
		cmd := exec.Command(e.BrowserBin, film.BrowserCommand(e.BrowserBin, page, png)[1:]...)
		cmd.Dir = dir
		if err := cmd.Run(); err != nil {
			os.RemoveAll(dir)
			continue
		}
		raw, err := os.ReadFile(png)
		os.RemoveAll(dir)
		if err != nil || len(raw) == 0 {
			continue
		}
		hash, err := e.PutBytes(raw)
		if err != nil {
			continue
		}
		p.Artifacts = append(p.Artifacts, film.Artifact{Name: "browser:" + shot.ID, Hash: hash, MediaType: "image/png"})
	}
	return p
}

// HandoffFilm 用已有的剪辑台打开这部翻拍的画面。优先用成片，其次任意图像，最后用参考片。都没有就拒绝。标题空着就用工程标题。什么时候打开剪辑台由调用方决定。
// id 传给 GetFilm。
// title 去掉空白后为空时改用工程标题，再传给 CreateCutProject 和 CutIngest。
func (e *Engine) HandoffFilm(id, title string) (CutProject, error) {
	p, err := e.GetFilm(id)
	if err != nil {
		return CutProject{}, err
	}
	hash := ""
	for _, a := range p.Artifacts {
		if strings.HasPrefix(a.MediaType, "video/") && a.Hash != "" {
			hash = a.Hash
			break
		}
	}
	if hash == "" {
		for _, a := range p.Artifacts {
			if strings.HasPrefix(a.MediaType, "image/") && a.Hash != "" {
				hash = a.Hash
				break
			}
		}
	}
	if hash == "" {
		hash = p.Reference.Hash
	}
	if strings.TrimSpace(title) == "" {
		title = p.Title
	}
	if hash == "" {
		return CutProject{}, fmt.Errorf("film has no picture to hand off")
	}
	cut, err := e.CreateCutProject(CutProject{Title: title})
	if err != nil {
		return CutProject{}, err
	}
	return e.CutIngest(cut.ID, hash, title)
}
