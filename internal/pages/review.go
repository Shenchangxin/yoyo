package pages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ReviewStatus string

const (
	ReviewAwaiting ReviewStatus = "awaiting_review"
	ReviewSaved    ReviewStatus = "saved"
	ReviewDeclined ReviewStatus = "declined"
	ReviewExpired  ReviewStatus = "expired"
)

type Review struct {
	ID                string       `json:"id"`
	ThreadID          string       `json:"thread_id,omitempty"`
	ToolCallID        string       `json:"tool_call_id,omitempty"`
	SessionID         string       `json:"session_id,omitempty"`
	SpaceID           string       `json:"space_id"`
	PageID            string       `json:"page_id,omitempty"`
	ParentID          string       `json:"parent_id,omitempty"`
	Title             string       `json:"title"`
	Content           string       `json:"content"`
	ExpectedRevision  int          `json:"expected_revision,omitempty"`
	Hash              string       `json:"hash"`
	Status            ReviewStatus `json:"status"`
	CreatedAt         time.Time    `json:"created_at"`
	ExpiresAt         time.Time    `json:"expires_at"`
	SavedPageID       string       `json:"saved_page_id,omitempty"`
}

func (s *Store) Propose(r Review) (Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.SpaceID = normalizeSpace(r.SpaceID)
	r.Title = clipTitle(r.Title)
	r.Content = clipBody(r.Content)
	r.Hash = hashDraft(r.Title, r.Content, r.PageID, r.SpaceID, r.ExpectedRevision)
	now := time.Now().UTC()
	if r.CreatedAt.IsZero() {
		r.CreatedAt = now
	}
	if r.ExpiresAt.IsZero() {
		r.ExpiresAt = now.Add(30 * time.Minute)
	}
	r.Status = ReviewAwaiting
	if r.ID == "" {
		r.ID = "rev-" + r.Hash[:12]
	}
	all := s.loadReviews()
	for i, existing := range all {
		if existing.Status != ReviewAwaiting {
			continue
		}
		if (r.ToolCallID != "" && existing.ToolCallID == r.ToolCallID && existing.ThreadID == r.ThreadID) || existing.Hash == r.Hash {
			if existing.SavedPageID != "" {
				return existing, nil
			}
			all[i] = r
			if err := s.flushReviews(all); err != nil {
				return Review{}, err
			}
			return r, nil
		}
	}
	all = append(all, r)
	if err := s.flushReviews(all); err != nil {
		return Review{}, err
	}
	return r, nil
}

func (s *Store) Pending() []Review {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Review
	now := time.Now().UTC()
	all := s.loadReviews()
	changed := false
	for i, r := range all {
		if r.Status == ReviewAwaiting && now.After(r.ExpiresAt) {
			all[i].Status = ReviewExpired
			changed = true
			continue
		}
		if r.Status == ReviewAwaiting {
			out = append(out, all[i])
		}
	}
	if changed {
		_ = s.flushReviews(all)
	}
	return out
}

func (s *Store) GetReview(id string) (Review, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.loadReviews() {
		if r.ID == id || r.Hash == id {
			return r, true
		}
	}
	return Review{}, false
}

func (s *Store) Decide(id, hash string, approve bool) (Page, Review, error) {
	s.mu.Lock()
	rev, idx, all, err := s.findReview(id)
	s.mu.Unlock()
	if err != nil {
		return Page{}, Review{}, err
	}
	if hash != "" && hash != rev.Hash {
		return Page{}, Review{}, fmt.Errorf("review hash mismatch")
	}
	if rev.Status == ReviewSaved && rev.SavedPageID != "" {
		pg, err := s.Get(rev.SpaceID, rev.SavedPageID)
		return pg, rev, err
	}
	if rev.Status != ReviewAwaiting {
		return Page{}, rev, fmt.Errorf("review is %s", rev.Status)
	}
	if !approve {
		s.mu.Lock()
		defer s.mu.Unlock()
		all = s.loadReviews()
		if idx >= 0 && idx < len(all) {
			all[idx].Status = ReviewDeclined
			_ = s.flushReviews(all)
			return Page{}, all[idx], nil
		}
		return Page{}, rev, nil
	}
	pg, err := s.ApplyApproved(rev.SpaceID, rev.PageID, rev.ParentID, rev.Title, rev.Content, rev.SessionID, rev.ExpectedRevision)
	if err != nil {
		return Page{}, rev, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all = s.loadReviews()
	for i := range all {
		if all[i].ID == rev.ID {
			all[i].Status = ReviewSaved
			all[i].SavedPageID = pg.ID
			rev = all[i]
			break
		}
	}
	_ = s.flushReviews(all)
	return pg, rev, nil
}

func (s *Store) findReview(id string) (Review, int, []Review, error) {
	all := s.loadReviews()
	id = strings.TrimSpace(id)
	for i, r := range all {
		if r.ID == id || r.Hash == id {
			return r, i, all, nil
		}
	}
	return Review{}, -1, all, fmt.Errorf("review not found")
}

func (s *Store) loadReviews() []Review {
	b, err := os.ReadFile(filepath.Join(s.root, "reviews.json"))
	if err != nil {
		return nil
	}
	var out []Review
	_ = json.Unmarshal(b, &out)
	return out
}

func (s *Store) flushReviews(all []Review) error {
	b, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(s.root, "reviews.json"), b, 0o644)
}
