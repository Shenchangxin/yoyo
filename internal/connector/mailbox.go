// mailbox.go 用 IMAP 读 QQ 邮箱、网易邮箱和自填的邮箱。令牌就是邮箱的授权码，不是登录密码。
// 读收件箱最新的 20 封：主题、发件人、日期和编号，新的在前。编码过的标题（=?UTF-8?B?…?=、GBK、GB18030）解开再给人看。
// 网易要求登录后先报一次客户端身份（IMAP ID），不报会回 Unsafe Login。
// 不拥有：发信（同一个授权码走 SMTP，见 mailbox_send.go）和正文。

package connector

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/mail"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/encoding/htmlindex"
)

// mailboxLatest 是一次读几封。
const mailboxLatest = 20

func isMailbox(provider string) bool {
	switch provider {
	case "qq-mail", "netease-mail", "local":
		return true
	default:
		return false
	}
}

func mailboxHost(provider, endpoint string) (host, user string, err error) {
	endpoint = strings.TrimSpace(endpoint)
	switch provider {
	case "qq-mail":
		host = "imap.qq.com"
	case "netease-mail":
		host = "imap.163.com"
		if strings.HasSuffix(strings.ToLower(endpoint), "@126.com") {
			host = "imap.126.com"
		} else if strings.HasSuffix(strings.ToLower(endpoint), "@yeah.net") {
			host = "imap.yeah.net"
		}
	default:
		if i := strings.LastIndex(endpoint, "@"); i > 0 && strings.Contains(endpoint[i:], ".") {
			host = "imap." + endpoint[i+1:]
		}
	}
	if strings.Contains(endpoint, "@") {
		user = endpoint
	}
	if host == "" || user == "" {
		return "", "", fmt.Errorf("connector: 邮箱要填写完整地址")
	}
	return host, user, nil
}

// mailboxDial 换掉连 IMAP 的方式。测试换成内存里的假服务器。
var mailboxDial = func(host string) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: 12 * time.Second}
	return tls.DialWithDialer(dialer, "tcp", host+":993", &tls.Config{ServerName: host})
}

func readMailbox(endpoint, password, provider, query string) ([]map[string]string, error) {
	host, user, err := mailboxHost(provider, endpoint)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("connector: no token")
	}
	conn, err := mailboxDial(host)
	if err != nil {
		return nil, fmt.Errorf("connector: %s", err.Error())
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(20 * time.Second))
	r := bufio.NewReader(conn)
	if _, err := readIMAP(r); err != nil {
		return nil, err
	}
	if _, err := imapCmd(conn, r, "a1", fmt.Sprintf("LOGIN %s %s", quoteIMAP(user), quoteIMAP(password))); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "login") || strings.Contains(err.Error(), " NO") {
			return nil, fmt.Errorf("connector: 邮箱登录失败。要填的是邮箱设置里开启 IMAP 后生成的授权码，不是登录密码")
		}
		return nil, err
	}
	_, _ = imapCmd(conn, r, "a2", `ID ("name" "yoyo" "version" "1.0" "vendor" "yoyo")`)
	lines, err := imapCmd(conn, r, "a3", "SELECT INBOX")
	if err != nil {
		return nil, err
	}
	total := existsCount(lines)
	if total == 0 {
		return []map[string]string{{"provider": provider, "note": "收件箱是空的"}}, nil
	}
	from := total - mailboxLatest + 1
	if from < 1 {
		from = 1
	}
	// 收件地址这几项（To、Cc、Delivered-To、X-Original-To）用来认出发给加号子地址 you+yoyo-工单号@域名 的回信，见 alias.go。
	if err := writeIMAP(conn, fmt.Sprintf("a4 FETCH %d:%d (UID BODY.PEEK[HEADER.FIELDS (SUBJECT FROM TO CC DELIVERED-TO X-ORIGINAL-TO DATE MESSAGE-ID)])", from, total)); err != nil {
		return nil, err
	}
	msgs, err := readFetch(r, "a4")
	if err != nil {
		return nil, err
	}
	_ = writeIMAP(conn, "a5 LOGOUT")
	items := mailItems(provider, msgs)
	if q := strings.ToLower(strings.TrimSpace(query)); q != "" {
		var kept []map[string]string
		for _, it := range items {
			if strings.Contains(strings.ToLower(it["subject"]+" "+it["from"]), q) {
				kept = append(kept, it)
			}
		}
		items = kept
		if len(items) == 0 {
			return []map[string]string{{"provider": provider, "note": "最新 20 封里没有对上的", "query": query}}, nil
		}
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("connector: 收件箱里没有读到邮件")
	}
	return items, nil
}

// imapCmd 发一条命令，读到这条的结果为止。返回中间的未标记行。
func imapCmd(conn net.Conn, r *bufio.Reader, tag, command string) ([]string, error) {
	if err := writeIMAP(conn, tag+" "+command); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, err := readIMAP(r)
		if err != nil {
			return lines, err
		}
		if strings.HasPrefix(line, tag+" OK") {
			return lines, nil
		}
		if strings.HasPrefix(line, tag+" NO") || strings.HasPrefix(line, tag+" BAD") {
			return lines, fmt.Errorf("connector: %s", line)
		}
		lines = append(lines, line)
	}
}

func writeIMAP(conn net.Conn, line string) error {
	_, err := conn.Write([]byte(line + "\r\n"))
	return err
}

func readIMAP(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("connector: imap %s", err.Error())
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func quoteIMAP(s string) string {
	return `"` + strings.ReplaceAll(strings.ReplaceAll(s, `\`, `\\`), `"`, `\"`) + `"`
}

var existsRe = regexp.MustCompile(`^\* (\d+) EXISTS`)

func existsCount(lines []string) int {
	for _, line := range lines {
		if m := existsRe.FindStringSubmatch(strings.ToUpper(line)); m != nil {
			n, _ := strconv.Atoi(m[1])
			return n
		}
	}
	return 0
}

// fetched 是一封信的序号、UID 和头部原文。
type fetched struct {
	Seq    int
	UID    string
	Header []byte
}

var (
	fetchRe   = regexp.MustCompile(`^\* (\d+) FETCH`)
	uidRe     = regexp.MustCompile(`UID (\d+)`)
	literalRe = regexp.MustCompile(`\{(\d+)\}$`)
)

// readFetch 读 FETCH 的回答，直到 tag 那一行。头部是 {n} 字面量，按字节数读，不按行猜。
func readFetch(r *bufio.Reader, tag string) ([]fetched, error) {
	var out []fetched
	var cur *fetched
	for {
		line, err := readIMAP(r)
		if err != nil {
			return out, err
		}
		if strings.HasPrefix(line, tag+" ") {
			if !strings.HasPrefix(line, tag+" OK") {
				return out, fmt.Errorf("connector: %s", line)
			}
			return out, nil
		}
		if m := fetchRe.FindStringSubmatch(line); m != nil {
			seq, _ := strconv.Atoi(m[1])
			out = append(out, fetched{Seq: seq})
			cur = &out[len(out)-1]
		}
		if cur != nil {
			if m := uidRe.FindStringSubmatch(line); m != nil {
				cur.UID = m[1]
			}
		}
		if m := literalRe.FindStringSubmatch(line); m != nil && cur != nil {
			n, _ := strconv.Atoi(m[1])
			if n > 256<<10 {
				return out, fmt.Errorf("connector: imap header too large")
			}
			buf := make([]byte, n)
			if _, err := io.ReadFull(r, buf); err != nil {
				return out, fmt.Errorf("connector: imap %s", err.Error())
			}
			cur.Header = buf
		}
	}
}

var mimeWords = &mime.WordDecoder{CharsetReader: func(charset string, input io.Reader) (io.Reader, error) {
	enc, err := htmlindex.Get(strings.ToLower(strings.TrimSpace(charset)))
	if err != nil {
		return nil, err
	}
	return enc.NewDecoder().Reader(input), nil
}}

// decodeWords 解开 =?charset?B?…?= 这种编码。解不开时原样返回。
func decodeWords(s string) string {
	out, err := mimeWords.DecodeHeader(s)
	if err != nil {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(out)
}

// mailItems 把头部收成条目，新的在前。
func mailItems(provider string, msgs []fetched) []map[string]string {
	sort.Slice(msgs, func(i, j int) bool { return msgs[i].Seq > msgs[j].Seq })
	var items []map[string]string
	for _, m := range msgs {
		if len(m.Header) == 0 {
			continue
		}
		msg, err := mail.ReadMessage(bytes.NewReader(append(append([]byte(nil), m.Header...), '\r', '\n')))
		if err != nil {
			continue
		}
		subject := decodeWords(msg.Header.Get("Subject"))
		from := decodeWords(msg.Header.Get("From"))
		if addr, err := (&mail.AddressParser{WordDecoder: mimeWords}).Parse(msg.Header.Get("From")); err == nil {
			if addr.Name != "" {
				from = addr.Name + " <" + addr.Address + ">"
			} else {
				from = addr.Address
			}
		}
		if subject == "" && from == "" {
			continue
		}
		it := map[string]string{"provider": provider, "kind": "mail", "subject": subject, "from": from, "title": firstNonEmpty(subject, from), "text": from}
		// 收件地址：To、Cc、Delivered-To、X-Original-To 合成一个 to 字段，用逗号隔开。
		// mail_alias.go 用它认出发给加号子地址 you+yoyo-工单号@域名 的回信。对方回信时子地址可能在 To 或 Cc 里；
		// 投递时服务器改写了收件头的，子地址只留在 Delivered-To / X-Original-To 里，所以这几项都收。
		var to []string
		for _, key := range []string{"To", "Cc", "Delivered-To", "X-Original-To"} {
			if v := strings.TrimSpace(decodeWords(msg.Header.Get(key))); v != "" {
				to = append(to, v)
			}
		}
		if len(to) > 0 {
			it["to"] = strings.Join(to, ", ")
		}
		if m.UID != "" {
			it["id"] = m.UID
		}
		if id := strings.TrimSpace(msg.Header.Get("Message-Id")); id != "" {
			it["message_id"] = id
		}
		if at, err := mail.ParseDate(msg.Header.Get("Date")); err == nil {
			it["date"] = at.UTC().Format(time.RFC3339)
		}
		items = append(items, it)
	}
	return items
}
