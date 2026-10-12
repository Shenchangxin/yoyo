export const LANGS = ["en", "zh", "ja"] as const;
export type Lang = (typeof LANGS)[number];
export type GateKind = "once" | "session" | "always" | "deny";
export type Scene = "ask" | "diff" | "drama" | "trust";

export type Copy = {
  meta: { title: string; description: string };
  nav: {
    workstation: string;
    video: string;
    model: string;
    trust: string;
    doors: string;
    download: string;
    menu: string;
    close: string;
    themeToDark: string;
    themeToLight: string;
  };
  hero: { title: string; dek: string; download: string; github: string; builds: string; chips: string[] };
  stage: { ask: string; diff: string; drama: string; trust: string };
  turn: { title: string; dek: string; steps: { title: string; body: string }[] };
  film: {
    title: string;
    dek: string;
    modes: { title: string; body: string }[];
    phasesTitle: string;
    phases: { title: string; body: string }[];
    apart: string;
  };
  limits: { title: string; dek: string; items: string[] };
  gate: {
    title: string;
    dek: string;
    once: string;
    session: string;
    always: string;
    deny: string;
    onceHint: string;
    sessionHint: string;
    alwaysHint: string;
    denyHint: string;
  };
  model: {
    title: string;
    dek: string;
    solver: string;
    solverBody: string;
    shape: string;
    shapeBody: string;
    evolve: string;
    evolveBody: string;
    weightsStill: string;
    modelName: string;
    lineLive: string;
    lineNext: string;
    l1: string;
    l1Body: string;
    l2: string;
    l2Body: string;
    l3: string;
    l3Body: string;
    l4: string;
    l4Body: string;
    fp: string;
    fpBody: string;
  };
  see: {
    title: string;
    dek: string;
    pin: string;
    pinBody: string;
    diff: string;
    diffBody: string;
    drama: string;
    dramaBody: string;
    folderAlt: string;
  };
  trust: {
    title: string;
    dek: string;
    staging: string;
    stagingHint: string;
    canary: string;
    canaryHint: string;
    active: string;
    activeHint: string;
    checkout: string;
    done: string;
  };
  doors: {
    title: string;
    dek: string;
    desktop: string;
    desktopBody: string;
    browser: string;
    browserBody: string;
    cli: string;
    cliBody: string;
    skills: string;
    skillsBody: string;
    budget: string;
    budgetBody: string;
  };
  get: {
    title: string;
    dek: string;
    premise: string;
    version: string;
    steps: string[];
    thisMachine: string;
    note: string;
  };
  pitch: { title: string; dek: string; cards: { title: string; body: string }[] };
    foot: { product: string; invariants: string; issues: string; license: string; stack: string };
  demo: {
    agent: string;
    review: string;
    video: string;
    harbor: string;
    module: string;
    chats: string;
    threadLive: string;
    threadLiveSub: string;
    threadIdle: string;
    newChat: string;
    hello: string;
    bubble: string;
    read: string;
    patch: string;
    needsOk: string;
    needsApproval: string;
    reading: string;
    patching: string;
    worked: string;
    plan: string;
    action: string;
    path: string;
    allowOnce: string;
    allowSession: string;
    allowAlways: string;
    reject: string;
    apply: string;
    hunks: string;
    composer: string;
    createTab: string;
    canvasTab: string;
    browserBar: string;
    browserBody: string;
    trailRead: string;
    trailPatch: string;
    queue: string;
    queueItem: string;
    statusFolder: string;
    local: string;
    kept: string;
    keptNote: string;
    createBody: string;
    canvasHint: string;
    dramaPrompt: string;
    ep1: string;
    ep2: string;
    ep3: string;
    shotCap: string;
    createHint: string;
  };
};

export const COPY: Record<Lang, Copy> = {
  zh: {
    meta: {
      title: "Yoyo — 本地智能体工作站",
      description: "一台本地的智能体工作站。在你指定的文件夹里改代码、审查改动、拍短剧。写之前先问你。新的干活方式要你亲自换成正在用的那一份。",
    },
    nav: {
      workstation: "工作站",
      video: "视频",
      model: "模型",
      trust: "信任",
      doors: "入口",
      download: "下载",
      menu: "菜单",
      close: "关闭菜单",
      themeToDark: "切换到深色",
      themeToLight: "切换到浅色",
    },
    hero: {
      title: "在你选的文件夹里，\n跑一轮智能体。",
      dek: "它改代码，审查每一处改动，也能把一章文字拍成短剧。写文件和跑命令之前先问你。新的干活方式先接受评测，再由你换成正在使用的那一份。",
      download: "下载",
      github: "GitHub",
      builds: "全部安装包",
      chips: ["Apache-2.0", "Windows · macOS · Linux", "桌面 · 浏览器 · 命令行", "自备模型"],
    },
    stage: {
      ask: "写文件或跑命令之前，问题出现在这一轮里。可以只放行这一次，记住这个对话，记在这台电脑上，或让这一轮停下。",
      diff: "每一处改动单独勾选。原来的文件如果已经不在，会直接告诉你失败，不会悄悄把仓库倒回去。",
      drama: "贴上一章，先确认分成几集，再开拍。创作、短剧和画布是同一套界面，目录和代码仓库分开。",
      trust: "评测通过，只说明这一份可以换。换上去，是你的动作。密钥和更新程序不在它能碰到的范围里。",
    },
    turn: {
      title: "一轮里，事情按这个顺序发生。",
      dek: "先点名文件，再决定能不能写。改动留在你看得见的地方。花费到顶，这一轮就停。",
      steps: [
        { title: "钉住上下文", body: "用 @ 点名一份文件或一个目录。打一个 @，不会把整个文件夹倒进提示词。" },
        { title: "先问再写", body: "写文件或跑命令时，问题出现在这一轮里。只放行这一次，记住这个对话，记在这台电脑上，或拒绝。" },
        { title: "按段留下", body: "右侧是待处理的队列。按片段勾选再应用。没勾的不会写进去。" },
        { title: "到顶就停", body: "预算不是一条提醒。到了就停。Stop 取消这一轮，已经写好的会留下。" },
      ],
    },
    film: {
      title: "视频是另一场对话。\n片子不进代码仓库。",
      dek: "创作、短剧、画布共用一套界面。短剧的家在对话里：先把一章拆成几集，确认之后才拍。要看一集，再把导演台从旁边拉开。",
      modes: [
        { title: "创作", body: "描述一段片子。图像、语音和视频适配器在设置里，画布用的是同一批。密钥留在这台电脑的保管库，不进对话。" },
        { title: "短剧", body: "贴上一章，先确认分成几集，再开拍。对话一直留着。打开一集，旁边才分出导演台。短剧里面没有第二块节点画布，也不是一条八步流水线。画幅锁在这一部上。" },
        { title: "画布", body: "镜头排在同一张空间桌面上。节点是画面。工具停在窗口边缘，画面才看得清。Agent 钉在检查器里，不是另一套聊天。" },
      ],
      phasesTitle: "导演台有四段。每一段只有一个往下走的动作。",
      phases: [
        { title: "剧本", body: "把这一章拆成集。计划没确认，不开拍。" },
        { title: "造型", body: "角色和场景的样子在这里定下来。" },
        { title: "分镜", body: "镜头排成一条胶片。板子上能看，也能改。" },
        { title: "成片", body: "看片和检查在同一张桌子上。胶片条还在。" },
      ],
      apart: "视频会话和写代码的会话不是同一条。Profile 管不到这里。片子有自己的目录，和代码仓库分开。",
    },
    limits: {
      title: "开始之前，这几件事先说清楚。",
      dek: "它是这台电脑上的工作站，不是一个免注册的云聊天。",
      items: [
        "密钥要你自己准备。对话不托管到别人的云上。",
        "不训练专属模型。权重留在你接上的服务里。",
        "技能包默认关闭。你为当前文件夹打开，它才进来。评测不加载它们。",
        "密钥、评测器和更新程序，它改不了。新的干活方式要你亲自换成正在用的那一份。",
        "Windows 和 Linux 安装包未签名。macOS 是 ad-hoc 签名，未经公证。",
      ],
    },
    gate: {
      title: "写下去之前，\n它先在对话里问你。",
      dek: "写文件或跑命令，问题就出现在这一轮里。你可以只放行这一次，记住整个对话，记在这台电脑上，或让这一轮停下。桌面应用默认是问你，不是自己做主。",
      once: "一次",
      session: "本对话",
      always: "总是",
      deny: "拒绝",
      onceHint: "只放行这一次",
      sessionHint: "这个对话里记住",
      alwaysHint: "记在这台电脑上",
      denyHint: "这一轮停下",
    },
    model: {
      title: "模型你自己带。\n变的是它怎么干活。",
      dek: "Yoyo 不训练一个专属模型，也不把提示词越写越长叫做变聪明。你接上 OpenAI 兼容的接口，权重留在原来的服务里。它能打磨的是提示词、工作手册和技能。打磨结果先停在一旁。你确认之后，才换成正在用的版本。",
      solver: "你带来的模型",
      solverBody: "出厂默认是 gpt-4.1-mini，也可以换成任何兼容接口。同一个模型可以准备两套干活方式，互不影响。",
      shape: "上下文有边界",
      shapeBody: "用 @ 钉住一份文件或一个目录。打一个 @ 不会把整个仓库塞进提示词。旧的工具输出会被裁短，把位置留给这一轮。",
      evolve: "先证明，再换上",
      evolveBody: "你点的赞只进草稿。自动打磨写出的是待晋升版本。老题目和新题目都不能变差，安全检查失败就停。正在使用的版本，只有你能切换。",
      weightsStill: "权重不动",
      modelName: "gpt-4.1-mini",
      lineLive: "错误要返回，不要吞掉。",
      lineNext: "同时写上调用方。",
      l1: "L1",
      l1Body: "提示词、playbook、技能。Evolve 可以改，必须过 Harbor。",
      l2: "L2",
      l2Body: "签名的 WASM。未签名进不了 active。没有文件系统。",
      l3: "L3",
      l3Body: "循环与策略拓扑。只有你能改。",
      l4: "L4",
      l4Body: "内核、评测器、密钥、更新器。智能体不行，evolve 也不行。",
      fp: "跟着模型走",
      fpBody: "换了模型，就换一套对应的干活方式。为这个模型调好的手册，不会硬套到另一个模型上。",
    },
    see: {
      title: "改动留在\n你看得见的地方。",
      dek: "代码按一小段一小段勾选，再决定留下哪些。短剧如果要拍，走自己的目录，不和写代码的工作区混在一起。",
      pin: "钉一份文件",
      pinBody: "工作从一份你点名的文件开始，而不是把整个文件夹倒进对话。",
      diff: "勾选再留下",
      diffBody: "每一处改动单独决定。原来的文件如果已经不在，会直接告诉你失败，不会悄悄把仓库倒回去。",
      drama: "短剧单独拍",
      dramaBody: "贴上一章，先确认分成几集，再开拍。创作、短剧和画布是同一套界面，目录和代码仓库分开。",
      folderAlt: "深色橡木桌上，一本合上的牛皮纸文件夹和一支铜笔。",
    },
    trust: {
      title: "它可以打磨自己。\n不能自己上岗。",
      dek: "草稿、待晋升、正在用，是三份不同的干活方式。评测通过，只说明这一份可以换。换上去，是你的动作。密钥和更新程序不在它能碰到的范围里。",
      staging: "草稿",
      stagingHint: "点赞只写到这里",
      canary: "待晋升",
      canaryHint: "评测过了，还没轮到你",
      active: "正在用",
      activeHint: "只有你能拨这根指针",
      checkout: "晋升",
      done: "指针是你拨的。不是它。",
    },
    doors: {
      title: "同一套能力。\n三扇门都能进。",
      dek: "桌面有窗口和托盘。浏览器开在你自己的电脑上。命令行走的是同一份本地数据。页面里没有第二套智能体。",
      desktop: "桌面",
      desktopBody: "原生窗口、菜单、托盘和通知。安装包就是这个应用。",
      browser: "浏览器",
      browserBody: "对话和评测都在这台电脑上。没有托盘，也没有云端托管。",
      cli: "命令行",
      cliBody: "跑一轮、做评测、打磨干活方式，或安装一个技能包。",
      skills: "技能默认关闭",
      skillsBody: "技能包先放在这台电脑上，不会自动参加每一轮。你为当前文件夹打开一个，它才进来。评测时不读它们。",
      budget: "花费到顶就停",
      budgetBody: "上限到了，这一轮就停，不是弹一条提醒。密钥留在本地。它改不了这些设置。",
    },
    get: {
      title: "装到这台电脑上，\n就可以开始。",
      dek: "不用注册，也不把对话托管到别人的云上。选一个文件夹，填入你自己的接口密钥，然后让它先问再写。",
      premise: "密钥要你自己准备。如果想要一个不用密钥、完全托管的聊天产品，这不是那一页。",
      version: "版本",
      steps: [
        "按系统安装桌面应用。Windows 不用管理员。",
        "设置里选工作文件夹。它只在这个文件夹里动手。",
        "设置里粘贴 OpenAI 兼容的 Key。",
        "输入任务。用 @ 钉上下文。写入先问你。",
      ],
      thisMachine: "这台电脑",
      note: "Windows / Linux 安装包未签名。SmartScreen 可能拦一下：更多信息 → 仍要运行。macOS 是 ad-hoc 签名，未经公证。源码 Apache-2.0，不舒服就自己编译。",
    },
    pitch: {
      title: "它实际会做的事",
      dek: "不是云上的另一个你。是这台电脑上的一轮工作：你给文件夹，你给模型，你决定什么能写进去。",
      cards: [
        { title: "改你文件夹里的代码", body: "选一个目录，输入任务。它边读边改。只要方案、先不落笔，就打开 Plan。停掉这一轮时，已经写好的会留下。" },
        { title: "先问，再写，再执行", body: "写文件或跑命令时，问题出现在对话里。只放行这一次，记住这次对话，记在这台电脑上，或者拒绝。" },
        { title: "每一处改动都看得见", body: "右侧是待处理的队列。按片段勾选再应用。原来的文件如果不在了，会直接失败，不会悄悄把仓库倒回去。" },
        { title: "短剧不和代码混放", body: "贴上一章，先确认分成几集，再开拍。创作、短剧、画布是同一套界面。片子有自己的目录。" },
        { title: "技能先关着", body: "别人写的技能包不会自动进入每一轮。你在这个工作区打开，它才进来。评测的时候一律不加载。" },
        { title: "钱花到上限就停", body: "预算不是一条提醒。到了就停。密钥留在这台电脑上。它改不了自己的设置，也换不了正在运行的程序。" },
      ],
    },
    foot: { product: "产品", invariants: "不变量", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
    demo: {
      agent: "Agent",
      review: "审查",
      video: "视频",
      harbor: "Harbor",
      module: "模块",
      chats: "会话",
      threadLive: "Parser fix",
      threadLiveSub: "apply_patch · 等你",
      threadIdle: "Harbor smoke",
      newChat: "新会话",
      hello: "下午好。",
      bubble: "把 parser 里那处错误返回补上",
      read: "已读取",
      patch: "apply_patch · +24 −6",
      needsOk: "等你点头",
      needsApproval: "需要审批",
      reading: "读取",
      patching: "应用补丁",
      worked: "工作了 4s",
      plan: "Plan",
      action: "write_file",
      path: "internal/runtime/loop.go",
      allowOnce: "允许一次",
      allowSession: "本对话",
      allowAlways: "总是允许",
      reject: "拒绝",
      apply: "应用",
      hunks: "处改动",
      composer: "修 loop.go 里的 parser",
      createTab: "创作",
      canvasTab: "画布",
      browserBar: "localhost · preview",
      browserBody: "解析器现在会把错误返回去。",
      trailRead: "Read",
      trailPatch: "apply_patch",
      queue: "队列",
      queueItem: "loop.go · +24 −6",
      statusFolder: "~/work/parser",
      local: "本地",
      kept: "已留下",
      keptNote: "勾上的留在工作区里。没勾的不会写进去。",
      createBody: "描述一段片子。图像、语音和视频适配器是你自己的。",
      canvasHint: "镜头排在同一张桌面上。目录和代码仓库分开。",
      dramaPrompt: "把这一章拍成三集。先确认分集，再拍。",
      ep1: "抵达",
      ep2: "灯",
      ep3: "离开",
      shotCap: "02 · 灯",
      createHint: "橡木桌上的一盏灯，黄昏。",
    },
  },
  en: {
    meta: {
      title: "Yoyo — a local agent workstation",
      description: "A local workstation. You bring the model; the weights stay put. Writes ask first. A new way of working is tested first. You switch it on.",
    },
    nav: {
      workstation: "Workstation",
      video: "Video",
      model: "Model",
      trust: "Trust",
      doors: "Doors",
      download: "Download",
      menu: "Menu",
      close: "Close menu",
      themeToDark: "Switch to dark",
      themeToLight: "Switch to light",
    },
    hero: {
      title: "Run an agent\nin a folder you choose.",
      dek: "It edits code, lets you keep each change, and can turn a chapter into a short film. It asks before it writes or runs a command. A new way of working is tested first. You are the one who switches it on.",
      download: "Download",
      github: "GitHub",
      builds: "All builds",
      chips: ["Apache-2.0", "Windows · macOS · Linux", "Desktop · browser · CLI", "Bring your own model"],
    },
    stage: {
      ask: "A write or a command asks in the turn. Allow it once, for this chat, on this machine, or stop the turn.",
      diff: "Keep each change on its own. If the file is already gone, restore fails out loud. Nothing is quietly rolled back.",
      drama: "Paste a chapter, confirm the episodes, then shoot. Create, drama, and canvas share one shell. The footage has its own folder.",
      trust: "A passing eval only means this one can be switched on. Switching it on is your action. Keys and the updater stay out of reach.",
    },
    turn: {
      title: "A turn happens in this order.",
      dek: "Name a file first. Then decide what may be written. Changes stay where you can see them. When the budget is spent, the turn stops.",
      steps: [
        { title: "Pin context", body: "Type @ to name one file or one folder. One @ does not pour the whole tree into the prompt." },
        { title: "Ask, then write", body: "A write or a command asks in the turn. Allow it once, for this chat, on this machine, or turn it down." },
        { title: "Keep by hunk", body: "The queue sits beside the thread. Tick the pieces you want. What you leave unticked is not written." },
        { title: "Stop at the cap", body: "The budget is not a warning. When it hits, the turn stops. Stop cancels the turn. What already landed stays." },
      ],
    },
    film: {
      title: "Video is another conversation.\nFootage stays out of the repo.",
      dek: "Create, drama, and canvas share one shell. Drama lives in the chat: split a chapter into episodes, confirm the plan, then shoot. Open an episode, and the director desk splits out beside it.",
      modes: [
        { title: "Create", body: "Describe a piece. Image, voice, and video adapters live in Settings. Canvas uses the same ones. Keys stay in the vault on this machine." },
        { title: "Drama", body: "Paste a chapter, confirm the episodes, then shoot. The chat stays mounted. Opening an episode splits a director desk beside it. There is no second node canvas inside drama, and it is not an eight-stage factory. Aspect is locked on the series." },
        { title: "Canvas", body: "Shots sit on one spatial desk. A node is a picture. Tools stay at the window edge so the frame stays clear. The agent pins to the inspector. It is not a second chat." },
      ],
      phasesTitle: "The director desk has four phases. Each one has a single next step.",
      phases: [
        { title: "Script", body: "Split the chapter into episodes. Nothing shoots before the plan is confirmed." },
        { title: "Looks", body: "Characters and places take their shape here." },
        { title: "Board", body: "Shots line up as a film strip. You can watch the board and change it." },
        { title: "Cut", body: "The viewer and the inspector share the desk. The film strip stays." },
      ],
      apart: "Video chats are not Agent chats. A profile does not apply on this channel. Footage has its own folder, apart from the code workspace.",
    },
    limits: {
      title: "Before you start, five things are already true.",
      dek: "This is a workstation on your machine. It is not a hosted chat you can use without a key.",
      items: [
        "You bring the key. The conversation is not hosted in someone else's cloud.",
        "Yoyo does not train a model of its own. The weights stay with the service you connect.",
        "Skill packs start off. You enable one for this folder. Evaluation never loads them.",
        "It cannot change keys, the evaluator, or the updater. You switch on a new way of working yourself.",
        "Windows and Linux packages are unsigned. macOS is ad-hoc signed, not notarized.",
      ],
    },
    gate: {
      title: "The question lands in the turn.",
      dek: "A write or a command asks in the thread: allow once, this chat, always, or reject. The desktop default is to ask.",
      once: "Once",
      session: "This chat",
      always: "Always",
      deny: "Deny",
      onceHint: "This write only",
      sessionHint: "Remembered in this chat",
      alwaysHint: "Kept on this machine",
      denyHint: "The turn stops",
    },
    model: {
      title: "You bring the model.\nWhat changes is how it works.",
      dek: "Yoyo does not train a model of its own, and a longer prompt is not getting smarter. You connect an OpenAI-compatible API. The weights stay with that service. It may refine the prompt, the working notes, and skills. The result waits aside. You switch it on.",
      solver: "Solver",
      solverBody: "In a turn the model reads context, calls tools, and writes prose. Stop cancels the turn. What already landed stays. Plan only proposes. Writes and shell stay off.",
      shape: "Shape",
      shapeBody: "Old tool results get shortened. Context has a budget. @file pins one file. It does not dump the tree into the prompt. That shape is the harness, not a personality.",
      evolve: "Evolve",
      evolveBody: "Thumbs write staging. Evolve writes canary. Held-in and held-out must not drop. A safety failure blocks. You check out, and only then the pointer is active.",
      weightsStill: "Weights stay put",
      modelName: "gpt-4.1-mini",
      lineLive: "Return the error. Do not swallow it.",
      lineNext: "Name the caller as well.",
      l1: "L1",
      l1Body: "Prompt, playbook, skills. Evolve may change them. Harbor must pass.",
      l2: "L2",
      l2Body: "Signed WASM. Unsigned never reaches active. No filesystem.",
      l3: "L3",
      l3Body: "Loop and policy. Only you change these.",
      l4: "L4",
      l4Body: "Kernel, evaluator, keys, updater. Not the agent. Not evolve.",
      fp: "Model fingerprint",
      fpBody: "refs/models/<fingerprint>/active. Same weights. The harness is per model.",
    },
    see: {
      title: "After you ask, the change stays where you can see it.",
      dek: "Pin a file. Tick the hunks you want. Video, if you shoot it, lives in its own folder.",
      pin: "Pin a file",
      pinBody: "Typing @ does not dump the tree into the prompt.",
      diff: "Tick, then keep",
      diffBody: "Accept by hunk. Restore fails out loud if the files are gone.",
      drama: "Video is a conversation",
      dramaBody: "Paste a chapter. Confirm the episodes, then shoot. Not mixed with the code workspace.",
      folderAlt: "A closed kraft folder and a brass pen on a dark oak desk.",
    },
    trust: {
      title: "It can get smarter. It cannot promote itself.",
      dek: "Thumbs write a draft. After eval passes, you check it out. The pointer moves a harness, not the model weights. Keys and the updater stay off limits.",
      staging: "Draft",
      stagingHint: "Thumbs write here only",
      canary: "Waiting",
      canaryHint: "Eval passed. You have not checked it out.",
      active: "Live",
      activeHint: "Only you move this pointer",
      checkout: "Promote",
      done: "You moved it. The agent did not.",
    },
    doors: {
      title: "One loop. Three doors.",
      dek: "Desktop has a window and a tray. The browser runs on localhost. The CLI shares the same home.",
      desktop: "Desktop",
      desktopBody: "Native window, menus, tray, notifications. The installer is this app.",
      browser: "Browser",
      browserBody: "Chat and eval on localhost. No tray.",
      cli: "CLI",
      cliBody: "Run a turn, eval, evolve, or install a pack.",
      skills: "Skills stay off",
      skillsBody: "Packs live in ~/.yoyo/packs/. You enable one for the workspace. Eval never loads them.",
      budget: "Budget is a hard stop",
      budgetBody: "When the cap hits, the loop stops. Keys stay in the local vault.",
    },
    get: {
      title: "No account. No cloud. Install and run.",
      dek: "Pick a folder. Paste an OpenAI-compatible key. Type a task, pin context with @, approve writes.",
      premise: "You need your own key. If you want a hosted chat with no keys, this page is not for that.",
      version: "Version",
      steps: [
        "Install the desktop app for your OS.",
        "Settings: one folder the agent may touch.",
        "Settings: paste an OpenAI-compatible key.",
        "Type a task. Pin context with @. Writes ask first.",
      ],
      thisMachine: "This machine",
      note: "Windows and Linux packages are unsigned. SmartScreen may ask for More info, then Run anyway. macOS is ad-hoc signed, not notarized. Source is Apache-2.0.",
    },
    pitch: {
      title: "What it actually does",
      dek: "Not another you in the cloud. A turn on this machine: you give the folder, you give the model, you decide what may be written.",
      cards: [
        { title: "Edit the folder you chose", body: "Pick a directory and type a task. It reads and edits as it goes. Plan proposes only. Stop keeps what already landed." },
        { title: "Ask, then write, then run", body: "A write or a command asks in the thread. Allow it once, for this chat, on this machine, or turn it down." },
        { title: "Every change stays visible", body: "The queue sits beside the thread. Keep changes piece by piece. If the file is gone, restore fails out loud." },
        { title: "Film stays out of the repo", body: "Paste a chapter, confirm the episodes, then shoot. Create, drama, and canvas share one shell. The footage has its own folder." },
        { title: "Skills start off", body: "Packs do not enter every turn. You enable one for this folder. Evaluation never loads them." },
        { title: "The budget is a stop", body: "When the cap hits, the turn stops. Keys stay on this machine. It cannot edit those settings or replace the running app." },
      ],
    },
    foot: { product: "Product", invariants: "Invariants", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
    demo: {
      agent: "Agent",
      review: "Review",
      video: "Video",
      harbor: "Harbor",
      module: "Module",
      chats: "Chats",
      threadLive: "Parser fix",
      threadLiveSub: "apply_patch · waiting",
      threadIdle: "Harbor smoke",
      newChat: "New chat",
      hello: "Good afternoon.",
      bubble: "Return the parse error instead of swallowing it",
      read: "Read",
      patch: "apply_patch · +24 −6",
      needsOk: "Needs your OK",
      needsApproval: "Needs approval",
      reading: "Reading",
      patching: "Patching",
      worked: "Worked for 4s",
      plan: "Plan",
      action: "write_file",
      path: "internal/runtime/loop.go",
      allowOnce: "Allow once",
      allowSession: "This chat",
      allowAlways: "Always allow",
      reject: "Reject",
      apply: "Apply",
      hunks: "hunks",
      composer: "Fix the parser in @file:loop.go",
      createTab: "Create",
      canvasTab: "Canvas",
      browserBar: "localhost · preview",
      browserBody: "The parser now returns the error.",
      trailRead: "Read",
      trailPatch: "apply_patch",
      queue: "Queue",
      queueItem: "loop.go · +24 −6",
      statusFolder: "~/work/parser",
      local: "Local",
      kept: "Kept",
      keptNote: "Ticked hunks stay in the folder. The rest are not written.",
      createBody: "Describe a piece. Image, voice, and video adapters are yours.",
      canvasHint: "Shots sit on one desk. The folder is not the code workspace.",
      dramaPrompt: "Shoot this chapter as three episodes. Confirm the plan, then shoot.",
      ep1: "Arrival",
      ep2: "The lamp",
      ep3: "Leaving",
      shotCap: "02 · The lamp",
      createHint: "A lamp on an oak desk, dusk.",
    },
  },
  ja: {
    meta: {
      title: "Yoyo — ローカルの作業台",
      description: "ローカルの作業台。モデルは自分で持つ。重みは動かない。書き込みの前に尋ねる。証明したあと、切り替えるのはあなた。",
    },
    nav: {
      workstation: "作業台",
      video: "ビデオ",
      model: "モデル",
      trust: "信頼",
      doors: "入口",
      download: "入手",
      menu: "メニュー",
      close: "メニューを閉じる",
      themeToDark: "ダークに切り替え",
      themeToLight: "ライトに切り替え",
    },
    hero: {
      title: "選んだフォルダで、\nエージェントを走らせる。",
      dek: "コードを直し、変更は断片ごとに残し、章を短編にもできる。書く前、コマンドの前に尋ねる。新しい働き方はまず評価され、切り替えるのはあなた。",
      download: "入手",
      github: "GitHub",
      builds: "すべてのビルド",
      chips: ["Apache-2.0", "Windows · macOS · Linux", "デスクトップ · ブラウザ · CLI", "モデルは自分で"],
    },
    stage: {
      ask: "書き込みもコマンドも、このターンの中で尋ねる。一度、この会話、このマシン、またはここで止める。",
      diff: "変更は断片ごとに残す。ファイルがもう無いときは、静かに戻さず、失敗してはっきり言う。",
      drama: "章を貼り、話数を確認してから撮る。作成、ドラマ、キャンバスは一つの殻。映像はコードの作業場と分ける。",
      trust: "評価が通ったのは、この一份が切り替えられるという意味だけ。切り替えるのはあなた。鍵とアップデータには触れない。",
    },
    turn: {
      title: "一つのターンは、この順で進む。",
      dek: "まずファイルを名指しする。それから書いてよいかを決める。変更は見える場所に残る。予算の上限で、ターンは止まる。",
      steps: [
        { title: "文脈をピンする", body: "@ で一つのファイルかフォルダを名指しする。@ を一つ打っても、木全体はプロンプトに入らない。" },
        { title: "聞いてから書く", body: "書き込みもコマンドも、このターンで尋ねる。一度、この会話、このマシン、または拒否。" },
        { title: "断片ごとに残す", body: "キューは会話のそば。残す断片だけを選ぶ。選ばなかったものは書き込まれない。" },
        { title: "上限で止まる", body: "予算は注意書きではない。達したら止まる。Stop はそのターンを止める。すでに書いたものは残る。" },
      ],
    },
    film: {
      title: "ビデオは別の会話。\n映像はリポジトリの外。",
      dek: "作成、ドラマ、キャンバスは一つの殻。ドラマの家は会話の中。章を話に分け、確認してから撮る。一話を開くと、隣に演出机が分かれる。",
      modes: [
        { title: "作成", body: "一本を説明する。画像、音声、動画のアダプタは設定にあり、キャンバスと同じものを使う。鍵はこのマシンの保管庫に残る。" },
        { title: "ドラマ", body: "章を貼り、話数を確認してから撮る。会話はそのまま残る。一話を開くと、隣に演出机が分かれる。ドラマの中に二つ目のノードキャンバスはなく、八段階の工場でもない。画角はそのシリーズにロックされる。" },
        { title: "キャンバス", body: "ショットは一つの空間の机に並ぶ。ノードは絵だ。道具は窓の縁に置き、画面を空けておく。Agent はインスペクタにピンされ、別のチャットではない。" },
      ],
      phasesTitle: "演出机は四つの段階。各段階の次の動作は一つ。",
      phases: [
        { title: "脚本", body: "章を話に分ける。計画を確認するまで撮らない。" },
        { title: "造型", body: "人物と場所の見た目をここで決める。" },
        { title: "絵コンテ", body: "ショットはフィルムの帯になる。板の上で見て、直せる。" },
        { title: "完成", body: "試写と検査は同じ机。フィルムの帯は残る。" },
      ],
      apart: "ビデオの会話は Agent の会話ではない。プロフィールはこの通道に届かない。映像はコードの作業場とは別のフォルダ。",
    },
    limits: {
      title: "始める前に、決まっていること。",
      dek: "これはこのマシンの作業台です。鍵なしで使えるホストされたチャットではありません。",
      items: [
        "鍵は自分で持つ。会話は他人のクラウドに預けない。",
        "自前のモデルは訓練しない。重みは接続したサービスに残る。",
        "スキルパックは最初オフ。このフォルダで有効化してから入る。評価は読み込まない。",
        "鍵、評価器、アップデータは変更できない。新しい働き方を切り替えるのはあなた。",
        "Windows と Linux は未署名。macOS はアドホック署名で、公証はない。",
      ],
    },
    gate: {
      title: "問いはターンの中に出る。",
      dek: "書き込みもコマンドも、スレッドで尋ねる。一度許可、この会話、常に許可、または拒否。デスクトップの既定は尋ねること。",
      once: "一度",
      session: "この会話",
      always: "常に",
      deny: "拒否",
      onceHint: "この書き込みだけ",
      sessionHint: "この会話で覚える",
      alwaysHint: "このマシンに記憶",
      denyHint: "このターンは止まる",
    },
    model: {
      title: "モデルは自分で持つ。\n変わるのは働き方。",
      dek: "Yoyo は自前モデルを訓練しない。プロンプトを長くしても、賢くなったことにはならない。OpenAI 互換の API につなぐ。重みは元のサービスに残る。磨けるのはプロンプト、手順、スキル。結果は脇に置かれ、切り替えるのはあなた。",
      solver: "ソルバー",
      solverBody: "ターンの中で、モデルは文脈を読み、ツールを呼び、文章を書く。Stop はそのターンを止める。すでに書いたものは残る。Plan は提案だけ。書き込みと shell は閉じたまま。",
      shape: "形",
      shapeBody: "古いツール結果は短く切られる。文脈には予算がある。@file は一つのファイルをピンする。木全体はプロンプトに入らない。これはハーネスの形で、個性ではない。",
      evolve: "進化",
      evolveBody: "評価は staging へ。Evolve は canary へ。保持セットもホールドアウトも下がってはならない。安全失敗は常に止める。checkout したあとだけ、ポインタは active。",
      weightsStill: "重みは動かない",
      modelName: "gpt-4.1-mini",
      lineLive: "エラーは返す。握りつぶさない。",
      lineNext: "呼び出し元の名前も書く。",
      l1: "L1",
      l1Body: "プロンプト、playbook、スキル。Evolve は変えられる。Harbor を通す。",
      l2: "L2",
      l2Body: "署名済み WASM。未署名は active に入らない。ファイルシステムはない。",
      l3: "L3",
      l3Body: "ループと方針。変えられるのは人だけ。",
      l4: "L4",
      l4Body: "カーネル、評価器、鍵、アップデータ。エージェントも evolve も不可。",
      fp: "モデル指紋",
      fpBody: "refs/models/<指紋>/active。同じ重みでも、ハーネスはモデルごと。",
    },
    see: {
      title: "頼んだあと、変更は見える場所に残る。",
      dek: "ファイルをピンする。残すハンクを選ぶ。動画は専用フォルダ。",
      pin: "ファイルをピン",
      pinBody: "@ を打っただけでリポジトリ全体は入らない。",
      diff: "選んで残す",
      diffBody: "ハンク単位で適用。見つからなければ失敗する。",
      drama: "動画は会話",
      dramaBody: "章を貼る。話数を確認してから撮る。コードの作業場とは混ぜない。",
      folderAlt: "暗いオークの机の上の、閉じたクラフト紙のフォルダと真鍮のペン。",
    },
    trust: {
      title: "賢くなってよい。自分では昇格できない。",
      dek: "評価は下書きへ。評価が通ったら、あなたが checkout する。動くのはハーネスで、モデルの重みではない。鍵とアップデータには触れない。",
      staging: "下書き",
      stagingHint: "評価はここだけ",
      canary: "待機",
      canaryHint: "評価は通った。まだ checkout していない",
      active: "使用中",
      activeHint: "動かすのは人だけ",
      checkout: "昇格",
      done: "動かしたのはあなた。エージェントではない。",
    },
    doors: {
      title: "ひとつのループ。三つの入口。",
      dek: "デスクトップは窓とトレイ。ブラウザは本機。CLI は同じ home。",
      desktop: "デスクトップ",
      desktopBody: "ネイティブ窓、メニュー、トレイ、通知。インストーラはこれ。",
      browser: "ブラウザ",
      browserBody: "会話と評価は本機。トレイはない。",
      cli: "CLI",
      cliBody: "ターン、評価、進化、パック。",
      skills: "スキルは既定オフ",
      skillsBody: "パックは ~/.yoyo/packs/。ワークスペースで有効化。評価は読まない。",
      budget: "予算はハードストップ",
      budgetBody: "上限に達したら止まる。鍵はローカル保管庫。",
    },
    get: {
      title: "アカウントもクラウドもない。入れて走らせる。",
      dek: "フォルダを選ぶ。OpenAI 互換キーを貼る。タスクを打ち、@ でピンし、書き込みは承認。",
      premise: "キーは自分で持つ。ホストされたチャットが欲しいなら、このページは対象外。",
      version: "バージョン",
      steps: [
        "OS のデスクトップアプリを入れる。",
        "設定で作業フォルダを選ぶ。そこだけ触る。",
        "設定に OpenAI 互換キーを貼る。",
        "タスクを打つ。@ でピン。書き込みは承認。",
      ],
      thisMachine: "このマシン",
      note: "Windows と Linux は未署名。SmartScreen は「詳細情報 → 実行」。macOS はアドホック署名で公証なし。ソースは Apache-2.0。",
    },
    pitch: {
      title: "実際にすること",
      dek: "クラウド上のもう一人ではない。このマシンでの一つのターン。フォルダもモデルも、書いてよいかも、決めるのはあなた。",
      cards: [
        { title: "選んだフォルダを直す", body: "ディレクトリを選び、タスクを打つ。読みながら直す。Plan は提案だけ。止めても、すでに書いたものは残る。" },
        { title: "聞いてから書き、実行する", body: "書き込みもコマンドも、スレッドで尋ねる。一度、この会話、このマシン、または拒否。" },
        { title: "変更は見えるところに残る", body: "キューは会話のそば。断片ごとに残す。ファイルがなければ、復元は失敗してはっきり言う。" },
        { title: "映像はリポジトリの外", body: "章を貼り、話数を確認してから撮る。作成、ドラマ、キャンバスは一つの殻。映像は別フォルダ。" },
        { title: "スキルは最初オフ", body: "パックは毎ターン入らない。このフォルダで有効化してから。評価は読まない。" },
        { title: "予算は停止", body: "上限に達したらターンは止まる。鍵はこのマシン。設定も実行中のアプリも、自分では替えられない。" },
      ],
    },
    foot: { product: "製品", invariants: "不変条件", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
    demo: {
      agent: "Agent",
      review: "審査",
      video: "ビデオ",
      harbor: "Harbor",
      module: "モジュール",
      chats: "チャット",
      threadLive: "Parser fix",
      threadLiveSub: "apply_patch · 待ち",
      threadIdle: "Harbor smoke",
      newChat: "新しい会話",
      hello: "こんにちは。",
      bubble: "parser のエラーを握りつぶさず返す",
      read: "読んだ",
      patch: "apply_patch · +24 −6",
      needsOk: "確認が必要",
      needsApproval: "承認が必要",
      reading: "読み取り",
      patching: "パッチ",
      worked: "4s 作業した",
      plan: "Plan",
      action: "write_file",
      path: "internal/runtime/loop.go",
      allowOnce: "一度許可",
      allowSession: "この会話",
      allowAlways: "常に許可",
      reject: "拒否",
      apply: "適用",
      hunks: "ハンク",
      composer: "@file:loop.go の parser を直す",
      createTab: "作成",
      canvasTab: "キャンバス",
      browserBar: "localhost · preview",
      browserBody: "パーサーはエラーを返すようになった。",
      trailRead: "Read",
      trailPatch: "apply_patch",
      queue: "キュー",
      queueItem: "loop.go · +24 −6",
      statusFolder: "~/work/parser",
      local: "ローカル",
      kept: "残した",
      keptNote: "選んだハンクはフォルダに残る。選ばなかったものは書き込まれない。",
      createBody: "一本を説明する。画像、音声、動画のアダプタは自分のもの。",
      canvasHint: "ショットは一つの机に並ぶ。フォルダはコードの作業場とは別。",
      dramaPrompt: "この章を三話で撮る。計画を確認してから撮る。",
      ep1: "到着",
      ep2: "ランプ",
      ep3: "出発",
      shotCap: "02 · ランプ",
      createHint: "樫の机のランプ、黄昏。",
    },
  },
};

export function detectLang(): Lang {
  try {
    const q = new URLSearchParams(window.location.search).get("lang");
    if (q && (LANGS as readonly string[]).includes(q)) return q as Lang;
    const saved = localStorage.getItem("yoyo-site-lang");
    if (saved && (LANGS as readonly string[]).includes(saved)) return saved as Lang;
  } catch {
    /* ignore */
  }
  const nav = (typeof navigator !== "undefined" ? navigator.language : "en").toLowerCase();
  if (nav.startsWith("zh")) return "zh";
  if (nav.startsWith("ja")) return "ja";
  return "en";
}

export function htmlLang(lang: Lang): string {
  return lang === "zh" ? "zh-Hans" : lang;
}
