// procgroup_test.go 锁住 Kill 会结束整棵进程树：测试进程自己扮演父进程和孙进程。

package procgroup

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

// helperEnv 告诉测试二进制该扮演哪一层。
const helperEnv = "PROCGROUP_HELPER"

// TestMain 在 helperEnv 设置时扮演父进程或孙进程，否则照常跑测试。
func TestMain(m *testing.M) {
	switch os.Getenv(helperEnv) {
	case "parent":
		child := exec.Command(os.Args[0], "-test.run=^$")
		child.Env = append(os.Environ(), helperEnv+"=child")
		if err := child.Start(); err != nil {
			fmt.Println("error", err)
			os.Exit(2)
		}
		fmt.Println("child", child.Process.Pid)
		time.Sleep(time.Minute)
		os.Exit(0)
	case "child":
		time.Sleep(time.Minute)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// TestKillEndsGrandchild 锁住 Kill 之后主进程和它派生的孙进程都不在了。
func TestKillEndsGrandchild(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	cmd.Env = append(os.Environ(), helperEnv+"=parent")
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	p, err := Start(cmd, Options{Hide: true})
	if err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(out).ReadString('\n')
	if err != nil {
		_ = p.Kill()
		t.Fatal(err)
	}
	fields := strings.Fields(line)
	if len(fields) != 2 || fields[0] != "child" {
		_ = p.Kill()
		t.Fatalf("helper said %q", line)
	}
	grandchild, _ := strconv.Atoi(fields[1])
	if !alive(grandchild) {
		_ = p.Kill()
		t.Fatal("grandchild did not start")
	}
	if err := p.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-p.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("parent still running after Kill")
	}
	deadline := time.Now().Add(5 * time.Second)
	for alive(grandchild) {
		if time.Now().After(deadline) {
			t.Fatalf("grandchild %d survived Kill", grandchild)
		}
		time.Sleep(50 * time.Millisecond)
	}
	if p.ExitCode() == 0 {
		t.Fatal("a killed parent should not exit cleanly")
	}
}
