// grade.go 用 ffmpeg 滤镜给成片套上预设调色。

package cut

import (
	"fmt"
	"strings"
)

// GradeFilter 返回 warm、cool、contrast、fade 对应的 ffmpeg 滤镜。其他预设用默认的对比度和饱和度。
// preset 去掉空白并转小写后匹配 warm、cool、contrast、fade，其余走 default。
func GradeFilter(preset string) string {
	switch strings.ToLower(strings.TrimSpace(preset)) {
	case "warm":
		return "eq=gamma=1.03:saturation=1.08:contrast=1.04,colorbalance=rs=0.05:gs=0.01:bs=-0.04"
	case "cool":
		return "eq=gamma=1.0:saturation=0.94:contrast=1.02,colorbalance=rs=-0.04:bs=0.06"
	case "contrast":
		return "eq=contrast=1.14:saturation=1.06"
	case "fade":
		return "eq=gamma=1.06:saturation=0.9:contrast=0.96"
	default:
		return "eq=contrast=1.06:saturation=1.04"
	}
}

// Look 一次重编码套上画面滤镜和音频滤镜。vf 空着时画面直接复制，af 空着时音频直接复制；两个都空时什么都不做。源片没有音轨时不处理音频。
// src 作为 -i。
// outPath 是输出。
// vf 是逗号连起来的画面滤镜。
// af 是音频滤镜。
func (b Bin) Look(src, outPath, vf, af string) error {
	if err := b.ok(); err != nil {
		return err
	}
	if vf == "" && af == "" {
		return fmt.Errorf("nothing to apply")
	}
	args := []string{"-y", "-i", src}
	if vf != "" {
		args = append(args, "-vf", vf, "-c:v", "libx264", "-preset", "veryfast", "-crf", "20", "-pix_fmt", "yuv420p")
	} else {
		args = append(args, "-c:v", "copy")
	}
	switch {
	case !b.HasAudio(src):
		args = append(args, "-an")
	case af != "":
		args = append(args, "-af", af, "-c:a", "aac", "-ar", "48000", "-b:a", "192k")
	default:
		args = append(args, "-c:a", "copy")
	}
	_, err := b.run(append(args, "-movflags", "+faststart", outPath)...)
	return err
}

// Grade 用 GradeFilter 重编码画面并复制音频。复制音频失败时改为不带音频再跑一次。
// src 作为 -i。
// outPath 作为两次 b.run 的输出路径。
// preset 传给 GradeFilter。
func (b Bin) Grade(src, outPath, preset string) error {
	if err := b.ok(); err != nil {
		return err
	}
	_, err := b.run("-y", "-i", src, "-vf", GradeFilter(preset), "-c:a", "copy", outPath)
	if err != nil {
		_, err = b.run("-y", "-i", src, "-vf", GradeFilter(preset), "-an", outPath)
	}
	return err
}
