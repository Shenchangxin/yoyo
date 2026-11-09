// install_test.go 锁住托管安装：Node 从第二个镜像回落下载并校验、npm 命令和环境的拼法、二进制归档的哈希校验，以及 npm 源的选择。
// 网络全部换成 httptest，npm 和 node 的执行换成假的 Run。

package buddy

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// sum 返回十六进制 sha256。
func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// tarGz 打一个 tar.gz，files 的值是内容。
func tarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		_, _ = tw.Write([]byte(body))
	}
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

// zipOf 打一个 zip。
func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(body))
	}
	_ = zw.Close()
	return buf.Bytes()
}

// useRegistry 把目录换成只含 agents 的测试注册表。
func (f *fixture) useRegistry(agents ...RegistryAgent) {
	f.m.mu.Lock()
	f.m.reg = Registry{Version: "1.0.0", Agents: agents}
	f.m.rebuildLocked()
	f.m.mu.Unlock()
}

// waitInstalled 等安装任务结束，返回最后的视图。
func (f *fixture) waitInstalled(t *testing.T, id string) View {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		f.m.mu.Lock()
		busy := f.m.jobs[id] != nil
		f.m.mu.Unlock()
		if !busy {
			return f.m.view(id)
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("install did not finish")
	return View{}
}

// distTags 是一个只回 dist-tags 的假 npm 源。
func distTags(latest string, delay time.Duration, hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		time.Sleep(delay)
		if latest == "" || !strings.HasSuffix(r.URL.Path, "/dist-tags") {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"latest": latest})
	}))
}

// TestManagedNodeAndNPM 锁住托管 Node 第一个镜像失败时用第二个、校验 sha256 并确认版本；npm 用 node 跑 npm-cli.js，
// 参数带 --prefix、--registry 和带版本的包名，环境是清洗过的白名单且 Node 目录在 PATH 最前面；装完记下 node 加 JS 入口。
func TestManagedNodeAndNPM(t *testing.T) {
	prefix := "node-" + nodeVersion + "-linux-x64/"
	archive := tarGz(t, map[string]string{prefix + "bin/node": "elf", prefix + "lib/node_modules/npm/bin/npm-cli.js": "// npm"})
	old := nodeSHA256["linux-x64.tar.gz"]
	nodeSHA256["linux-x64.tar.gz"] = sum(archive)
	t.Cleanup(func() { nodeSHA256["linux-x64.tar.gz"] = old })
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "down", http.StatusBadGateway) }))
	defer bad.Close()
	var nodeHits atomic.Int32
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/"+nodeVersion+"/node-"+nodeVersion+"-linux-x64.tar.gz" {
			http.NotFound(w, r)
			return
		}
		nodeHits.Add(1)
		_, _ = w.Write(archive)
	}))
	defer good.Close()
	npmjs := distTags("1.2.3", 0, nil)
	defer npmjs.Close()
	mirror := distTags("", 0, nil)
	defer mirror.Close()

	var mu sync.Mutex
	var npmRun Cmd
	f := newFixture(t, func(o *Options) {
		o.HTTP = &http.Client{}
		o.NodeMirrors = []string{bad.URL + "/", good.URL + "/"}
		o.NPMRegistries = map[string]string{"npmjs": npmjs.URL + "/", "npmmirror": mirror.URL + "/"}
		o.BaseEnv = func() []string {
			return []string{"PATH=/usr/bin", "HOME=/home/me", "YOYO_TOKEN=x", "OPENAI_API_KEY=sk-should-not-pass", "LC_ALL=C"}
		}
		o.Run = func(ctx context.Context, c Cmd) ([]byte, error) {
			if len(c.Launch.Args) == 1 && c.Launch.Args[0] == "--version" {
				return []byte(nodeVersion + "\n"), nil
			}
			mu.Lock()
			npmRun = c
			mu.Unlock()
			dir := c.Launch.Args[3]
			pkg := filepath.Join(dir, "node_modules", "fake-acp")
			_ = os.MkdirAll(filepath.Join(pkg, "dist"), 0o755)
			_ = os.WriteFile(filepath.Join(pkg, "package.json"), []byte(`{"name":"fake-acp","bin":{"fake-acp":"dist/index.js"}}`), 0o644)
			_ = os.WriteFile(filepath.Join(pkg, "dist", "index.js"), []byte("#!/usr/bin/env node\n"), 0o644)
			return nil, nil
		}
	})
	f.useRegistry(RegistryAgent{ID: "fake", Name: "Fake", Version: "1.2.3", Distribution: Distribution{NPX: &PackageTarget{Package: "fake-acp@1.2.3", Args: []string{"--acp"}}}})
	if _, err := f.m.Install("fake", "managed"); err != nil {
		t.Fatal(err)
	}
	v := f.waitInstalled(t, "fake")
	if v.Install.Kind != "managed" || v.Install.Source != "npm" || v.Version != "1.2.3" {
		t.Fatalf("view %+v detail %q", v.Install, v.Detail)
	}
	if nodeHits.Load() != 1 {
		t.Fatalf("node downloads %d", nodeHits.Load())
	}
	nodeExe := f.m.managedNodeExe()
	mu.Lock()
	run := npmRun
	mu.Unlock()
	args := run.Launch.Args
	if run.Launch.Command != nodeExe || !strings.HasSuffix(filepath.ToSlash(args[0]), "lib/node_modules/npm/bin/npm-cli.js") || args[1] != "install" || args[2] != "--prefix" {
		t.Fatalf("npm launch %+v", run.Launch)
	}
	if args[len(args)-1] != "fake-acp@1.2.3" || args[len(args)-2] != npmjs.URL+"/" || args[len(args)-3] != "--registry" {
		t.Fatalf("npm args %v", args)
	}
	env := strings.Join(run.Launch.Env, "\n")
	if strings.Contains(env, "YOYO_TOKEN") || strings.Contains(env, "sk-should-not-pass") || !strings.Contains(env, "LC_ALL=C") || !strings.Contains(env, "npm_config_cache=") {
		t.Fatalf("npm env %v", run.Launch.Env)
	}
	if path := getEnv(run.Launch.Env, "PATH"); !strings.HasPrefix(path, filepath.Dir(nodeExe)) {
		t.Fatalf("PATH %s", path)
	}
	rec, _ := f.m.readInstall("fake")
	if rec == nil || rec.Command != nodeExe || rec.Entry == "" || len(rec.Args) != 2 || rec.Args[0] != rec.Entry || rec.Args[1] != "--acp" {
		t.Fatalf("record %+v", rec)
	}
	l, err := f.m.Launch("fake")
	if err != nil || l.Command != nodeExe || l.Args[0] != rec.Entry {
		t.Fatalf("launch %+v %v", l, err)
	}
}

// TestNPMSourcePick 锁住 npm 源的选择：auto 时取先回且有这个版本的源；镜像落后时用 npmjs；偏好写死时不发请求。
func TestNPMSourcePick(t *testing.T) {
	fast := distTags("1.2.3", 0, nil)
	defer fast.Close()
	slow := distTags("1.2.3", 400*time.Millisecond, nil)
	defer slow.Close()
	lagging := distTags("1.1.0", 0, nil)
	defer lagging.Close()
	pick := func(npmjs, mirror *httptest.Server, pref string) string {
		f := newFixture(t, func(o *Options) {
			o.HTTP = &http.Client{}
			o.NPMRegistries = map[string]string{"npmjs": npmjs.URL + "/", "npmmirror": mirror.URL + "/"}
		})
		_ = f.settings.save(func(s *Settings) { s.NPMRegistry = pref })
		return f.m.npmRegistryFor(context.Background(), "@scope/fake-acp", "1.2.3")
	}
	if got := pick(slow, fast, ""); got != fast.URL+"/" {
		t.Fatalf("fast mirror not picked: %s", got)
	}
	if got := pick(slow, lagging, ""); got != slow.URL+"/" {
		t.Fatalf("lagging mirror picked: %s", got)
	}
	var hits atomic.Int32
	counted := distTags("1.2.3", 0, &hits)
	defer counted.Close()
	if got := pick(counted, counted, "npmmirror"); got != counted.URL+"/" || hits.Load() != 0 {
		t.Fatalf("explicit pref %s hits %d", got, hits.Load())
	}
	if !versionAtMost("1.2.3", "1.10.0") || versionAtMost("2.0.0", "1.9.9") || !versionAtMost("1.2.3-beta.1", "1.2.3") {
		t.Fatal("version compare")
	}
}

// TestBinaryInstallVerifiesHash 锁住二进制归档按 sha256 校验：对得上就解开并记下已校验；对不上就失败、不留安装记录、回到未安装。
func TestBinaryInstallVerifiesHash(t *testing.T) {
	archive := zipOf(t, map[string]string{"dist/bin/fake-acp": "binary"})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) }))
	defer srv.Close()
	target := func(hash string) RegistryAgent {
		return RegistryAgent{ID: "bin", Name: "Bin", Version: "2.0.0", Distribution: Distribution{Binary: map[string]BinaryTarget{
			"linux-x86_64": {Archive: srv.URL + "/fake.zip", Cmd: `./dist\bin/fake-acp`, Args: []string{"acp"}, SHA256: hash},
		}}}
	}

	f := newFixture(t, func(o *Options) { o.HTTP = srv.Client() })
	f.useRegistry(target(sum(archive)))
	if _, err := f.m.Install("bin", ""); err != nil {
		t.Fatal(err)
	}
	v := f.waitInstalled(t, "bin")
	rec, _ := f.m.readInstall("bin")
	if rec == nil || !rec.Verified || rec.SHA256 != sum(archive) || v.Install.Unverified || filepath.Base(rec.Command) != "fake-acp" {
		t.Fatalf("record %+v view %+v detail %q", rec, v.Install, v.Detail)
	}
	if l, err := f.m.Launch("bin"); err != nil || l.Command != rec.Command || len(l.Args) != 1 || l.Args[0] != "acp" {
		t.Fatalf("launch %+v %v", l, err)
	}

	f2 := newFixture(t, func(o *Options) { o.HTTP = srv.Client() })
	f2.useRegistry(target(strings.Repeat("0", 64)))
	if _, err := f2.m.Install("bin", "managed"); err != nil {
		t.Fatal(err)
	}
	v = f2.waitInstalled(t, "bin")
	if rec, _ := f2.m.readInstall("bin"); rec != nil || v.State != StateNotInstalled || !strings.Contains(v.Detail, "sha256") {
		t.Fatalf("mismatch accepted: rec %+v view %s %q", rec, v.State, v.Detail)
	}

	f3 := newFixture(t, func(o *Options) { o.HTTP = srv.Client() })
	f3.useRegistry(target(""))
	_, _ = f3.m.Install("bin", "managed")
	f3.waitInstalled(t, "bin")
	if rec, _ := f3.m.readInstall("bin"); rec == nil || rec.Verified || rec.SHA256 != sum(archive) {
		t.Fatalf("unverified record %+v", rec)
	}
	if v, _ := f3.m.Get("bin"); !v.Install.Unverified {
		t.Fatal("unverified binary not flagged")
	}
}

// rewrite 把对 github.com 的请求判成失败，其余主机都转到本地的测试服务，并记下请求过的主机。
type rewrite struct {
	target *httptest.Server
	mu     sync.Mutex
	hosts  []string
}

// RoundTrip 记下主机后转发或失败。
func (r *rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	r.mu.Lock()
	r.hosts = append(r.hosts, req.URL.Host)
	r.mu.Unlock()
	if req.URL.Host == "github.com" {
		return nil, errors.New("connection reset")
	}
	out := req.Clone(req.Context())
	out.URL.Scheme = "https"
	out.URL.Host = strings.TrimPrefix(r.target.URL, "https://")
	out.Host = ""
	return r.target.Client().Transport.RoundTrip(out)
}

// TestBinaryMirrorOnlyWithHash 锁住有 sha256 的 GitHub 归档在原地址失败时回落到 ghfast；没有 sha256 的不走镜像。
func TestBinaryMirrorOnlyWithHash(t *testing.T) {
	archive := zipOf(t, map[string]string{"fake-acp": "binary"})
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) }))
	defer srv.Close()
	agent := func(hash string) RegistryAgent {
		return RegistryAgent{ID: "gh", Name: "GH", Version: "1.0.0", Distribution: Distribution{Binary: map[string]BinaryTarget{
			"linux-x86_64": {Archive: "https://github.com/o/r/releases/download/v1/fake.zip", Cmd: "./fake-acp", SHA256: hash},
		}}}
	}
	rt := &rewrite{target: srv}
	f := newFixture(t, func(o *Options) { o.HTTP = &http.Client{Transport: rt} })
	f.useRegistry(agent(sum(archive)))
	_, _ = f.m.Install("gh", "managed")
	f.waitInstalled(t, "gh")
	if rec, _ := f.m.readInstall("gh"); rec == nil || !rec.Verified {
		t.Fatalf("mirror install failed, hosts %v", rt.hosts)
	}
	if len(rt.hosts) < 2 || rt.hosts[0] != "github.com" || rt.hosts[1] != "ghfast.top" {
		t.Fatalf("hosts %v", rt.hosts)
	}

	rt2 := &rewrite{target: srv}
	f2 := newFixture(t, func(o *Options) { o.HTTP = &http.Client{Transport: rt2} })
	f2.useRegistry(agent(""))
	_, _ = f2.m.Install("gh", "managed")
	v := f2.waitInstalled(t, "gh")
	if rec, _ := f2.m.readInstall("gh"); rec != nil || v.State != StateNotInstalled {
		t.Fatalf("unverified archive used a mirror: %+v", rec)
	}
	for _, h := range rt2.hosts {
		if h != "github.com" {
			t.Fatalf("hosts %v", rt2.hosts)
		}
	}
}

// TestInstallRejectsPlainHTTP 锁住二进制归档地址不是 https 时直接失败。
func TestInstallRejectsPlainHTTP(t *testing.T) {
	f := newFixture(t, nil)
	f.useRegistry(RegistryAgent{ID: "bin", Name: "Bin", Version: "1", Distribution: Distribution{Binary: map[string]BinaryTarget{
		"linux-x86_64": {Archive: "http://example.com/a.zip", Cmd: "a", SHA256: strings.Repeat("a", 64)},
	}}})
	_, _ = f.m.Install("bin", "managed")
	if v := f.waitInstalled(t, "bin"); v.State != StateNotInstalled || !strings.Contains(v.Detail, "https") {
		t.Fatalf("view %s %q", v.State, v.Detail)
	}
}
