package desktop

import (
	"context"
	"fmt"
	"time"

	"github.com/Shenchangxin/yoyo/internal/im"
)

// IMSnapshot 返回消息目录、规则、账号和状态。密钥原文不在里面。
func (s *Service) IMSnapshot() (im.Snapshot, error) {
	if s.RPC != nil {
		return decode[im.Snapshot](s.call("im.snapshot", nil))
	}
	if s.App == nil || s.App.IM == nil {
		return im.Snapshot{}, fmt.Errorf("messaging is not running")
	}
	return s.App.IM.Snapshot(), nil
}

// IMSaveSettings 写下共用规则。正在跑的账号会按新规则重载。
func (s *Service) IMSaveSettings(settings im.Settings) error {
	if s.RPC != nil {
		_, err := s.call("im.settings", settings)
		return err
	}
	if s.App == nil || s.App.IM == nil {
		return fmt.Errorf("messaging is not running")
	}
	return s.App.IM.SaveSettings(settings)
}

// IMSaveAccount 新增或替换一个机器人。secrets 里的空字符串保留原密钥。
func (s *Service) IMSaveAccount(account im.Account, secrets map[string]string) (im.Account, error) {
	if s.RPC != nil {
		return decode[im.Account](s.call("im.account.save", map[string]any{
			"account": account,
			"secrets": secrets,
		}))
	}
	if s.App == nil || s.App.IM == nil {
		return im.Account{}, fmt.Errorf("messaging is not running")
	}
	return s.App.IM.SaveAccount(account, secrets)
}

// IMDeleteAccount 停掉并忘掉一个机器人，同时清掉它的密钥。
func (s *Service) IMDeleteAccount(id string) error {
	if s.RPC != nil {
		_, err := s.call("im.account.delete", map[string]string{"id": id})
		return err
	}
	if s.App == nil || s.App.IM == nil {
		return fmt.Errorf("messaging is not running")
	}
	return s.App.IM.DeleteAccount(id)
}

// IMSend 用一个已启用的机器人主动发一条文本。
func (s *Service) IMSend(accountID, chatID, text string) error {
	if s.RPC != nil {
		_, err := s.call("im.send", map[string]string{
			"account_id": accountID,
			"chat_id":    chatID,
			"text":       text,
		})
		return err
	}
	if s.App == nil || s.App.IM == nil {
		return fmt.Errorf("messaging is not running")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	return s.App.IM.Push(ctx, accountID, chatID, text)
}
