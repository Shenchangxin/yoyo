// bridge.go 拥有 stdio 到控制面的桥：只会启动 stdio MCP 服务器的 agent 把 yoyo 自己当服务器启动（YOYO_MCP_BRIDGE=1），
// 桥把每条消息转成对 server.go 的 HTTP 请求，再把响应写回。不拥有工具和登记（server.go），也不决定何时启动它（acpworker 写进 mcpServers）。
// stdout 只写协议；桥自己不打日志。

package mcpserve

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// maxReply 是桥从控制面读一个响应的字节上限。
const maxReply = 16 << 20

// RunBridge 逐条读 stdin 里的 JSON-RPC 消息，POST 给 endpoint，把响应写成一行输出到 stdout。
// 消息按行分隔；以 Content-Length 头开头的按头里的长度读。endpoint 必须是本机回环地址上的 http 地址，token 作为 Bearer 发出。
// 记住 initialize 回的会话号，之后每个请求都带上。stdin 读完时结束会话并返回 nil。
// 只有读 stdin 或写 stdout 出错才返回错误；单条请求失败时回一条 JSON-RPC 错误，通知失败时不回。
func RunBridge(stdin io.Reader, stdout io.Writer, endpoint, token string) error {
	if err := loopbackURL(endpoint); err != nil {
		return err
	}
	if strings.TrimSpace(token) == "" {
		return errors.New("mcpserve: bridge needs a token")
	}
	b := &bridge{
		endpoint: endpoint,
		token:    token,
		client:   &http.Client{Timeout: 2 * time.Minute, Transport: &http.Transport{Proxy: nil}},
	}
	in := bufio.NewReaderSize(stdin, 64<<10)
	for {
		msg, err := readFrame(in)
		if len(msg) > 0 {
			if werr := b.forward(stdout, msg); werr != nil {
				return werr
			}
		}
		if errors.Is(err, io.EOF) {
			b.end()
			return nil
		}
		if err != nil {
			return err
		}
	}
}

// bridge 是桥的状态：控制面地址、token、HTTP 客户端，以及 initialize 回的会话号。
type bridge struct {
	endpoint string
	token    string
	client   *http.Client
	session  string
}

// forward 把一条消息交给控制面，并把要回的响应写进 out。
func (b *bridge) forward(out io.Writer, msg []byte) error {
	var head message
	isReq := json.Unmarshal(msg, &head) == nil && head.request()
	resp, err := b.send(context.Background(), http.MethodPost, msg)
	if err != nil {
		return reject(out, head.ID, isReq, "yoyo control plane unreachable")
	}
	defer resp.Body.Close()
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		b.session = sid
	} else if resp.StatusCode == http.StatusNotFound {
		b.session = ""
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxReply))
	if err != nil {
		return reject(out, head.ID, isReq, "yoyo control plane read failed")
	}
	var line bytes.Buffer
	if json.Compact(&line, data) == nil && line.Len() > 0 {
		line.WriteByte('\n')
		_, err := out.Write(line.Bytes())
		return err
	}
	if resp.StatusCode >= 300 {
		return reject(out, head.ID, isReq, fmt.Sprintf("yoyo control plane: HTTP %d", resp.StatusCode))
	}
	return nil
}

// end 结束会话。控制面不在时不管。
func (b *bridge) end() {
	if b.session == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if resp, err := b.send(ctx, http.MethodDelete, nil); err == nil {
		resp.Body.Close()
	}
	b.session = ""
}

// send 带上 token 和会话号发一个请求。
func (b *bridge) send(ctx context.Context, method string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, b.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if b.session != "" {
		req.Header.Set("Mcp-Session-Id", b.session)
	}
	return b.client.Do(req)
}

// reject 在 isReq 为真时给 id 回一条 JSON-RPC 错误。通知和响应失败时什么都不写。
func reject(out io.Writer, id json.RawMessage, isReq bool, text string) error {
	if !isReq {
		return nil
	}
	data, err := json.Marshal(errorReply(id, codeInternal, text))
	if err != nil {
		return err
	}
	_, err = out.Write(append(data, '\n'))
	return err
}

// readFrame 读一条消息，空行跳过。行首是 Content-Length 头时读完其余头，再按长度读正文。
// 最后一行没有换行时照样返回它，下一次再返回 io.EOF。
func readFrame(r *bufio.Reader) ([]byte, error) {
	for {
		line, err := r.ReadBytes('\n')
		text := bytes.TrimSpace(line)
		if len(text) == 0 {
			if err != nil {
				return nil, err
			}
			continue
		}
		n, framed := contentLength(text)
		if !framed {
			return text, nil
		}
		if err != nil {
			return nil, err
		}
		for {
			h, herr := r.ReadBytes('\n')
			if herr != nil {
				return nil, herr
			}
			if len(bytes.TrimSpace(h)) == 0 {
				break
			}
		}
		if n < 0 || n > MaxBody {
			return nil, fmt.Errorf("mcpserve: frame of %d bytes", n)
		}
		body := make([]byte, n)
		if _, err := io.ReadFull(r, body); err != nil {
			return nil, err
		}
		return body, nil
	}
}

// contentLength 在 line 是 Content-Length 头时返回长度和真。
func contentLength(line []byte) (int, bool) {
	name, value, ok := strings.Cut(string(line), ":")
	if !ok || !strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return -1, true
	}
	return n, true
}

// loopbackURL 检查 endpoint 是本机回环地址上的 http 地址，免得 token 发到别处。
func loopbackURL(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "http" || !loopbackName(u.Hostname()) {
		return errors.New("mcpserve: bridge endpoint must be http on the loopback address")
	}
	return nil
}
