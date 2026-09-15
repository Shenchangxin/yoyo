package evolve

import (
	"testing"

	"github.com/Shenchangxin/yoyo/internal/artifact"
)

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
