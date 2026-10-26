// 锁住策展会追加洞察，并把已有条目的 Helpful 计数加上去，而不是用一条新文本换掉整本剧本。

package evolve

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

// TestCurateDoesNotCollapse 要求洞察变成新增条目，且原条目的 Helpful 至少变成 2。
func TestCurateDoesNotCollapse(t *testing.T) {
	pb := artifact.Playbook{ID: "main", Bullets: []artifact.PlaybookBullet{
		{ID: "b1", Text: "create output early", Helpful: 1},
	}}
	next := Curate(pb, Reflection{
		Insights:   []string{"missing_artifact: tools=0 errors=0"},
		BulletTags: []BulletTag{{ID: "b1", Tag: "helpful"}},
	})
	if len(next.Bullets) < 2 {
		t.Fatalf("expected delta add, got %+v", next)
	}
	if next.Bullets[0].Helpful < 2 {
		t.Fatalf("helpful counter %+v", next.Bullets[0])
	}
}
