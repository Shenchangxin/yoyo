package app

import (
	"path/filepath"
	"testing"

	"github.com/Shenchangxin/yoyo/internal/session"
)

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
