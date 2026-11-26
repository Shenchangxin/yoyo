// 学习门面。状态仍在 App 上。

package app

import (
	"context"

	"github.com/Shenchangxin/yoyo/internal/learn"
	"github.com/Shenchangxin/yoyo/internal/runtime"
)

// learnHost 学习门面。状态仍在 App 上。
type learnHost struct{ app *App }

// LearnCall 转发到 learnHost。
func (a *App) LearnCall(method string, params map[string]any) (any, error) {
	return (&learnHost{app: a}).LearnCall(method, params)
}

// learnCoreCall 转发到 learnHost。
func (a *App) learnCoreCall(method string, params map[string]any) (any, bool, error) {
	return (&learnHost{app: a}).learnCoreCall(method, params)
}

// SetSessionLearn 转发到 learnHost。
func (a *App) SetSessionLearn(session string, bind LearnBinding) (SessionMeta, error) {
	return (&learnHost{app: a}).SetSessionLearn(session, bind)
}

// publishLearn 转发到 learnHost。
func (a *App) publishLearn(c learn.Change) {
	(&learnHost{app: a}).publishLearn(c)
}

// learnBooksCall 转发到 learnHost。
func (a *App) learnBooksCall(method string, params map[string]any) (any, bool, error) {
	return (&learnHost{app: a}).learnBooksCall(method, params)
}

// learnLibraryCall 转发到 learnHost。
func (a *App) learnLibraryCall(method string, params map[string]any) (any, bool, error) {
	return (&learnHost{app: a}).learnLibraryCall(method, params)
}

// learnImportURL 转发到 learnHost。
func (a *App) learnImportURL(libraryID, rawURL string) (learn.Document, error) {
	return (&learnHost{app: a}).learnImportURL(libraryID, rawURL)
}

// learnMasteryCall 转发到 learnHost。
func (a *App) learnMasteryCall(method string, params map[string]any) (any, bool, error) {
	return (&learnHost{app: a}).learnMasteryCall(method, params)
}

// learnPracticeCall 转发到 learnHost。
func (a *App) learnPracticeCall(method string, params map[string]any) (any, bool, error) {
	return (&learnHost{app: a}).learnPracticeCall(method, params)
}

// learnComplete 转发到 learnHost。
func (a *App) learnComplete(ctx context.Context, req learn.CompleteRequest) (string, error) {
	return (&learnHost{app: a}).learnComplete(ctx, req)
}

// embedTexts 转发到 learnHost。
func (a *App) embedTexts(model string, texts []string) ([][]float32, error) {
	return (&learnHost{app: a}).embedTexts(model, texts)
}

// embedBatch 转发到 learnHost。
func (a *App) embedBatch(model, key string, texts []string) ([][]float32, error) {
	return (&learnHost{app: a}).embedBatch(model, key, texts)
}

// learnTurnBrief 转发到 learnHost。
func (a *App) learnTurnBrief(meta SessionMeta) string {
	return (&learnHost{app: a}).learnTurnBrief(meta)
}

// learnPathBrief 转发到 learnHost。
func (a *App) learnPathBrief(meta SessionMeta) string {
	return (&learnHost{app: a}).learnPathBrief(meta)
}

// attachLearn 转发到 learnHost。
func (a *App) attachLearn(tools *runtime.WorkspaceTools) {
	(&learnHost{app: a}).attachLearn(tools)
}

// mountJournalTool 转发到 learnHost。
func (a *App) mountJournalTool(m *learnMount) {
	(&learnHost{app: a}).mountJournalTool(m)
}

// mountPathTools 转发到 learnHost。
func (a *App) mountPathTools(m *learnMount, pathID string) {
	(&learnHost{app: a}).mountPathTools(m, pathID)
}

// needMode 转发到 learnHost。
func (a *App) needMode(pathID, tool string, allowed ...string) error {
	return (&learnHost{app: a}).needMode(pathID, tool, allowed...)
}

// reviewScope 转发到 learnHost。
func (a *App) reviewScope(pathID, pointID string) error {
	return (&learnHost{app: a}).reviewScope(pathID, pointID)
}

// mountBookTools 转发到 learnHost。
func (a *App) mountBookTools(m *learnMount, book string) {
	(&learnHost{app: a}).mountBookTools(m, book)
}

// learnBookBrief 转发到 learnHost。
func (a *App) learnBookBrief(meta SessionMeta) string {
	return (&learnHost{app: a}).learnBookBrief(meta)
}

// mountLibraryTools 转发到 learnHost。
func (a *App) mountLibraryTools(m *learnMount, lib string) {
	(&learnHost{app: a}).mountLibraryTools(m, lib)
}

// mountMaterialTools 转发到 learnHost。
func (a *App) mountMaterialTools(m *learnMount, mat string) {
	(&learnHost{app: a}).mountMaterialTools(m, mat)
}

// setLearnViewport 转发到 learnHost。
func (a *App) setLearnViewport(session string, vp learnViewport) {
	(&learnHost{app: a}).setLearnViewport(session, vp)
}

// learnViewportOf 转发到 learnHost。
func (a *App) learnViewportOf(session string) (learnViewport, bool) {
	return (&learnHost{app: a}).learnViewportOf(session)
}

// learnMaterialBrief 转发到 learnHost。
func (a *App) learnMaterialBrief(meta SessionMeta) string {
	return (&learnHost{app: a}).learnMaterialBrief(meta)
}

// mountPracticeTools 转发到 learnHost。
func (a *App) mountPracticeTools(m *learnMount) {
	(&learnHost{app: a}).mountPracticeTools(m)
}
