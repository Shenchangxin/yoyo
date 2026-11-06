// studio_state_test.go 锁住短片工程的读改写：并发写回不丢字段、口播比较去掉首尾空白、镜头合并认得出没带编号的镜头、闸门不越级、镜头时长覆盖每一镜。

package video

import (
	"fmt"
	"sync"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/video/cut"
)

// approvedStudio 建一条已锁定口播的短片，镜头由口播自动拆出。
func approvedStudio(t *testing.T, e *Engine, script string) StudioProject {
	t.Helper()
	p, err := e.CreateStudioProject(StudioProject{Title: "并发"})
	if err != nil {
		t.Fatal(err)
	}
	p, err = e.ApproveStudioScriptText(p.ID, script)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// TestStudioJobWritesDoNotLoseEachOther 确认多镜同时落图、同时有界面保存时，每一镜的图和保存的字段都还在。
func TestStudioJobWritesDoNotLoseEachOther(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。第二句。第三句。第四句。第五句。第六句。")
	shots := p.Storyboard.Shots
	if len(shots) != 6 {
		t.Fatalf("shots %d", len(shots))
	}
	var wg sync.WaitGroup
	for i, s := range shots {
		wg.Add(2)
		go func(i int, s StudioShot) {
			defer wg.Done()
			j := Job{ID: fmt.Sprintf("job-%d", i), Type: "image", ResultHash: fmt.Sprintf("hash-%d", i)}
			if err := e.applyStudioJob(j, JobParams{StudioProjectID: p.ID, StudioShotID: s.ID, StudioKind: "shot"}); err != nil {
				t.Error(err)
			}
		}(i, s)
		go func(i int) {
			defer wg.Done()
			if _, err := e.UpdateStudioProject(StudioProject{ID: p.ID, Audience: fmt.Sprintf("观众%d", i)}); err != nil {
				t.Error(err)
			}
		}(i)
	}
	wg.Wait()
	got, err := e.GetStudioProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range got.Storyboard.Shots {
		if s.ImageHash != fmt.Sprintf("hash-%d", i) {
			t.Fatalf("shot %d lost its image: %+v", i, s)
		}
	}
	if got.Audience == "" {
		t.Fatal("audience save lost")
	}
}

// TestStudioScriptTrailingNewlineKeepsApproval 确认已锁定的口播只多一个换行再保存时，仍是已锁定。
func TestStudioScriptTrailingNewlineKeepsApproval(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "今天说三件事。第一件很重要。")
	got, err := e.UpdateStudioProject(StudioProject{ID: p.ID, Script: "今天说三件事。第一件很重要。\n"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ScriptStatus != ScriptApproved {
		t.Fatalf("trailing newline reverted approval: %s", got.ScriptStatus)
	}
	got, err = e.UpdateStudioProject(StudioProject{ID: p.ID, Script: "今天说两件事。"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ScriptStatus != ScriptNeedsReview {
		t.Fatalf("a real edit should need review: %s", got.ScriptStatus)
	}
	cleared, err := e.UpdateStudioProjectWith(StudioProject{ID: p.ID}, true)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.Script != "" || cleared.ScriptStatus != ScriptNeedsReview {
		t.Fatalf("clearing the script %q %s", cleared.Script, cleared.ScriptStatus)
	}
}

// TestStudioShotMergeKeepsMediaWithoutIDs 确认助手重写分镜不带编号时按字幕对上原镜头，图不丢；客户端手里的旧过期标记不算；新传的图算手动换画面。
func TestStudioShotMergeKeepsMediaWithoutIDs(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "开场一句。中间一句。")
	for i, s := range p.Storyboard.Shots {
		j := Job{ID: fmt.Sprintf("job-%d", i), Type: "image", ResultHash: fmt.Sprintf("img-%d", i)}
		if err := e.applyStudioJob(j, JobParams{StudioProjectID: p.ID, StudioShotID: s.ID, StudioKind: "shot"}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := e.UpdateStudioProject(StudioProject{ID: p.ID, Storyboard: StudioStoryboard{Shots: []StudioShot{
		{Caption: "中间一句。", Duration: 4, Stale: true},
		{Caption: "开场一句。", Duration: 2},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	shots := got.Storyboard.Shots
	if shots[0].ImageHash != "img-1" || shots[1].ImageHash != "img-0" {
		t.Fatalf("media lost on reorder: %+v", shots)
	}
	if shots[0].Stale {
		t.Fatal("a stale flag from the client must not mark a landed image stale")
	}
	if got.ShotsFrom != "" {
		t.Fatal("written shots should stop following the script")
	}
	swapped, err := e.UpdateStudioProject(StudioProject{ID: p.ID, Storyboard: StudioStoryboard{Shots: []StudioShot{
		{ID: shots[0].ID, Caption: shots[0].Caption, ImageHash: "upload-1"},
		shots[1],
	}}})
	if err != nil {
		t.Fatal(err)
	}
	up := swapped.Storyboard.Shots[0]
	if up.ImageHash != "upload-1" || up.PrevImage != "img-1" || up.JobID != manualShotJob || up.Stale {
		t.Fatalf("manual swap %+v", up)
	}
	late := Job{ID: "job-1", Type: "image", ResultHash: "late"}
	if err := e.applyStudioJob(late, JobParams{StudioProjectID: p.ID, StudioShotID: up.ID, StudioKind: "shot"}); err != nil {
		t.Fatal(err)
	}
	after, _ := e.GetStudioProject(p.ID)
	if after.Storyboard.Shots[0].ImageHash != "upload-1" {
		t.Fatal("a late generation overwrote the manual picture")
	}
}

// TestSplitSpeakKeepsNumbersTogether 确认版本号、小数和序号不会被英文句点切开，真正的句末照样切。
func TestSplitSpeakKeepsNumbersTogether(t *testing.T) {
	got := splitSpeak("iOS 18.1 更新了三点：\n1. 续航更长。\n2. 价格 3.5 折! Next one. 最后")
	want := []string{"iOS 18.1 更新了三点：", "1. 续航更长。", "2. 价格 3.5 折!", "Next one.", "最后"}
	if fmt.Sprintf("%q", got) != fmt.Sprintf("%q", want) {
		t.Fatalf("split %q, want %q", got, want)
	}
}

// TestStudioStaleClientKeepsLandedImage 确认界面拿着生图前的旧图保存字幕时，刚落地的新图不会被旧图换回去，字幕照样存上。
func TestStudioStaleClientKeepsLandedImage(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "开场一句。中间一句。")
	shot := p.Storyboard.Shots[0]
	first := Job{ID: "job-a", Type: "image", ResultHash: "img-a"}
	if err := e.applyStudioJob(first, JobParams{StudioProjectID: p.ID, StudioShotID: shot.ID, StudioKind: "shot"}); err != nil {
		t.Fatal(err)
	}
	seen, _ := e.GetStudioProject(p.ID)
	stale := seen.Storyboard.Shots
	if _, err := e.mutateStudio(p.ID, func(cur *StudioProject) error {
		cur.Storyboard.Shots[0].JobID = "job-b"
		cur.Storyboard.Shots[0].Stale = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	regen := Job{ID: "job-b", Type: "image", ResultHash: "img-b"}
	if err := e.applyStudioJob(regen, JobParams{StudioProjectID: p.ID, StudioShotID: shot.ID, StudioKind: "shot"}); err != nil {
		t.Fatal(err)
	}
	edited := append([]StudioShot(nil), stale...)
	edited[0].Caption = "改过的开场"
	got, err := e.UpdateStudioProject(StudioProject{ID: p.ID, Storyboard: StudioStoryboard{Shots: edited}})
	if err != nil {
		t.Fatal(err)
	}
	s := got.Storyboard.Shots[0]
	if s.ImageHash != "img-b" || s.JobID != "job-b" {
		t.Fatalf("stale client swapped the landed image back: %+v", s)
	}
	if s.Caption != "改过的开场" {
		t.Fatalf("caption lost: %+v", s)
	}
}

// TestApproveFollowsNewScriptOnlyForSplitShots 确认按口播自动拆出的镜头会跟上新口播并留下对得上的画面，手写镜头不动。
func TestApproveFollowsNewScriptOnlyForSplitShots(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。第二句。")
	if p.ShotsFrom == "" {
		t.Fatal("split shots should remember their script")
	}
	first := p.Storyboard.Shots[0]
	if err := e.applyStudioJob(Job{ID: "j0", Type: "image", ResultHash: "keep-0"}, JobParams{StudioProjectID: p.ID, StudioShotID: first.ID, StudioKind: "shot"}); err != nil {
		t.Fatal(err)
	}
	next, err := e.ApproveStudioScriptText(p.ID, "第一句。新的第二句。第三句。")
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Storyboard.Shots) != 3 {
		t.Fatalf("shots should follow the new script: %+v", next.Storyboard.Shots)
	}
	if next.Storyboard.Shots[0].ImageHash != "keep-0" {
		t.Fatal("the matching shot lost its picture")
	}
	written, err := e.UpdateStudioProject(StudioProject{ID: p.ID, Storyboard: StudioStoryboard{Shots: []StudioShot{{Caption: "手写镜头", Visual: "雨夜街角"}}}})
	if err != nil {
		t.Fatal(err)
	}
	again, err := e.ApproveStudioScriptText(written.ID, "完全不同的一句。")
	if err != nil {
		t.Fatal(err)
	}
	if len(again.Storyboard.Shots) != 1 || again.Storyboard.Shots[0].Visual != "雨夜街角" {
		t.Fatalf("written shots must stay: %+v", again.Storyboard.Shots)
	}
}

// TestStudioGateCannotJumpAhead 确认没配音的工程不能被标到合成。
func TestStudioGateCannotJumpAhead(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "一句话。")
	got, err := e.UpdateStudioProject(StudioProject{ID: p.ID, Gate: GateCompose})
	if err != nil {
		t.Fatal(err)
	}
	if got.Gate == GateCompose {
		t.Fatal("gate jumped past audio")
	}
	got, err = e.UpdateStudioProject(StudioProject{ID: p.ID, Gate: GateSetup})
	if err != nil {
		t.Fatal(err)
	}
	if got.Gate != GateSetup {
		t.Fatalf("going back should work: %s", got.Gate)
	}
}

// TestRecoverStudioAutoMarksInterruptedRuns 确认重启后停在制作中的一键成片会被标成失败，可以再点一次。
func TestRecoverStudioAutoMarksInterruptedRuns(t *testing.T) {
	e := testEngine(t)
	p, err := e.CreateStudioProject(StudioProject{Title: "中断"})
	if err != nil {
		t.Fatal(err)
	}
	e.markAuto(p.ID, "visuals")
	e.recoverStudioAuto()
	got, err := e.GetStudioProject(p.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.AutoPhase != "failed" || got.AutoError == "" {
		t.Fatalf("phase %q error %q", got.AutoPhase, got.AutoError)
	}
	if !filmNeedsRun(got) {
		t.Fatal("an interrupted run should be runnable again")
	}
}

// TestShotDurationsCoverEveryShot 确认字幕条数和镜头对不上时每一镜都分到时长，总长等于配音；对得上时停顿算进前一镜。
func TestShotDurationsCoverEveryShot(t *testing.T) {
	shots := []StudioShot{{Caption: "很长很长的一句话在这里"}, {Caption: "短"}, {Caption: "中等长度的一句"}}
	durs := shotDurations(shots, []float64{4, 6}, 12)
	if len(durs) != 3 {
		t.Fatalf("durations %v", durs)
	}
	var sum float64
	for _, d := range durs {
		if d < 1 {
			t.Fatalf("shot shorter than a second: %v", durs)
		}
		sum += d
	}
	if sum < 11.9 || sum > 12.1 {
		t.Fatalf("total %v want 12", sum)
	}
	spans := cueSpans([]cut.Cue{{Start: 0.4, End: 2}, {Start: 3, End: 5}}, 6)
	if len(spans) != 2 || spans[0] != 3 || spans[1] != 3 {
		t.Fatalf("spans %v", spans)
	}
}
