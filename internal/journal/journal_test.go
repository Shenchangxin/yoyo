package journal

import (
	"testing"
)

func TestChainCommitAndVerify(t *testing.T) {
	dir := t.TempDir()
	l, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	r1, err := l.Append("intent", map[string]string{"op": "load"})
	if err != nil {
		t.Fatal(err)
	}
	if r1.Committed {
		t.Fatal("intent should start uncommitted")
	}
	r2, err := l.Append("intent", map[string]string{"op": "eval"})
	if err != nil {
		t.Fatal(err)
	}
	if r2.PrevHash != r1.Hash {
		t.Fatal("chain broken")
	}
	if err := l.Commit(r1.Seq); err != nil {
		t.Fatal(err)
	}
	if err := l.Verify(); err != nil {
		t.Fatal(err)
	}
	pending, err := l.Pending()
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 1 || pending[0].Seq != r2.Seq {
		t.Fatalf("%+v", pending)
	}
}
