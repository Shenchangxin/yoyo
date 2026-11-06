// Package schedule 把任务写在 Open 收到的目录里的 jobs.json。桌面应用把该目录设成主目录的 schedule/。
// Create 把 Enabled 设为真。KindFollow 和 IsStudioJob 为真时 Isolate 为假，其余任务 Isolate 为真。
package schedule

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Kind 是任务如何到期。
type Kind string

const (
	// KindCron 按本地时刻或一段时长重复，值为 cron。写法见 cronDue。
	KindCron Kind = "cron"
	// KindOnce 在 Spec 的 RFC3339 时间到达后跑一次，值为 once。
	KindOnce Kind = "once"
	// KindHeartbeat 按间隔重复，值为 heartbeat。Spec 解析失败时按 30 分钟。
	KindHeartbeat Kind = "heartbeat"
	// KindWebhook 不因时间到期，只能 Trigger，值为 webhook。
	KindWebhook Kind = "webhook"
	// KindFollow 在会话里延迟跟进，值为 follow。Create 强制 Isolate 为假。
	KindFollow Kind = "follow"
)

// Job 是一条已保存的定时任务，含提示词、工作区和是否隔离。
type Job struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Spec      string    `json:"spec"`
	Prompt    string    `json:"prompt"`
	Workspace string    `json:"workspace,omitempty"`
	Harness   string    `json:"harness,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	Channel   string    `json:"channel,omitempty"`
	TopicID   string    `json:"topic_id,omitempty"`
	StudioID  string    `json:"studio_id,omitempty"`
	Aspect    string    `json:"aspect,omitempty"`
	Enabled   bool      `json:"enabled"`
	Isolate   bool      `json:"isolate"`
	Status    string    `json:"status,omitempty"`
	LastRun   time.Time `json:"last_run,omitempty"`
	LastErr   string    `json:"last_err,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// Service 把任务切片读写到 path 指向的 jobs.json。
type Service struct {
	mu   sync.Mutex
	path string
	jobs []Job
}

// Open 按 0o755 创建目录，路径为 dir/jobs.json。读文件失败时仍返回空服务。
func Open(dir string) (*Service, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Service{path: filepath.Join(dir, "jobs.json")}
	_ = s.load()
	return s, nil
}

// Create 生成空着的 ID，并把 Enabled 设为真。Kind 为空时用 KindOnce。KindFollow 和短片任务 Isolate 为假，其他为真。follow 的间隔短于 1 分钟时改成 1m。
// j 的 ID 为空时写成当前 UTC 的 job-20060102T150405.000000000。CreatedAt 为零时写成当前 UTC。Enabled 设为 true。Kind 为空时用 KindOnce。Kind 为 KindFollow 时 Isolate 为 false，Spec 能解析且时长短于 time.Minute 则改成 1m。否则 IsStudioJob 为真时 Isolate 为 false，其余为 true。
func (s *Service) Create(j Job) Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j.ID == "" {
		j.ID = time.Now().UTC().Format("job-20060102T150405.000000000")
	}
	if j.CreatedAt.IsZero() {
		j.CreatedAt = time.Now().UTC()
	}
	j.Enabled = true
	if j.Kind == "" {
		j.Kind = KindOnce
	}
	if j.Kind == KindFollow {
		j.Isolate = false
		if d, err := time.ParseDuration(j.Spec); err == nil && d < time.Minute {
			j.Spec = "1m"
		}
	} else if IsStudioJob(j) {
		// 短片任务留在视频通道。拷进临时工作区会藏起 studio_* 工具，并触发 shell 门。
		j.Isolate = false
	} else {
		j.Isolate = true
	}
	s.jobs = append(s.jobs, j)
	_ = s.flush()
	return j
}

// Cancel 按 ID 从列表删除任务。找到则返回真。
// id 与每条任务的 ID 比较，相等的从列表删掉。
func (s *Service) Cancel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.jobs[:0]
	ok := false
	for _, j := range s.jobs {
		if j.ID == id {
			ok = true
			continue
		}
		out = append(out, j)
	}
	s.jobs = out
	_ = s.flush()
	return ok
}

// List 按 CreatedAt 从新到旧返回副本。
func (s *Service) List() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Job(nil), s.jobs...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

// Due 返回当前已启用且到期的任务，并把它们的 LastRun 写成 now。
func (s *Service) Due(now time.Time) []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Job
	for i := range s.jobs {
		j := s.jobs[i]
		if !j.Enabled {
			continue
		}
		if due(j, now) {
			s.jobs[i].LastRun = now
			out = append(out, s.jobs[i])
		}
	}
	_ = s.flush()
	return out
}

// Trigger 按 ID 或 Spec 找到 webhook、once、heartbeat 或 follow。未启用或不属于这些种类则返回假。命中时更新 LastRun。
// id 与任务的 ID 或 Spec 比较。
func (s *Service) Trigger(id string) (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.jobs {
		if s.jobs[i].ID != id && s.jobs[i].Spec != id {
			continue
		}
		if s.jobs[i].Kind != KindWebhook && s.jobs[i].Kind != KindOnce && s.jobs[i].Kind != KindHeartbeat && s.jobs[i].Kind != KindFollow {
			continue
		}
		if !s.jobs[i].Enabled {
			return Job{}, false
		}
		s.jobs[i].LastRun = time.Now().UTC()
		j := s.jobs[i]
		_ = s.flush()
		return j, true
	}
	return Job{}, false
}

// Interrupt 把该任务的 Status 写成 interrupted，并记下原因和 LastRun。
// id 与任务的 ID 比较，命中时 Status 写成 interrupted。
// reason 写入命中任务的 LastErr。
func (s *Service) Interrupt(id, reason string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.jobs {
		if s.jobs[i].ID != id {
			continue
		}
		s.jobs[i].Status = "interrupted"
		s.jobs[i].LastErr = reason
		s.jobs[i].LastRun = time.Now().UTC()
	}
	_ = s.flush()
}

// Retry 清掉 Status 和 LastErr，重新启用，并把 LastRun 置零。找不到返回假。
// id 与任务的 ID 比较，命中时清掉 Status 和 LastErr，Enabled 设为 true，LastRun 置零。
func (s *Service) Retry(id string) (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.jobs {
		if s.jobs[i].ID != id {
			continue
		}
		s.jobs[i].Status = ""
		s.jobs[i].LastErr = ""
		s.jobs[i].Enabled = true
		s.jobs[i].LastRun = time.Time{}
		j := s.jobs[i]
		_ = s.flush()
		return j, true
	}
	return Job{}, false
}

// Record 更新 LastRun。err 非 nil 时写下 LastErr，否则清空。once 和 follow 随后把 Enabled 设为假。
// id 与任务的 ID 比较，命中时更新 LastRun，并按 err 写 LastErr。
func (s *Service) Record(id string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.jobs {
		if s.jobs[i].ID != id {
			continue
		}
		s.jobs[i].LastRun = time.Now().UTC()
		if err != nil {
			s.jobs[i].LastErr = err.Error()
		} else {
			s.jobs[i].LastErr = ""
		}
		if s.jobs[i].Kind == KindOnce || s.jobs[i].Kind == KindFollow {
			s.jobs[i].Enabled = false
		}
	}
	_ = s.flush()
}

// due 判断一条任务在 now 是否到期。webhook 和未知种类返回假。follow 在 interrupted 时返回假，间隔短于 1 分钟时按 1 分钟。
// j 的 Kind 决定分支。once 解析 Spec 的 RFC3339。heartbeat 解析 Spec 的时长，失败则用 30 分钟，并看 LastRun。follow 在 Status 为 interrupted 时返回 false，否则解析 Spec。cron 把 Spec 和 LastRun 传给 cronDue。webhook 与其他种类返回 false。
func due(j Job, now time.Time) bool {
	switch j.Kind {
	case KindOnce:
		t, err := time.Parse(time.RFC3339, j.Spec)
		if err != nil {
			return false
		}
		return !t.After(now) && j.LastRun.IsZero()
	case KindHeartbeat:
		d, err := time.ParseDuration(j.Spec)
		if err != nil {
			d = 30 * time.Minute
		}
		if j.LastRun.IsZero() {
			return true
		}
		return now.Sub(j.LastRun) >= d
	case KindFollow:
		if j.Status == "interrupted" {
			return false
		}
		d, err := time.ParseDuration(j.Spec)
		if err != nil {
			switch strings.ToLower(strings.TrimSpace(j.Spec)) {
			case "1h", "hourly":
				d = time.Hour
			case "1d", "daily":
				d = 24 * time.Hour
			case "1w", "weekly":
				d = 7 * 24 * time.Hour
			default:
				d = time.Hour
			}
		}
		if d < time.Minute {
			d = time.Minute
		}
		anchor := j.LastRun
		if anchor.IsZero() {
			anchor = j.CreatedAt
		}
		if anchor.IsZero() {
			return false
		}
		return now.Sub(anchor) >= d
	case KindCron:
		return cronDue(j.Spec, j.LastRun, j.CreatedAt, now)
	case KindWebhook:
		return false
	default:
		return false
	}
}

// IsStudioJob 在 Channel 为 video、带了 StudioID 或 TopicID，或提示词以「做成短片」开头时返回真。
// j 的 Channel 去掉空白后不区分大小写等于 video 时返回 true。StudioID 或 TopicID 去掉空白后非空也返回 true。Prompt 去掉空白后以前缀「做成短片」开头也返回 true。
func IsStudioJob(j Job) bool {
	if strings.EqualFold(strings.TrimSpace(j.Channel), "video") {
		return true
	}
	if strings.TrimSpace(j.StudioID) != "" || strings.TrimSpace(j.TopicID) != "" {
		return true
	}
	p := strings.TrimSpace(j.Prompt)
	return strings.HasPrefix(p, "做成短片")
}

// cronGrace 是错过一个排好的时刻之后还补跑的时长。电脑睡过了头，醒来六小时内补一次，再晚就等下一次。
const cronGrace = 6 * time.Hour

// cronDue 判断一条 cron 任务在 now 是否到期。
// 按间隔的写法：hourly、weekly，或 time.ParseDuration 认得的一段时长，离上一次运行够久就到期。
// 按本地时刻的写法：daily@HH:MM、weekdays@HH:MM、weekly@D@HH:MM（D 取 0–6，0 是周日），五段 cron「分 时 日 月 周」，
// 以及旧写法 daily（每天 18:00）和 weekday-1800（工作日 18:00）。最近一个排到的时刻晚于上一次运行和创建时刻、
// 并且离 now 不超过 cronGrace 时到期。时刻按本机时区算，now 是 UTC 也一样。
func cronDue(spec string, last, created, now time.Time) bool {
	spec = strings.TrimSpace(strings.ToLower(spec))
	switch spec {
	case "hourly":
		return last.IsZero() || now.Sub(last) >= time.Hour
	case "weekly":
		return last.IsZero() || now.Sub(last) >= 7*24*time.Hour
	}
	if m, ok := parseClock(spec); ok {
		slot := m.lastSlot(now.In(time.Local))
		if slot.IsZero() {
			return false
		}
		anchor := last
		if created.After(anchor) {
			anchor = created
		}
		return slot.After(anchor) && now.Sub(slot) <= cronGrace
	}
	d, err := time.ParseDuration(spec)
	if err == nil {
		return last.IsZero() || now.Sub(last) >= d
	}
	return false
}

// clock 是一组按本地时刻排的分钟。按 Vixie cron 的习惯：日和周有一段以 * 开头时两段都要对上，否则任一对上就算。
type clock struct {
	minute  [60]bool
	hour    [24]bool
	dom     [32]bool
	month   [13]bool
	dow     [7]bool
	starDom bool
	starDow bool
}

// parseClock 认按本地时刻排的写法。认不出时 ok 为假。
func parseClock(spec string) (clock, bool) {
	switch spec {
	case "daily":
		spec = "0 18 * * *"
	case "weekday-1800":
		spec = "0 18 * * 1-5"
	}
	if strings.Contains(spec, "@") {
		return parseAt(spec)
	}
	fields := strings.Fields(spec)
	if len(fields) != 5 {
		return clock{}, false
	}
	var c clock
	ok := fillField(c.minute[:], fields[0], 0, 59) &&
		fillField(c.hour[:], fields[1], 0, 23) &&
		fillField(c.dom[:], fields[2], 1, 31) &&
		fillField(c.month[:], fields[3], 1, 12)
	if !ok {
		return clock{}, false
	}
	var dow [8]bool
	if !fillField(dow[:], fields[4], 0, 7) {
		return clock{}, false
	}
	for d := 0; d < 7; d++ {
		c.dow[d] = dow[d]
	}
	if dow[7] {
		c.dow[0] = true
	}
	c.starDom = strings.HasPrefix(fields[2], "*")
	c.starDow = strings.HasPrefix(fields[4], "*")
	return c, true
}

// parseAt 认 daily@HH:MM、weekdays@HH:MM 和 weekly@D@HH:MM。
func parseAt(spec string) (clock, bool) {
	parts := strings.Split(spec, "@")
	hm := parts[len(parts)-1]
	h, m, ok := parseHM(hm)
	if !ok {
		return clock{}, false
	}
	days := "*"
	switch {
	case len(parts) == 2 && parts[0] == "daily":
	case len(parts) == 2 && parts[0] == "weekdays":
		days = "1-5"
	case len(parts) == 3 && parts[0] == "weekly":
		days = parts[1]
	default:
		return clock{}, false
	}
	return parseClock(strconv.Itoa(m) + " " + strconv.Itoa(h) + " * * " + days)
}

func parseHM(text string) (int, int, bool) {
	hs, ms, ok := strings.Cut(strings.TrimSpace(text), ":")
	if !ok {
		return 0, 0, false
	}
	h, err1 := strconv.Atoi(hs)
	m, err2 := strconv.Atoi(ms)
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}

// fillField 按一段 cron 字段标出 out 里的取值。认 *、单个数、a-b、a,b 和 /n 步长。越界或写错时返回假。
func fillField(out []bool, field string, lo, hi int) bool {
	for _, part := range strings.Split(field, ",") {
		step := 1
		if base, s, ok := strings.Cut(part, "/"); ok {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 {
				return false
			}
			part, step = base, n
		}
		from, to := lo, hi
		switch {
		case part == "*":
		case strings.Contains(part, "-"):
			a, b, _ := strings.Cut(part, "-")
			x, err1 := strconv.Atoi(a)
			y, err2 := strconv.Atoi(b)
			if err1 != nil || err2 != nil {
				return false
			}
			from, to = x, y
		default:
			x, err := strconv.Atoi(part)
			if err != nil {
				return false
			}
			from, to = x, x
			if step > 1 {
				to = hi
			}
		}
		if from < lo || to > hi || from > to {
			return false
		}
		for v := from; v <= to; v += step {
			out[v] = true
		}
	}
	return true
}

// lastSlot 返回不晚于 now 的最近一个排到的时刻。往回看八天，都没有时返回零。
func (c clock) lastSlot(now time.Time) time.Time {
	for back := 0; back <= 8; back++ {
		day := time.Date(now.Year(), now.Month(), now.Day()-back, 0, 0, 0, 0, now.Location())
		if !c.onDay(day) {
			continue
		}
		for h := 23; h >= 0; h-- {
			if !c.hour[h] {
				continue
			}
			for m := 59; m >= 0; m-- {
				if !c.minute[m] {
					continue
				}
				at := time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, now.Location())
				if !at.After(now) {
					return at
				}
			}
		}
	}
	return time.Time{}
}

func (c clock) onDay(day time.Time) bool {
	if !c.month[int(day.Month())] {
		return false
	}
	dom := c.dom[day.Day()]
	dow := c.dow[int(day.Weekday())]
	if c.starDom || c.starDow {
		return dom && dow
	}
	return dom || dow
}

// load 读取 jobs.json 并解进任务切片。
func (s *Service) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.jobs)
}

// flush 把任务写成缩进 JSON，先写临时文件，权限 0o644，再改名替换。
func (s *Service) flush() error {
	b, err := json.MarshalIndent(s.jobs, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
