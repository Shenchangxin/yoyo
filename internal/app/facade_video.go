// 视频门面。状态仍在 App 上。

package app

import (
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/schedule"
	"github.com/Shenchangxin/yoyo/internal/trace"
	"github.com/Shenchangxin/yoyo/internal/video"
)

// videoHost 视频门面。状态仍在 App 上。
type videoHost struct{ app *App }

// startCanvasAgent 转发到 videoHost。
func (a *App) startCanvasAgent(turn video.CanvasAgentTurn) error {
	return (&videoHost{app: a}).startCanvasAgent(turn)
}

// stopCanvasAgent 转发到 videoHost。
func (a *App) stopCanvasAgent(sessionID string) {
	(&videoHost{app: a}).stopCanvasAgent(sessionID)
}

// decideCanvasAgent 转发到 videoHost。
func (a *App) decideCanvasAgent(approvalID, decision string) error {
	return (&videoHost{app: a}).decideCanvasAgent(approvalID, decision)
}

// canvasAgentSession 转发到 videoHost。
func (a *App) canvasAgentSession(canvasID string) (string, error) {
	return (&videoHost{app: a}).canvasAgentSession(canvasID)
}

// ensureCanvasAgentWatch 转发到 videoHost。
func (a *App) ensureCanvasAgentWatch(sessionID string) error {
	return (&videoHost{app: a}).ensureCanvasAgentWatch(sessionID)
}

// canvasAgentLoop 转发到 videoHost。
func (a *App) canvasAgentLoop(sessionID string, ch <-chan trace.Event, unsub func()) {
	(&videoHost{app: a}).canvasAgentLoop(sessionID, ch, unsub)
}

// OpenStudioPublish 转发到 videoHost。
func (a *App) OpenStudioPublish(id string, analytics bool) (map[string]any, error) {
	return (&videoHost{app: a}).OpenStudioPublish(id, analytics)
}

// CreateStudioPersona 转发到 videoHost。
func (a *App) CreateStudioPersona(name, topicID string) (map[string]any, error) {
	return (&videoHost{app: a}).CreateStudioPersona(name, topicID)
}

// enrichPublish 转发到 videoHost。
func (a *App) enrichPublish(p video.Publish) map[string]any {
	return (&videoHost{app: a}).enrichPublish(p)
}

// openVideo 转发到 videoHost。
func (a *App) openVideo() error {
	return (&videoHost{app: a}).openVideo()
}

// onCanvasHub 转发到 videoHost。
func (a *App) onCanvasHub(kind string, payload map[string]any) {
	(&videoHost{app: a}).onCanvasHub(kind, payload)
}

// onVideoJob 转发到 videoHost。
func (a *App) onVideoJob(j video.Job) {
	(&videoHost{app: a}).onVideoJob(j)
}

// mediaURL 转发到 videoHost。
func (a *App) mediaURL(hash string) string {
	return (&videoHost{app: a}).mediaURL(hash)
}

// journalVideo 转发到 videoHost。
func (a *App) journalVideo(kind string, payload map[string]any) {
	(&videoHost{app: a}).journalVideo(kind, payload)
}

// requireVideo 转发到 videoHost。
func (a *App) requireVideo() (*video.Engine, error) {
	return (&videoHost{app: a}).requireVideo()
}

// VideoStatus 转发到 videoHost。
func (a *App) VideoStatus() map[string]any {
	return (&videoHost{app: a}).VideoStatus()
}

// attachDramaTools 转发到 videoHost。
func (a *App) attachDramaTools(tools *runtime.WorkspaceTools, sessionID string) {
	(&videoHost{app: a}).attachDramaTools(tools, sessionID)
}

// attachCanvasTools 转发到 videoHost。
func (a *App) attachCanvasTools(tools *runtime.WorkspaceTools, sessionID string) {
	(&videoHost{app: a}).attachCanvasTools(tools, sessionID)
}

// videoChannel 转发到 videoHost。
func (a *App) videoChannel(sessionID string) bool {
	return (&videoHost{app: a}).videoChannel(sessionID)
}

// studioLane 转发到 videoHost。
func (a *App) studioLane(sessionID string) bool {
	return (&videoHost{app: a}).studioLane(sessionID)
}

// conductorLane 转发到 videoHost。
func (a *App) conductorLane(sessionID string) bool {
	return (&videoHost{app: a}).conductorLane(sessionID)
}

// attachCutTools 转发到 videoHost。
func (a *App) attachCutTools(tools *runtime.WorkspaceTools, sessionID string) {
	(&videoHost{app: a}).attachCutTools(tools, sessionID)
}

// attachConductorTools 转发到 videoHost。
func (a *App) attachConductorTools(tools *runtime.WorkspaceTools, sessionID string) {
	(&videoHost{app: a}).attachConductorTools(tools, sessionID)
}

// attachRoutedWorkshop 转发到 videoHost。
func (a *App) attachRoutedWorkshop(tools *runtime.WorkspaceTools, sessionID string) string {
	return (&videoHost{app: a}).attachRoutedWorkshop(tools, sessionID)
}

// attachStudioTools 转发到 videoHost。
func (a *App) attachStudioTools(tools *runtime.WorkspaceTools, sessionID string) {
	(&videoHost{app: a}).attachStudioTools(tools, sessionID)
}

// pinStudioLane 转发到 videoHost。
func (a *App) pinStudioLane(tools *runtime.WorkspaceTools) {
	(&videoHost{app: a}).pinStudioLane(tools)
}

// pinConductorLane 转发到 videoHost。
func (a *App) pinConductorLane(tools *runtime.WorkspaceTools) {
	(&videoHost{app: a}).pinConductorLane(tools)
}

// RunDramaStage 转发到 videoHost。
func (a *App) RunDramaStage(sessionID, episodeID, stage, dramaID string) error {
	return (&videoHost{app: a}).RunDramaStage(sessionID, episodeID, stage, dramaID)
}

// DramaBind 转发到 videoHost。
func (a *App) DramaBind(sessionID, episodeID, dramaID string) error {
	return (&videoHost{app: a}).DramaBind(sessionID, episodeID, dramaID)
}

// VideoCall 转发到 videoHost。
func (a *App) VideoCall(method string, params map[string]any) (any, error) {
	return (&videoHost{app: a}).VideoCall(method, params)
}

// CanvasBind 转发到 videoHost。
func (a *App) CanvasBind(sessionID, canvasID string) error {
	return (&videoHost{app: a}).CanvasBind(sessionID, canvasID)
}

// enrichBundle 转发到 videoHost。
func (a *App) enrichBundle(b video.EpisodeBundle) map[string]any {
	return (&videoHost{app: a}).enrichBundle(b)
}

// pendingTopicJob 转发到 videoHost。
func (a *App) pendingTopicJob(topicID string) (schedule.Job, bool) {
	return (&videoHost{app: a}).pendingTopicJob(topicID)
}

// cancelPendingTopicJobs 转发到 videoHost。
func (a *App) cancelPendingTopicJobs(topicID string) int {
	return (&videoHost{app: a}).cancelPendingTopicJobs(topicID)
}

// runStudioJob 转发到 videoHost。
func (a *App) runStudioJob(j schedule.Job) {
	(&videoHost{app: a}).runStudioJob(j)
}

// enrichStudio 转发到 videoHost。
func (a *App) enrichStudio(p video.StudioProject) map[string]any {
	return (&videoHost{app: a}).enrichStudio(p)
}

// enrichMedia 转发到 videoHost。
func (a *App) enrichMedia(ref video.MediaRef) map[string]any {
	return (&videoHost{app: a}).enrichMedia(ref)
}

// enrichCut 转发到 videoHost。
func (a *App) enrichCut(p video.CutProject) map[string]any {
	return (&videoHost{app: a}).enrichCut(p)
}

// conductorBrief 转发到 videoHost。
func (a *App) conductorBrief(tools *runtime.WorkspaceTools, args map[string]any) runtime.ToolResult {
	return (&videoHost{app: a}).conductorBrief(tools, args)
}
