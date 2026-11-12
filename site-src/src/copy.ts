/**
 * 拥有：产品站的英文、简体中文和日文文案。
 * 不拥有：页面怎么排。下面的对象是数据，不为每一句文案再写注释。
 */

export const LANGS = ["en", "zh", "ja"] as const;
export type Lang = (typeof LANGS)[number];

export const ROOMS = ["today", "tasks", "skills", "video", "study", "idol"] as const;
export type Room = (typeof ROOMS)[number];

/** 截图文件放在 public/shots，三种语言共用这一套文件名。原图在仓库根目录的 assets 里。 */
export const ROOM_SHOT: Record<Room, string> = {
  today: "today.webp",
  tasks: "tasks.webp",
  skills: "skills.webp",
  video: "video.webp",
  study: "study.webp",
  idol: "idol.webp",
};

/** 截图的像素尺寸。写在 img 上，图片还没到时版面先留好位置。 */
export const SHOT_SIZE = { width: 1740, height: 1060 } as const;

export type Copy = {
  meta: { title: string; description: string };
  nav: {
    home: string;
    rooms: string;
    why: string;
    doors: string;
    download: string;
    menu: string;
    close: string;
    themeToDark: string;
    themeToLight: string;
  };
  hero: { title: string; dek: string; download: string; github: string; builds: string; chips: string[]; shotAlt: string };
  rooms: {
    title: string;
    dek: string;
    label: string;
    items: { id: Room; name: string; line: string; points: string[]; alt: string }[];
  };
  why: { title: string; dek: string; items: { tag: string; title: string; body: string }[] };
  doors: { title: string; dek: string; items: { name: string; body: string; hint: string }[] };
  get: {
    title: string;
    dek: string;
    premise: string;
    version: string;
    steps: string[];
    thisMachine: string;
    phone: string;
    note: string;
  };
  foot: { product: string; issues: string; license: string; stack: string };
};

export const COPY: Record<Lang, Copy> = {
  zh: {
    meta: {
      title: "Yoyo — 开源的桌面超级智能体",
      description:
        "开源的桌面超级智能体。今天、任务、技能、视频、学习和偶像在一个窗口里，还能把活派给 Claude Code、Codex 这些本机 agent。模型你自己接，对话留在这台电脑上。",
    },
    nav: {
      home: "今天",
      rooms: "模块",
      why: "能力",
      doors: "入口",
      download: "下载",
      menu: "菜单",
      close: "关闭菜单",
      themeToDark: "切换到深色",
      themeToLight: "切换到浅色",
    },
    hero: {
      title: "你的这一天，\n在这一个窗口里。",
      dek: "Yoyo 是开源的桌面超级智能体。它盯着今天要你看的事，把活派给自己和 Claude Code、Codex 这些本机 agent，也做视频、陪你学习、在桌面上陪你说话。",
      download: "下载",
      github: "GitHub",
      builds: "全部安装包",
      chips: ["Windows · macOS · Linux · Android", "模型你自己接", "Apache-2.0", "Go · Wails 3"],
      shotAlt: "Yoyo 的今天。左边是晚间简报，右边是这一天的对话。",
    },
    rooms: {
      title: "同一个窗口里的几面。",
      dek: "今天和任务是同一间屋子。技能、视频、学习和偶像从底栏进去。",
      label: "模块",
      items: [
        {
          id: "today",
          name: "今天",
          line: "到期的和要你回答的，出现在这一天的对话里。晚间简报写好就留在这里。",
          points: ["晨间和晚间各一份简报，写完就不再改", "只在到期、卡住、等你的时候开口", "记忆只留下能核对的事实"],
          alt: "Yoyo 的今天。左边是晚间简报，右边是这一天的对话。",
        },
        {
          id: "tasks",
          name: "任务",
          line: "还是这一间屋子。任务是对话交出去的一件工作，做完的、停下的、在等下一条消息的，列在旁边。",
          points: ["一句话交出去，进展列在旁边", "可以 @ 一个 agent，也可以拆给几个一起做", "同一件事让几个 agent 各做一份，再挑一份"],
          alt: "Yoyo 的任务。左边列出交出去的工作，右边仍是对话。",
        },
        {
          id: "skills",
          name: "技能",
          line: "已安装的技能可以在对话里用 @ 调用。到市场里安装，不会自己闯进每一轮。",
          points: ["十一个技能目录，在一处搜索", "装好以后，在对话里 @ 它", "技能包默认关闭，你打开才进这一轮"],
          alt: "Yoyo 的技能。市场里可以搜索目录并安装。",
        },
        {
          id: "video",
          name: "视频",
          line: "创作、短片、短剧和画布在同一间工作台。先说想做什么片子，案例只是把做法放进输入框。",
          points: ["选片、翻拍参考片、剪视频、生图、生视频、配音", "即梦、可灵、通义万相、Vidu、Veo 等 90 多个模型插件", "短片、短剧、画布做完，送到成片台收尾"],
          alt: "Yoyo 的视频工作台。创作页问今天想做一条什么样的片子。",
        },
        {
          id: "study",
          name: "学习",
          line: "先选一条路，每一页只做一件事。计划、文库、课本和练习都在这台机器上，旁边有一位导师。",
          points: ["先定大纲，再一次做一个知识点", "在文库里检索，按页阅读，摘录编成课本", "先做到期的题，一次只答一题"],
          alt: "Yoyo 的学习。计划、阅读、课本和练习，旁边是导师。",
        },
        {
          id: "idol",
          name: "偶像",
          line: "选一位，或者新建。可以说话；你让她动手时，用的是这台电脑上的 Yoyo。",
          points: ["选一位，或者新建一位", "可以打字，也可以通话", "可以待在桌面上。让她干活时，交给 Yoyo"],
          alt: "Yoyo 的偶像。左边选一位，中间可以和她说话。",
        },
      ],
    },
    why: {
      title: "不只是一个聊天框。",
      dek: "下面这几件，是 Yoyo 已经做熟的。",
      items: [
        {
          tag: "Buddy",
          title: "一个路由，一群 agent。",
          body: "Claude Code、Codex、Gemini CLI、Qwen Code 这些支持 ACP 的本机 agent，装好、登录，就能接 Yoyo 派的工单。可以分工，也可以同一件事各做一份再比。",
        },
        {
          tag: "今天",
          title: "主动，但先想好要不要开口。",
          body: "到期的、卡住的、等你回答的才会出现。晨间和晚间各一份简报。主动程度和安静时段由你调。",
        },
        {
          tag: "技能",
          title: "十一个技能目录，一处安装。",
          body: "WorkBuddy、SkillHub、ClawHub、Anthropic、Vercel 等目录在同一个市场里搜索和安装。装好以后，在对话里用 @ 调用。",
        },
        {
          tag: "视频",
          title: "从一句话到成片。",
          body: "创作、短片、短剧和画布共用一间工作台。生图、生视频、配音和剪辑接你自己的账号，仓库里带着 90 多个模型插件。",
        },
        {
          tag: "审批",
          title: "写之前先问你。",
          body: "写文件和跑命令先问：一次、本对话、总是，或拒绝。改动按处审查。预算到顶，这一轮就停。密钥进本地保管库。",
        },
        {
          tag: "实验室",
          title: "先证明，再换上。",
          body: "提示词和工作手册可以自己打磨，但要先过评测。正在用的那一份由你切换。内核、评测器、密钥库和更新器，它改不了。",
        },
      ],
    },
    doors: {
      title: "人不在电脑前，\n也叫得到它。",
      dek: "手机、聊天软件和终端连的都是电脑上的同一个 Yoyo。活在电脑上干。",
      items: [
        {
          name: "手机",
          body: "同一个局域网里，装 Android 客户端，扫码配对。不装 App，用微信扫同一个码也行。",
          hint: "在设置里显示配对码",
        },
        {
          name: "聊天软件",
          body: "飞书、钉钉、企业微信、QQ、Telegram、Slack、Discord 等消息平台可以接进来，在那边和它说话。",
          hint: "在设置里添加机器人",
        },
        {
          name: "终端和浏览器",
          body: "命令行可以跑一轮、做评测、管理技能包。同一套界面也能开在本机浏览器里。",
          hint: "yoyo run · yoyo serve",
        },
      ],
    },
    get: {
      title: "装到这台电脑上，\n就可以开始。",
      dek: "不用注册。选你的系统，打开今天，接上你自己的模型。",
      premise: "密钥要你自己准备。如果想要一个不用密钥、完全托管的聊天产品，这不是那一页。",
      version: "版本",
      steps: [
        "按系统安装桌面应用。",
        "打开「今天」，告诉它先盯哪件事。",
        "在设置里接上你自己的模型。",
        "技能、视频、学习和偶像从底栏进去。",
      ],
      thisMachine: "这台设备",
      phone: "Android 是同一局域网里的手机端，要先在电脑上装好 Yoyo。",
      note: "Windows / Linux 安装包未签名。SmartScreen 可能拦一下：更多信息 → 仍要运行。macOS 是 ad-hoc 签名，未经公证。源码 Apache-2.0，不舒服就自己编译。",
    },
    foot: { product: "产品", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
  },
  en: {
    meta: {
      title: "Yoyo — the open-source desktop super agent",
      description:
        "An open-source super agent for the desktop. Today, tasks, skills, video, study, and a companion in one window, with Claude Code, Codex, and other local agents as workers. You bring the model. Chats stay on this machine.",
    },
    nav: {
      home: "Today",
      rooms: "Places",
      why: "Strengths",
      doors: "Doors",
      download: "Download",
      menu: "Menu",
      close: "Close menu",
      themeToDark: "Switch to dark",
      themeToLight: "Switch to light",
    },
    hero: {
      title: "Your day,\nin one window.",
      dek: "Yoyo is an open-source super agent for the desktop. It watches what needs you today, hands work to itself and to local agents like Claude Code and Codex, makes video, helps you study, and keeps you company on the desktop.",
      download: "Download",
      github: "GitHub",
      builds: "All builds",
      chips: ["Windows · macOS · Linux · Android", "Bring your own model", "Apache-2.0", "Go · Wails 3"],
      shotAlt: "Yoyo’s Today. An evening brief on the left, the day’s conversation on the right.",
    },
    rooms: {
      title: "A few faces of one window.",
      dek: "Today and Tasks share a room. Skills, Video, Study, and Idol open from the dock.",
      label: "Places",
      items: [
        {
          id: "today",
          name: "Today",
          line: "What’s due and what needs an answer show up in the day’s conversation. The evening brief stays here once it’s written.",
          points: [
            "A brief each morning and evening, fixed once written",
            "It speaks up only when something is due, stuck, or waiting on you",
            "Memory keeps only facts that can be checked",
          ],
          alt: "Yoyo’s Today. An evening brief on the left, the day’s conversation on the right.",
        },
        {
          id: "tasks",
          name: "Tasks",
          line: "Same room. A task is work the conversation handed off — finished, paused, or waiting for the next message.",
          points: [
            "Hand work off in one sentence and watch it beside the chat",
            "@ one agent, or split the work across several",
            "Give several agents the same job, then pick one result",
          ],
          alt: "Yoyo’s Tasks. Handed-off work listed on the left, the conversation still on the right.",
        },
        {
          id: "skills",
          name: "Skills",
          line: "Installed skills can be called with @ in the chat. Install from the catalog; they don’t walk into every turn on their own.",
          points: [
            "Eleven skill catalogs, searched in one place",
            "Once installed, call it with @ in the chat",
            "Skill packs stay off until you turn them on",
          ],
          alt: "Yoyo’s skill catalog, with search and install.",
        },
        {
          id: "video",
          name: "Video",
          line: "Create, short film, drama, and canvas share one desk. Say what you want to make. A case only fills the box.",
          points: [
            "Pick a clip, remake a reference, cut, generate images and video, dub",
            "90+ model plugins, including Jimeng, Kling, Wan, Vidu, and Veo",
            "Shorts, drama, and canvas send their work to the finish desk",
          ],
          alt: "Yoyo’s video desk, asking what kind of film to make today.",
        },
        {
          id: "study",
          name: "Study",
          line: "Pick one path, and do one thing per page. Plan, library, textbook, and practice stay on this machine, with a mentor beside them.",
          points: [
            "Set the outline first, then take one knowledge point at a time",
            "Search the library, read by page, turn excerpts into a short book",
            "Due questions come first, one answer at a time",
          ],
          alt: "Yoyo’s study plan, with reading, a textbook, practice, and a mentor.",
        },
        {
          id: "idol",
          name: "Idol",
          line: "Pick one, or make one. You can talk. When you ask her to do something, she uses the Yoyo on this computer.",
          points: [
            "Pick someone, or create your own",
            "Type, or call and talk",
            "She can stay on the desktop. Work you ask for goes to Yoyo",
          ],
          alt: "Yoyo’s Idol. Choose someone on the left and talk with her.",
        },
      ],
    },
    why: {
      title: "More than a chat box.",
      dek: "These are the parts of Yoyo that are already well worn.",
      items: [
        {
          tag: "Buddy",
          title: "One router, many agents.",
          body: "Claude Code, Codex, Gemini CLI, Qwen Code, and other local agents that speak ACP can take tickets from Yoyo once they are installed and signed in. Split the work, or give each the same job and compare.",
        },
        {
          tag: "Today",
          title: "Proactive, after deciding whether to speak.",
          body: "Only what is due, stuck, or waiting on you shows up. One brief in the morning, one in the evening. You set how proactive it is, and the quiet hours.",
        },
        {
          tag: "Skills",
          title: "Eleven catalogs, one place to install.",
          body: "WorkBuddy, SkillHub, ClawHub, Anthropic, Vercel, and more are searched and installed from one market. Call what you installed with @ in the chat.",
        },
        {
          tag: "Video",
          title: "From one sentence to a finished film.",
          body: "Create, Shorts, Drama, and Canvas share one desk. Images, video, voice, and cutting run on your own accounts, with 90+ model plugins in the repository.",
        },
        {
          tag: "Approval",
          title: "It asks before it writes.",
          body: "Writes and shell commands ask first: once, this chat, always, or deny. Changes are reviewed piece by piece. A spending cap stops the turn. Keys go into a local vault.",
        },
        {
          tag: "Lab",
          title: "Prove it first, then switch.",
          body: "Prompts and the playbook can refine themselves, but only past evaluation. You switch the copy in use. The kernel, evaluator, vault, and updater are out of its reach.",
        },
      ],
    },
    doors: {
      title: "Not at the desk?\nStill in reach.",
      dek: "Phone, chat apps, and the terminal all reach the same Yoyo on your computer. The work runs there.",
      items: [
        {
          name: "Phone",
          body: "On the same LAN, install the Android app and pair it with a code. Without the app, scan the same code with WeChat or the camera.",
          hint: "Show the pairing code in Settings",
        },
        {
          name: "Chat apps",
          body: "Connect Feishu, DingTalk, WeCom, QQ, Telegram, Slack, Discord, and other messaging platforms, and talk to it from there.",
          hint: "Add a bot in Settings",
        },
        {
          name: "Terminal and browser",
          body: "The CLI runs a turn, evaluates, and manages skill packs. The same UI can also be served to a browser on this computer.",
          hint: "yoyo run · yoyo serve",
        },
      ],
    },
    get: {
      title: "Install it here,\nthen start.",
      dek: "No account. Pick your system, open Today, connect your model.",
      premise: "You bring the key. If you want a hosted chat with no key, this is not that page.",
      version: "Version",
      steps: [
        "Install the desktop app for your system.",
        "Open Today and say what it should watch first.",
        "Connect your own model in Settings.",
        "Skills, Video, Study, and Idol open from the dock.",
      ],
      thisMachine: "This device",
      phone: "The Android app is a client on the same LAN. Install Yoyo on a computer first.",
      note: "Windows and Linux installers are unsigned. SmartScreen may pause on More info → Run anyway. macOS is ad-hoc signed and not notarized. The source is Apache-2.0 if you’d rather build it.",
    },
    foot: { product: "Product", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
  },
  ja: {
    meta: {
      title: "Yoyo — オープンソースのデスクトップ・スーパーエージェント",
      description:
        "オープンソースのデスクトップ・スーパーエージェント。今日、タスク、スキル、映像、学習、アイドルが一つのウィンドウにあり、Claude Code や Codex などローカルのエージェントにも仕事を渡せます。モデルは自分でつなぎ、会話はこのマシンに残ります。",
    },
    nav: {
      home: "今日",
      rooms: "場所",
      why: "強み",
      doors: "入口",
      download: "ダウンロード",
      menu: "メニュー",
      close: "メニューを閉じる",
      themeToDark: "ダークに切り替える",
      themeToLight: "ライトに切り替える",
    },
    hero: {
      title: "今日の用事は、\nこの一つの\nウィンドウで。",
      dek: "Yoyo はオープンソースのデスクトップ・スーパーエージェントです。今日見てほしいことを見張り、仕事を自分と Claude Code や Codex などローカルのエージェントに渡し、映像を作り、学習に付き合い、デスクトップで話し相手にもなります。",
      download: "ダウンロード",
      github: "GitHub",
      builds: "すべてのビルド",
      chips: ["Windows · macOS · Linux · Android", "モデルは持参", "Apache-2.0", "Go · Wails 3"],
      shotAlt: "Yoyo の今日。左に夜の報告、右にその日の会話。",
    },
    rooms: {
      title: "一つのウィンドウの、いくつかの面。",
      dek: "今日とタスクは同じ部屋です。スキル、映像、学習、アイドルは下のドックから開きます。",
      label: "場所",
      items: [
        {
          id: "today",
          name: "今日",
          line: "期限が来たものと、返事が要るものが、その日の会話に出ます。夜の報告は書いたらここに残ります。",
          points: ["朝と夜に報告が一通ずつ。書いたら書き換えない", "声をかけるのは、期限、行き詰まり、返事待ちのときだけ", "記憶に残すのは、照合できる事実だけ"],
          alt: "Yoyo の今日。左に夜の報告、右にその日の会話。",
        },
        {
          id: "tasks",
          name: "タスク",
          line: "同じ部屋です。タスクは会話が手渡した一つの仕事。終わったもの、止めたもの、次のメッセージを待っているものが並びます。",
          points: ["一言で手渡し、進み具合は横に並ぶ", "エージェントを @ で一人指名するか、数人に分担させる", "同じ仕事を複数のエージェントに任せ、一つを選ぶ"],
          alt: "Yoyo のタスク。左に手渡した仕事、右は同じ会話。",
        },
        {
          id: "skills",
          name: "スキル",
          line: "インストールしたスキルは、会話で @ から呼べます。カタログから入れるもので、勝手に毎ターン入ってはきません。",
          points: ["11 のスキルカタログを一か所で検索", "入れたあとは、会話で @ から呼ぶ", "スキルパックは、自分でオンにするまでオフ"],
          alt: "Yoyo のスキル。カタログを検索してインストールできる。",
        },
        {
          id: "video",
          name: "映像",
          line: "創作、短片、短劇、キャンバスは同じ作業台です。先に作りたいものを言う。事例は入力欄を埋めるだけです。",
          points: ["クリップ選択、参考映像のリメイク、カット、画像と動画の生成、吹き替え", "即夢、Kling、Wan、Vidu、Veo など 90 以上のモデルプラグイン", "短片、短劇、キャンバスの成果は仕上げ台へ送る"],
          alt: "Yoyo の映像作業台。今日どんな映像を作るか聞いている。",
        },
        {
          id: "study",
          name: "学習",
          line: "一つの道を選び、一ページでは一つのことだけ。計画、文庫、教科書、練習はこのマシンにあり、隣にメンターがいます。",
          points: ["先に目次を決め、知識点を一つずつ進める", "文庫を検索し、ページごとに読み、抜き書きを短い本にまとめる", "期限の来た問題から、一問ずつ答える"],
          alt: "Yoyo の学習。計画、読書、教科書、練習と、隣のメンター。",
        },
        {
          id: "idol",
          name: "アイドル",
          line: "一人を選ぶか、新しく作る。話せます。何かを頼むと、使うのはこのパソコンの Yoyo です。",
          points: ["一人を選ぶか、新しく作る", "文字でも、通話でも話せる", "デスクトップに置ける。頼んだ仕事は Yoyo に渡る"],
          alt: "Yoyo のアイドル。左で一人を選び、話しかける。",
        },
      ],
    },
    why: {
      title: "チャットだけではない。",
      dek: "次の部分は、Yoyo がすでに使い込んでいるところです。",
      items: [
        {
          tag: "Buddy",
          title: "ルーターは一つ、エージェントは何人でも。",
          body: "Claude Code、Codex、Gemini CLI、Qwen Code など ACP に対応したローカルのエージェントは、入れてログインすれば Yoyo のチケットを受けられます。分担させることも、同じ仕事を任せて比べることもできます。",
        },
        {
          tag: "今日",
          title: "自分から動く。ただし、声をかけるかは先に決める。",
          body: "出てくるのは、期限が来たもの、行き詰まったもの、返事を待っているものだけ。朝と夜に報告が一通ずつ。積極さと静かな時間帯は自分で決めます。",
        },
        {
          tag: "スキル",
          title: "11 のカタログを、一か所で入れる。",
          body: "WorkBuddy、SkillHub、ClawHub、Anthropic、Vercel などを一つのマーケットで検索して入れられます。入れたスキルは会話で @ から呼びます。",
        },
        {
          tag: "映像",
          title: "一言から、完成した映像まで。",
          body: "創作、短片、短劇、キャンバスは同じ作業台です。画像、動画、音声、編集は自分のアカウントで動き、リポジトリには 90 以上のモデルプラグインがあります。",
        },
        {
          tag: "承認",
          title: "書く前に、まず聞く。",
          body: "ファイルの書き込みとコマンドの実行は先に聞きます。一度だけ、この会話、常に、拒否。変更は箇所ごとに確認できます。予算の上限でそのターンは止まります。鍵はローカルの保管庫に入ります。",
        },
        {
          tag: "ラボ",
          title: "先に証明し、それから切り替える。",
          body: "プロンプトと手順書は自分で磨けますが、評価を通ってからです。使っているものを切り替えるのは自分。カーネル、評価器、保管庫、アップデータには手を出せません。",
        },
      ],
    },
    doors: {
      title: "席を離れても、\n呼べます。",
      dek: "スマホ、チャットアプリ、ターミナルがつながるのは、パソコンにある同じ Yoyo です。仕事はパソコンで進みます。",
      items: [
        {
          name: "スマホ",
          body: "同じ LAN で Android アプリを入れ、コードを読み取ってペアリングします。アプリなしでも、同じコードを WeChat やカメラで読み取れば使えます。",
          hint: "設定でペアリングコードを表示",
        },
        {
          name: "チャットアプリ",
          body: "Feishu、DingTalk、WeCom、QQ、Telegram、Slack、Discord などのメッセージングをつなぎ、そこから話しかけられます。",
          hint: "設定でボットを追加",
        },
        {
          name: "ターミナルとブラウザ",
          body: "CLI で一ターン実行し、評価し、スキルパックを管理できます。同じ画面をこのパソコンのブラウザで開くこともできます。",
          hint: "yoyo run · yoyo serve",
        },
      ],
    },
    get: {
      title: "このパソコンに入れて、\n始められます。",
      dek: "登録はいりません。システムを選び、今日を開き、自分のモデルをつなぎます。",
      premise: "鍵は自分で用意します。鍵なしの完全なホスト型チャットが欲しいなら、このページではありません。",
      version: "バージョン",
      steps: [
        "自分のシステムのデスクトップアプリを入れる。",
        "「今日」を開き、最初に何を見ていてほしいか伝える。",
        "設定で自分のモデルをつなぐ。",
        "スキル、映像、学習、アイドルはドックから入る。",
      ],
      thisMachine: "このデバイス",
      phone: "Android 版は同じ LAN の中のクライアントです。先にパソコンへ Yoyo を入れてください。",
      note: "Windows / Linux のインストーラは未署名です。SmartScreen では「詳細情報 → 実行」が必要なことがあります。macOS は ad-hoc 署名で、公証されていません。ソースは Apache-2.0 なので、自分でビルドできます。",
    },
    foot: { product: "製品", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
  },
};

/** detectLang 先看地址栏的 lang，再看本地保存的 yoyo-site-lang，最后看浏览器语言。中文给 zh，日文给 ja，其余给 en。读本地存储失败就当没有保存。 */
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

/** htmlLang 把站内的 zh 写成 html 的 zh-Hans，其余语言代码原样返回。lang 为 zh 时返回 zh-Hans，否则返回 lang。 */
export function htmlLang(lang: Lang): string {
  return lang === "zh" ? "zh-Hans" : lang;
}
