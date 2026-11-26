// brief_test.go 锁住简报：意图和片型对不上就报错；缺账号写成缺；语音没有账号走本机；有图像账号才算花额度；确认卡是 hold；答复读成 go、edit、stop 或 unconfirmed。

package video

import (
	"strings"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/connection"
)

// TestBriefRejectsUnknownIntentAndFormat 确认对不上的意图和片型返回错误，不猜。
func TestBriefRejectsUnknownIntentAndFormat(t *testing.T) {
	e := testEngine(t)
	if _, err := e.VideoBrief(map[string]any{"intent": "dance"}); err == nil {
		t.Fatal("unknown intent must fail")
	}
	if _, err := e.VideoBrief(map[string]any{"intent": "film", "format": "opera"}); err == nil {
		t.Fatal("unknown format must fail")
	}
}

// TestNormalizeBriefDefaults 确认片型默认讲解，越界时长回到默认 30 秒，16x9 收成 16:9，非成片不带片型。
func TestNormalizeBriefDefaults(t *testing.T) {
	b, err := NormalizeBrief(map[string]any{"intent": " Film ", "seconds": float64(999), "aspect": "16x9"})
	if err != nil {
		t.Fatal(err)
	}
	if b.Intent != "film" || b.Format != "explainer" || b.Seconds != 30 || b.Aspect != "16:9" {
		t.Fatalf("%+v", b)
	}
	c, err := NormalizeBrief(map[string]any{"intent": "still", "format": "ranking", "seconds": float64(20), "aspect": "4:3"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Format != "" || c.Seconds != 0 || c.Aspect != "9:16" {
		t.Fatalf("%+v", c)
	}
}

// TestBriefWithoutAccountsListsMissing 确认没有账号时成片缺图像、语音走本机、不算花额度也不就绪；剪辑什么都不需要。
func TestBriefWithoutAccountsListsMissing(t *testing.T) {
	e := testEngine(t)
	plan, err := e.VideoBrief(map[string]any{"intent": "film", "subject": "早起"})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Ready || plan.Spend || len(plan.Missing) != 1 || plan.Missing[0] != connection.CapImage {
		t.Fatalf("missing %+v", plan)
	}
	if len(plan.Local) != 1 || plan.Local[0] != connection.CapSpeech || len(plan.Have) != 0 {
		t.Fatalf("local %+v", plan)
	}
	cut, err := e.VideoBrief(map[string]any{"intent": "cut"})
	if err != nil {
		t.Fatal(err)
	}
	if cut.Spend || !cut.Ready || len(cut.Needs) != 0 {
		t.Fatalf("cut %+v", cut)
	}
}

// TestBriefWithImageAccountSpendsAndHolds 确认有图像账号时成片就绪且花额度，确认卡是 hold，题干写出账号名。
func TestBriefWithImageAccountSpendsAndHolds(t *testing.T) {
	e := testEngine(t)
	reg, err := connection.Open(t.TempDir(), e.Vault)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reg.Upsert(connection.Connection{
		ID: "img", Name: "Kling", Vendor: "kling",
		Capabilities: []string{connection.CapImage}, Active: true,
	}, "k"); err != nil {
		t.Fatal(err)
	}
	e.Conn = reg
	plan, err := e.VideoBrief(map[string]any{"intent": "film", "format": "ranking", "subject": "三个天天用的东西"})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Ready || !plan.Spend || len(plan.Have[connection.CapImage]) != 1 || plan.Have[connection.CapImage][0] != "Kling" {
		t.Fatalf("%+v", plan)
	}
	args := e.BriefConfirmArgs(plan)
	if args["hold"] != true {
		t.Fatalf("confirm card must hold %+v", args)
	}
	qs, _ := args["questions"].([]any)
	if len(qs) != 1 {
		t.Fatalf("questions %+v", qs)
	}
	q, _ := qs[0].(map[string]any)
	prompt, _ := q["prompt"].(string)
	if !strings.Contains(prompt, "Kling") || !strings.Contains(prompt, "三个天天用的东西") {
		t.Fatalf("prompt %q", prompt)
	}
}

// TestBriefDecision 确认答复读成 go、edit、stop；未确认、拒绝和空答复都是 unconfirmed。
func TestBriefDecision(t *testing.T) {
	cases := map[string]string{
		"go: go — 开拍\nnote: operator answered":     "go",
		"go: edit — 改一下":                           "edit",
		"go: stop — 先不做":                           "stop",
		"not confirmed: nobody answered this card": "unconfirmed",
		"operator decision: deny":                  "unconfirmed",
		"":                                         "unconfirmed",
	}
	for in, want := range cases {
		if got := BriefDecision(in); got != want {
			t.Fatalf("%q => %q, want %q", in, got, want)
		}
	}
}
