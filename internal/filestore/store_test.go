package filestore

import (
	"path/filepath"
	"testing"
)

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
