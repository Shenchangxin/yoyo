package connector

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"time"
)

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
	default:
		if i := strings.LastIndex(endpoint, "@"); i > 0 && strings.Contains(endpoint[i:], ".") {
			domain := endpoint[i+1:]
			host = "imap." + domain
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

func readMailbox(endpoint, password, provider string) ([]map[string]string, error) {
	host, user, err := mailboxHost(provider, endpoint)
	if err != nil {
		return nil, err
	}
	dialer := &net.Dialer{Timeout: 12 * time.Second}
	conn, err := tls.DialWithDialer(dialer, "tcp", host+":993", &tls.Config{ServerName: host})
	if err != nil {
		return nil, fmt.Errorf("connector: %s", err.Error())
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(12 * time.Second))
	r := bufio.NewReader(conn)
	if _, err := readIMAP(r); err != nil {
		return nil, err
	}
	if err := imapCmd(conn, r, "a1", fmt.Sprintf("LOGIN %s %s", quoteIMAP(user), quoteIMAP(password))); err != nil {
		return nil, err
	}
	if err := imapCmd(conn, r, "a2", "SELECT INBOX"); err != nil {
		return nil, err
	}
	if err := writeIMAP(conn, "a3 FETCH 1:8 BODY.PEEK[HEADER.FIELDS (SUBJECT FROM)]"); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, err := readIMAP(r)
		if err != nil {
			return nil, err
		}
		lines = append(lines, line)
		if strings.HasPrefix(line, "a3 ") {
			break
		}
	}
	items := parseMailHeaders(provider, strings.Join(lines, "\n"))
	if len(items) == 0 {
		return nil, fmt.Errorf("connector: 收件箱里没有读到邮件")
	}
	return items, nil
}

func imapCmd(conn net.Conn, r *bufio.Reader, tag, command string) error {
	if err := writeIMAP(conn, tag+" "+command); err != nil {
		return err
	}
	for {
		line, err := readIMAP(r)
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, tag+" OK") {
			return nil
		}
		if strings.HasPrefix(line, tag+" NO") || strings.HasPrefix(line, tag+" BAD") {
			return fmt.Errorf("connector: %s", line)
		}
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
	return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"`
}

func parseMailHeaders(provider, raw string) []map[string]string {
	var items []map[string]string
	var subject, from string
	flush := func() {
		if subject == "" && from == "" {
			return
		}
		title := subject
		if title == "" {
			title = from
		}
		items = append(items, map[string]string{"provider": provider, "title": title, "text": from})
		subject, from = "", ""
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "subject:"):
			subject = strings.TrimSpace(line[len("subject:"):])
		case strings.HasPrefix(lower, "from:"):
			from = strings.TrimSpace(line[len("from:"):])
		case strings.HasPrefix(line, "a3 "):
			flush()
		}
	}
	flush()
	return items
}
