// 待审阅草稿的状态和形状。审阅的存取由 Catalog 的实现负责。

package pages

import (
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
