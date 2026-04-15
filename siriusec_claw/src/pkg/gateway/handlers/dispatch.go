package handlers

// NewRegistry constructs a handler Registry with all method mappings.
func NewRegistry(ctx *Context) Registry {
	reg := make(Registry)

	// --- Health & Status ---
	reg["health"] = HealthHandler
	reg["status"] = StatusHandler

	// --- Config ---
	reg["config.get"] = ConfigGetHandler
	reg["config.env"] = ConfigEnvHandler
	reg["config.set"] = ConfigSetHandler
	reg["config.apply"] = ConfigSetHandler
	reg["config.patch"] = ConfigPatchHandler
	reg["config.schema"] = ConfigSchemaHandler

	// --- Sessions ---
	reg["sessions.list"] = SessionsListHandler
	reg["sessions.create"] = SessionsCreateHandler
	reg["sessions.ensure"] = SessionsEnsureHandler
	reg["sessions.preview"] = SessionsPreviewHandler
	reg["sessions.patch"] = SessionsPatchHandler
	reg["sessions.reset"] = SessionsResetHandler
	reg["sessions.delete"] = SessionsDeleteHandler
	reg["sessions.compact"] = SessionsCompactHandler
	reg["sessions.usage"] = SessionsUsageHandler
	reg["sessions.usage.timeseries"] = SessionsUsageTimeseriesHandler
	reg["sessions.usage.logs"] = SessionsUsageLogsHandler

	// --- Chat ---
	reg["chat.send"] = ChatSendHandler
	reg["chat.history"] = ChatHistoryHandler
	reg["chat.abort"] = ChatAbortHandler
	reg["chat.inject"] = ChatInjectHandler

	// --- Agents ---
	reg["agents.list"] = AgentsListHandler
	reg["agents.create"] = AgentsCreateHandler
	reg["agents.update"] = AgentsUpdateHandler
	reg["agents.delete"] = AgentsDeleteHandler
	reg["agents.files.list"] = AgentsFilesListHandler
	reg["agents.files.get"] = AgentsFilesGetHandler
	reg["agents.files.set"] = AgentsFilesSetHandler

	// --- Models ---
	reg["models.list"] = ModelsListHandler

	// --- Skills ---
	reg["skills.status"] = SkillsStatusHandler
	reg["skills.getDoc"] = SkillsGetDocHandler
	reg["skills.install"] = SkillsInstallHandler
	reg["skills.delete"] = SkillsDeleteHandler
	reg["skills.update"] = SkillsUpdateHandler
	reg["skills.bins"] = SkillsBinsHandler
	reg["skills.getConfig"] = SkillsGetConfigHandler
	reg["skills.setConfig"] = SkillsSetConfigHandler

	// --- Employees ---
	reg["employees.list"] = EmployeesListHandler
	reg["employees.get"] = EmployeesGetHandler
	reg["employees.create"] = EmployeesCreateHandler
	reg["employees.update"] = EmployeesUpdateHandler
	reg["employees.upload"] = EmployeesUploadHandler
	reg["employees.install"] = EmployeesInstallHandler
	reg["employees.delete"] = EmployeesDeleteHandler

	// --- MCP ---
	reg["mcp.list"] = MCPListHandler
	reg["mcp.get"] = MCPGetHandler
	reg["mcp.add"] = MCPAddHandler
	reg["mcp.update"] = MCPUpdateHandler
	reg["mcp.delete"] = MCPDeleteHandler
	reg["mcp.install"] = MCPInstallHandler

	// --- Cron ---
	reg["cron.list"] = CronListHandler
	reg["cron.status"] = CronStatusHandler
	reg["cron.add"] = CronAddHandler
	reg["cron.remove"] = CronRemoveHandler
	reg["cron.update"] = CronUpdateHandler
	reg["cron.run"] = CronRunHandler
	reg["cron.runs"] = CronRunsHandler

	// --- Channels ---
	reg["channels.status"] = ChannelsStatusHandler
	reg["channels.logout"] = ChannelsLogoutHandler
	reg["channels.start"] = ChannelsStartHandler
	reg["channels.restart"] = ChannelsRestartHandler
	reg["channels.send"] = ChannelsSendHandler
	reg["channels.startAll"] = ChannelsStartAllHandler
	reg["channels.stopAll"] = ChannelsStopAllHandler
	reg["channels.configure"] = ChannelsConfigureHandler

	// --- Approvals ---
	reg["approvals.list"] = ApprovalsListHandler
	reg["approvals.approve"] = ApprovalsApproveHandler
	reg["approvals.deny"] = ApprovalsDenyHandler
	reg["approvals.whitelistSession"] = ApprovalsWhitelistSessionHandler

	// --- Memory ---
	reg["memory.list"] = MemoryListHandler
	reg["memory.get"] = MemoryGetHandler
	reg["memory.add"] = MemoryAddHandler
	reg["memory.update"] = MemoryUpdateHandler
	reg["memory.delete"] = MemoryDeleteHandler
	reg["memory.search"] = MemorySearchHandler

	// --- Context ---
	reg["context.list"] = ContextListHandler
	reg["context.detail"] = ContextDetailHandler

	// --- Stub handlers for unimplemented methods ---
	stubMethods := []string{
		"logs.tail",
		"channels.wework.qr.start", "channels.wework.qr.poll",
		"channels.weixin.qr.start", "channels.weixin.qr.poll",
		"usage.status", "usage.cost",
		"tts.status", "tts.providers", "tts.enable", "tts.disable", "tts.convert", "tts.setProvider",
		"exec.approvals.get", "exec.approvals.set", "exec.approvals.node.get", "exec.approvals.node.set",
		"exec.approval.request", "exec.approval.resolve",
		"wizard.start", "wizard.next", "wizard.cancel", "wizard.status",
		"talk.mode",
		"files.read",
		"update.run", "voicewake.get", "voicewake.set",
		"trace.list", "trace.content",
		"last-heartbeat", "set-heartbeats", "wake",
		"node.pair.request", "node.pair.list", "node.pair.approve", "node.pair.reject", "node.pair.verify",
		"device.pair.list", "device.pair.approve", "device.pair.reject",
		"device.token.rotate", "device.token.revoke",
		"node.rename", "node.list", "node.describe", "node.invoke", "node.invoke.result", "node.event",
		"system-presence", "system-event", "send", "agent",
		"agent.identity.get", "agent.wait", "browser.request",
		"web.login.start", "web.login.wait",
	}
	for _, m := range stubMethods {
		if _, exists := reg[m]; !exists {
			reg[m] = StubHandler
		}
	}

	return reg
}
