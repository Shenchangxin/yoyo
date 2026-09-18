package isolation

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"sync"
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

func Run(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if cmd == nil {
		return nil, os.ErrInvalid
	}
	apply(cmd)
	var buf bytes.Buffer
	if cmd.Stdout == nil {
		cmd.Stdout = &buf
	}
	if cmd.Stderr == nil {
		cmd.Stderr = &buf
	}
	if err := cmd.Start(); err != nil {
		return buf.Bytes(), err
	}
	assign(cmd)
	defer release(cmd)
	wait := make(chan error, 1)
	go func() { wait <- cmd.Wait() }()
	if ctx == nil {
		err := <-wait
		return buf.Bytes(), err
	}
	select {
	case err := <-wait:
		return buf.Bytes(), err
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-wait
		return buf.Bytes(), ctx.Err()
	}
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
