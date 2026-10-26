// Package isolation 汇总当前平台的进程隔离探针，并等待 shell 子进程。
//
// 各平台 probe 填写 Kind、Sandbox 和 Available。Sandbox 为 true 时，调用方才可以画沙箱徽标。
// Status 再写上 GOOS 和签名。Kind 为空时改为 none，说明只剩路径与拒绝名单，不是 Seatbelt 或 bubblewrap。
// 空闲或墙钟到达时可以提前交还控制权并留下进程；ctx 取消一定会杀掉进程。
// 路径与拒绝名单在 runtime。runtime 的 shell、app 的隔离报告和更新检查会调用本包。
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

// Report 是隔离探针和签名状态的如实快照。Sandbox 为 false 时，界面不得画沙箱徽标。
type Report struct {
	// Kind 是平台 probe 填的后端名。Status 在它为空时改成 none。
	Kind string `json:"kind"`
	// Sandbox 为 true 才表示这次探针允许沙箱徽标。
	Sandbox bool `json:"sandbox"`
	// Available 表示该后端在当前 GOOS 上是否可用。
	Available bool `json:"available"`
	// Signed 是签名状态，由 Status 调用 SigningStatus 写入。
	Signed string `json:"signed"`
	// Note 是给人读的说明，不参与徽标判断。
	Note string `json:"note"`
	// GOOS 由 Status 写成 runtime.GOOS。
	GOOS string `json:"goos"`
}

// Status 调用当前平台的 probe，写入 GOOS 和签名状态。
// Kind 为空时改为 none，说明只剩路径与拒绝名单，不是 Seatbelt 或 bubblewrap。
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

// Outcome 是一次 shell 等待的结果。
// Background 为 true 表示进程仍在跑且没有被杀掉，调用方已经拿回控制权。
type Outcome struct {
	// Output 是截至返回时缓冲里的输出。
	Output []byte
	// Err 是启动、等待或取消的错误。提前放回后台时为 nil。
	Err error
	// PID 是已启动进程的进程号；还没启动则为 0。
	PID int
	// Background 为 true 时进程仍在运行，这次等待没有杀它。
	Background bool
}

// Run 以 idle 和 block 都为 0 调用 Wait，只把输出和错误交回去。
// ctx 传给 Wait。
// cmd 传给 Wait。
func Run(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	o := Wait(ctx, cmd, 0, 0)
	return o.Output, o.Err
}

// Wait 运行 cmd，直到进程退出或 ctx 取消。
// idle 或 block 大于 0 时，静默或墙钟到达就交还控制权，进程继续留着，对应 Cursor 的 block_until_ms。
// idle 和 block 都为 0 时一直等到退出或取消，与 Run 相同。ctx 取消一定会杀掉进程。
func Wait(ctx context.Context, cmd *exec.Cmd, idle, block time.Duration) Outcome {
	return WaitNotify(ctx, cmd, idle, block, nil)
}

// liveStdoutCap 是实时界面推送的字节上限，值为 32 * 1024。
// 完整输出仍留在工具结果里，超过该上限的字节不再作为增量送出。
const liveStdoutCap = 32 * 1024

// WaitNotify 在等待之外把 stdout 和 stderr 的片段交给 onChunk，供界面实时显示。
// onChunk 不得阻塞：Hub 会合并片段，而 shell 线程还握着管道。
// ctx 为 nil 且 idle、block 都不大于 0 时，一直等到进程退出；取消时杀掉进程，并把对外错误换成 ctx.Err。
// cmd 为 nil 时返回 os.ErrInvalid；否则先 apply、preparePythonUTF8、preparePwshEnv，再 Start，成功后 assign。
// idle 大于 0 时，距上次输出达到它就交还控制权且不杀进程；与 block 都不大于 0 时不建轮询。
// block 大于 0 时，从启动起达到它就交还控制权且不杀进程。
func WaitNotify(ctx context.Context, cmd *exec.Cmd, idle, block time.Duration, onChunk func([]byte)) Outcome {
	if cmd == nil {
		return Outcome{Err: os.ErrInvalid}
	}
	// 启动前改写命令或创建标志；没有后端的平台上 apply 留空。
	apply(cmd)
	preparePythonUTF8(cmd)
	preparePwshEnv(cmd)
	// last 从现在起算，一直没有输出也会累积空闲。
	sp := &spool{last: time.Now(), onChunk: onChunk}
	// 调用方已经接过的标准输出或标准错误不再进入 spool，空闲熔断也看不到那些写入。
	if cmd.Stdout == nil {
		cmd.Stdout = sp
	}
	if cmd.Stderr == nil {
		cmd.Stderr = sp
	}
	if err := cmd.Start(); err != nil {
		return Outcome{Output: sp.Bytes(), Err: err}
	}
	// Start 成功后才交给平台记账。
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
		// 空闲和墙钟熔断每 50ms 看一次。
		tk := time.NewTicker(50 * time.Millisecond)
		defer tk.Stop()
		poll = tk.C
	}

	finish := func(err error, kill bool) Outcome {
		release(cmd, kill)
		// kill 为 true 且 ctx 已取消时，对外错误用 ctx 的错误。
		if kill && ctx != nil && ctx.Err() != nil {
			err = ctx.Err()
		}
		return Outcome{Output: sp.Bytes(), Err: err, PID: pid}
	}
	background := func() Outcome {
		// 不杀进程。另起 goroutine 收走 Wait 的结果，调用方先拿回控制权。
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
			// 命令已退出。ctx 也已取消时，release 的 kill 仍为 true。
			kill := ctx != nil && ctx.Err() != nil
			return finish(err, kill)
		case <-ctxDone(ctx):
			// 取消先杀掉当前进程，再让 release 以 kill 为 true 收尾。
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

// preparePythonUTF8 在环境里尚未设置时补上 PYTHONUTF8=1 和 PYTHONIOENCODING=utf-8。
// cmd 为 nil 时直接返回；Env 为 nil 时先换成 os.Environ，再补那两个变量。
func preparePythonUTF8(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}
	env = withDefaultEnv(env, "PYTHONUTF8", "1")
	env = withDefaultEnv(env, "PYTHONIOENCODING", "utf-8")
	cmd.Env = env
}

// preparePwshEnv 在环境里尚未设置时补上 POWERSHELL_TELEMETRY_OPTOUT=1 和 POWERSHELL_UPDATECHECK=Off。
// cmd 为 nil 时直接返回；Env 为 nil 时先换成 os.Environ，再补那两个变量。
func preparePwshEnv(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	env := cmd.Env
	if env == nil {
		env = os.Environ()
	}
	env = withDefaultEnv(env, "POWERSHELL_TELEMETRY_OPTOUT", "1")
	env = withDefaultEnv(env, "POWERSHELL_UPDATECHECK", "Off")
	cmd.Env = env
}

// withDefaultEnv 仅在环境中还没有该键时追加 key=val，键名按 ASCII 大小写不敏感比较。
// env 里已有该键时原样返回，否则把 key=val 追加到末尾。
func withDefaultEnv(env []string, key, val string) []string {
	prefix := key + "="
	for _, e := range env {
		if len(e) >= len(prefix) && stringsEqualFoldPrefix(e, prefix) {
			return env
		}
	}
	return append(env, prefix+val)
}

// stringsEqualFoldPrefix 判断 s 是否以 prefix 开头，只折叠 A 到 Z 的大小写。
func stringsEqualFoldPrefix(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		a, b := s[i], prefix[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		if b >= 'A' && b <= 'Z' {
			b += 'a' - 'A'
		}
		if a != b {
			return false
		}
	}
	return true
}

// ctxDone 在 ctx 为 nil 时返回 nil，select 不会选中这一支。
func ctxDone(ctx context.Context) <-chan struct{} {
	if ctx == nil {
		return nil
	}
	return ctx.Done()
}

// spool 攒住子进程的全部输出，并记下最后一次非空写入，供空闲熔断读取。
type spool struct {
	mu      sync.Mutex
	buf     bytes.Buffer
	last    time.Time    // 最后一次非空写入的时间
	liveN   int          // 已经交给 onChunk 的字节数，不超过 liveStdoutCap
	onChunk func([]byte) // 实时片段回调，可为 nil
}

// Write 追加到缓冲。非空写入更新 last。实时回调只覆盖前 liveStdoutCap 个字节，并在解锁之后调用。
func (s *spool) Write(p []byte) (int, error) {
	s.mu.Lock()
	if len(p) > 0 {
		s.last = time.Now()
	}
	n, err := s.buf.Write(p)
	emit := s.onChunk != nil && s.liveN < liveStdoutCap && n > 0
	var chunk []byte
	if emit {
		take := n
		if remain := liveStdoutCap - s.liveN; take > remain {
			take = remain
		}
		chunk = make([]byte, take)
		copy(chunk, p[:take])
		s.liveN += take
	}
	s.mu.Unlock()
	// 回调放在解锁之后，避免 onChunk 堵住读管道。
	if len(chunk) > 0 {
		s.onChunk(chunk)
	}
	return n, err
}

// Bytes 返回当前缓冲的拷贝。
func (s *spool) Bytes() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]byte, s.buf.Len())
	copy(out, s.buf.Bytes())
	return out
}

// Last 返回最后一次非空写入的时间。
func (s *spool) Last() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last
}

// SigningStatus 在环境变量 YOYO_SIGNED 为 1 时返回 env-asserted，否则返回平台签名探针的缓存结果。
func SigningStatus() string {
	if os.Getenv("YOYO_SIGNED") == "1" {
		return "env-asserted"
	}
	return signingCached()
}

// signingOnce 保证 signingProbe 在进程里只跑一次。
var signingOnce = sync.OnceValue(signingProbe)

// signingCached 返回只计算一次的 signingProbe 结果。
func signingCached() string {
	return signingOnce()
}
