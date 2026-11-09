// helper.go 拥有假 agent 的子进程版：测试二进制看到 ACPTEST_FAKE 就按名字选内置脚本，在 stdin/stdout 上扮演 agent。
// 用它的测试包要在 TestMain 里先调 RunIfHelper。

package acptest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/Shenchangxin/yoyo/internal/acp"
)

// HelperEnv 选内置脚本的名字。
const HelperEnv = "ACPTEST_FAKE"

// CapsEnv 用 JSON 覆盖内置脚本的 agentCapabilities。
const CapsEnv = "ACPTEST_CAPS"

// named 是子进程版能用的内置脚本。函数每次返回一份新的。
var named = map[string]func() Script{
	"echo": func() Script { return Script{} },
	"crash": func() Script {
		return Script{OnPrompt: func(t *Turn) acp.StopReason {
			t.Say("about to fail")
			t.Crash(3, "boom: model exploded")
			return acp.StopEndTurn
		}}
	},
	"auth": func() Script {
		return Script{NeedAuth: true, AuthMethods: []acp.AuthMethod{
			{ID: "login", Name: "Log in"},
			{ID: "term", Name: "Log in from a terminal", Type: "terminal", Args: []string{"--login"}},
		}}
	},
	"slow": func() Script {
		return Script{OnPrompt: func(t *Turn) acp.StopReason {
			if t.WaitCancel(time.Minute) {
				return acp.StopCancelled
			}
			return acp.StopEndTurn
		}}
	},
	"tools": func() Script { return Script{OnPrompt: ToolsTurn} },
	"write": func() Script {
		return Script{OnPrompt: func(t *Turn) acp.StopReason {
			if err := t.WriteFile(filepath.Join(t.Session.Cwd, "hello.txt"), "hi from fake"); err != nil {
				t.Say("write failed: " + err.Error())
			} else {
				t.Say("wrote hello.txt")
			}
			return acp.StopEndTurn
		}}
	},
	"perm": func() Script {
		return Script{OnPrompt: func(t *Turn) acp.StopReason {
			out, err := t.Permission(map[string]any{"toolCallId": "p1", "title": "rm -rf build", "kind": "execute", "rawInput": map[string]any{"command": "rm -rf build"}}, DefaultPermOptions())
			if err != nil {
				t.Say("permission error: " + err.Error())
				return acp.StopEndTurn
			}
			t.Say("outcome " + out.Outcome + " " + out.OptionID)
			return acp.StopEndTurn
		}}
	},
	"models": func() Script {
		return Script{Models: []acp.ModelInfo{{ModelID: "m1", Name: "Model One"}, {ModelID: "m2", Name: "Model Two"}}}
	},
	"badversion": func() Script { return Script{Version: 2} },
}

// Register 登记一个内置脚本。测试包在 TestMain 之前（init 里）调用它，子进程版才能按名字找到。
func Register(name string, mk func() Script) { named[name] = mk }

// RunIfHelper 在 HelperEnv 设置时用对应脚本在 stdin/stdout 上跑假 agent，连接断开后退出进程。没设置时返回假。
func RunIfHelper() bool {
	name := os.Getenv(HelperEnv)
	if name == "" {
		return false
	}
	mk := named[name]
	if mk == nil {
		exitProcess(2, "acptest: unknown script "+name)
		return true
	}
	script := mk()
	if raw := os.Getenv(CapsEnv); raw != "" {
		var caps map[string]any
		if json.Unmarshal([]byte(raw), &caps) == nil {
			script.Caps = caps
		}
	}
	a := newAgent(script, os.Stdin, os.Stdout, exitProcess)
	<-a.Done()
	os.Exit(0)
	return true
}

// Command 返回启动内置脚本 name 的命令、参数和环境。caps 非 nil 时覆盖能力。
func Command(name string, caps map[string]any) (string, []string, []string) {
	env := append(os.Environ(), HelperEnv+"="+name)
	if caps != nil {
		raw, _ := json.Marshal(caps)
		env = append(env, CapsEnv+"="+string(raw))
	}
	return os.Args[0], []string{"-test.run=^$"}, env
}

// ToolsTurn 是一轮典型的改文件：正文、一次 edit 工具调用和它的 diff、计划、用量，再收尾一句。
func ToolsTurn(t *Turn) acp.StopReason {
	path := filepath.Join(t.Session.Cwd, "a.txt")
	t.Say("先看一下。")
	t.Think("想想怎么改")
	t.Tool(map[string]any{"toolCallId": "t1", "title": "Edit a.txt", "kind": "edit", "status": "pending", "locations": []map[string]any{{"path": path}}})
	t.ToolUpdate(map[string]any{"toolCallId": "t1", "status": "completed", "content": []map[string]any{{"type": "diff", "path": path, "oldText": "a", "newText": "b"}}})
	t.Update(map[string]any{"sessionUpdate": acp.UpdatePlan, "entries": []map[string]any{{"content": "改 a.txt", "status": "completed"}, {"content": "跑测试", "status": "in_progress"}}})
	t.Update(map[string]any{"sessionUpdate": acp.UpdateUsage, "used": 1200, "size": 200000, "cost": map[string]any{"amount": 0.02, "currency": "USD"}})
	t.Say("改好了。")
	return acp.StopEndTurn
}
