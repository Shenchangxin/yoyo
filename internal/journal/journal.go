// Package journal 维护主目录里的 JSONL 链 chain.jsonl。
// 应用启动时 Open。业务代码用 Append 在末尾追加记录。演化流程用 Commit 把某个序号标成已提交。
// Committed 不参与哈希。Commit 会把整份链用临时文件重写，所以文件字节不是只追加。
package journal

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/zeebo/blake3"
)

// Record 是链上的一条 JSON 记录。
type Record struct {
	// Append 成功时从 1 起。空链上 Last 得到的零值是 0。
	Seq uint64 `json:"seq"`
	// 上一条的 Hash。第一条是空字符串。
	PrevHash string `json:"prev_hash"`
	// hashRecord 的结果，计算时不读 Committed。
	Hash string `json:"hash"`
	Type string `json:"type"`
	// Append 写入的 UTC 时间。哈希按 RFC3339Nano 格式化它。
	Time time.Time `json:"time"`
	// 载荷的 JSON 字节，原样进入哈希。
	Payload json.RawMessage `json:"payload"`
	// Commit 把它写成 true。零值 false。这个字段不进入哈希。
	Committed bool `json:"committed"`
}

// Log 读写 chain.jsonl。Path 由 Open 设为这个文件。
type Log struct {
	Path string
	mu   sync.Mutex
}

// Open 创建 dir，权限 0o755，并把 Path 设为 dir/chain.jsonl。
// 这时不创建链文件，第一次 Append 才创建。目录创建失败时返回错误，不返回 Log。
func Open(dir string) (*Log, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Log{Path: filepath.Join(dir, "chain.jsonl")}, nil
}

// hashRecord 用 BLAKE3 计算十六进制摘要。
// 输入是序号、上一条哈希、类型、该时间的 UTC RFC3339Nano。四段用竖线拼接，末尾再加一条竖线，然后接上载荷原始字节。
// Committed 不参与哈希。
func hashRecord(seq uint64, prev, typ string, t time.Time, payload []byte) string {
	h := blake3.New()
	fmt.Fprintf(h, "%d|%s|%s|%s|", seq, prev, typ, t.UTC().Format(time.RFC3339Nano))
	_, _ = h.Write(payload)
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}

// Last 返回最后一条。链为空或文件还不存在时返回零值 Record 和 nil，不把没有记录当成错误。
func (l *Log) Last() (Record, error) {
	recs, err := l.ReadAll()
	if err != nil {
		return Record{}, err
	}
	if len(recs) == 0 {
		return Record{}, nil
	}
	return recs[len(recs)-1], nil
}

// Append 追加一条 Committed 为 false 的记录，不改已有行。
// 载荷先做 JSON 编码，编码失败时文件不动。
// 在锁内读完整条链：空链的序号是 1、PrevHash 是空字符串；否则序号是最后一条 Seq 加 1，PrevHash 是最后一条的 Hash。
// 然后以 O_APPEND 把这一行写到末尾，文件权限 0o644。
// 写盘失败时返回零值 Record，不回滚已经写出的字节。
func (l *Log) Append(typ string, payload any) (Record, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Record{}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	recs, err := l.readUnlocked()
	if err != nil {
		return Record{}, err
	}
	prev := ""
	seq := uint64(1)
	if n := len(recs); n > 0 {
		prev = recs[n-1].Hash
		seq = recs[n-1].Seq + 1
	}
	now := time.Now().UTC()
	rec := Record{
		Seq:       seq,
		PrevHash:  prev,
		Type:      typ,
		Time:      now,
		Payload:   raw,
		Committed: false,
	}
	rec.Hash = hashRecord(rec.Seq, rec.PrevHash, rec.Type, rec.Time, rec.Payload)
	if err := l.write(rec); err != nil {
		return Record{}, err
	}
	return rec, nil
}

// Commit 把序号等于 seq 的记录标成 Committed，再把整份链用临时文件重写。
// Append 在锁里按上一条 Seq+1 编号，所以正常追加不会写出重复序号。
// Commit 的循环不在第一条匹配处停下；文件里若已有相同序号，会全部标上。
// 找不到时返回错误，chain.jsonl 保持原样。
// 只改 Committed，不重算 Hash。重写还没改名就失败时，原文件保持原样，内存里翻过的标志也不留。
// 改名成功后，Path 上是整份新内容，文件字节不是只追加。
func (l *Log) Commit(seq uint64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	recs, err := l.readUnlocked()
	if err != nil {
		return err
	}
	var found bool
	for i := range recs {
		if recs[i].Seq == seq {
			// Committed 不参与哈希，这里不重算 Hash。
			recs[i].Committed = true
			found = true
		}
	}
	if !found {
		return fmt.Errorf("journal: seq %d not found", seq)
	}
	return l.rewrite(recs)
}

// Pending 按链上的顺序返回 Committed 为 false 的记录。
func (l *Log) Pending() ([]Record, error) {
	recs, err := l.ReadAll()
	if err != nil {
		return nil, err
	}
	var out []Record
	for _, r := range recs {
		if !r.Committed {
			out = append(out, r)
		}
	}
	return out, nil
}

// ReadAll 在锁内读出全部记录。文件不存在时当作空链，返回 nil 和 nil。
func (l *Log) ReadAll() ([]Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.readUnlocked()
}

// Verify 检查每条的 PrevHash 是否等于上一条的 Hash，以及 Hash 是否等于 hashRecord 的现算值。
// 第一条用来衔接的上一条哈希是空字符串。空链通过。
// 不检查序号是否连续，也不读 Committed。断链或哈希不符时返回错误，不改文件。
func (l *Log) Verify() error {
	recs, err := l.ReadAll()
	if err != nil {
		return err
	}
	prev := ""
	for _, r := range recs {
		if r.PrevHash != prev {
			return fmt.Errorf("journal: broken chain at seq %d", r.Seq)
		}
		want := hashRecord(r.Seq, r.PrevHash, r.Type, r.Time, r.Payload)
		if want != r.Hash {
			return fmt.Errorf("journal: bad hash at seq %d", r.Seq)
		}
		prev = r.Hash
	}
	return nil
}

// write 以追加方式把一条记录写成一行 JSON，权限 0o644。由 Append 在持有 l.mu 时调用，这里不再加锁。
func (l *Log) write(rec Record) error {
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(rec)
}

// rewrite 把整份链写到 Path 加上 .tmp，再改名替换 Path。这不是追加。
// 编码或关闭失败时不改名，原文件保留。临时文件可能留在磁盘上，这里不删除它。
func (l *Log) rewrite(recs []Record) error {
	tmp := l.Path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(f)
	for _, r := range recs {
		if err := enc.Encode(r); err != nil {
			_ = f.Close()
			return err
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, l.Path)
}

// readUnlocked 逐行解析 JSON。空行跳过。文件不存在时返回 nil 和 nil。
// 某一行 JSON 不合法时返回 nil 和错误，不带回已经解析的记录。
// 扫描器出错时带回已经解析的记录和该错误。
// 调用方必须已经持有 l.mu，这里不再加锁。
func (l *Log) readUnlocked() ([]Record, error) {
	f, err := os.Open(l.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []Record
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if len(sc.Bytes()) == 0 {
			continue
		}
		var r Record
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, sc.Err()
}
