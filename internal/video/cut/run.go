// run.go 用 ffmpeg 和 ffprobe 路径执行剪辑命令。

package cut

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Bin 是剪辑原语共用的 ffmpeg、ffprobe 和 whisper 路径。
type Bin struct {
	FFmpeg  string
	FFprobe string
	Whisper string
	// WhisperModel 是 whisper.cpp 的 ggml 模型文件。
	WhisperModel string
	// WhisperLang 是转写语言，空着按 auto。
	WhisperLang string
	// Ctx 非空时，命令跟着它取消。引擎关闭时正在跑的 ffmpeg 一起停下。
	Ctx context.Context
}

// command 按 Ctx 造命令。Ctx 为空时用普通的 exec.Command。
func (b Bin) command(name string, args ...string) *exec.Cmd {
	if b.Ctx != nil {
		return exec.CommandContext(b.Ctx, name, args...)
	}
	return exec.Command(name, args...)
}

// HasAudio 用 ffprobe 看文件里有没有音轨。没有单独的 ffprobe 或读不出时按有音轨算，交给后面的命令报错。
// path 传给 ffprobe。
func (b Bin) HasAudio(path string) bool {
	probe := strings.TrimSpace(b.FFprobe)
	if probe == "" || strings.Contains(strings.ToLower(filepath.Base(probe)), "ffmpeg") {
		return true
	}
	out, err := b.probe("-v", "error", "-select_streams", "a", "-show_entries", "stream=index", "-of", "csv=p=0", path)
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(out)) != ""
}

// ok 在 ffmpeg 路径为空时返回 ffmpeg missing。
func (b Bin) ok() error {
	if strings.TrimSpace(b.FFmpeg) == "" {
		return fmt.Errorf("ffmpeg missing")
	}
	return nil
}

// probeOK 在 ffprobe 和 ffmpeg 都为空时返回 ffprobe missing。
func (b Bin) probeOK() error {
	if strings.TrimSpace(b.FFprobe) == "" && strings.TrimSpace(b.FFmpeg) == "" {
		return fmt.Errorf("ffprobe missing")
	}
	return nil
}

// run 用 ffmpeg 跑参数并收集合并输出。失败时经 ffmpegErr 收成错误文本。
// args 展开传给 ffmpeg。
func (b Bin) run(args ...string) ([]byte, error) {
	if err := b.ok(); err != nil {
		return nil, err
	}
	cmd := quiet(b.command(b.FFmpeg, args...))
	out, err := cmd.CombinedOutput()
	if err != nil {
		if b.Ctx != nil && b.Ctx.Err() != nil {
			return out, b.Ctx.Err()
		}
		return out, fmt.Errorf("%s", ffmpegErr(out, err))
	}
	return out, nil
}

// probe 优先用 ffprobe，路径为空时改用 ffmpeg。两者都空则返回 ffprobe missing。
// args 展开传给 ffprobe；FFprobe 为空时改传给 ffmpeg。
func (b Bin) probe(args ...string) ([]byte, error) {
	bin := b.FFprobe
	if bin == "" {
		bin = b.FFmpeg
	}
	if bin == "" {
		return nil, fmt.Errorf("ffprobe missing")
	}
	cmd := quiet(b.command(bin, args...))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s", ffmpegErr(out, err))
	}
	return out, nil
}

// ffmpegErr 取命令输出的尾部。输出含换行且长于 400 字节时只留最后 8 行。输出为空时用原错误，原错误也空则返回 ffmpeg failed。
// out 转成字符串并去掉首尾空白；含换行且长于 400 字节时只留最后 8 行。
// err 在输出为空且非 nil 时返回 Error，输出和 err 都空则返回 ffmpeg failed。
func ffmpegErr(out []byte, err error) string {
	tail := strings.TrimSpace(string(out))
	if n := strings.LastIndex(tail, "\n"); n >= 0 && len(tail) > 400 {
		lines := strings.Split(tail, "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		tail = strings.Join(lines, "\n")
	}
	if tail == "" {
		if err != nil {
			return err.Error()
		}
		return "ffmpeg failed"
	}
	return tail
}
