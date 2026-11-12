// 批量导入题目：先预览，再提交。预览把 CSV、TSV 或 JSON 读成一行行题目，逐行校验并标出重复，合格的暂存在 practice_imports 里；
// 提交按暂存的内容写入题库，同一个令牌只写一次。读文件和解析都不持有库锁。
// 题目的校验在 practice_draft.go，写入走 bank.go 的 bankInsert。这里不排复习。

package learn

import (
	"crypto/rand"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
	"unicode/utf8"
)

const (
	maxImportBytes = 2 << 20
	maxImportRows  = 500
	// importKeep 是暂存的预览留多久。更早的在下一次预览时删掉。
	importKeep = 24 * time.Hour
)

// ImportRequest 是预览导入的入参。Path 是这台机器上的 .csv、.tsv 或 .json 文件，Text 是粘贴的内容，两者给一个，都给时用 Path。
// Format 是 csv、tsv 或 json。为空时文件看扩展名；粘贴的内容以方括号或花括号开头算 json，否则算 csv。
type ImportRequest struct {
	Path   string `json:"path"`
	Text   string `json:"text"`
	Format string `json:"format"`
}

// ImportRow 是预览里的一行。Line 是这一行在文件里的行号，表头是第 1 行；JSON 里是第几个对象。
// Question 是校验之后的题目，Error 是这一行为什么不能导入，两者只有一个有值。
// Duplicate 为真表示这道题已经在题库里，或者在这份文件里出现过，提交时不会导入。
type ImportRow struct {
	Line      int            `json:"line"`
	Question  *DraftQuestion `json:"question,omitempty"`
	Error     string         `json:"error,omitempty"`
	Duplicate bool           `json:"duplicate,omitempty"`
}

// ImportPreview 是一次预览的结果。Valid 是提交时会导入的行数，Invalid 是校验没过的行数，Duplicates 是重复的行数，三者之和是总行数。
// Token 交给 CommitImport。
type ImportPreview struct {
	Token      string      `json:"token"`
	Rows       []ImportRow `json:"rows"`
	Valid      int         `json:"valid"`
	Invalid    int         `json:"invalid"`
	Duplicates int         `json:"duplicates"`
}

// ImportResult 是一次提交的结果。Skipped 是提交时发现已经在题库里而没写的题数。
type ImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

// importLine 是从文件里读出的一行，还没校验。Err 非空表示这一行连读都读不出来。
type importLine struct {
	Line  int
	Draft DraftQuestion
	Err   string
}

// stagedImport 是暂存在 practice_imports 里的内容。提交之后 Items 清空，Result 记下当时的结果，好让重复提交拿到同一个答复。
type stagedImport struct {
	Items  []stagedItem  `json:"items"`
	Result *ImportResult `json:"result,omitempty"`
}

// stagedItem 是暂存的一道题和它的来源键。
type stagedItem struct {
	Key      string        `json:"key"`
	Question DraftQuestion `json:"question"`
}

// PreviewImport 读一份题目文件或一段粘贴的内容，逐行校验，返回每一行的结果，并把合格且不重复的行暂存起来等提交。
// 文件和内容最大 2 MiB，最多 500 行。CSV 和 TSV 的第一行是表头，列名见 draftField；JSON 是对象数组，键名相同。
// 重复有两种：来源键（import: 加内容指纹）已经在题库里或在这份文件里出现过，或者题库里已有题干和题型都相同的题。
// 超过 24 小时的暂存在这里顺手删掉。
// req 的用法见 ImportRequest。
func (s *Store) PreviewImport(req ImportRequest) (ImportPreview, error) {
	if s == nil {
		return ImportPreview{}, fmt.Errorf("learn store unavailable")
	}
	text, format, filename, err := importSource(req)
	if err != nil {
		return ImportPreview{}, err
	}
	var lines []importLine
	switch format {
	case "json":
		lines, err = parseJSONItems(text)
	case "tsv":
		lines, err = parseDelimited(text, '\t')
	default:
		lines, err = parseDelimited(text, sniffComma(text))
	}
	if err != nil {
		return ImportPreview{}, err
	}
	if len(lines) == 0 {
		return ImportPreview{}, fmt.Errorf("no questions found in the import")
	}
	if len(lines) > maxImportRows {
		return ImportPreview{}, fmt.Errorf("an import holds at most %d questions; split the file", maxImportRows)
	}
	out := ImportPreview{Rows: make([]ImportRow, 0, len(lines))}
	keys := make([]string, len(lines))
	for i, line := range lines {
		row := ImportRow{Line: line.Line, Error: line.Err}
		if row.Error == "" {
			draft, err := normalizeDraft(line.Draft)
			if err != nil {
				row.Error = err.Error()
			} else {
				row.Question = &draft
				keys[i] = "import:" + contentKey(draft)
			}
		}
		out.Rows = append(out.Rows, row)
	}
	token, err := importToken()
	if err != nil {
		return ImportPreview{}, err
	}
	now := s.clock()
	staged := stagedImport{Items: []stagedItem{}}
	seen := map[string]bool{}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range out.Rows {
		row := &out.Rows[i]
		if row.Question == nil {
			out.Invalid++
			continue
		}
		// 同一份文件再导一次靠来源键认出来；别处已经录过的同一道题靠题干和题型认出来。
		var existing string
		if seen[keys[i]] || s.db.QueryRow(`SELECT id FROM questions WHERE origin_key = ? OR (stem = ? AND qtype = ?) LIMIT 1`,
			keys[i], row.Question.Stem, row.Question.Type).Scan(&existing) == nil {
			row.Duplicate = true
			out.Duplicates++
			continue
		}
		seen[keys[i]] = true
		out.Valid++
		staged.Items = append(staged.Items, stagedItem{Key: keys[i], Question: *row.Question})
	}
	payload, err := json.Marshal(staged)
	if err != nil {
		return ImportPreview{}, err
	}
	err = s.inTx(func(tx *sql.Tx) error {
		if _, err := tx.Exec(`DELETE FROM practice_imports WHERE created_at < ?`, formatTime(now.Add(-importKeep))); err != nil {
			return err
		}
		_, err := tx.Exec(`INSERT INTO practice_imports (token, filename, payload_json, created_at, committed) VALUES (?, ?, ?, ?, 0)`,
			token, filename, string(payload), formatTime(now))
		return err
	})
	if err != nil {
		return ImportPreview{}, err
	}
	out.Token = token
	return out, nil
}

// CommitImport 把一次预览暂存的题写进题库，来源记成 import。token 是预览给的令牌，不存在或已过期时返回错误。
// tags 加在每道题自己的标签后面。asMistakes 为真时这些题记成错题，一天后到期。
// 同一个令牌再提交一次不会再写，返回第一次的结果。
func (s *Store) CommitImport(token string, tags []string, asMistakes bool) (ImportResult, error) {
	if s == nil {
		return ImportResult{}, fmt.Errorf("learn store unavailable")
	}
	token = strings.TrimSpace(token)
	now := s.clock()
	s.mu.Lock()
	var payload string
	var committed int
	err := s.db.QueryRow(`SELECT payload_json, committed FROM practice_imports WHERE token = ?`, token).Scan(&payload, &committed)
	if err != nil {
		s.mu.Unlock()
		return ImportResult{}, fmt.Errorf("this import preview was not found or has expired; preview the file again")
	}
	var staged stagedImport
	if err := json.Unmarshal([]byte(payload), &staged); err != nil {
		s.mu.Unlock()
		return ImportResult{}, fmt.Errorf("this import preview is unreadable; preview the file again")
	}
	if committed != 0 {
		s.mu.Unlock()
		if staged.Result == nil {
			return ImportResult{}, nil
		}
		return *staged.Result, nil
	}
	var result ImportResult
	err = s.inTx(func(tx *sql.Tx) error {
		for _, item := range staged.Items {
			entry := bankEntryOf(item.Question, "import", item.Key)
			entry.Tags = mergeTags(item.Question.Tags, tags)
			entry.Mistake = asMistakes
			_, added, err := bankInsert(tx, entry, now)
			if err != nil {
				return err
			}
			if added {
				result.Imported++
			} else {
				result.Skipped++
			}
		}
		done, _ := json.Marshal(stagedImport{Items: []stagedItem{}, Result: &result})
		_, err := tx.Exec(`UPDATE practice_imports SET committed = 1, payload_json = ? WHERE token = ?`, string(done), token)
		return err
	})
	s.mu.Unlock()
	if err != nil {
		return ImportResult{}, err
	}
	if result.Imported > 0 {
		s.notify("practice", "", "imported")
	}
	return result, nil
}

// importToken 生成一个随机的预览令牌。
func importToken() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return "imp-" + hex.EncodeToString(raw[:]), nil
}

// importSource 取出要导入的文字、格式和文件名。req.Path 非空时读那个文件，扩展名只能是 .csv、.tsv 或 .json，大小不超过 2 MiB；
// 否则用 req.Text，文件名为空。req.Format 非空时以它为准，只能是 csv、tsv 或 json。
func importSource(req ImportRequest) (text, format, filename string, err error) {
	format = strings.ToLower(strings.TrimSpace(req.Format))
	switch format {
	case "", "csv", "tsv", "json":
	default:
		return "", "", "", fmt.Errorf("format must be csv, tsv, or json")
	}
	var raw []byte
	if path := strings.TrimSpace(req.Path); path != "" {
		ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
		if ext != "csv" && ext != "tsv" && ext != "json" {
			return "", "", "", fmt.Errorf("only .csv, .tsv, and .json files can be imported")
		}
		info, statErr := os.Stat(path)
		if statErr != nil || info.IsDir() {
			return "", "", "", fmt.Errorf("the import file could not be opened")
		}
		if info.Size() > maxImportBytes {
			return "", "", "", fmt.Errorf("the import file is larger than 2 MiB; split it")
		}
		if raw, err = os.ReadFile(path); err != nil {
			return "", "", "", fmt.Errorf("the import file could not be read")
		}
		filename = filepath.Base(path)
		if format == "" {
			format = ext
		}
	} else {
		if len(req.Text) > maxImportBytes {
			return "", "", "", fmt.Errorf("the pasted text is larger than 2 MiB; split it")
		}
		raw = []byte(req.Text)
	}
	text, err = decodeImportText(raw)
	if err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(text) == "" {
		return "", "", "", fmt.Errorf("nothing to import: give a file path or paste the questions")
	}
	if format == "" {
		format = "csv"
		if lead := strings.TrimSpace(text); strings.HasPrefix(lead, "[") || strings.HasPrefix(lead, "{") {
			format = "json"
		}
	}
	return text, format, filename, nil
}

// decodeImportText 把文件的字节读成文字。UTF-8 的 BOM 去掉；带 BOM 的 UTF-16 转成 UTF-8；其余必须是合法的 UTF-8，否则返回错误。
func decodeImportText(raw []byte) (string, error) {
	switch {
	case len(raw) >= 3 && raw[0] == 0xEF && raw[1] == 0xBB && raw[2] == 0xBF:
		raw = raw[3:]
	case len(raw) >= 2 && ((raw[0] == 0xFF && raw[1] == 0xFE) || (raw[0] == 0xFE && raw[1] == 0xFF)):
		little := raw[0] == 0xFF
		units := make([]uint16, 0, len(raw)/2)
		for i := 2; i+1 < len(raw); i += 2 {
			if little {
				units = append(units, uint16(raw[i])|uint16(raw[i+1])<<8)
			} else {
				units = append(units, uint16(raw[i])<<8|uint16(raw[i+1]))
			}
		}
		return string(utf16.Decode(units)), nil
	}
	if !utf8.Valid(raw) {
		return "", fmt.Errorf("the import is not UTF-8 text; save the file as UTF-8 (in Excel: CSV UTF-8) and try again")
	}
	return string(raw), nil
}

// sniffComma 看 CSV 的第一行用什么分隔。没有逗号而有制表符时用制表符，没有逗号而有分号时用分号，其余用逗号。
func sniffComma(text string) rune {
	head := text
	if cut := strings.IndexAny(head, "\r\n"); cut >= 0 {
		head = head[:cut]
	}
	switch {
	case strings.Contains(head, ","):
		return ','
	case strings.Contains(head, "\t"):
		return '\t'
	case strings.Contains(head, ";"):
		return ';'
	}
	return ','
}

// parseDelimited 把 CSV 或 TSV 读成一行行题目。comma 是分隔符。第一行是表头，列名按 draftField 认，必须有题干列。
// 全空的行跳过。读不出来的行留下行号和原因，不让整份文件失败。A 到 H 的列有内容时以它们为选项，否则用整列选项拆出来的。
func parseDelimited(text string, comma rune) ([]importLine, error) {
	r := csv.NewReader(strings.NewReader(text))
	r.Comma = comma
	r.FieldsPerRecord = -1
	r.LazyQuotes = true
	// 制表符本身是空白。分隔符是它时不能去掉字段开头的空白，否则空单元格会被吞掉。
	r.TrimLeadingSpace = comma != '\t'
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("the import is empty; the first row must be a header such as question,type,A,B,C,D,answer")
	}
	type column struct{ field, label string }
	cols := make([]column, len(header))
	hasStem := false
	for i, name := range header {
		field, label := draftField(name)
		cols[i] = column{field, label}
		if field == "stem" {
			hasStem = true
		}
	}
	if !hasStem {
		return nil, fmt.Errorf("the header row has no question column; name one column question, stem, 题目, or 题干")
	}
	var out []importLine
	for len(out) <= maxImportRows {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			var parseErr *csv.ParseError
			if !errors.As(err, &parseErr) {
				return nil, err
			}
			out = append(out, importLine{Line: parseErr.StartLine, Err: "this row could not be read: " + parseErr.Err.Error()})
			continue
		}
		line, _ := r.FieldPos(0)
		var d DraftQuestion
		lettered := map[string]string{}
		blank := true
		set := func(dst *string, value string) {
			if *dst == "" {
				*dst = value
			}
		}
		for i, cell := range record {
			cell = strings.TrimSpace(cell)
			if cell == "" || i >= len(cols) {
				continue
			}
			blank = false
			switch cols[i].field {
			case "stem":
				set(&d.Stem, cell)
			case "qtype":
				set(&d.Type, cell)
			case "answer":
				set(&d.Answer, cell)
			case "explanation":
				set(&d.Explanation, cell)
			case "difficulty":
				set(&d.Difficulty, cell)
			case "tags":
				d.Tags = append(d.Tags, splitTags(cell)...)
			case "options":
				if len(d.Options) == 0 {
					d.Options = splitOptions(cell)
				}
			case "option":
				lettered[cols[i].label] = cell
			}
		}
		if blank {
			continue
		}
		if len(lettered) > 0 {
			d.Options = letteredOptions(lettered)
		}
		out = append(out, importLine{Line: line, Draft: d})
	}
	return out, nil
}

// parseJSONItems 把 JSON 读成一个个题目。text 是对象数组，或者把数组放在 questions 或 items 键下的对象。
// 不是对象的项留下序号和原因。Line 是从 1 起的序号。
func parseJSONItems(text string) ([]importLine, error) {
	list, err := jsonItemList([]byte(strings.TrimSpace(text)))
	if err != nil {
		return nil, err
	}
	out := make([]importLine, 0, len(list))
	for i, raw := range list {
		var obj map[string]json.RawMessage
		if json.Unmarshal(raw, &obj) != nil {
			out = append(out, importLine{Line: i + 1, Err: "this item is not a JSON object"})
			continue
		}
		out = append(out, importLine{Line: i + 1, Draft: draftFromJSON(obj)})
	}
	return out, nil
}

// jsonItemList 从 data 里取出题目数组。data 是数组时直接用；是对象时取它的 questions 或 items 键。其余返回错误。
func jsonItemList(data []byte) ([]json.RawMessage, error) {
	var list []json.RawMessage
	if json.Unmarshal(data, &list) == nil {
		return list, nil
	}
	var wrap map[string]json.RawMessage
	if json.Unmarshal(data, &wrap) == nil {
		for _, key := range []string{"questions", "items"} {
			if raw, ok := wrap[key]; ok && json.Unmarshal(raw, &list) == nil {
				return list, nil
			}
		}
	}
	return nil, fmt.Errorf("the JSON must be an array of question objects")
}
