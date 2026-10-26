package desktop

import (
	"encoding/base64"
	"errors"
	"strings"

	"github.com/Shenchangxin/yoyo/internal/app"
	"github.com/Shenchangxin/yoyo/internal/skin"
)

// ListSkins 在 RPC 非空时调用 skins.list。结果为 nil 或没有 App 时返回空切片。否则走 App.ListSkins。
func (s *Service) ListSkins() []skin.Info {
	if s.RPC != nil {
		v, err := s.call("skins.list", nil)
		out, _ := decode[[]skin.Info](v, err)
		if out == nil {
			return []skin.Info{}
		}
		return out
	}
	if s.App == nil {
		return []skin.Info{}
	}
	return s.App.ListSkins()
}

// GetSkin 在 RPC 非空时调用 skins.get。没有 App 时返回 no app。否则走 App.GetSkin。
func (s *Service) GetSkin(id string) (map[string]any, error) {
	if s.RPC != nil {
		v, err := s.call("skins.get", map[string]any{"id": id})
		return decode[map[string]any](v, err)
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	return s.App.GetSkin(id)
}

// ImportSkin 在 RPC 非空时调用 skins.import，字段是 bytes_b64。同进程先 decodeB64，再 App.ImportSkin。没有 App 时返回 no app。
func (s *Service) ImportSkin(b64 string) (skin.Info, error) {
	if s.RPC != nil {
		v, err := s.call("skins.import", map[string]any{"bytes_b64": b64})
		return decode[skin.Info](v, err)
	}
	if s.App == nil {
		return skin.Info{}, errors.New("no app")
	}
	raw, err := decodeB64(b64)
	if err != nil {
		return skin.Info{}, err
	}
	return s.App.ImportSkin(raw)
}

// ImportSkinPath 在 RPC 非空时调用 skins.import，字段是 path。没有 App 时返回 no app。否则走 App.ImportSkinPath。
func (s *Service) ImportSkinPath(path string) (skin.Info, error) {
	if s.RPC != nil {
		v, err := s.call("skins.import", map[string]any{"path": path})
		return decode[skin.Info](v, err)
	}
	if s.App == nil {
		return skin.Info{}, errors.New("no app")
	}
	return s.App.ImportSkinPath(path)
}

// ExportSkin 在 RPC 非空时调用 skins.export。同进程返回 filename 和 bytes_b64。没有 App 时返回 no app。
func (s *Service) ExportSkin(id string) (map[string]any, error) {
	if s.RPC != nil {
		v, err := s.call("skins.export", map[string]any{"id": id})
		return decode[map[string]any](v, err)
	}
	if s.App == nil {
		return nil, errors.New("no app")
	}
	name, body, err := s.App.ExportSkin(id)
	if err != nil {
		return nil, err
	}
	return map[string]any{"filename": name, "bytes_b64": base64.StdEncoding.EncodeToString(body)}, nil
}

// DeleteSkin 在 RPC 非空时调用 skins.delete。没有 App 时返回 no app。否则走 App.DeleteSkin。
func (s *Service) DeleteSkin(id string) error {
	if s.RPC != nil {
		_, err := s.call("skins.delete", map[string]any{"id": id})
		return err
	}
	if s.App == nil {
		return errors.New("no app")
	}
	return s.App.DeleteSkin(id)
}

// ActivateSkin 在 RPC 非空时调用 skins.activate。没有 App 时返回 no app。否则走 App.ActivateSkin。
func (s *Service) ActivateSkin(id string) error {
	if s.RPC != nil {
		_, err := s.call("skins.activate", map[string]any{"id": id})
		return err
	}
	if s.App == nil {
		return errors.New("no app")
	}
	return s.App.ActivateSkin(id)
}

// ResolveSkin 在 RPC 非空时调用 skins.resolve。没有 App 时返回 no app。否则走 App.ResolveSkin。
func (s *Service) ResolveSkin(id, mode string) (skin.Resolved, error) {
	if s.RPC != nil {
		v, err := s.call("skins.resolve", map[string]any{"id": id, "mode": mode})
		return decode[skin.Resolved](v, err)
	}
	if s.App == nil {
		return skin.Resolved{}, errors.New("no app")
	}
	return s.App.ResolveSkin(id, mode)
}

// SaveSkin 在 RPC 非空时调用 skins.save。没有 App 时返回 no app。否则走 App.SaveSkin。
func (s *Service) SaveSkin(in app.SkinSaveIn) (skin.Info, error) {
	if s.RPC != nil {
		v, err := s.call("skins.save", in)
		return decode[skin.Info](v, err)
	}
	if s.App == nil {
		return skin.Info{}, errors.New("no app")
	}
	return s.App.SaveSkin(in)
}

// decodeB64 去掉空白。空字符串返回 empty。逗号前带 base64 时去掉 data URL 头，再用标准 Base64 解码。
func decodeB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, errors.New("empty")
	}
	if i := strings.Index(s, ","); i >= 0 && strings.Contains(s[:i], "base64") {
		s = s[i+1:]
	}
	return base64.StdEncoding.DecodeString(s)
}
