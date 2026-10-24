package remote

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Shenchangxin/yoyo/internal/app"
)

func TestBrowserDoor(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	a.Config.SearchKey = "search-secret"
	n, err := Bind(a)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Close() })

	offer, err := n.BeginPair()
	if err != nil {
		t.Fatal(err)
	}
	uri := offer["uri"].(string)
	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if parsed.Scheme != "http" || !strings.HasPrefix(parsed.Path, "/open") || q.Get("n") == "" || q.Get("fp") == "" || q.Get("p") == "" {
		t.Fatalf("uri %s", uri)
	}
	if parsed.Port() == q.Get("p") {
		t.Fatalf("browser port and tls port must differ: %s", uri)
	}
	local, err := url.Parse(uri)
	if err != nil {
		t.Fatal(err)
	}
	local.Host = "127.0.0.1:" + parsed.Port()

	base := "http://" + local.Host
	bare := &http.Client{Timeout: 20 * time.Second}
	res, err := bare.Get(base + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("health before allow %d", res.StatusCode)
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Jar: jar, Timeout: 20 * time.Second}
	opened, err := client.Get(local.String())
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(opened.Body)
	opened.Body.Close()
	if opened.StatusCode != http.StatusOK || !strings.Contains(string(page), "等这台电脑允许") {
		t.Fatalf("open %d %s", opened.StatusCode, page)
	}

	var id string
	for _, item := range n.Status()["pending"].([]map[string]any) {
		if item["name"] == "浏览器" {
			id = fmt.Sprint(item["id"])
		}
	}
	if id == "" {
		t.Fatal("browser pending missing")
	}
	if _, err := n.Approve(id); err != nil {
		t.Fatal(err)
	}
	connected, _ := n.Status()["browsers"].([]map[string]any)
	if len(connected) != 1 || connected[0]["name"] != "浏览器" {
		t.Fatalf("connected browser %#v", n.Status()["browsers"])
	}
	for _, item := range n.Status()["pending"].([]map[string]any) {
		if item["name"] == "浏览器" {
			t.Fatal("approved browser still pending")
		}
	}
	waitRes, err := client.Get(base + "/open/wait")
	if err != nil {
		t.Fatal(err)
	}
	waitBody, _ := io.ReadAll(waitRes.Body)
	waitRes.Body.Close()
	if waitRes.StatusCode != http.StatusOK || !strings.Contains(string(waitBody), `"ok":true`) {
		t.Fatalf("wait %d %s", waitRes.StatusCode, waitBody)
	}

	cfg, err := client.Get(base + "/api/config")
	if err != nil {
		t.Fatal(err)
	}
	cfgBody, _ := io.ReadAll(cfg.Body)
	cfg.Body.Close()
	if cfg.StatusCode != http.StatusOK || strings.Contains(string(cfgBody), "search-secret") || strings.Contains(string(cfgBody), "search_key") {
		t.Fatalf("config %d %s", cfg.StatusCode, cfgBody)
	}

	cross, err := http.NewRequest(http.MethodGet, base+"/api/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	cross.Header.Set("Origin", "https://evil.example")
	crossed, err := client.Do(cross)
	if err != nil {
		t.Fatal(err)
	}
	crossed.Body.Close()
	if crossed.StatusCode != http.StatusUnauthorized {
		t.Fatalf("cross origin %d", crossed.StatusCode)
	}

	start, err := client.Post(base+"/api/rpc", "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"thread.start","params":{"workspace":"","channel":"agent"}}`))
	if err != nil {
		t.Fatal(err)
	}
	startBody, _ := io.ReadAll(start.Body)
	start.Body.Close()
	if start.StatusCode != http.StatusOK || strings.Contains(string(startBody), "search-secret") {
		t.Fatalf("start %d %s", start.StatusCode, startBody)
	}
	var started struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(startBody, &started); err != nil {
		t.Fatal(err)
	}
	if started.Error != nil || started.Result.ID == "" {
		t.Fatalf("start result %s", startBody)
	}
	modelBody := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"thread.model.set","params":{"session":%q,"model":"gpt-browser"}}`, started.Result.ID)
	model, err := client.Post(base+"/api/rpc", "application/json", strings.NewReader(modelBody))
	if err != nil {
		t.Fatal(err)
	}
	modelRaw, _ := io.ReadAll(model.Body)
	model.Body.Close()
	var modeled struct {
		Error any `json:"error"`
	}
	if err := json.Unmarshal(modelRaw, &modeled); err != nil {
		t.Fatal(err)
	}
	if model.StatusCode != http.StatusOK || modeled.Error != nil {
		t.Fatalf("model %d %s", model.StatusCode, modelRaw)
	}
	meta, err := a.GetSession(started.Result.ID)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Model != "gpt-browser" {
		t.Fatalf("pc model %q", meta.Model)
	}

	again, err := client.Get(local.String())
	if err != nil {
		t.Fatal(err)
	}
	again.Body.Close()
	if again.StatusCode != http.StatusBadRequest {
		t.Fatalf("reused nonce %d", again.StatusCode)
	}

	// A phone that never stored the fetch Set-Cookie still gets in through the
	// document navigation, and a missing or port-less Origin is the same page.
	jar2, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	phone := &http.Client{Jar: jar2, Timeout: 20 * time.Second}
	entered, err := phone.Get(base + "/open/enter?id=" + url.QueryEscape(id))
	if err != nil {
		t.Fatal(err)
	}
	enterBody, _ := io.ReadAll(entered.Body)
	entered.Body.Close()
	if entered.StatusCode != http.StatusOK || !strings.Contains(string(enterBody), `sessionStorage.setItem("yoyo_browser"`) {
		t.Fatalf("enter %d %s", entered.StatusCode, enterBody)
	}
	nullOrigin, err := http.NewRequest(http.MethodGet, base+"/api/sessions", nil)
	if err != nil {
		t.Fatal(err)
	}
	nullOrigin.Header.Set("Origin", "null")
	listed, err := phone.Do(nullOrigin)
	if err != nil {
		t.Fatal(err)
	}
	listBody, _ := io.ReadAll(listed.Body)
	listed.Body.Close()
	if listed.StatusCode != http.StatusOK || !strings.Contains(string(listBody), started.Result.ID) || strings.Contains(string(listBody), "search-secret") {
		t.Fatalf("sessions %d %s", listed.StatusCode, listBody)
	}

	u, err := url.Parse(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	var token string
	for _, c := range jar2.Cookies(u) {
		if c.Name == "yoyo_session" {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("session cookie missing")
	}
	headerOnly := &http.Client{Timeout: 20 * time.Second}
	byHeader, err := http.NewRequest(http.MethodGet, base+"/api/sessions", nil)
	if err != nil {
		t.Fatal(err)
	}
	byHeader.Header.Set("X-Yoyo-Session", token)
	byHeader.Header.Set("Origin", "null")
	headed, err := headerOnly.Do(byHeader)
	if err != nil {
		t.Fatal(err)
	}
	headedBody, _ := io.ReadAll(headed.Body)
	headed.Body.Close()
	if headed.StatusCode != http.StatusOK || !strings.Contains(string(headedBody), started.Result.ID) {
		t.Fatalf("header session %d %s", headed.StatusCode, headedBody)
	}
	evilHeader, err := http.NewRequest(http.MethodGet, base+"/api/sessions", nil)
	if err != nil {
		t.Fatal(err)
	}
	evilHeader.Header.Set("X-Yoyo-Session", token)
	evilHeader.Header.Set("Origin", "https://evil.example")
	evilRes, err := headerOnly.Do(evilHeader)
	if err != nil {
		t.Fatal(err)
	}
	evilRes.Body.Close()
	if evilRes.StatusCode != http.StatusUnauthorized {
		t.Fatalf("evil header origin %d", evilRes.StatusCode)
	}
}

func TestBrowserHTMLDisablesDrag(t *testing.T) {
	h := markBrowser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Content-Type", "text/css")
			_, _ = io.WriteString(w, ".drag{color:red}")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, "<!doctype html><html><head><title>Yoyo</title></head><body><div class=\"drag\"></div></body></html>")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "yoyo-browser-touch") || !strings.Contains(body, "no-drag") || !strings.Contains(body, `dataset.browser="1"`) {
		t.Fatalf("html %d %s", rec.Code, body)
	}
	if rec.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache %s", rec.Header().Get("Cache-Control"))
	}
	css := httptest.NewRecorder()
	h.ServeHTTP(css, httptest.NewRequest(http.MethodGet, "/app.css", nil))
	if css.Body.String() != ".drag{color:red}" || strings.Contains(css.Body.String(), "yoyo-browser-touch") {
		t.Fatalf("css rewritten: %s", css.Body.String())
	}
}

func TestBrowserDoorRejectsPublicPeer(t *testing.T) {
	a, err := app.Open(t.TempDir(), evalsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	n, err := Bind(a)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = n.Close() })
	req := httptest.NewRequest(http.MethodGet, "/open?n=nope", nil)
	req.RemoteAddr = "8.8.8.8:443"
	rec := httptest.NewRecorder()
	n.serveBrowserHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("public %d", rec.Code)
	}
}
