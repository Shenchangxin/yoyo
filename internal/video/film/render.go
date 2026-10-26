// 用代码画出一帧，或把当前的词和贴纸画到已有静帧上。
// 不调用生成模型。本机浏览器要截的页面也在这里拼。

package film

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// RenderCode 画出这一镜最后一个词的画面，不调用生成模型。
func RenderCode(p Project, shot Shot) ([]byte, error) {
	return RenderMoment(p, shot, Word{})
}

// RenderMoment 画出这一词上的板、分屏、揭示、字幕和贴纸。
func RenderMoment(p Project, shot Shot, word Word) ([]byte, error) {
	return RenderMomentImages(p, shot, word, nil)
}

// RenderMomentImages 画出同一帧，并把排行榜格子里已有的静帧填进去。
func RenderMomentImages(p Project, shot Shot, word Word, pictures map[string][]byte) ([]byte, error) {
	w, h := frameSize(p.Brief.Aspect)
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	frame := compose(p, shot, word)
	frame.Pictures = pictures
	paintFrame(img, frame, shot.Region)
	return encodePNG(img)
}

// Annotate 把当前的词和贴纸画到一张已有静帧上。图解不开就改画代码画面。
func Annotate(raw []byte, p Project, shot Shot, word Word) ([]byte, error) {
	src, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return RenderMoment(p, shot, word)
	}
	b := src.Bounds()
	img := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, src.At(x, y))
		}
	}
	frame := compose(p, shot, word)
	drawCaption(img, frame, shot.Region)
	drawSticker(img, frame.Sticker)
	return encodePNG(img)
}

// CompositeInsert 把一张生成图叠到代码画面的下半部分。叠图解不开就返回原来的画面。
func CompositeInsert(basePNG, insert []byte) ([]byte, error) {
	base, err := png.Decode(bytes.NewReader(basePNG))
	if err != nil {
		return nil, err
	}
	over, _, err := image.Decode(bytes.NewReader(insert))
	if err != nil {
		return basePNG, nil
	}
	b := base.Bounds()
	img := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, base.At(x, y))
		}
	}
	iw := b.Dx() * 7 / 10
	ih := b.Dy() / 3
	if iw < 2 {
		iw = 2
	}
	if ih < 2 {
		ih = 2
	}
	blit(img, over, b.Min.X+(b.Dx()-iw)/2, b.Max.Y-ih-12, iw, ih)
	return encodePNG(img)
}

// RenderHTML 把这一镜最后一个词写成一页 HTML，给本机浏览器截图。
func RenderHTML(p Project, shot Shot) string {
	return RenderHTMLMoment(p, shot, Word{})
}

// RenderHTMLMoment 把这一词的构图写成 HTML。
func RenderHTMLMoment(p Project, shot Shot, word Word) string {
	frame := compose(p, shot, word)
	var b strings.Builder
	b.WriteString("<!doctype html><meta charset=utf-8><title>")
	b.WriteString(htmlText(frame.Title))
	b.WriteString("</title><style>")
	b.WriteString("body{margin:0;background:#12161c;color:#e8dcc4;font:16px sans-serif}")
	b.WriteString("main{min-height:100vh;padding:24px}")
	b.WriteString(".row{margin:8px 0;padding:8px 12px;background:#243048}")
	b.WriteString(".on{background:#78a89c;color:#12161c}")
	b.WriteString(".flash{background:#e8c460;color:#12161c}")
	b.WriteString(".split{display:flex;gap:8px}.split section{flex:1;min-height:160px;padding:12px;background:#243040}")
	b.WriteString(".split .on{background:#78a89c;color:#12161c}")
	b.WriteString(".caption{margin-top:16px}.caption mark{background:#e8c460;color:#12161c}")
	b.WriteString(".sticker{display:inline-block;margin-top:12px;padding:4px 8px;border:1px solid #e8dcc4}")
	b.WriteString("</style><body><main data-visual=\"")
	b.WriteString(htmlText(frame.Visual))
	b.WriteString("\" data-structure=\"")
	b.WriteString(htmlText(p.Structure))
	b.WriteString("\" data-word=\"")
	b.WriteString(htmlText(frame.Word))
	b.WriteString("\" data-speaker=\"")
	b.WriteString(htmlText(frame.Speaker))
	b.WriteString("\"><h1>")
	b.WriteString(htmlText(frame.Title))
	b.WriteString("</h1>")
	if frame.Left.Name != "" || frame.Right.Name != "" {
		b.WriteString("<div class=split><section")
		if frame.Left.On {
			b.WriteString(" class=on")
		}
		b.WriteString("><strong>")
		b.WriteString(htmlText(frame.Left.Name))
		b.WriteString("</strong><p>")
		b.WriteString(htmlText(frame.Left.Text))
		b.WriteString("</p></section><section")
		if frame.Right.On {
			b.WriteString(" class=on")
		}
		b.WriteString("><strong>")
		b.WriteString(htmlText(frame.Right.Name))
		b.WriteString("</strong><p>")
		b.WriteString(htmlText(frame.Right.Text))
		b.WriteString("</p></section></div>")
	}
	if frame.Product != "" || frame.Price != "" || frame.CTA != "" {
		b.WriteString("<p class=handoff>")
		b.WriteString(htmlText(strings.TrimSpace(frame.Product + " " + frame.Price + " " + frame.CTA)))
		b.WriteString("</p>")
	}
	if len(frame.Rows) > 0 {
		b.WriteString("<ol>")
		for _, row := range frame.Rows {
			b.WriteString("<li class=\"row")
			if row.Flash {
				b.WriteString(" flash")
			} else if row.On {
				b.WriteString(" on")
			}
			b.WriteString("\">")
			b.WriteString(htmlText(row.Label))
			b.WriteString("</li>")
		}
		b.WriteString("</ol>")
	}
	if len(frame.Caption) > 0 {
		b.WriteString("<p class=caption>")
		for _, piece := range frame.Caption {
			if piece.On {
				b.WriteString("<mark>")
				b.WriteString(htmlText(piece.Text))
				b.WriteString("</mark>")
				continue
			}
			b.WriteString(htmlText(piece.Text))
		}
		b.WriteString("</p>")
	}
	if frame.Sticker != "" {
		b.WriteString("<aside class=sticker>")
		b.WriteString(htmlText(frame.Sticker))
		b.WriteString("</aside>")
	}
	b.WriteString("</main></body>")
	return b.String()
}

type frame struct {
	Visual   string
	Title    string
	Word     string
	Speaker  string
	Rows     []boardRow
	Left     column
	Right    column
	Product  string
	Price    string
	CTA      string
	Caption  []piece
	Sticker  string
	Insert   bool
	Pictures map[string][]byte
}

type boardRow struct {
	ID    string
	Label string
	On    bool
	Flash bool
}

type column struct {
	Name string
	Text string
	On   bool
}

type piece struct {
	Text string
	On   bool
}

// compose 把工程、镜头和当前词收成一帧要画的内容。词是空的就用这一镜最后一个词。
func compose(p Project, shot Shot, word Word) frame {
	words := ShotWords(p, shot)
	if word.ID == "" && len(words) > 0 {
		word = words[len(words)-1]
	}
	out := frame{
		Visual: shot.Visual, Title: p.Title, Word: word.Text, Speaker: word.Speaker,
		Caption: captionPieces(p, word), Sticker: activeSticker(p, word),
	}
	switch shot.Visual {
	case "split":
		out.Left, out.Right = splitColumns(p, word)
	case "handoff":
		out.Product = p.Brief.Product
		out.Price = p.Brief.Price
		out.CTA = p.Brief.CTA
		if out.Product == "" {
			out.Product = shotText(p, shot)
		}
	default:
		if shot.Visual == "broll" && p.Structure != "ranking" {
			out.Insert = true
			break
		}
		out.Rows = boardRows(p, shot)
	}
	return out
}

// VisibleEntries 返回到这一镜为止该露出的行。最后一镜露出全部。没有条目时用每一行的句子代替。
func VisibleEntries(p Project, shot Shot) []Entry {
	entries := p.Entries
	if len(entries) == 0 {
		for _, line := range LineTexts(p) {
			entries = append(entries, Entry{ID: line.ID, Label: line.Text})
		}
	}
	if len(entries) == 0 || len(p.Shots) == 0 {
		return nil
	}
	n := shotIndex(p, shot) + 1
	if shot.ID == p.Shots[len(p.Shots)-1].ID {
		n = len(entries)
	}
	if n > len(entries) {
		n = len(entries)
	}
	if n < 0 {
		n = 0
	}
	return append([]Entry(nil), entries[:n]...)
}

// boardRows 把已露出的行收成板。口播里提到的行会点亮，最后一行在板或揭示上会闪一下。
func boardRows(p Project, shot Shot) []boardRow {
	entries := VisibleEntries(p, shot)
	if len(entries) == 0 {
		return nil
	}
	spoken := shotText(p, shot)
	rows := make([]boardRow, 0, len(entries))
	for i, entry := range entries {
		label := entry.Label
		on := strings.Contains(spoken, label) || strings.Contains(label, spoken)
		rows = append(rows, boardRow{
			ID: entry.ID, Label: label, On: on,
			Flash: i == len(entries)-1 && (shot.Visual == "reveal" || shot.Visual == "board"),
		})
	}
	return rows
}

// splitColumns 按说话人分成左右两栏，并标出当前词是谁在说。只取前两个说话人。
func splitColumns(p Project, word Word) (column, column) {
	var names []string
	text := map[string]string{}
	var buf = map[string][]Word{}
	for _, w := range p.Words {
		if w.Speaker == "" {
			continue
		}
		if _, ok := buf[w.Speaker]; !ok {
			names = append(names, w.Speaker)
		}
		if word.ID == "" || w.Start <= word.End {
			buf[w.Speaker] = append(buf[w.Speaker], w)
		}
	}
	for name, words := range buf {
		text[name] = joinTokens(words)
	}
	left, right := column{}, column{}
	if len(names) > 0 {
		left = column{Name: names[0], Text: text[names[0]], On: word.Speaker == names[0]}
	}
	if len(names) > 1 {
		right = column{Name: names[1], Text: text[names[1]], On: word.Speaker == names[1]}
	}
	return left, right
}

// captionPieces 列出这一行的词，并标出当前这个词。没有行号时只留这个词自己。
func captionPieces(p Project, word Word) []piece {
	if word.LineID == "" && word.Text == "" {
		return nil
	}
	var out []piece
	for _, w := range p.Words {
		if word.LineID != "" && w.LineID != word.LineID {
			continue
		}
		if word.LineID == "" && w.ID != word.ID {
			continue
		}
		out = append(out, piece{Text: w.Text, On: w.ID == word.ID})
	}
	return out
}

// activeSticker 返回已经说到的最新一条贴纸。还没说到的不算。
func activeSticker(p Project, word Word) string {
	var text string
	var at float64
	for _, sticker := range p.Stickers {
		for _, w := range p.Words {
			if w.ID != sticker.WordID {
				continue
			}
			if word.ID != "" && w.Start > word.Start {
				continue
			}
			if text == "" || w.Start >= at {
				text = sticker.Text
				at = w.Start
			}
		}
	}
	return text
}

// shotIndex 返回这一镜在工程里的位置。找不到就返回 0。
func shotIndex(p Project, shot Shot) int {
	for i := range p.Shots {
		if p.Shots[i].ID == shot.ID {
			return i
		}
	}
	return 0
}

// frameSize 按画幅给代码画面的像素。16:9 是 640×360，1:1 是 480，4:5 是 480×600，其余是 360×640。
func frameSize(aspect string) (int, int) {
	switch clean(aspect) {
	case "16:9":
		return 640, 360
	case "1:1":
		return 480, 480
	case "4:5":
		return 480, 600
	default:
		return 360, 640
	}
}

// paintFrame 把一帧画到图上，包括标题、分屏、产品、排行和字幕。
func paintFrame(img *image.RGBA, frame frame, region *Box) {
	b := img.Bounds()
	fill(img, b, visualColor(frame.Visual))
	face := textFace(18)
	title := textFace(26)
	y := b.Min.Y + 36
	y = drawLine(img, title, b.Min.X+20, y, b.Dx()-40, color.RGBA{232, 220, 196, 255}, frame.Title)
	y += 8
	if frame.Left.Name != "" || frame.Right.Name != "" {
		mid := b.Min.X + b.Dx()/2
		left := color.RGBA{36, 48, 64, 255}
		right := color.RGBA{48, 36, 52, 255}
		if frame.Left.On {
			left = color.RGBA{120, 168, 156, 255}
		}
		if frame.Right.On {
			right = color.RGBA{120, 168, 156, 255}
		}
		rect(img, b.Min.X+12, y, mid-8, y+b.Dy()/3, left)
		rect(img, mid+8, y, b.Max.X-12, y+b.Dy()/3, right)
		ink := color.RGBA{232, 220, 196, 255}
		drawLine(img, face, b.Min.X+20, y+28, mid-b.Min.X-36, ink, frame.Left.Name+" "+frame.Left.Text)
		drawLine(img, face, mid+16, y+28, b.Max.X-mid-28, ink, frame.Right.Name+" "+frame.Right.Text)
		y += b.Dy()/3 + 12
	}
	if frame.Product != "" || frame.Price != "" || frame.CTA != "" {
		rect(img, b.Min.X+24, y, b.Max.X-24, y+72, color.RGBA{232, 196, 96, 255})
		drawLine(img, title, b.Min.X+36, y+46, b.Dx()-72, color.RGBA{18, 22, 28, 255}, strings.TrimSpace(frame.Product+" "+frame.Price+" "+frame.CTA))
		y += 84
	}
	for _, row := range frame.Rows {
		col := color.RGBA{36, 48, 72, 255}
		ink := color.RGBA{232, 220, 196, 255}
		if row.Flash {
			col = color.RGBA{232, 196, 96, 255}
			ink = color.RGBA{18, 22, 28, 255}
		} else if row.On {
			col = color.RGBA{120, 168, 156, 255}
			ink = color.RGBA{18, 22, 28, 255}
		}
		well := 36
		rect(img, b.Min.X+24, y, b.Max.X-24-well-8, y+36, col)
		rect(img, b.Max.X-24-well, y, b.Max.X-24, y+36, color.RGBA{12, 14, 18, 255})
		if frame.Pictures != nil {
			if raw := frame.Pictures[row.ID]; len(raw) > 0 {
				if src, _, err := image.Decode(bytes.NewReader(raw)); err == nil {
					blit(img, src, b.Max.X-24-well, y, well, 36)
				}
			}
		}
		drawLine(img, face, b.Min.X+36, y+24, b.Dx()-well-80, ink, row.Label)
		y += 44
	}
	drawCaption(img, frame, region)
	drawSticker(img, frame.Sticker)
}

// drawCaption 在画面底部或区域下沿画出字幕，当前词用另一色。
func drawCaption(img *image.RGBA, frame frame, region *Box) {
	if len(frame.Caption) == 0 {
		return
	}
	b := img.Bounds()
	x, y := b.Min.X+16, b.Max.Y-56
	w := b.Dx() - 32
	if region != nil && region.W > 0 && region.H > 0 {
		x = b.Min.X + int(region.X*float64(b.Dx()))
		y = b.Min.Y + int((region.Y+region.H)*float64(b.Dy())) - 8
		w = int(region.W * float64(b.Dx()))
		if w < 40 {
			w = 40
		}
		if y > b.Max.Y-20 {
			y = b.Max.Y - 20
		}
	}
	rect(img, x, y-22, x+w, y+8, color.RGBA{12, 14, 18, 220})
	face := textFace(16)
	cursor := x + 6
	for _, piece := range frame.Caption {
		ink := color.RGBA{232, 220, 196, 255}
		if piece.On {
			ink = color.RGBA{232, 196, 96, 255}
		}
		cursor = drawInline(img, face, cursor, y, ink, piece.Text)
	}
}

// drawSticker 在右上角画一条贴纸。文字是空的就什么都不画。
func drawSticker(img *image.RGBA, text string) {
	if text == "" {
		return
	}
	b := img.Bounds()
	rect(img, b.Max.X-160, b.Min.Y+12, b.Max.X-12, b.Min.Y+44, color.RGBA{232, 220, 196, 255})
	drawLine(img, textFace(14), b.Max.X-152, b.Min.Y+34, 132, color.RGBA{18, 22, 28, 255}, text)
}

// visualColor 按画面种类给底色。认不出的用近黑。
func visualColor(visual string) color.RGBA {
	switch visual {
	case "board":
		return color.RGBA{24, 32, 48, 255}
	case "split":
		return color.RGBA{28, 24, 36, 255}
	case "reveal":
		return color.RGBA{48, 24, 28, 255}
	case "handoff":
		return color.RGBA{42, 32, 18, 255}
	default:
		return color.RGBA{18, 22, 28, 255}
	}
}

// fill 把一块矩形涂成纯色。
func fill(img *image.RGBA, b image.Rectangle, col color.RGBA) {
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetRGBA(x, y, col)
		}
	}
}

// rect 在图里画一块矩形，超出画面的部分会裁掉。
func rect(img *image.RGBA, x0, y0, x1, y1 int, col color.RGBA) {
	b := img.Bounds()
	if x0 < b.Min.X {
		x0 = b.Min.X
	}
	if y0 < b.Min.Y {
		y0 = b.Min.Y
	}
	if x1 > b.Max.X {
		x1 = b.Max.X
	}
	if y1 > b.Max.Y {
		y1 = b.Max.Y
	}
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			img.SetRGBA(x, y, col)
		}
	}
}

// drawLine 按最大宽度把文字画成一行或多行。没有字体时用一条色块代替。返回下一行的纵坐标。
func drawLine(img *image.RGBA, face font.Face, x, y, maxW int, col color.Color, text string) int {
	if text == "" {
		return y
	}
	if face == nil {
		width := len([]rune(text)) * 8
		if width > maxW {
			width = maxW
		}
		rect(img, x, y-10, x+width, y-4, color.RGBA{232, 220, 196, 255})
		return y + 18
	}
	d := &font.Drawer{Dst: img, Src: image.NewUniform(col), Face: face}
	var line []rune
	flush := func() {
		if len(line) == 0 {
			return
		}
		d.Dot = fixed.P(x, y)
		d.DrawString(string(line))
		y += 22
		line = line[:0]
	}
	for _, r := range text {
		line = append(line, r)
		if d.MeasureString(string(line)).Ceil() > maxW && len(line) > 1 {
			last := line[len(line)-1]
			line = line[:len(line)-1]
			flush()
			line = append(line, last)
		}
	}
	flush()
	return y
}

// drawInline 在同一行接着画文字，返回画完后的横坐标。没有字体时用色块占位。
func drawInline(img *image.RGBA, face font.Face, x, y int, col color.Color, text string) int {
	if text == "" {
		return x
	}
	if face == nil {
		width := len([]rune(text)) * 8
		rect(img, x, y-12, x+width, y-6, color.RGBA{232, 220, 196, 255})
		return x + width
	}
	d := &font.Drawer{Dst: img, Src: image.NewUniform(col), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(text)
	return d.Dot.X.Ceil()
}

// blend 按透明度把一张图贴进目标。完全透明的像素留着原来的画面。
func blend(dst *image.RGBA, src image.Image, x, y, w, h int) {
	sb := src.Bounds()
	if sb.Dx() == 0 || sb.Dy() == 0 || w <= 0 || h <= 0 {
		return
	}
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			dx, dy := x+xx, y+yy
			if !image.Pt(dx, dy).In(dst.Bounds()) {
				continue
			}
			sx := sb.Min.X + xx*sb.Dx()/w
			sy := sb.Min.Y + yy*sb.Dy()/h
			sr, sg, sbC, sa := src.At(sx, sy).RGBA()
			if sa == 0 {
				continue
			}
			dr, dg, db, _ := dst.At(dx, dy).RGBA()
			inv := 65535 - sa
			dst.SetRGBA(dx, dy, color.RGBA{
				R: uint8(((sr*sa + dr*inv) / 65535) >> 8),
				G: uint8(((sg*sa + dg*inv) / 65535) >> 8),
				B: uint8(((sbC*sa + db*inv) / 65535) >> 8),
				A: 255,
			})
		}
	}
}

// blit 把一张图缩放贴进目标，不处理透明度。
func blit(dst *image.RGBA, src image.Image, x, y, w, h int) {
	sb := src.Bounds()
	if sb.Dx() == 0 || sb.Dy() == 0 || w <= 0 || h <= 0 {
		return
	}
	for yy := 0; yy < h; yy++ {
		for xx := 0; xx < w; xx++ {
			sx := sb.Min.X + xx*sb.Dx()/w
			sy := sb.Min.Y + yy*sb.Dy()/h
			dst.Set(x+xx, y+yy, src.At(sx, sy))
		}
	}
}

// encodePNG 把图编成 PNG 字节。
func encodePNG(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// htmlText 转义 &、<、> 和引号。
func htmlText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

// DownloadCommand 拼本机 yt-dlp 下载参考地址的参数，不拉播放列表。
func DownloadCommand(bin, pageURL, out string) []string {
	return []string{bin, "-o", out, "--no-playlist", pageURL}
}

// StampCommand 拼 whisperx 的参数，把词级 JSON 写到输出目录，文件名沿用媒体主名。
func StampCommand(bin, media, outDir string) []string {
	return []string{bin, media, "--model", "small", "--output_format", "json", "--output_dir", outDir, "--compute_type", "int8"}
}

// RegionCommand 拼本机区域程序的参数，把区域写到指定文件。
func RegionCommand(bin, media, out string) []string {
	return []string{bin, media, "--regions", out}
}

// BrowserCommand 拼当前无头 Chrome 或 Edge 的参数，把一页 HTML 截成 PNG。
func BrowserCommand(bin, htmlPath, pngPath string) []string {
	return []string{bin, "--headless=new", "--disable-gpu", "--window-size=720,1280", "--screenshot=" + pngPath, htmlPath}
}

// CompositeMatte 把抠出的图按透明度贴进镜头区域。透明像素留着原来的画面。抠图解不开就返回原来的画面。
func CompositeMatte(basePNG, cutout []byte, box *Box) ([]byte, error) {
	base, err := png.Decode(bytes.NewReader(basePNG))
	if err != nil {
		return nil, err
	}
	over, _, err := image.Decode(bytes.NewReader(cutout))
	if err != nil {
		return basePNG, nil
	}
	b := base.Bounds()
	img := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.Set(x, y, base.At(x, y))
		}
	}
	x, y, w, h := b.Min.X, b.Min.Y, b.Dx()*2/3, b.Dy()*2/3
	if w < 2 {
		w = 2
	}
	if h < 2 {
		h = 2
	}
	x += (b.Dx() - w) / 2
	if box != nil && box.W > 0 && box.H > 0 {
		x = b.Min.X + int(box.X*float64(b.Dx()))
		y = b.Min.Y + int(box.Y*float64(b.Dy()))
		w = int(box.W * float64(b.Dx()))
		h = int(box.H * float64(b.Dy()))
		if w < 1 {
			w = 1
		}
		if h < 1 {
			h = 1
		}
	}
	blend(img, over, x, y, w, h)
	return encodePNG(img)
}

// ParseRegions 读区域程序写出的 JSON，取出 regions 数组。
func ParseRegions(raw []byte) ([]Box, error) {
	var doc struct {
		Regions []Box `json:"regions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	return doc.Regions, nil
}

// ParseCrop 把 ffmpeg cropdetect 的 crop=宽:高:x:y 收成 0 到 1 的区域。宽或高不是正数，或画面尺寸无效，就返回假。
func ParseCrop(spec string, width, height int) (Box, bool) {
	spec = strings.TrimSpace(spec)
	if i := strings.Index(spec, "crop="); i >= 0 {
		spec = spec[i+len("crop="):]
	}
	if i := strings.IndexAny(spec, " \t"); i >= 0 {
		spec = spec[:i]
	}
	parts := strings.Split(spec, ":")
	if len(parts) != 4 || width <= 0 || height <= 0 {
		return Box{}, false
	}
	vals := [4]int{}
	for i, part := range parts {
		n := 0
		if part == "" {
			return Box{}, false
		}
		for _, r := range part {
			if r < '0' || r > '9' {
				return Box{}, false
			}
			n = n*10 + int(r-'0')
		}
		vals[i] = n
	}
	if vals[0] <= 0 || vals[1] <= 0 {
		return Box{}, false
	}
	return Box{
		X: float64(vals[2]) / float64(width), Y: float64(vals[3]) / float64(height),
		W: float64(vals[0]) / float64(width), H: float64(vals[1]) / float64(height),
		Label: "画面",
	}, true
}
