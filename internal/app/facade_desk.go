// 桌面简报门面。状态仍在 App 上。

package app

import (
	"time"

	"github.com/Shenchangxin/yoyo/internal/dshdesk"
	"github.com/Shenchangxin/yoyo/internal/initiative"
	"github.com/Shenchangxin/yoyo/internal/poi"
	"github.com/Shenchangxin/yoyo/internal/reach"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// deskHost 桌面简报门面。状态仍在 App 上。
type deskHost struct{ app *App }

// ComposeToday 转发到 deskHost。
func (a *App) ComposeToday(slot string) TodayBrief {
	return (&deskHost{app: a}).ComposeToday(slot)
}

// briefMaterial 转发到 deskHost。
func (a *App) briefMaterial(slot string, now time.Time) *briefMaterial {
	return (&deskHost{app: a}).briefMaterial(slot, now)
}

// briefDir 转发到 deskHost。
func (a *App) briefDir() string {
	return (&deskHost{app: a}).briefDir()
}

// briefPath 转发到 deskHost。
func (a *App) briefPath(slot, day string) string {
	return (&deskHost{app: a}).briefPath(slot, day)
}

// briefRecorded 转发到 deskHost。
func (a *App) briefRecorded(slot, day string) bool {
	return (&deskHost{app: a}).briefRecorded(slot, day)
}

// rememberSessionBrief 转发到 deskHost。
func (a *App) rememberSessionBrief(sessionID, id string) {
	(&deskHost{app: a}).rememberSessionBrief(sessionID, id)
}

// rememberBrief 转发到 deskHost。
func (a *App) rememberBrief(note DeskNote, at time.Time) {
	(&deskHost{app: a}).rememberBrief(note, at)
}

// saveBriefRecord 转发到 deskHost。
func (a *App) saveBriefRecord(rec BriefRecord) {
	(&deskHost{app: a}).saveBriefRecord(rec)
}

// BriefList 转发到 deskHost。
func (a *App) BriefList() []BriefRecord {
	return (&deskHost{app: a}).BriefList()
}

// sealBriefs 转发到 deskHost。
func (a *App) sealBriefs() {
	(&deskHost{app: a}).sealBriefs()
}

// agentDesk 转发到 deskHost。
func (a *App) agentDesk() *dshdesk.Host {
	return (&deskHost{app: a}).agentDesk()
}

// orchestrator 转发到 deskHost。
func (a *App) orchestrator() *superagent.Runtime {
	return (&deskHost{app: a}).orchestrator()
}

// workerProfile 转发到 deskHost。
func (a *App) workerProfile() string {
	return (&deskHost{app: a}).workerProfile()
}

// relayWaiting 转发到 deskHost。
func (a *App) relayWaiting() {
	(&deskHost{app: a}).relayWaiting()
}

// stopOrchestrator 转发到 deskHost。
func (a *App) stopOrchestrator() {
	(&deskHost{app: a}).stopOrchestrator()
}

// OrchestratorView 转发到 deskHost。
func (a *App) OrchestratorView() superagent.View {
	return (&deskHost{app: a}).OrchestratorView()
}

// OrchestratorDecide 转发到 deskHost。
func (a *App) OrchestratorDecide(triggerID, kind, reason string) error {
	return (&deskHost{app: a}).OrchestratorDecide(triggerID, kind, reason)
}

// OrchestratorMode 转发到 deskHost。
func (a *App) OrchestratorMode(mode string) error {
	return (&deskHost{app: a}).OrchestratorMode(mode)
}

// OrchestratorPaused 转发到 deskHost。
func (a *App) OrchestratorPaused(paused bool) error {
	return (&deskHost{app: a}).OrchestratorPaused(paused)
}

// OrchestratorDeliver 转发到 deskHost。
func (a *App) OrchestratorDeliver() int {
	return (&deskHost{app: a}).OrchestratorDeliver()
}

// WorkerTranscript 转发到 deskHost。
func (a *App) WorkerTranscript(ticketID string) []superagent.TranscriptLine {
	return (&deskHost{app: a}).WorkerTranscript(ticketID)
}

// OrchestratorAct 转发到 deskHost。
func (a *App) OrchestratorAct(kind, ticketID, triggerID, text, reason string) error {
	return (&deskHost{app: a}).OrchestratorAct(kind, ticketID, triggerID, text, reason)
}

// OrchestratorAdmission 转发到 deskHost。
func (a *App) OrchestratorAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed string) error {
	return (&deskHost{app: a}).OrchestratorAdmission(listedUnaddressed, unlistedAddressed, unlistedUnaddressed)
}

// DeskState 转发到 deskHost。
func (a *App) DeskState() DeskState {
	return (&deskHost{app: a}).DeskState()
}

// ensureDeskLoop 转发到 deskHost。
func (a *App) ensureDeskLoop() {
	(&deskHost{app: a}).ensureDeskLoop()
}

// stopDeskLoop 转发到 deskHost。
func (a *App) stopDeskLoop() {
	(&deskHost{app: a}).stopDeskLoop()
}

// deskLoop 转发到 deskHost。
func (a *App) deskLoop(stop <-chan struct{}) {
	(&deskHost{app: a}).deskLoop(stop)
}

// deskTick 转发到 deskHost。
func (a *App) deskTick(now time.Time) {
	(&deskHost{app: a}).deskTick(now)
}

// deliverSpeaks 转发到 deskHost。
func (a *App) deliverSpeaks(sessionID string, b initiative.Briefing, now time.Time) []initiative.Signal {
	return (&deskHost{app: a}).deliverSpeaks(sessionID, b, now)
}

// DeskPush 转发到 deskHost。
func (a *App) DeskPush(note DeskNote) bool {
	return (&deskHost{app: a}).DeskPush(note)
}

// traceHasID 转发到 deskHost。
func (a *App) traceHasID(sessionID, id string) bool {
	return (&deskHost{app: a}).traceHasID(sessionID, id)
}

// PoiRefresh 转发到 deskHost。
func (a *App) PoiRefresh() poi.Snapshot {
	return (&deskHost{app: a}).PoiRefresh()
}

// PoiAck 转发到 deskHost。
func (a *App) PoiAck(sig string) bool {
	return (&deskHost{app: a}).PoiAck(sig)
}

// touchPoi 转发到 deskHost。
func (a *App) touchPoi() {
	(&deskHost{app: a}).touchPoi()
}

// poiObservations 转发到 deskHost。
func (a *App) poiObservations() ([]poi.Observation, []string) {
	return (&deskHost{app: a}).poiObservations()
}

// fileCensus 转发到 deskHost。
func (a *App) fileCensus(c poi.Census) {
	(&deskHost{app: a}).fileCensus(c)
}

// deskBook 转发到 deskHost。
func (a *App) deskBook() (*initiative.Store, error) {
	return (&deskHost{app: a}).deskBook()
}

// assessDesk 转发到 deskHost。
func (a *App) assessDesk(snap poi.Snapshot) (initiative.Briefing, bool) {
	return (&deskHost{app: a}).assessDesk(snap)
}

// keepBriefing 转发到 deskHost。
func (a *App) keepBriefing(b initiative.Briefing, at time.Time) {
	(&deskHost{app: a}).keepBriefing(b, at)
}

// DeskFeedback 转发到 deskHost。
func (a *App) DeskFeedback(className, kind, memoryID string) bool {
	return (&deskHost{app: a}).DeskFeedback(className, kind, memoryID)
}

// noteSchedule 转发到 deskHost。
func (a *App) noteSchedule(id, title string) {
	(&deskHost{app: a}).noteSchedule(id, title)
}

// scheduleAllowed 转发到 deskHost。
func (a *App) scheduleAllowed() bool {
	return (&deskHost{app: a}).scheduleAllowed()
}

// connectorFacts 转发到 deskHost。
func (a *App) connectorFacts() []initiative.Fact {
	return (&deskHost{app: a}).connectorFacts()
}

// orchestratorFacts 转发到 deskHost。
func (a *App) orchestratorFacts(now time.Time) []initiative.Fact {
	return (&deskHost{app: a}).orchestratorFacts(now)
}

// reachFacts 转发到 deskHost。
func (a *App) reachFacts(seen map[string]string, now time.Time) []initiative.Fact {
	return (&deskHost{app: a}).reachFacts(seen, now)
}

// reachLines 转发到 deskHost。
func (a *App) reachLines() []string {
	return (&deskHost{app: a}).reachLines()
}

// runReach 转发到 deskHost。
func (a *App) runReach(probe reach.Probe, seen map[string]bool, pace string) {
	(&deskHost{app: a}).runReach(probe, seen, pace)
}

// reachPace 转发到 deskHost。
func (a *App) reachPace(now time.Time) string {
	return (&deskHost{app: a}).reachPace(now)
}

// reachBudget 转发到 deskHost。
func (a *App) reachBudget(now time.Time) (reach.Allowance, bool) {
	return (&deskHost{app: a}).reachBudget(now)
}

// reachBusyNow 转发到 deskHost。
func (a *App) reachBusyNow() bool {
	return (&deskHost{app: a}).reachBusyNow()
}

// reachUserAt 转发到 deskHost。
func (a *App) reachUserAt() time.Time {
	return (&deskHost{app: a}).reachUserAt()
}

// touchReachUser 转发到 deskHost。
func (a *App) touchReachUser(now time.Time) {
	(&deskHost{app: a}).touchReachUser(now)
}

// reachPath 转发到 deskHost。
func (a *App) reachPath() string {
	return (&deskHost{app: a}).reachPath()
}

// readReachLog 转发到 deskHost。
func (a *App) readReachLog() reach.Log {
	return (&deskHost{app: a}).readReachLog()
}

// readReachLogLocked 转发到 deskHost。
func (a *App) readReachLogLocked() reach.Log {
	return (&deskHost{app: a}).readReachLogLocked()
}

// updateReachLog 转发到 deskHost。
func (a *App) updateReachLog(fn func(*reach.Log)) {
	(&deskHost{app: a}).updateReachLog(fn)
}

// writeReachLogLocked 转发到 deskHost。
func (a *App) writeReachLogLocked(log reach.Log) {
	(&deskHost{app: a}).writeReachLogLocked(log)
}

// reachNote 转发到 deskHost。
func (a *App) reachNote(item reach.Item) string {
	return (&deskHost{app: a}).reachNote(item)
}
