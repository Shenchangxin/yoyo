// run.go 用 ffmpeg 和 ffprobe 路径执行剪辑命令。

package cut

import (
	"fmt"
	"os/exec"
	"strings"
)

// Bin 是剪辑原语共用的 ffmpeg、ffprobe 和 whisper 路径。
type Bin struct {
	FFmpeg  string
	FFprobe string
	Whisper string
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
func (b Bin) run(args ...string) ([]byte, error) {
	if err := b.ok(); err != nil {
		return nil, err
	}
	cmd := quiet(exec.Command(b.FFmpeg, args...))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s", ffmpegErr(out, err))
	}
	return out, nil
}

// probe 优先用 ffprobe，路径为空时改用 ffmpeg。两者都空则返回 ffprobe missing。
func (b Bin) probe(args ...string) ([]byte, error) {
	bin := b.FFprobe
	if bin == "" {
		bin = b.FFmpeg
	}
	if bin == "" {
		return nil, fmt.Errorf("ffprobe missing")
	}
	cmd := quiet(exec.Command(bin, args...))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s", ffmpegErr(out, err))
	}
	return out, nil
}

// ffmpegErr 取命令输出的尾部。输出含换行且长于 400 字节时只留最后 8 行。输出为空时用原错误，原错误也空则返回 ffmpeg failed。
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
