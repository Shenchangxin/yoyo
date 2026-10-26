// 本文件锁住 Put、Get、Delete、IDs 和 LoadAll 能对同一条 JSON 往返。
// 不覆盖锁的并发。由 go test 调用。

package filestore

import (
	"path/filepath"
	"testing"
)

// TestPutGetDelete 写入一条记录后读回标题，列出 id，再用 LoadAll 拿到它，删除后 Exists 为假，文件名是 abc.json。
func TestPutGetDelete(t *testing.T) {
	s, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	type rec struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := s.Put(Rel("dramas", "abc"), rec{ID: "abc", Title: "雨巷"}); err != nil {
		t.Fatal(err)
	}
	var got rec
	if err := s.Get(Rel("dramas", "abc"), &got); err != nil {
		t.Fatal(err)
	}
	if got.Title != "雨巷" {
		t.Fatalf("got %+v", got)
	}
	ids, err := s.IDs("dramas")
	if err != nil || len(ids) != 1 || ids[0] != "abc" {
		t.Fatalf("ids=%v err=%v", ids, err)
	}
	all, err := LoadAll[rec](s, "dramas")
	if err != nil || len(all) != 1 {
		t.Fatalf("all=%v err=%v", all, err)
	}
	if err := s.Delete(Rel("dramas", "abc")); err != nil {
		t.Fatal(err)
	}
	if s.Exists(Rel("dramas", "abc")) {
		t.Fatal("still exists")
	}
	if filepath.Base(s.Abs(Rel("dramas", "abc"))) != "abc.json" {
		t.Fatal(s.Abs(Rel("dramas", "abc")))
	}
}
