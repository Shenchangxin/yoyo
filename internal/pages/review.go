// 待审阅草稿写在 root/reviews.json。批准才调用 applyApprovedLocked，拒绝只改状态。

package pages

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReviewStatus 是审阅状态。
type ReviewStatus string

const (
	// ReviewAwaiting 是等待审阅，值为 awaiting_review。
	ReviewAwaiting ReviewStatus = "awaiting_review"
	// ReviewSaved 是已写入页面，值为 saved。
	ReviewSaved ReviewStatus = "saved"
	// ReviewDeclined 是已拒绝，值为 declined。
	ReviewDeclined ReviewStatus = "declined"
	// ReviewExpired 是已过期，值为 expired。
	ReviewExpired ReviewStatus = "expired"
)

// Review 是一份待批准的页面草稿。Hash 覆盖标题、正文、页面 id、空间和期望修订号。
type Review struct {
	ID               string       `json:"id"`
	ThreadID         string       `json:"thread_id,omitempty"`
	ToolCallID       string       `json:"tool_call_id,omitempty"`
	SessionID        string       `json:"session_id,omitempty"`
	SpaceID          string       `json:"space_id"`
	PageID           string       `json:"page_id,omitempty"`
	ParentID         string       `json:"parent_id,omitempty"`
	Title            string       `json:"title"`
	Content          string       `json:"content"`
	ExpectedRevision int          `json:"expected_revision,omitempty"`
	Hash             string       `json:"hash"`
	Status           ReviewStatus `json:"status"`
	CreatedAt        time.Time    `json:"created_at"`
	ExpiresAt        time.Time    `json:"expires_at"`
	SavedPageID      string       `json:"saved_page_id,omitempty"`
}

// Propose 规范化字段并计算哈希，状态设为 awaiting。ExpiresAt 为空时是现在加 30 分钟。同一工具调用或同一哈希的待审阅会被替换，但保留原来的 ID 和创建时间。
func (s *Store) Propose(r Review) (Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r.SpaceID = normalizeSpace(r.SpaceID)
	r.PageID = normalizePageID(r.PageID)
	r.ParentID = normalizePageID(r.ParentID)
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
		r.ID = reviewID(r.Hash)
	}
	all := s.loadReviews()
	for i, existing := range all {
		if existing.Status != ReviewAwaiting {
			continue
		}
		sameCall := r.ToolCallID != "" && existing.ToolCallID == r.ToolCallID && existing.ThreadID == r.ThreadID
		if !sameCall && existing.Hash != r.Hash {
			continue
		}
		if existing.SavedPageID != "" {
			return existing, nil
		}
		id, created := existing.ID, existing.CreatedAt
		all[i] = r
		all[i].ID = id
		all[i].CreatedAt = created
		if err := s.flushReviews(all); err != nil {
			return Review{}, err
		}
		return all[i], nil
	}
	all = append(all, r)
	if err := s.flushReviews(all); err != nil {
		return Review{}, err
	}
	return r, nil
}

// Pending 返回仍在等待的审阅。已过期的从 awaiting 改成 expired 并写回。
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

// GetReview 按 ID 或哈希查找审阅。
// id 与每条的 ID 或 Hash 比较。
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

// Decide 在哈希非空且不符时返回错误。已保存则再读出页面。approve 为假则标成 declined，不写页面。为真时以修订号 0 调用 applyApprovedLocked，再标成 saved。
// hash 非空且与找到的审阅 Hash 不同时返回 review hash mismatch。
func (s *Store) Decide(id, hash string, approve bool) (Page, Review, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rev, idx, all, err := s.findReview(id)
	if err != nil {
		return Page{}, Review{}, err
	}
	if hash != "" && hash != rev.Hash {
		return Page{}, Review{}, fmt.Errorf("review hash mismatch")
	}
	if rev.Status == ReviewSaved && rev.SavedPageID != "" {
		pg, err := s.getLocked(rev.SpaceID, rev.SavedPageID)
		return pg, rev, err
	}
	if rev.Status != ReviewAwaiting {
		return Page{}, rev, fmt.Errorf("review is %s", rev.Status)
	}
	if !approve {
		if idx >= 0 && idx < len(all) {
			all[idx].Status = ReviewDeclined
			_ = s.flushReviews(all)
			return Page{}, all[idx], nil
		}
		return Page{}, rev, nil
	}
	// 操作者批准才写入。这里把 expected revision 传 0，并发改过页面也不会把审阅卡死。
	pg, err := s.applyApprovedLocked(rev.SpaceID, rev.PageID, rev.ParentID, rev.Title, rev.Content, rev.SessionID, 0)
	if err != nil {
		return Page{}, rev, err
	}
	if idx >= 0 && idx < len(all) && all[idx].ID == rev.ID {
		all[idx].Status = ReviewSaved
		all[idx].SavedPageID = pg.ID
		rev = all[idx]
	} else {
		for i := range all {
			if all[i].ID == rev.ID {
				all[i].Status = ReviewSaved
				all[i].SavedPageID = pg.ID
				rev = all[i]
				break
			}
		}
	}
	_ = s.flushReviews(all)
	return pg, rev, nil
}

// reviewID 用哈希前 12 位生成 rev- 前缀。哈希为空时用当前 UTC 时间。
// hash 长度大于等于 12 时取前 12 位接在 rev- 后面。为空时改用当前 UTC 时间。否则整段接在 rev- 后面。
func reviewID(hash string) string {
	if len(hash) >= 12 {
		return "rev-" + hash[:12]
	}
	if hash == "" {
		return "rev-" + time.Now().UTC().Format("20060102T150405")
	}
	return "rev-" + hash
}

// findReview 按 ID 或哈希查找，返回条目、下标和整份列表。找不到返回 review not found。
// id 去掉首尾空白后与每条的 ID 或 Hash 比较。
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

// loadReviews 读取 reviews.json。文件不存在或 JSON 无效时返回 nil。
func (s *Store) loadReviews() []Review {
	b, err := os.ReadFile(filepath.Join(s.root, "reviews.json"))
	if err != nil {
		return nil
	}
	var out []Review
	_ = json.Unmarshal(b, &out)
	return out
}

// flushReviews 把审阅列表原子写到 reviews.json，权限 0o644。
// all 交给 json.MarshalIndent，前缀是空串，缩进是两个空格，再传给 atomicWrite，权限是 0o644。
func (s *Store) flushReviews(all []Review) error {
	b, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(s.root, "reviews.json"), b, 0o644)
}
