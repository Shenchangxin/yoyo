// 偶像门面。状态仍在 App 上。

package app

import (
	"context"
	"time"

	"github.com/Shenchangxin/yoyo/internal/idol"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// idolHost 偶像门面。状态仍在 App 上。
type idolHost struct{ app *App }

// idolEngine 转发到 idolHost。
func (a *App) idolEngine() *idol.Engine {
	return (&idolHost{app: a}).idolEngine()
}

// cancelIdolTurn 转发到 idolHost。
func (a *App) cancelIdolTurn() {
	(&idolHost{app: a}).cancelIdolTurn()
}

// IdolCall 转发到 idolHost。
func (a *App) IdolCall(op, payload string) (string, error) {
	return (&idolHost{app: a}).IdolCall(op, payload)
}

// idolTurn 转发到 idolHost。
func (a *App) idolTurn(payload string) (string, error) {
	return (&idolHost{app: a}).idolTurn(payload)
}

// idolNarrate 转发到 idolHost。
func (a *App) idolNarrate(payload string) (string, error) {
	return (&idolHost{app: a}).idolNarrate(payload)
}

// idolReach 转发到 idolHost。
func (a *App) idolReach(idolID string, now time.Time) (string, error) {
	return (&idolHost{app: a}).idolReach(idolID, now)
}

// startIdolWork 转发到 idolHost。
func (a *App) startIdolWork(idolID, instruction string) (string, error) {
	return (&idolHost{app: a}).startIdolWork(idolID, instruction)
}

// lastAssistant 转发到 idolHost。
func (a *App) lastAssistant(sessionID string) string {
	return (&idolHost{app: a}).lastAssistant(sessionID)
}

// openIdolTurn 转发到 idolHost。
func (a *App) openIdolTurn(in idol.TurnInput) (*idolLiveTurn, error) {
	return (&idolHost{app: a}).openIdolTurn(in)
}

// runIdolTurn 转发到 idolHost。
func (a *App) runIdolTurn(ctx context.Context, t *idolLiveTurn, in idol.TurnInput) {
	(&idolHost{app: a}).runIdolTurn(ctx, t, in)
}

// watchIdolWork 转发到 idolHost。
func (a *App) watchIdolWork(idolID, sid, recordID string) *idolWorkWatch {
	return (&idolHost{app: a}).watchIdolWork(idolID, sid, recordID)
}

// waitIdolPhoto 转发到 idolHost。
func (a *App) waitIdolPhoto(recordID string, timeout time.Duration) (string, string) {
	return (&idolHost{app: a}).waitIdolPhoto(recordID, timeout)
}

// idolSettingsView 转发到 idolHost。
func (a *App) idolSettingsView() idolSettingsView {
	return (&idolHost{app: a}).idolSettingsView()
}

// idolSaveSettings 转发到 idolHost。
func (a *App) idolSaveSettings(payload string) (string, error) {
	return (&idolHost{app: a}).idolSaveSettings(payload)
}

// putIdolKey 转发到 idolHost。
func (a *App) putIdolKey(name, value string) {
	(&idolHost{app: a}).putIdolKey(name, value)
}

// vaultIdol 转发到 idolHost。
func (a *App) vaultIdol(name string) string {
	return (&idolHost{app: a}).vaultIdol(name)
}

// idolSnapshot 转发到 idolHost。
func (a *App) idolSnapshot() (string, error) {
	return (&idolHost{app: a}).idolSnapshot()
}

// idolSave 转发到 idolHost。
func (a *App) idolSave(payload string) (string, error) {
	return (&idolHost{app: a}).idolSave(payload)
}

// idolSaveCircle 转发到 idolHost。
func (a *App) idolSaveCircle(payload string) (string, error) {
	return (&idolHost{app: a}).idolSaveCircle(payload)
}

// idolSaveLore 转发到 idolHost。
func (a *App) idolSaveLore(payload string) (string, error) {
	return (&idolHost{app: a}).idolSaveLore(payload)
}

// idolVoice 转发到 idolHost。
func (a *App) idolVoice(idolID string) string {
	return (&idolHost{app: a}).idolVoice(idolID)
}

// idolClient 转发到 idolHost。
func (a *App) idolClient(kind string) (*runtime.OpenAIClient, string, error) {
	return (&idolHost{app: a}).idolClient(kind)
}

// idolSessionClient 转发到 idolHost。
func (a *App) idolSessionClient(picked string) (*runtime.OpenAIClient, string, error) {
	return (&idolHost{app: a}).idolSessionClient(picked)
}

// idolLegacyClient 转发到 idolHost。
func (a *App) idolLegacyClient(base, picked string) (*runtime.OpenAIClient, string, error) {
	return (&idolHost{app: a}).idolLegacyClient(base, picked)
}

// idolConfigClient 转发到 idolHost。
func (a *App) idolConfigClient(picked string) (*runtime.OpenAIClient, string, error) {
	return (&idolHost{app: a}).idolConfigClient(picked)
}

// fallbackChatKey 转发到 idolHost。
func (a *App) fallbackChatKey() string {
	return (&idolHost{app: a}).fallbackChatKey()
}

// idolSpeechCreds 转发到 idolHost。
func (a *App) idolSpeechCreds(ignored1 string) (base, key, model string, err error) {
	return (&idolHost{app: a}).idolSpeechCreds(ignored1)
}

// idolSpeak 转发到 idolHost。
func (a *App) idolSpeak(ctx context.Context, idolID, text, cue string) ([]byte, string, error) {
	return (&idolHost{app: a}).idolSpeak(ctx, idolID, text, cue)
}

// idolASRCreds 转发到 idolHost。
func (a *App) idolASRCreds() (base, key, model string, err error) {
	return (&idolHost{app: a}).idolASRCreds()
}

// idolTranscribe 转发到 idolHost。
func (a *App) idolTranscribe(ctx context.Context, filename string, audio []byte, hint string) (string, error) {
	return (&idolHost{app: a}).idolTranscribe(ctx, filename, audio, hint)
}

// idolImageCreds 转发到 idolHost。
func (a *App) idolImageCreds() (base, key, model string, err error) {
	return (&idolHost{app: a}).idolImageCreds()
}

// idolPaint 转发到 idolHost。
func (a *App) idolPaint(ctx context.Context, prompt string) ([]byte, error) {
	return (&idolHost{app: a}).idolPaint(ctx, prompt)
}

// idolCanPaint 转发到 idolHost。
func (a *App) idolCanPaint() bool {
	return (&idolHost{app: a}).idolCanPaint()
}

// idolMediaFields 转发到 idolHost。
func (a *App) idolMediaFields(kind string) (base, key, model, provider string) {
	return (&idolHost{app: a}).idolMediaFields(kind)
}

// borrowMediaKey 转发到 idolHost。
func (a *App) borrowMediaKey(cap, provider, endpoint, key string) string {
	return (&idolHost{app: a}).borrowMediaKey(cap, provider, endpoint, key)
}

// leaseMediaKey 转发到 idolHost。
func (a *App) leaseMediaKey(cap, vendor, endpoint string, strict bool) string {
	return (&idolHost{app: a}).leaseMediaKey(cap, vendor, endpoint, strict)
}

// idolMediaFallback 转发到 idolHost。
func (a *App) idolMediaFallback() idolMediaFallback {
	return (&idolHost{app: a}).idolMediaFallback()
}

// activeMedia 转发到 idolHost。
func (a *App) activeMedia(cap string) (name, vendor, endpoint, model string) {
	return (&idolHost{app: a}).activeMedia(cap)
}

// idolPing 转发到 idolHost。
func (a *App) idolPing(payload string) (string, error) {
	return (&idolHost{app: a}).idolPing(payload)
}
