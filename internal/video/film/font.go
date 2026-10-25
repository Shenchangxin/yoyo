package film

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

var (
	fontOnce sync.Once
	fontData *opentype.Font
	faceMu   sync.Mutex
	faces    map[float64]font.Face
)

func textFace(size float64) font.Face {
	fontOnce.Do(loadFont)
	if fontData == nil || size <= 0 {
		return nil
	}
	faceMu.Lock()
	defer faceMu.Unlock()
	if faces == nil {
		faces = map[float64]font.Face{}
	}
	if face, ok := faces[size]; ok {
		return face
	}
	face, err := opentype.NewFace(fontData, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil
	}
	faces[size] = face
	return face
}

func loadFont() {
	for _, path := range fontCandidates() {
		raw, err := os.ReadFile(path)
		if err != nil || len(raw) == 0 {
			continue
		}
		if col, err := opentype.ParseCollection(raw); err == nil && col != nil {
			if parsed, err := col.Font(0); err == nil {
				fontData = parsed
				return
			}
		}
		if parsed, err := opentype.Parse(raw); err == nil {
			fontData = parsed
			return
		}
	}
}

func fontCandidates() []string {
	switch runtime.GOOS {
	case "windows":
		root := os.Getenv("WINDIR")
		if root == "" {
			root = `C:\Windows`
		}
		dir := filepath.Join(root, "Fonts")
		return []string{
			filepath.Join(dir, "msyh.ttc"),
			filepath.Join(dir, "msyh.ttf"),
			filepath.Join(dir, "simhei.ttf"),
			filepath.Join(dir, "simsun.ttc"),
		}
	case "darwin":
		return []string{
			"/System/Library/Fonts/PingFang.ttc",
			"/System/Library/Fonts/Hiragino Sans GB.ttc",
			"/System/Library/Fonts/STHeiti Medium.ttc",
		}
	default:
		return []string{
			"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
			"/usr/share/fonts/truetype/wqy/wqy-microhei.ttc",
		}
	}
}
