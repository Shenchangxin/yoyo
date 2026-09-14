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

type Record struct {
	Seq       uint64          `json:"seq"`
	PrevHash  string          `json:"prev_hash"`
	Hash      string          `json:"hash"`
	Type      string          `json:"type"`
	Time      time.Time       `json:"time"`
	Payload   json.RawMessage `json:"payload"`
	Committed bool            `json:"committed"`
}

type Log struct {
	Path string
	mu   sync.Mutex
}

func Open(dir string) (*Log, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Log{Path: filepath.Join(dir, "chain.jsonl")}, nil
}

func hashRecord(seq uint64, prev, typ string, t time.Time, payload []byte) string {
	h := blake3.New()
	fmt.Fprintf(h, "%d|%s|%s|%s|", seq, prev, typ, t.UTC().Format(time.RFC3339Nano))
	_, _ = h.Write(payload)
	sum := h.Sum(nil)
	return hex.EncodeToString(sum)
}

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
			recs[i].Committed = true
			found = true
		}
	}
	if !found {
		return fmt.Errorf("journal: seq %d not found", seq)
	}
	return l.rewrite(recs)
}

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

func (l *Log) ReadAll() ([]Record, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.readUnlocked()
}

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

func (l *Log) write(rec Record) error {
	f, err := os.OpenFile(l.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	return json.NewEncoder(f).Encode(rec)
}

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
