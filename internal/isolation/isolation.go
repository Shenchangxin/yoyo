package isolation

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"
)

// Report is the honest isolation/signing/vault picture. The UI must not
// render a "sandboxed" badge unless Sandbox is true.
type Report struct {
	Kind      string `json:"kind"`
	Sandbox   bool   `json:"sandbox"`
	Available bool   `json:"available"`
	Signed    string `json:"signed"`
	Note      string `json:"note"`
	GOOS      string `json:"goos"`
}

func Status() Report {
	r := probe()
	r.GOOS = runtime.GOOS
	r.Signed = SigningStatus()
	if r.Kind == "" {
		r.Kind = "none"
		r.Note = "policy jail only (path + deny-list). not Seatbelt/bubblewrap."
	}
	return r
}

// Outcome is one shell wait. Background means the process is still running
// and was not killed: the agent loop got control back.
type Outcome struct {
	Output     []byte
	Err        error
	PID        int
	Background bool
}

func Run(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	o := Wait(ctx, cmd, 0, 0)
	return o.Output, o.Err
}

// Wait runs cmd. idle/block are fuses that return while the process lives
// (Cursor block_until_ms). ctx cancel always kills. idle=block=0 waits
// until exit or ctx, matching Run.
func Wait(ctx context.Context, cmd *exec.Cmd, idle, block time.Duration) Outcome {
	if cmd == nil {
		return Outcome{Err: os.ErrInvalid}
	}
	apply(cmd)
	sp := &spool{last: time.Now()}
	if cmd.Stdout == nil {
		cmd.Stdout = sp
	}
	if cmd.Stderr == nil {
		cmd.Stderr = sp
	}
	if err := cmd.Start(); err != nil {
		return Outcome{Output: sp.Bytes(), Err: err}
	}
	assign(cmd)
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()

	started := time.Now()
	var poll <-chan time.Time
	if idle > 0 || block > 0 {
		tk := time.NewTicker(50 * time.Millisecond)
		defer tk.Stop()
		poll = tk.C
	}

	finish := func(err error, kill bool) Outcome {
		release(cmd, kill)
		if kill && ctx != nil && ctx.Err() != nil {
			err = ctx.Err()
		}
		return Outcome{Output: sp.Bytes(), Err: err, PID: pid}
	}
	background := func() Outcome {
		release(cmd, false)
		go func() { <-wait }()
		return Outcome{Output: sp.Bytes(), PID: pid, Background: true}
	}

	if ctx == nil && poll == nil {
		return finish(<-wait, false)
	}

	for {
		select {
		case err := <-wait:
			kill := ctx != nil && ctx.Err() != nil
			return finish(err, kill)
		case <-ctxDone(ctx):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			<-wait
			return finish(ctx.Err(), true)
		case <-poll:
			now := time.Now()
			if block > 0 && now.Sub(started) >= block {
				return background()
			}
			if idle > 0 && now.Sub(sp.Last()) >= idle {
				return background()
			}
		}
	}
}

func ctxDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return nil
	}
	return ctx.Done()
}

type spool struct {
	mu   sync.Mutex
	buf  bytes.Buffer
	last time.Time
}

func (s *spool) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(p) > 0 {
		s.last = time.Now()
	}
	return s.buf.Write(p)
}

func (s *spool) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, s.buf.Len())
	copy(out, s.buf.Bytes())
	return out
}

func (s *spool) Last() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

func SigningStatus() string {
	if os.Getenv("YOYO_SIGNED") == "1" {
		return "env-asserted"
	}
	return signingCached()
}

var signingOnce = sync.OnceValue(signingProbe)

func signingCached() string {
	return signingOnce()
}
