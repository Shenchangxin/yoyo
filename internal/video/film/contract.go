package film

// Row is one line of the parity contract.
// State is "implemented" or "excluded". Pending is not a finished contract.
type Row struct {
	ID    string
	Name  string
	State string
	Note  string
}

// Contract is the closed list. A capability that changes the film is implemented.
// Anything that must not be copied is excluded, and the list of exclusions does not grow quietly.
func Contract() []Row {
	return []Row{
		{ID: "workflow", Name: "参考片或描述变成可再跑的工程", State: "implemented", Note: "FromReference 与 FromTemplate 会保存词、镜头和待办"},
		{ID: "words", Name: "词轴，改一行只重排这一行", State: "implemented", Note: "词级时间来自标注 JSON 或按字均分；本机有 whisper.cpp 时用它的句子再切开"},
		{ID: "variant", Name: "换宿主、产品、语言、画幅、开头时只重做变的步骤", State: "implemented", Note: "开头只改第一行。价格和收口只改最后的画面。变体另存，原片不动。换语言时用对话连接逐行重写并重排时间"},
		{ID: "structure", Name: "排行榜、播客分屏、街采揭示、评论贴纸、区域字幕、卡点", State: "implemented", Note: "排行榜每一行有一张图的位置，图从图像连接来。播客两边都在，中间交出产品。街采逐条揭示。贴纸和字幕跟当前词走。切点落在词上"},
		{ID: "code", Name: "代码画面，没有生成账号也能出图", State: "implemented", Note: "进程内按词画出画面；本机有 Chrome 或 Edge 时另存一张浏览器截图。有垫乐时混在人声下面"},
		{ID: "models", Name: "图、视频、语音、抠像走已启用的连接", State: "implemented", Note: "拒绝名称或地址里带 hypihub 的连接。抠像预设火山引擎主体分割、remove.bg、Photoroom，并按各自的公开接口发请求。配乐走 MiniMax、Mureka、ElevenLabs、Stability。Lyria 和 Suno 仍是预设，没有统一请求体时该步等待"},
		{ID: "plan", Name: "计划、花费、构建、失败证据", State: "implemented", Note: "构建留下任务；失败的一步不抹掉其余产物"},
		{ID: "tools", Name: "媒体、词级标注、下载、区域、浏览器", State: "implemented", Note: "ffmpeg 沿用现有安装。yt-dlp 可从官方发布下载到本机 bin。没有区域程序时用 ffmpeg cropdetect。词级标注调用 whisperx 的 json 输出。浏览器从 PATH 或 YOYO_BROWSER 读取"},
		{ID: "identity", Name: "其他产品的名称、命令和清单格式", State: "excluded", Note: "翻译树不进安装包"},
		{ID: "gateway", Name: "托管模型网关客户端", State: "excluded", Note: "生成只走 Yoyo 连接"},
		{ID: "tenancy", Name: "多租户托管这套引擎", State: "excluded", Note: "每个用户在自己的机器上跑"},
		{ID: "foreign-ui", Name: "另一套工作室的界面源码", State: "excluded", Note: "预览、任务和产物在创作台"},
		{ID: "runtime", Name: "npm 与 Node 模块运行时", State: "excluded", Note: "工程是 Go 结构"},
		{ID: "vault", Name: "第二套密钥库", State: "excluded", Note: "账号留在连接库"},
		{ID: "samples", Name: "上游例子里的具体片子", State: "excluded", Note: "结构要有，题目用 Yoyo 自己的"},
	}
}

// Pending reports rows that are neither implemented nor excluded.
func Pending() []Row {
	var out []Row
	for _, row := range Contract() {
		if row.State != "implemented" && row.State != "excluded" {
			out = append(out, row)
		}
	}
	return out
}
