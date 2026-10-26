// 本文件锁住 resolveURL 和 URL 对协议的取舍：https 原样返回，file 当成路径，javascript、data、ftp 和无协议被拒绝。
// 不启动系统程序。由 go test 调用。

package hostopen

import (
	"runtime"
	"strings"
	"testing"
)

// TestResolveURLHTTP 要求 https 地址原样返回，并且 file 为假。
func TestResolveURLHTTP(t *testing.T) {
	got, file, err := resolveURL("https://example.com/x?q=1")
	if err != nil {
		t.Fatal(err)
	}
	if file {
		t.Fatal("https should not be treated as a file")
	}
	if got != "https://example.com/x?q=1" {
		t.Fatalf("got %q", got)
	}
}

// TestResolveURLRejects 要求空串、无协议、javascript、data、ftp 和没有主机的 http 都返回错误。
func TestResolveURLRejects(t *testing.T) {
	for _, raw := range []string{
		"",
		"example.com",
		"javascript:alert(1)",
		"data:text/html,hi",
		"ftp://files.example",
		"http://",
	} {
		if _, _, err := resolveURL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}

// TestResolveURLFile 要求 file URL 的 file 为真。Windows 上路径含 page.html 且盘符形式可被接受，其他系统的结果里要有 page.html。
func TestResolveURLFile(t *testing.T) {
	got, file, err := resolveURL("file:///C:/tmp/page.html")
	if err != nil {
		t.Fatal(err)
	}
	if !file {
		t.Fatal("file:// should resolve as a path")
	}
	if runtime.GOOS == "windows" {
		if !strings.EqualFold(got, `C:\tmp\page.html`) && !strings.EqualFold(got, `C:/tmp/page.html`) {
			t.Fatalf("windows path %q", got)
		}
		return
	}
	if !strings.Contains(got, "page.html") {
		t.Fatalf("unix path %q", got)
	}
}

// TestURLRejectsUnsafeSchemes 要求 URL 对 javascript、data 和空串返回错误。
func TestURLRejectsUnsafeSchemes(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "data:text/html,x", ""} {
		if err := URL(raw); err == nil {
			t.Fatalf("expected error for %q", raw)
		}
	}
}
