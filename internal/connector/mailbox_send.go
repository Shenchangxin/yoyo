// mailbox_send.go 用 SMTP 给 QQ 邮箱、网易邮箱（163 / 126 / yeah.net）和自填的邮箱发信。
// 令牌和读信（mailbox.go）是同一个授权码：在邮箱设置里开启 IMAP/SMTP 服务后生成，不是登录密码。
// 连接走 465 端口的直连 TLS（QQ、网易都支持，也是它们推荐的方式），不走 25 / 587 的明文升级，免得授权码在升级前被看到。
// 信的正文和附件复用 mimeMail（和 Gmail 发信同一种写法），这里只补 SMTP 需要而 Gmail 会自己补的几项头：
//   - From：QQ、网易都要求和登录的邮箱一致，不一致直接退信；
//   - Date、Message-ID：没有的话部分收件方会判成垃圾邮件；
//   - 主题里的中文按 RFC 2047 编码，老的收件端才不会显示乱码；
//   - Bcc 只进信封（RCPT TO），从信头里去掉，不然暗送的人会被所有收件人看到。
// 不拥有：草稿和工单标记（connector.go、alias.go）、读信（mailbox.go）。

package connector

import (
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// smtpDial 换掉连 SMTP 的方式。默认是 465 端口直连 TLS；测试换成内存里的假服务器。
var smtpDial = func(host string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 12 * time.Second}
	return tls.DialWithDialer(dialer, "tcp", host+":465", &tls.Config{ServerName: host})
}

// smtpHost 按邮箱种类和地址找出 SMTP 服务器，规则和 IMAP 一样，只是前缀换成 smtp.：
// QQ 是 smtp.qq.com；网易按域名分 163 / 126 / yeah.net；自填的邮箱用 smtp.<域名>。
// user 是完整邮箱地址，同时用作登录名和发件人。地址不完整时返回错误。
func smtpHost(provider, endpoint string) (host, user string, err error) {
	imapHost, user, err := mailboxHost(provider, endpoint)
	if err != nil {
		return "", "", err
	}
	return "smtp." + strings.TrimPrefix(imapHost, "imap."), user, nil
}

// mailboxAuth 是 SMTP 的 AUTH PLAIN。
// 不用 net/smtp 自带的 PlainAuth：它要求连接是 *tls.Conn 或者服务器在本机，测试里的内存连接过不了这一关。
// 生产路径上 smtpDial 总是直连 TLS，所以去掉这层检查不会让授权码走明文。
type mailboxAuth struct {
	user, password string
}

// Start 一次把「\0用户名\0授权码」作为初始响应发过去，QQ 和网易都支持这种写法。
func (m mailboxAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "PLAIN", []byte("\x00" + m.user + "\x00" + m.password), nil
}

// Next 在服务器还要追问时报错：PLAIN 只有一来一回，追问说明服务器不按 PLAIN 走。
func (m mailboxAuth) Next(_ []byte, more bool) ([]byte, error) {
	if more {
		return nil, fmt.Errorf("connector: smtp server asked for more during AUTH PLAIN")
	}
	return nil, nil
}

// mailRecipients 把 To、Cc、Bcc 里的地址拆成信封收件人，只留邮箱地址本身，按出现顺序去重。
// 能按 RFC 5322 解析的用解析结果（带显示名的「张三 <a@b.com>」也认）；解析不了的按逗号、分号切开，留下带 @ 的那几段。
func mailRecipients(fields ...string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(addr string) {
		addr = strings.TrimSpace(addr)
		key := strings.ToLower(addr)
		if addr == "" || !strings.Contains(addr, "@") || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, addr)
	}
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		if list, err := mail.ParseAddressList(strings.ReplaceAll(field, ";", ",")); err == nil {
			for _, a := range list {
				add(a.Address)
			}
			continue
		}
		for _, part := range strings.FieldsFunc(field, func(r rune) bool { return r == ',' || r == ';' }) {
			if a, err := mail.ParseAddress(part); err == nil {
				add(a.Address)
			} else {
				add(part)
			}
		}
	}
	return out
}

// smtpMessage 编出要交给 SMTP 的整封信：在 mimeMail 的结果上补 From、Date、Message-ID，
// 去掉 Bcc 头，并把含非 ASCII 字符的主题按 RFC 2047（UTF-8 + base64）编码。正文和附件原样保留。
// from 是登录的邮箱地址；now 写进 Date，测试可以固定它。
func smtpMessage(from string, d Draft, now time.Time) string {
	raw := mimeMail(d)
	head, body := raw, ""
	if i := strings.Index(raw, "\r\n\r\n"); i >= 0 {
		head, body = raw[:i], raw[i+4:]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "From: %s\r\nDate: %s\r\nMessage-ID: %s\r\n", from, now.Format(time.RFC1123Z), messageID(from))
	for _, line := range strings.Split(head, "\r\n") {
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "bcc:"):
			// 暗送只进信封，不留在信头里。
			continue
		case strings.HasPrefix(lower, "subject:"):
			subject := strings.TrimSpace(line[len("subject:"):])
			fmt.Fprintf(&b, "Subject: %s\r\n", mime.BEncoding.Encode("utf-8", subject))
			continue
		}
		if line != "" {
			b.WriteString(line + "\r\n")
		}
	}
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

// messageID 生成 <随机串@发件域名>。随机数取不到时退回用纳秒时间，只要不和别的信重复就行。
func messageID(from string) string {
	domain := "yoyo.local"
	if i := strings.LastIndex(from, "@"); i >= 0 && i+1 < len(from) {
		domain = from[i+1:]
	}
	var r [12]byte
	if _, err := rand.Read(r[:]); err != nil {
		return fmt.Sprintf("<%d.yoyo@%s>", time.Now().UnixNano(), domain)
	}
	return "<" + hex.EncodeToString(r[:]) + ".yoyo@" + domain + ">"
}

// sendMailbox 用 SMTP 发出一份草稿。endpoint 是账号上填的邮箱地址，password 是授权码，provider 决定服务器。
// 顺序：连 465 → 有 AUTH 就登录 → MAIL FROM 自己 → 每个收件人 RCPT TO → DATA 写整封信 → QUIT。
// 登录被拒时说清楚要填授权码、要先开 SMTP 服务；某个收件人被拒时带上是哪个地址。
// 任何一步失败都不会算发出去：Broker.Send 只在这里返回 nil 之后才把草稿标成已发送。
func sendMailbox(endpoint, password, provider string, d Draft) error {
	host, user, err := smtpHost(provider, endpoint)
	if err != nil {
		return err
	}
	if strings.TrimSpace(password) == "" {
		return fmt.Errorf("connector: no token")
	}
	rcpts := mailRecipients(d.To, d.Cc, d.Bcc)
	if len(rcpts) == 0 {
		return fmt.Errorf("connector: 这封信没有收件人")
	}
	conn, err := smtpDial(host)
	if err != nil {
		return fmt.Errorf("connector: %s", err.Error())
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(60 * time.Second))
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("connector: smtp greeting: %s", err.Error())
	}
	defer c.Close()
	if ok, _ := c.Extension("AUTH"); ok {
		if err := c.Auth(mailboxAuth{user: user, password: password}); err != nil {
			return fmt.Errorf("connector: 邮箱发信登录失败（%s）。要填的是邮箱设置里开启 IMAP/SMTP 服务后生成的授权码，不是登录密码", strings.TrimSpace(err.Error()))
		}
	}
	if err := c.Mail(user); err != nil {
		return fmt.Errorf("connector: smtp MAIL FROM: %s", err.Error())
	}
	for _, r := range rcpts {
		if err := c.Rcpt(r); err != nil {
			return fmt.Errorf("connector: 收件人 %s 被拒：%s", r, err.Error())
		}
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("connector: smtp DATA: %s", err.Error())
	}
	if _, err := w.Write([]byte(smtpMessage(user, d, time.Now()))); err != nil {
		_ = w.Close()
		return fmt.Errorf("connector: smtp write: %s", err.Error())
	}
	// Close 才真正收到服务器对整封信的答复（250 表示已经收下），这里出错就是没发出去。
	if err := w.Close(); err != nil {
		return fmt.Errorf("connector: smtp send: %s", err.Error())
	}
	// QUIT 失败不影响：信已经被服务器收下了。
	_ = c.Quit()
	return nil
}
