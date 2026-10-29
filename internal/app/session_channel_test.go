// 视频频道与代理频道各自独立，并使用自己的工作区。

package app

import (
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/session"
)

// TestVideoChannelIndependentOfAgent 确认视频会话和代理会话 id 不同，分叉后仍是视频车道。
func TestVideoChannelIndependentOfAgent(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	agent, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	video, err := a.NewSessionOn(ws, session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	if session.NormalizeChannel(agent.Channel) != session.ChannelAgent {
		t.Fatalf("agent channel %q", agent.Channel)
	}
	if video.Channel != session.ChannelVideo {
		t.Fatalf("video channel %q", video.Channel)
	}
	if agent.ID == video.ID {
		t.Fatal("channels must not share an id")
	}
	fork, err := a.ForkSession(video.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fork.Channel != session.ChannelVideo {
		t.Fatalf("fork channel %q", fork.Channel)
	}
	if fork.ID == video.ID {
		t.Fatal("fork reused id")
	}
	got, err := a.GetSession(video.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Channel != session.ChannelVideo {
		t.Fatalf("reload channel %q", got.Channel)
	}
}

// TestVideoSessionUsesOwnWorkspace 确认空工作区的视频会话用视频工作区，并和代理会话的工作区分开。
func TestVideoSessionUsesOwnWorkspace(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	video, err := a.NewSessionOn("", session.ChannelVideo)
	if err != nil {
		t.Fatal(err)
	}
	agent, err := a.NewSessionOn("", session.ChannelAgent)
	if err != nil {
		t.Fatal(err)
	}
	if video.Workspace != a.VideoWorkspace() {
		t.Fatalf("video workspace %q want %q", video.Workspace, a.VideoWorkspace())
	}
	if agent.Workspace != a.Workspace() {
		t.Fatalf("agent workspace %q want %q", agent.Workspace, a.Workspace())
	}
	if video.Workspace == agent.Workspace {
		t.Fatal("channels must not share a default workspace")
	}
}

// TestLearnChannelIndependentOfAgent 确认学习会话和代理会话不是同一条，分叉后仍是学习频道，并且技能目录仍走代理车道。
func TestLearnChannelIndependentOfAgent(t *testing.T) {
	a, err := Open(t.TempDir(), filepath.Join("..", "..", "evals"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ws := t.TempDir()
	agent, err := a.NewSession(ws)
	if err != nil {
		t.Fatal(err)
	}
	learned, err := a.NewSessionOn(ws, session.ChannelLearn)
	if err != nil {
		t.Fatal(err)
	}
	if learned.Channel != session.ChannelLearn {
		t.Fatalf("learn channel %q", learned.Channel)
	}
	if learned.ID == agent.ID {
		t.Fatal("learn channel reused the agent id")
	}
	if learned.Workspace != agent.Workspace {
		t.Fatalf("learn workspace %q want %q", learned.Workspace, agent.Workspace)
	}
	fork, err := a.ForkSession(learned.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fork.Channel != session.ChannelLearn {
		t.Fatalf("fork channel %q", fork.Channel)
	}
	if session.NormalizeChannel("Learn") != session.ChannelLearn {
		t.Fatal("normalize learn")
	}
	if session.CatalogLane(session.ChannelLearn) != session.ChannelAgent {
		t.Fatal("learn catalog must stay on the agent lane")
	}
	empty, err := a.NewSessionOn("", session.ChannelLearn)
	if err != nil {
		t.Fatal(err)
	}
	if empty.Workspace != a.Workspace() {
		t.Fatalf("default learn workspace %q", empty.Workspace)
	}
	if empty.Workspace == a.VideoWorkspace() && a.VideoWorkspace() != a.Workspace() {
		t.Fatal("learn session used the video workspace")
	}
}
