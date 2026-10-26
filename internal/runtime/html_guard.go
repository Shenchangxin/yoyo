// 拒绝会把正文留在 </html> 之后的 HTML 追加，并拒绝改写封存的 game-world 可玩文件。
// 不读写文件本身。write_file 在追加和覆盖时调用这里。覆盖只警告，追加才拒绝。

package runtime

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// htmlCloseTag 匹配任意大小写的文档结束标签。
var htmlCloseTag = regexp.MustCompile(`(?i)</html\s*>`)

// isHTMLDocPath 判断路径扩展名是不是 .html 或 .htm。
// rel 取扩展名并转小写，是 .html 或 .htm 时返回 true。
func isHTMLDocPath(rel string) bool {
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".html", ".htm":
		return true
	default:
		return false
	}
}

// htmlTailAfterClose 返回最后一个 </html> 之后的非空白文本，没有则返回空串。那里的文字在文档外面：浏览器把它渲染在页面之后，下一段追加也会落在它后面。
// doc 查找全部结束标签，没有则返回空串，否则返回最后一处之后去掉空白的文本。
func htmlTailAfterClose(doc string) string {
	locs := htmlCloseTag.FindAllStringIndex(doc, -1)
	if len(locs) == 0 {
		return ""
	}
	return strings.TrimSpace(doc[locs[len(locs)-1][1]:])
}

// htmlTailSnippet 把散落尾巴收成一行，超过 80 个符文则截断，供错误信息使用。
// tail 压成一行，超过 80 个符文则截断。
func htmlTailSnippet(tail string) string {
	r := []rune(strings.Join(strings.Fields(tail), " "))
	if len(r) > 80 {
		return string(r[:80]) + "..."
	}
	return string(r)
}

// 分段重写 HTML 曾反复提前结束文档：一段以 </script></body></html> 开头，下一段就变成页面之后的文字。页面会静默坏掉，而代理自己后来的检查仍能在残骸上通过。追加若会把文字留在 </html> 之后则拒绝；覆盖允许，只给警告。
// rel 先交给 isHTMLDocPath，不是 HTML 时返回空串，否则写进拒绝句。
// want 交给 htmlTailAfterClose，尾巴为空时返回空串。
func htmlAppendRefusal(rel, want string) string {
	if !isHTMLDocPath(rel) {
		return ""
	}
	tail := htmlTailAfterClose(want)
	if tail == "" {
		return ""
	}
	return "refusing to append to " + rel + ": the result would put text after </html> (\"" + htmlTailSnippet(tail) + "\"), which is outside the document. " +
		"Put the new code before the closing </script> with str_replace, or rewrite the whole file with write_file mode=overwrite."
}

// sealedGameWorldWrite 拒绝改写残留的 game-world 可玩文件，使代理不能重写封存的 HTML 或 runtime.js。
// 路径里要有 game-worlds。其下直接的 index.html，以及 playable 里的 runtime.js、index.html、game.json，被拒绝。
// rel 转成斜杠后按段查找 game-worlds，再看其后的 index.html 或 playable 下的指定文件。
func sealedGameWorldWrite(rel string) error {
	slash := filepath.ToSlash(rel)
	parts := strings.Split(slash, "/")
	idx := -1
	for i, p := range parts {
		if strings.EqualFold(p, "game-worlds") {
			idx = i
			break
		}
	}
	if idx < 0 || idx+1 >= len(parts) {
		return nil
	}
	rest := parts[idx+2:]
	if len(rest) == 1 && strings.EqualFold(rest[0], "index.html") {
		return fmt.Errorf("refusing write %s: game-world HTML is not a supported product", rel)
	}
	if len(rest) >= 2 && strings.EqualFold(rest[0], "playable") {
		leaf := strings.ToLower(rest[len(rest)-1])
		switch leaf {
		case "runtime.js", "index.html", "game.json":
			return fmt.Errorf("refusing write %s: sealed playable is not writable", rel)
		}
	}
	return nil
}

// htmlOverwriteWarning 在写好的 HTML 于 </html> 之后还有文字时返回警告，文档干净结束则返回空串。
// rel 先交给 isHTMLDocPath，不是 HTML 时返回空串，否则写进警告句。
// want 交给 htmlTailAfterClose，尾巴为空时返回空串。
func htmlOverwriteWarning(rel, want string) string {
	if !isHTMLDocPath(rel) {
		return ""
	}
	tail := htmlTailAfterClose(want)
	if tail == "" {
		return ""
	}
	return "warning: " + rel + " has text after </html> (\"" + htmlTailSnippet(tail) + "\"). Browsers render it outside the page; move it inside the document."
}
