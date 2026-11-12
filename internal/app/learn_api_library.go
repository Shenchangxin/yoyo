// 文库和阅读的桌面接口：建库、导入、检索、按单元读，以及阅读台上的位置、书签和批注。
// 这里只做参数转接，规则都在 internal/learn。从链接存网页是唯一在这里做的事：抓取要出网，学习库自己不碰网络。
// 形状见 docs/architecture/learn.md。

package app

import (
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Shenchangxin/yoyo/internal/learn"
)

const (
	// learnPageBytes 是从链接存网页时最多读多少字节。
	learnPageBytes = 8 << 20
	// learnSelectionRunes 是记下的选中文字最多多少个符文。
	learnSelectionRunes = 1200
)

// learnLibraryCall 处理文库和阅读的方法。ok 为假表示这个方法不归它管。
func (a *App) learnLibraryCall(method string, params map[string]any) (any, bool, error) {
	s := a.Learn
	lib := strParam(params, "library_id")
	doc := strParam(params, "document_id")
	mat := strParam(params, "material_id")
	switch method {
	case "learn.libraries":
		return s.Libraries(), true, nil
	case "learn.library.create":
		out, err := s.CreateLibrary(strParam(params, "name"), strParam(params, "kind"), strParam(params, "path"))
		if err == nil && strParam(params, "description") != "" {
			description := strParam(params, "description")
			if err = s.UpdateLibrary(out.ID, nil, &description); err == nil {
				out, err = s.Library(out.ID)
			}
		}
		return out, true, err
	case "learn.library.update":
		var name, description *string
		if hasParam(params, "name") {
			v := strParam(params, "name")
			name = &v
		}
		if hasParam(params, "description") {
			v := strParam(params, "description")
			description = &v
		}
		return okMap(), true, s.UpdateLibrary(strParam(params, "id"), name, description)
	case "learn.library.delete":
		return okMap(), true, s.DeleteLibrary(strParam(params, "id"))
	case "learn.library.embed":
		return okMap(), true, s.SetEmbedModel(strParam(params, "id"), strParam(params, "model"))
	case "learn.documents":
		out, err := s.Documents(lib)
		return out, true, err
	case "learn.import":
		paths := strSlice(params, "paths")
		if path := strParam(params, "path"); path != "" {
			paths = append(paths, path)
		}
		out, err := s.ImportPaths(lib, paths)
		return out, true, err
	case "learn.import.url":
		out, err := a.learnImportURL(lib, strParam(params, "url"))
		return out, true, err
	case "learn.rebuild":
		return okMap(), true, s.Rebuild(lib)
	case "learn.document.remove":
		return okMap(), true, s.RemoveDocument(lib, doc)
	case "learn.document.rename":
		return okMap(), true, s.RenameDocument(lib, doc, strParam(params, "title"))
	case "learn.document.blob":
		name, mimeType, data, err := s.DocumentBlob(lib, doc)
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"name": name, "mime": mimeType, "base64": base64.StdEncoding.EncodeToString(data)}, true, nil
	case "learn.search":
		hits, msg, err := s.SearchIn(lib, strParam(params, "query"), intParam(params, "limit"))
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"hits": hits, "message": msg}, true, nil
	case "learn.read":
		out, err := s.ReadUnit(lib, doc, intParam(params, "page"), strParam(params, "source_hash"))
		return out, true, err
	case "learn.materials":
		return s.Materials(), true, nil
	case "learn.material.open":
		out, err := s.OpenMaterial(lib, doc)
		return out, true, err
	case "learn.material.close":
		return okMap(), true, s.CloseMaterial(mat)
	case "learn.material.position":
		return okMap(), true, s.SavePosition(mat, intParam(params, "locator"), floatParam(params, "percent"))
	case "learn.annotations":
		return s.Annotations(mat), true, nil
	case "learn.annotate":
		start := -1
		if hasParam(params, "start") {
			start = intParam(params, "start")
		}
		out, err := s.Annotate(learn.AnnotateRequest{
			MaterialID: mat, Kind: strParam(params, "kind"), Color: strParam(params, "color"), Locator: intParam(params, "locator"),
			Quote: strParam(params, "quote"), Note: strParam(params, "note"), Start: start,
			Prefix: strParam(params, "prefix"), Suffix: strParam(params, "suffix"),
		})
		return out, true, err
	case "learn.annotation.update":
		text := func(key string) *string {
			if !hasParam(params, key) {
				return nil
			}
			v := strParam(params, key)
			return &v
		}
		out, err := s.UpdateAnnotation(strParam(params, "id"), text("note"), text("color"), text("kind"))
		return out, true, err
	case "learn.annotation.delete":
		return okMap(), true, s.DeleteAnnotation(strParam(params, "id"))
	case "learn.bookmark.toggle":
		on, marks, err := s.ToggleBookmark(mat, intParam(params, "locator"), strParam(params, "label"))
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"bookmarked": on, "bookmarks": marks}, true, nil
	case "learn.locate":
		hits, msg, err := s.Locate(mat, strParam(params, "query"))
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"hits": hits, "message": msg}, true, nil
	case "learn.material.export":
		name, body, err := s.ExportMaterial(mat)
		if err != nil {
			return nil, true, err
		}
		return map[string]any{"filename": name, "markdown": body}, true, nil
	case "learn.viewport":
		a.setLearnViewport(strParam(params, "session"), learnViewport{
			MaterialID: mat, Locator: intParam(params, "locator"), Selection: clipLearn(strings.TrimSpace(strParam(params, "selection")), learnSelectionRunes),
		})
		return okMap(), true, nil
	}
	return nil, false, nil
}

// learnImportURL 把一个网页的正文存进文库。只认 http 和 https，最多读 8 MiB，只收网页和纯文本。
// libraryID 是要存进去的 indexed 库，rawURL 是网页地址。抓取由操作者在界面上发起，模型的工具里没有这一项。
func (a *App) learnImportURL(libraryID, rawURL string) (learn.Document, error) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return learn.Document{}, fmt.Errorf("give a full web address starting with http or https")
	}
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return learn.Document{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; yoyo-reader)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9,*/*;q=0.5")
	res, err := client.Do(req)
	if err != nil {
		return learn.Document{}, fmt.Errorf("the page could not be fetched: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return learn.Document{}, fmt.Errorf("the page answered with status %d", res.StatusCode)
	}
	contentType := res.Header.Get("Content-Type")
	kind := strings.ToLower(contentType)
	if kind != "" && !strings.Contains(kind, "html") && !strings.Contains(kind, "text/") {
		return learn.Document{}, fmt.Errorf("only web pages and plain text can be saved from a link; download the file and import it instead")
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, learnPageBytes+1))
	if err != nil {
		return learn.Document{}, fmt.Errorf("the page could not be read: %v", err)
	}
	if len(body) > learnPageBytes {
		return learn.Document{}, fmt.Errorf("the page is larger than 8 MiB")
	}
	title, text := "", string(body)
	if kind == "" || strings.Contains(kind, "html") {
		title, text = learn.PageFromHTML(body, contentType, u.String())
	}
	if strings.TrimSpace(text) == "" {
		return learn.Document{}, fmt.Errorf("no readable text was found on that page")
	}
	if strings.TrimSpace(title) == "" {
		title = strings.Trim(u.Host+u.Path, "/")
	}
	return a.Learn.ImportText(libraryID, title, text, u.String())
}
