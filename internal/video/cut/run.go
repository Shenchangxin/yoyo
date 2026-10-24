package cut

import (
	"fmt"
	"os/exec"
	"strings"
)

// Bin is the FFmpeg pair used by every cut primitive.
type Bin struct {
	FFmpeg  string
	FFprobe string
	Whisper string
}

func (b Bin) ok() error {
	if strings.TrimSpace(b.FFmpeg) == "" {
		return fmt.Errorf("ffmpeg missing")
	}
	return nil
}

func (b Bin) probeOK() error {
	if strings.TrimSpace(b.FFprobe) == "" && strings.TrimSpace(b.FFmpeg) == "" {
		return fmt.Errorf("ffprobe missing")
	}
	return nil
}

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
