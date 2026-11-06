// studio_bgm_test.go 锁住一键成片的自动配乐：目录里有曲子就配上；操作者清掉过就不再配；目录空着、也没有配乐连接时留一条说明。

package video

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAutoStudioBGMPicksLibraryTrack 确认配乐目录里有曲子时自动配上一首，音量垫在口播下面。
func TestAutoStudioBGMPicksLibraryTrack(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。")
	if err := os.MkdirAll(e.bgmDir(), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(e.bgmDir(), "calm.mp3"), fakeMP3, 0o644); err != nil {
		t.Fatal(err)
	}
	e.autoStudioBGM(p.ID)
	got, _ := e.GetStudioProject(p.ID)
	if got.BGMHash == "" || got.Storyboard.BGM != got.BGMHash || got.Storyboard.BGMVolume <= 0 || got.Storyboard.BGMVolume > 0.5 {
		t.Fatalf("bgm not attached: %+v", got.Storyboard)
	}
}

// TestClearStudioBGMRemembersNoMusic 确认操作者清掉配乐后，一键成片不再自动配上；再选一首会取消这个记号。
func TestClearStudioBGMRemembersNoMusic(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。")
	_ = os.MkdirAll(e.bgmDir(), 0o755)
	_ = os.WriteFile(filepath.Join(e.bgmDir(), "calm.mp3"), fakeMP3, 0o644)
	if _, err := e.ClearStudioBGM(p.ID); err != nil {
		t.Fatal(err)
	}
	e.autoStudioBGM(p.ID)
	got, _ := e.GetStudioProject(p.ID)
	if got.BGMHash != "" || !got.NoBGM {
		t.Fatalf("cleared bgm came back: %+v", got)
	}
	picked, err := e.AttachStudioBGM(p.ID, "calm.mp3")
	if err != nil {
		t.Fatal(err)
	}
	if picked.NoBGM || picked.BGMHash == "" {
		t.Fatalf("choosing music should lift no_bgm: %+v", picked)
	}
}

// TestAutoStudioBGMExplainsEmptyLibrary 确认目录空着、也没有配乐连接时不报错，只留一条怎么加配乐的说明。
func TestAutoStudioBGMExplainsEmptyLibrary(t *testing.T) {
	e := testEngine(t)
	p := approvedStudio(t, e, "第一句。")
	e.autoStudioBGM(p.ID)
	got, _ := e.GetStudioProject(p.ID)
	if got.BGMHash != "" {
		t.Fatal("no track should be attached")
	}
	found := false
	for _, a := range got.Activities {
		if strings.Contains(a.Message, "配乐目录是空的") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a note about the empty folder: %+v", got.Activities)
	}
}

// TestStudioBGMPromptFollowsTone 确认生成配乐的描述跟着语气走，并写明不要人声。
func TestStudioBGMPromptFollowsTone(t *testing.T) {
	s := studioBGMPrompt(StudioProject{Tone: "energetic", Title: "早餐三分钟"})
	if !strings.Contains(s, "节奏明快") || !strings.Contains(s, "没有人声") || !strings.Contains(s, "早餐三分钟") {
		t.Fatal(s)
	}
}
