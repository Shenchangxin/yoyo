// inbox.go 是驱动器私有的两列收件箱：下一回合，以及下一步。
// 它不唤醒代理。Agent 在拼接之后决定要不要开回合。

package dsh

// InboxTarget 是收件箱列。NextTurn 是下一回合的一条普通消息。NextStep 在最近的步骤边界取走。
type InboxTarget string

const (
	// NextTurn 每次 claim 只取走一条，作为该回合的普通消息。
	NextTurn InboxTarget = "next-turn"
	// NextStep 在步骤边界整列取走。steer 和 inject 都进这一列。
	NextStep InboxTarget = "next-step"
)

// UserMessage 是一条待投递文本。ID 在两列里都唯一。Source 标记来源，例如 user、steer、inject、job。
// Parts 是跟这条文本一起交给模型的多模态片段。
type UserMessage struct {
	ID     string
	Text   string
	Source string
	Parts  []Part
}

// Inbox 保存尚未被步骤取走的消息。
type Inbox struct {
	nextTurn []UserMessage
	nextStep []UserMessage
}

// HasPending 在任一列还有消息时为真。
func (n *Inbox) HasPending() bool {
	return len(n.nextTurn) > 0 || len(n.nextStep) > 0
}

// NextStep 返回下一步列的副本。
func (n *Inbox) NextStep() []UserMessage {
	return append([]UserMessage(nil), n.nextStep...)
}

// Append 把 message 加到 target 列尾。ID 与已有消息重复时返回错误。
func (n *Inbox) Append(target InboxTarget, message UserMessage) error {
	if message.ID == "" {
		return errInbox("empty message id")
	}
	if n.hasID(message.ID) {
		return errInbox("message " + message.ID + " is already pending")
	}
	switch target {
	case NextTurn:
		n.nextTurn = append(n.nextTurn, message)
	case NextStep:
		n.nextStep = append(n.nextStep, message)
	default:
		return errInbox("unknown inbox target")
	}
	return nil
}

// Claim 取走全部下一步消息。target 为 NextTurn 时再取走下一回合列的第一条。turn 只用于调用方记账。
func (n *Inbox) Claim(target InboxTarget, turn int) []UserMessage {
	_ = turn
	claimed := append([]UserMessage(nil), n.nextStep...)
	n.nextStep = nil
	if target == NextTurn && len(n.nextTurn) > 0 {
		claimed = append(claimed, n.nextTurn[0])
		n.nextTurn = n.nextTurn[1:]
	}
	return claimed
}

// Clear 先清空下一步，再清空下一回合。与取消时的顺序一致。
func (n *Inbox) Clear() {
	n.nextStep = nil
	n.nextTurn = nil
}

// hasID 在两列里已经有 id 时为真。
func (n *Inbox) hasID(id string) bool {
	for _, m := range n.nextTurn {
		if m.ID == id {
			return true
		}
	}
	for _, m := range n.nextStep {
		if m.ID == id {
			return true
		}
	}
	return false
}

// inboxError 是收件箱拒绝一条消息时的文本。
type inboxError string

// Error 返回这条收件箱错误的文本。
func (e inboxError) Error() string { return string(e) }

// errInbox 用 text 造一条收件箱错误。
func errInbox(text string) error { return inboxError(text) }
