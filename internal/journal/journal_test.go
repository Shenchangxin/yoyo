// 锁住这些不变量：新记录的 Committed 为 false；下一条的 PrevHash 等于上一条的 Hash；
// Commit 第一条之后 Verify 仍然通过；Pending 只剩下未提交的第二条。
// 不拥有哈希公式，也不检查临时文件里的字节。

package journal

import (
	"testing"
)

// TestChainCommitAndVerify 锁住未提交起点、前后哈希衔接、Commit 后校验仍通过，以及 Pending 只留后一条。
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
