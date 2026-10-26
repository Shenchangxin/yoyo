// 本文件维护 jsonl 旁边的 .idx。版本号是 2。追加时文件头里的 size 可以过期，是否对齐看最后一条是否盖住 jsonl 的长度。

package trace

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
)

// idxVersion 是索引文件第一行的 v。不是 2 就整份作废并重建。
const idxVersion = 2

// idxEntry 是一行 JSONL 的位置。Seq 给界面分页用，不是实时 Hub 的序号。
type idxEntry struct {
	// Seq 从 1 起，按文件顺序递增。为 0 的行会让 loadIdxFile 放弃整份索引。
	Seq int64 `json:"seq"`
	// Off 是这一行在 jsonl 里的起始字节。
	Off int64 `json:"off"`
	// N 是这一行的字节数，含换行。
	N int `json:"n"`
	// Type 是事件类型，用来不用解开前缀就找到 compaction。
	Type EventType `json:"t"`
	// Kind 是载荷里的 kind。checkpoint 和 steer 用它。
	Kind string `json:"k,omitempty"`
	// Source 是来源。steer 让分页不必再读正文。
	Source string `json:"s,omitempty"`
}

// operatorUser 在类型是 user，且 Source、Kind 都不是 steer 时返回 true。
func (e idxEntry) operatorUser() bool {
	if e.Type != TypeUser {
		return false
	}
	return e.Source != "steer" && e.Kind != "steer"
}

// sessionIndex 是一个会话的内存索引。
type sessionIndex struct {
	// size 是索引覆盖的 jsonl 字节数。
	size int64
	// entries 按文件顺序排列。
	entries []idxEntry
}

// idxPath 是该会话 jsonl 路径再加 .idx。
func (s *Store) idxPath(sessionID string) string {
	return s.path(sessionID) + ".idx"
}

// payloadKind 返回载荷里的 kind 字符串。没有载荷时返回空。
func payloadKind(ev Event) string {
	if ev.Payload == nil {
		return ""
	}
	k, _ := ev.Payload["kind"].(string)
	return k
}

// entrySource 在 Source 为 steer，或载荷 name、kind 为 steer 时返回 steer。否则返回事件自己的 Source。
func entrySource(ev Event) string {
	if ev.Source == "steer" {
		return "steer"
	}
	if ev.Payload != nil {
		if n, _ := ev.Payload["name"].(string); n == "steer" {
			return "steer"
		}
		if k, _ := ev.Payload["kind"].(string); k == "steer" {
			return "steer"
		}
	}
	return ev.Source
}

// indexLocked 返回内存索引。缓存的 size 和 jsonl 当前长度一致就复用，否则载入或重建。调用方须已持有 mu。
func (s *Store) indexLocked(sessionID string) *sessionIndex {
	if s.mem == nil {
		s.mem = map[string]*sessionIndex{}
	}
	if idx := s.mem[sessionID]; idx != nil && s.indexFreshLocked(sessionID, idx) {
		return idx
	}
	idx := s.loadOrRebuildLocked(sessionID)
	s.mem[sessionID] = idx
	return idx
}

// indexFreshLocked 在 jsonl 长度等于 idx.size 时为真。文件不存在时，只有索引 size 为 0 且没有条目才为真。
func (s *Store) indexFreshLocked(sessionID string, idx *sessionIndex) bool {
	st, err := os.Stat(s.path(sessionID))
	if err != nil {
		return os.IsNotExist(err) && idx.size == 0 && len(idx.entries) == 0
	}
	return st.Size() == idx.size
}

// loadOrRebuildLocked 在 .idx 能读、版本对，且最后一条刚好盖住 jsonl 长度时复用磁盘索引，并把 size 设成文件长度。否则重建。jsonl 不存在时返回空索引。
func (s *Store) loadOrRebuildLocked(sessionID string) *sessionIndex {
	p := s.path(sessionID)
	st, err := os.Stat(p)
	if err != nil {
		return &sessionIndex{}
	}
	idx := loadIdxFile(s.idxPath(sessionID))
	if idx != nil && lastCover(idx, st.Size()) {
		idx.size = st.Size()
		return idx
	}
	return s.rebuildIndexLocked(sessionID)
}

// lastCover 报告最后一条的 Off+N 是否等于 size。没有条目时只有 size 为 0 才为真。
func lastCover(idx *sessionIndex, size int64) bool {
	if len(idx.entries) == 0 {
		return size == 0
	}
	last := idx.entries[len(idx.entries)-1]
	return last.Off+int64(last.N) == size
}

// loadIdxFile 读索引。第一行的 v 必须是 2。其后每行 Seq 不能为 0。扫描缓冲从 64 KiB 起，单行上限 1 MiB。任一处不合就返回 nil。
func loadIdxFile(path string) *sessionIndex {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	idx := &sessionIndex{}
	line := 0
	for sc.Scan() {
		b := sc.Bytes()
		if len(b) == 0 {
			continue
		}
		line++
		if line == 1 {
			var head struct {
				V    int   `json:"v"`
				Size int64 `json:"size"`
			}
			if json.Unmarshal(b, &head) != nil || head.V != idxVersion {
				return nil
			}
			idx.size = head.Size
			continue
		}
		var e idxEntry
		if json.Unmarshal(b, &e) != nil || e.Seq == 0 {
			return nil
		}
		idx.entries = append(idx.entries, e)
	}
	if sc.Err() != nil {
		return nil
	}
	return idx
}

// rebuildIndexLocked 重扫 jsonl。序号从 1 起。载荷 name 或 kind 为 steer 时 Source 写成 steer。读失败返回空索引。成功后把索引写到 .idx，写失败被丢掉。
func (s *Store) rebuildIndexLocked(sessionID string) *sessionIndex {
	p := s.path(sessionID)
	f, err := os.Open(p)
	if err != nil {
		return &sessionIndex{}
	}
	defer f.Close()
	idx := &sessionIndex{}
	r := bufio.NewReaderSize(f, 64*1024)
	var off int64
	var seq int64
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			seq++
			n := len(line)
			var peek struct {
				Type    EventType      `json:"type"`
				Source  string         `json:"source"`
				Payload map[string]any `json:"payload"`
			}
			_ = json.Unmarshal(line, &peek)
			kind := ""
			source := peek.Source
			if peek.Payload != nil {
				kind, _ = peek.Payload["kind"].(string)
				if n, _ := peek.Payload["name"].(string); n == "steer" {
					source = "steer"
				}
				if kind == "steer" {
					source = "steer"
				}
			}
			idx.entries = append(idx.entries, idxEntry{
				Seq: seq, Off: off, N: n, Type: peek.Type, Kind: kind, Source: source,
			})
			off += int64(n)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return &sessionIndex{}
		}
	}
	idx.size = off
	_ = writeIdxFile(s.idxPath(sessionID), idx)
	return idx
}

// writeIdxFile 先写版本和 size，再写每条索引，经 .tmp 改名，权限 0o644。
func writeIdxFile(path string, idx *sessionIndex) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(map[string]any{"v": idxVersion, "size": idx.size}); err != nil {
		return err
	}
	for i := range idx.entries {
		if err := enc.Encode(idx.entries[i]); err != nil {
			return err
		}
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// recordAppendLocked 在内存索引的 size 正好等于这次写入的 off 时追加一条，并 appendIdxLine。对不上就整份重建。
func (s *Store) recordAppendLocked(sessionID string, ev Event, off int64, n int) {
	if s.mem == nil {
		s.mem = map[string]*sessionIndex{}
	}
	idx := s.mem[sessionID]
	if idx == nil || idx.size != off {
		s.mem[sessionID] = s.rebuildIndexLocked(sessionID)
		return
	}
	e := idxEntry{
		Seq:    int64(len(idx.entries)) + 1,
		Off:    off,
		N:      n,
		Type:   ev.Type,
		Kind:   payloadKind(ev),
		Source: entrySource(ev),
	}
	if len(idx.entries) > 0 {
		e.Seq = idx.entries[len(idx.entries)-1].Seq + 1
	}
	idx.entries = append(idx.entries, e)
	idx.size = off + int64(n)
	_ = appendIdxLine(s.idxPath(sessionID), idx.size, e)
}

// appendIdxLine 往 .idx 末尾追加一条。文件还是空的才写头，头里的 size 是这次的长度；之后追加不再改头。是否仍可用由 lastCover 对照 jsonl 长度决定。
func appendIdxLine(path string, size int64, e idxEntry) error {
	// 只追加这一条。头里的 size 会停到下次重建；是否对齐看 lastCover 对照 jsonl 长度。
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	st, _ := f.Stat()
	enc := json.NewEncoder(f)
	if st.Size() == 0 {
		if err := enc.Encode(map[string]any{"v": idxVersion, "size": size}); err != nil {
			return err
		}
	}
	return enc.Encode(e)
}

// dropIndexLocked 从内存删掉该会话索引，并删除 .idx。删除错误被丢掉。
func (s *Store) dropIndexLocked(sessionID string) {
	if s.mem != nil {
		delete(s.mem, sessionID)
	}
	_ = os.Remove(s.idxPath(sessionID))
}

// readRangeLocked 从 off 读到 end。end 小于等于 off 时读到文件尾。文件不存在返回 nil, nil。JSON 无效的行跳过。扫描缓冲从 64 KiB 起，单行上限 8 MiB。
func (s *Store) readRangeLocked(sessionID string, off, end int64) ([]Event, error) {
	p := s.path(sessionID)
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	if off > 0 {
		if _, err := f.Seek(off, io.SeekStart); err != nil {
			return nil, err
		}
	}
	var r io.Reader = f
	if end > off {
		r = io.LimitReader(f, end-off)
	}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	var out []Event
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var ev Event
		if err := json.Unmarshal(line, &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, sc.Err()
}
