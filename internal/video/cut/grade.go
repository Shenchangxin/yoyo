package cut

import "strings"

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
