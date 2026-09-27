package runtime

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestChatStreamStopsOnCancel(t *testing.T) {
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		close(started)
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-started
		cancel()
	}()
	client := NewOpenAIClient(srv.URL, "")
	done := make(chan error, 1)
	go func() {
		_, err := client.ChatStream(ctx, ChatRequest{Model: "m"}, nil)
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("canceled stream returned nil error")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("stream did not stop after cancel")
	}
}
