// 路由门面。状态仍在 App 上。

package app

import (
	"time"

	"github.com/Shenchangxin/yoyo/internal/agentloop"
	"github.com/Shenchangxin/yoyo/internal/buddy"
	"github.com/Shenchangxin/yoyo/internal/runtime"
	"github.com/Shenchangxin/yoyo/internal/superagent"
)

// routeHost 路由门面。状态仍在 App 上。
type routeHost struct{ app *App }

// beginRoute 转发到 routeHost。
func (a *App) beginRoute(tools *runtime.WorkspaceTools, meta SessionMeta, sessionID, turnID string) bool {
	return (&routeHost{app: a}).beginRoute(tools, meta, sessionID, turnID)
}

// routeSession 转发到 routeHost。
func (a *App) routeSession(meta SessionMeta, sessionID string) bool {
	return (&routeHost{app: a}).routeSession(meta, sessionID)
}

// isMainSession 转发到 routeHost。
func (a *App) isMainSession(sessionID string) bool {
	return (&routeHost{app: a}).isMainSession(sessionID)
}

// chatSlotFor 转发到 routeHost。
func (a *App) chatSlotFor(meta SessionMeta, sessionID string) SessionMeta {
	return (&routeHost{app: a}).chatSlotFor(meta, sessionID)
}

// adoptChatDefault 转发到 routeHost。
func (a *App) adoptChatDefault(meta SessionMeta) {
	(&routeHost{app: a}).adoptChatDefault(meta)
}

// compactOptsFor 转发到 routeHost。
func (a *App) compactOptsFor(routing bool) runtime.CompactOpts {
	return (&routeHost{app: a}).compactOptsFor(routing)
}

// intakeDesk 转发到 routeHost。
func (a *App) intakeDesk(sessionID, turnID, text string) string {
	return (&routeHost{app: a}).intakeDesk(sessionID, turnID, text)
}

// routerExtrasFor 转发到 routeHost。
func (a *App) routerExtrasFor(sessionID, turnID string) map[string]runtime.ExtraTool {
	return (&routeHost{app: a}).routerExtrasFor(sessionID, turnID)
}

// heard 转发到 routeHost。
func (a *App) heard(sessionID, text string) {
	(&routeHost{app: a}).heard(sessionID, text)
}

// heardIn 转发到 routeHost。
func (a *App) heardIn(sessionID string) []string {
	return (&routeHost{app: a}).heardIn(sessionID)
}

// routeTool 转发到 routeHost。
func (a *App) routeTool(sessionID, turnID string) runtime.ExtraTool {
	return (&routeHost{app: a}).routeTool(sessionID, turnID)
}

// routeOne 转发到 routeHost。
func (a *App) routeOne(rt *superagent.Runtime, sessionID, turnID string, item map[string]any) (string, error) {
	return (&routeHost{app: a}).routeOne(rt, sessionID, turnID, item)
}

// bindRouteTrigger 转发到 routeHost。
func (a *App) bindRouteTrigger(rt *superagent.Runtime, sessionID, turnID, given string) (string, error) {
	return (&routeHost{app: a}).bindRouteTrigger(rt, sessionID, turnID, given)
}

// statusTool 转发到 routeHost。
func (a *App) statusTool() runtime.ExtraTool {
	return (&routeHost{app: a}).statusTool()
}

// taskRoster 转发到 routeHost。
func (a *App) taskRoster(rt *superagent.Runtime) string {
	return (&routeHost{app: a}).taskRoster(rt)
}

// noteTool 转发到 routeHost。
func (a *App) noteTool() runtime.ExtraTool {
	return (&routeHost{app: a}).noteTool()
}

// stopTool 转发到 routeHost。
func (a *App) stopTool() runtime.ExtraTool {
	return (&routeHost{app: a}).stopTool()
}

// memoryTool 转发到 routeHost。
func (a *App) memoryTool() runtime.ExtraTool {
	return (&routeHost{app: a}).memoryTool()
}

// memoryWriteTool 转发到 routeHost。
func (a *App) memoryWriteTool(sessionID string) runtime.ExtraTool {
	return (&routeHost{app: a}).memoryWriteTool(sessionID)
}

// memoryForgetTool 转发到 routeHost。
func (a *App) memoryForgetTool() runtime.ExtraTool {
	return (&routeHost{app: a}).memoryForgetTool()
}

// mountWorker 转发到 routeHost。
func (a *App) mountWorker(reg *agentloop.Registry, agent *agentloop.Agent, dir, ticketID string) (func() int, func(time.Duration)) {
	return (&routeHost{app: a}).mountWorker(reg, agent, dir, ticketID)
}

// namedWorkRoots 转发到 routeHost。
func (a *App) namedWorkRoots(ticketID, home string) []string {
	return (&routeHost{app: a}).namedWorkRoots(ticketID, home)
}

// relayNotice 转发到 routeHost。
func (a *App) relayNotice(sessionID, text string) {
	(&routeHost{app: a}).relayNotice(sessionID, text)
}

// relayReport 转发到 routeHost。
func (a *App) relayReport(ticketID, state, summary, question string) {
	(&routeHost{app: a}).relayReport(ticketID, state, summary, question)
}

// rememberTask 转发到 routeHost。
func (a *App) rememberTask(ticketID, state, text string) {
	(&routeHost{app: a}).rememberTask(ticketID, state, text)
}

// connectorTriggers 转发到 routeHost。
func (a *App) connectorTriggers() []superagent.Trigger {
	return (&routeHost{app: a}).connectorTriggers()
}

// agentIntake 转发到 routeHost。
func (a *App) agentIntake(text string) ([]string, string) {
	return (&routeHost{app: a}).agentIntake(text)
}

// groupsRunning 转发到 routeHost。
func (a *App) groupsRunning() bool {
	return (&routeHost{app: a}).groupsRunning()
}

// agentRoster 转发到 routeHost。
func (a *App) agentRoster(mentioned bool, s buddy.Settings) string {
	return (&routeHost{app: a}).agentRoster(mentioned, s)
}

// routeAgent 转发到 routeHost。
func (a *App) routeAgent(rt *superagent.Runtime, triggerID, agent string) (string, bool, error) {
	return (&routeHost{app: a}).routeAgent(rt, triggerID, agent)
}

// switchAgent 转发到 routeHost。
func (a *App) switchAgent(rt *superagent.Runtime, ticket, trigger, agent, brief string) (string, error) {
	return (&routeHost{app: a}).switchAgent(rt, ticket, trigger, agent, brief)
}

// agentChoices 转发到 routeHost。
func (a *App) agentChoices(mentioned []string, def string, auto bool) string {
	return (&routeHost{app: a}).agentChoices(mentioned, def, auto)
}

// agentRoutable 转发到 routeHost。
func (a *App) agentRoutable(id string) string {
	return (&routeHost{app: a}).agentRoutable(id)
}

// routableIDs 转发到 routeHost。
func (a *App) routableIDs() []string {
	return (&routeHost{app: a}).routableIDs()
}

// splitTool 转发到 routeHost。
func (a *App) splitTool(sessionID, turnID string) runtime.ExtraTool {
	return (&routeHost{app: a}).splitTool(sessionID, turnID)
}

// splitOne 转发到 routeHost。
func (a *App) splitOne(rt *superagent.Runtime, sessionID, turnID string, args map[string]any) (string, error) {
	return (&routeHost{app: a}).splitOne(rt, sessionID, turnID, args)
}

// splitReceipt 转发到 routeHost。
func (a *App) splitReceipt(g superagent.Group, members []superagent.Ticket, mentioned []string) string {
	return (&routeHost{app: a}).splitReceipt(g, members, mentioned)
}

// groupCard 转发到 routeHost。
func (a *App) groupCard(g superagent.Group, members []superagent.Ticket) bool {
	return (&routeHost{app: a}).groupCard(g, members)
}

// agentDisplay 转发到 routeHost。
func (a *App) agentDisplay(agent string) string {
	return (&routeHost{app: a}).agentDisplay(agent)
}

// pickTool 转发到 routeHost。
func (a *App) pickTool() runtime.ExtraTool {
	return (&routeHost{app: a}).pickTool()
}

// recommendTool 转发到 routeHost。
func (a *App) recommendTool() runtime.ExtraTool {
	return (&routeHost{app: a}).recommendTool()
}

// readTool 转发到 routeHost。
func (a *App) readTool() runtime.ExtraTool {
	return (&routeHost{app: a}).readTool()
}

// ticketReadText 转发到 routeHost。
func (a *App) ticketReadText(read superagent.TicketRead) string {
	return (&routeHost{app: a}).ticketReadText(read)
}

// wakeRouter 转发到 routeHost。
func (a *App) wakeRouter(session, group, state, digest string) error {
	return (&routeHost{app: a}).wakeRouter(session, group, state, digest)
}

// wakeSession 转发到 routeHost。
func (a *App) wakeSession(session string) string {
	return (&routeHost{app: a}).wakeSession(session)
}

// pumpWake 转发到 routeHost。
func (a *App) pumpWake(sessionID string) {
	(&routeHost{app: a}).pumpWake(sessionID)
}

// runSlotHeld 转发到 routeHost。
func (a *App) runSlotHeld(sessionID string) bool {
	return (&routeHost{app: a}).runSlotHeld(sessionID)
}

// launchWake 转发到 routeHost。
func (a *App) launchWake(sessionID, group, state, digest string, client runtime.Client) error {
	return (&routeHost{app: a}).launchWake(sessionID, group, state, digest, client)
}
