package schedule

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Kind string

const (
	KindCron      Kind = "cron"
	KindOnce      Kind = "once"
	KindHeartbeat Kind = "heartbeat"
	KindWebhook   Kind = "webhook"
)

type Job struct {
	ID        string    `json:"id"`
	Kind      Kind      `json:"kind"`
	Spec      string    `json:"spec"`
	Prompt    string    `json:"prompt"`
	Workspace string    `json:"workspace,omitempty"`
	Harness   string    `json:"harness,omitempty"`
	Enabled   bool      `json:"enabled"`
	Isolate   bool      `json:"isolate"`
	LastRun   time.Time `json:"last_run,omitempty"`
	LastErr   string    `json:"last_err,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type Service struct {
	mu   sync.Mutex
	path string
	jobs []Job
}

func Open(dir string) (*Service, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	s := &Service{path: filepath.Join(dir, "jobs.json")}
	_ = s.load()
	return s, nil
}

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
	j.Isolate = true
	s.jobs = append(s.jobs, j)
	_ = s.flush()
	return j
}

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

func (s *Service) List() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Job(nil), s.jobs...)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out
}

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

func (s *Service) Trigger(id string) (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.jobs {
		if s.jobs[i].ID != id && s.jobs[i].Spec != id {
			continue
		}
		if s.jobs[i].Kind != KindWebhook && s.jobs[i].Kind != KindOnce && s.jobs[i].Kind != KindHeartbeat {
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
		if s.jobs[i].Kind == KindOnce {
			s.jobs[i].Enabled = false
		}
	}
	_ = s.flush()
}

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
	case KindCron:
		return cronDue(j.Spec, j.LastRun, now)
	case KindWebhook:
		return false
	default:
		return false
	}
}

func cronDue(spec string, last, now time.Time) bool {
	spec = strings.TrimSpace(strings.ToLower(spec))
	if spec == "hourly" {
		return last.IsZero() || now.Sub(last) >= time.Hour
	}
	if spec == "daily" || spec == "0 18 * * 5" || spec == "weekday-1800" {
		if last.IsZero() {
			return now.Hour() == 18 && now.Minute() < 5
		}
		return now.Sub(last) >= 20*time.Hour && now.Hour() == 18
	}
	if spec == "weekly" {
		return last.IsZero() || now.Sub(last) >= 7*24*time.Hour
	}
	d, err := time.ParseDuration(spec)
	if err == nil {
		return last.IsZero() || now.Sub(last) >= d
	}
	return false
}

func (s *Service) load() error {
	b, err := os.ReadFile(s.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, &s.jobs)
}

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
