package connector

import (
	"bufio"
	"encoding/base64"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// smtpSeen 是假 SMTP 服务器收到的东西：登录用的 PLAIN 载荷、信封发件人和收件人、整封信。
type smtpSeen struct {
	mu    sync.Mutex
	auth  string
	from  string
	rcpts []string
	data  string
}

// fakeSMTP 是只认这一套对话的假服务器：问候 → EHLO（宣告 AUTH PLAIN）→ AUTH → MAIL → RCPT → DATA → QUIT。
// rejectAuth 为真时 AUTH 回 535，模拟授权码填错。
func fakeSMTP(t *testing.T, conn net.Conn, seen *smtpSeen, rejectAuth bool) {
	t.Helper()
	defer conn.Close()
	w := bufio.NewWriter(conn)
	r := bufio.NewReader(conn)
	say := func(s string) { _, _ = w.WriteString(s + "\r\n"); _ = w.Flush() }
	say("220 smtp.qq.com ready")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			say("250-smtp.qq.com")
			say("250 AUTH PLAIN LOGIN")
		case strings.HasPrefix(upper, "AUTH PLAIN"):
			seen.mu.Lock()
			seen.auth = strings.TrimSpace(line[len("AUTH PLAIN"):])
			seen.mu.Unlock()
			if rejectAuth {
				say("535 Login fail. Please enter your authorization code to login")
			} else {
				say("235 Authentication successful")
			}
		case strings.HasPrefix(upper, "MAIL FROM:"):
			seen.mu.Lock()
			seen.from = line[len("MAIL FROM:"):]
			seen.mu.Unlock()
			say("250 OK")
		case strings.HasPrefix(upper, "RCPT TO:"):
			seen.mu.Lock()
			seen.rcpts = append(seen.rcpts, line[len("RCPT TO:"):])
			seen.mu.Unlock()
			say("250 OK")
		case upper == "DATA":
			say("354 End data with <CR><LF>.<CR><LF>")
			var data strings.Builder
			for {
				l, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if l == ".\r\n" {
					break
				}
				data.WriteString(l)
			}
			seen.mu.Lock()
			seen.data = data.String()
			seen.mu.Unlock()
			say("250 OK: queued")
		case upper == "QUIT":
			say("221 Bye")
			return
		default:
			// 包括 AUTH 失败后客户端发的「*」取消：回 501，客户端接着 QUIT。
			say("501 unknown")
		}
	}
}

// withFakeSMTP 把 smtpDial 换成连假服务器，并检查连的是哪台主机。
func withFakeSMTP(t *testing.T, wantHost string, rejectAuth bool) *smtpSeen {
	t.Helper()
	seen := &smtpSeen{}
	prev := smtpDial
	t.Cleanup(func() { smtpDial = prev })
	smtpDial = func(host string) (net.Conn, error) {
		if host != wantHost {
			t.Errorf("smtp host = %s, want %s", host, wantHost)
		}
		client, server := net.Pipe()
		go fakeSMTP(t, server, seen, rejectAuth)
		return client, nil
	}
	return seen
}

// TestSendMailboxOverSMTP 确认 QQ 邮箱发信：用授权码 PLAIN 登录，信封收件人含 To / Cc / Bcc（去重），
// 信头有 From、Date、Message-ID 和工单的 Reply-To，中文主题编过码，Bcc 不出现在信头里，正文开头的点被转义后原样送到。
func TestSendMailboxOverSMTP(t *testing.T) {
	seen := withFakeSMTP(t, "smtp.qq.com", false)
	d := Draft{
		To: "张三 <zhang@corp.com>, li@corp.com", Cc: "li@corp.com; boss@corp.com", Bcc: "me+copy@qq.com",
		Subject: "报价确认 [yoyo:tkt-12]", Body: "您好，报价见下。\r\n.以点开头的一行\r\n", Alias: "ada+yoyo-tkt-12@qq.com",
	}
	if err := sendMailbox("ada@qq.com", "auth-code", "qq-mail", d); err != nil {
		t.Fatal(err)
	}
	seen.mu.Lock()
	defer seen.mu.Unlock()
	if plain, _ := base64.StdEncoding.DecodeString(seen.auth); string(plain) != "\x00ada@qq.com\x00auth-code" {
		t.Fatalf("auth = %q", plain)
	}
	if !strings.Contains(seen.from, "ada@qq.com") {
		t.Fatalf("mail from = %q", seen.from)
	}
	want := []string{"zhang@corp.com", "li@corp.com", "boss@corp.com", "me+copy@qq.com"}
	if len(seen.rcpts) != len(want) {
		t.Fatalf("rcpts = %q", seen.rcpts)
	}
	for i, w := range want {
		if !strings.Contains(seen.rcpts[i], w) {
			t.Fatalf("rcpt %d = %q, want %q", i, seen.rcpts[i], w)
		}
	}
	for _, h := range []string{"From: ada@qq.com\r\n", "Date: ", "Message-ID: <", "Reply-To: ada+yoyo-tkt-12@qq.com\r\n", "Subject: =?utf-8?b?"} {
		if !strings.Contains(seen.data, h) {
			t.Fatalf("data missing %q:\n%s", h, seen.data)
		}
	}
	if strings.Contains(strings.ToLower(seen.data), "bcc:") || strings.Contains(seen.data, "me+copy@qq.com") {
		t.Fatalf("bcc leaked into headers:\n%s", seen.data)
	}
	// 服务器收到的是转义过的「..」开头；去掉转义后就是原文。
	if !strings.Contains(seen.data, "\r\n..以点开头的一行\r\n") {
		t.Fatalf("dot-stuffing:\n%s", seen.data)
	}
}

// TestSendMailboxHostsAndErrors 确认网易按域名选 SMTP 服务器、授权码被拒时说清要填授权码，
// 以及没有收件人、没有授权码时不去连服务器。
func TestSendMailboxHostsAndErrors(t *testing.T) {
	for endpoint, want := range map[string]string{"a@163.com": "smtp.163.com", "a@126.com": "smtp.126.com", "a@yeah.net": "smtp.yeah.net"} {
		if host, _, err := smtpHost("netease-mail", endpoint); err != nil || host != want {
			t.Errorf("smtpHost(%s) = %q %v, want %q", endpoint, host, err, want)
		}
	}
	if host, _, err := smtpHost("local", "me@corp.com"); err != nil || host != "smtp.corp.com" {
		t.Errorf("custom mailbox host = %q %v", host, err)
	}

	withFakeSMTP(t, "smtp.163.com", true)
	err := sendMailbox("a@163.com", "wrong", "netease-mail", Draft{To: "b@c.com", Subject: "hi", Body: "x"})
	if err == nil || !strings.Contains(err.Error(), "授权码") {
		t.Fatalf("rejected auth err = %v", err)
	}

	// 下面两种不该连服务器：换一个一连就报错的拨号。
	smtpDial = func(string) (net.Conn, error) {
		t.Error("must not dial")
		return nil, net.ErrClosed
	}
	if err := sendMailbox("a@qq.com", "code", "qq-mail", Draft{Subject: "hi"}); err == nil || !strings.Contains(err.Error(), "收件人") {
		t.Fatalf("no recipient err = %v", err)
	}
	if err := sendMailbox("a@qq.com", " ", "qq-mail", Draft{To: "b@c.com"}); err == nil {
		t.Fatal("missing auth code must fail")
	}
}

// TestSMTPMessageKeepsASCIISubject 确认纯英文主题不编码，信头和正文之间只有一个空行。
func TestSMTPMessageKeepsASCIISubject(t *testing.T) {
	msg := smtpMessage("a@qq.com", Draft{To: "b@c.com", Subject: "Weekly report", Body: "hello"}, time.Date(2026, 11, 28, 9, 0, 0, 0, time.UTC))
	if !strings.Contains(msg, "Subject: Weekly report\r\n") || !strings.Contains(msg, "Date: Sat, 28 Nov 2026 09:00:00 +0000\r\n") {
		t.Fatalf("msg = %q", msg)
	}
	if !strings.HasSuffix(msg, "\r\n\r\nhello") || strings.Contains(msg, "\r\n\r\n\r\n") {
		t.Fatalf("header/body split = %q", msg)
	}
}
