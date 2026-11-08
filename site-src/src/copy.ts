/**
 * 拥有：产品站的英文、简体中文和日文文案。
 * 不拥有：页面怎么排。下面的对象是数据，不为每一句文案再写注释。
 */

export const LANGS = ["en", "zh", "ja"] as const;
export type Lang = (typeof LANGS)[number];

export const ROOMS = ["today", "tasks", "skills", "video", "study", "idol"] as const;
export type Room = (typeof ROOMS)[number];

/** 截图文件放在 public/shots，六种语言共用这一套文件名。 */
export const ROOM_SHOT: Record<Room, string> = {
  today: "today.png",
  tasks: "tasks.png",
  skills: "skills.png",
  video: "video.png",
  study: "study.png",
  idol: "idol.png",
};

export type Copy = {
  meta: { title: string; description: string };
  nav: {
    home: string;
    rooms: string;
    local: string;
    download: string;
    menu: string;
    close: string;
    themeToDark: string;
    themeToLight: string;
  };
  hero: { title: string; dek: string; download: string; github: string; builds: string; chips: string[]; shotAlt: string };
  rooms: { title: string; dek: string; label: string; items: { id: Room; name: string; line: string; alt: string }[] };
  local: { title: string; dek: string; items: string[] };
  get: {
    title: string;
    dek: string;
    premise: string;
    version: string;
    steps: string[];
    thisMachine: string;
    note: string;
  };
  foot: { product: string; issues: string; license: string; stack: string };
};

export const COPY: Record<Lang, Copy> = {
  zh: {
    meta: {
      title: "Yoyo — 这台电脑上的助手",
      description: "一个窗口里的今天、任务、技能、视频、学习和偶像。模型你自己接，对话留在这台电脑上。",
    },
    nav: {
      home: "今天",
      rooms: "模块",
      local: "这台电脑",
      download: "下载",
      menu: "菜单",
      close: "关闭菜单",
      themeToDark: "切换到深色",
      themeToLight: "切换到浅色",
    },
    hero: {
      title: "你的这一天，\n在这一个窗口里。",
      dek: "今天要你看的、交出去的任务、能调用的技能、要做的片子、学习计划和桌面上的偶像。模型你自己接。",
      download: "下载",
      github: "GitHub",
      builds: "全部安装包",
      chips: ["Windows · macOS · Linux", "模型你自己接", "Apache-2.0"],
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
          alt: "Yoyo 的今天。左边是晚间简报，右边是这一天的对话。",
        },
        {
          id: "tasks",
          name: "任务",
          line: "还是这一间屋子。任务是对话交出去的一件工作，做完的、停下的、在等下一条消息的，列在旁边。",
          alt: "Yoyo 的任务。左边列出交出去的工作，右边仍是对话。",
        },
        {
          id: "skills",
          name: "技能",
          line: "已安装的技能可以在对话里用 @ 调用。到市场里安装，不会自己闯进每一轮。",
          alt: "Yoyo 的技能。市场里可以搜索目录并安装。",
        },
        {
          id: "video",
          name: "视频",
          line: "创作、短片、短剧和画布在同一间工作台。先说想做什么片子，案例只是把做法放进输入框。",
          alt: "Yoyo 的视频工作台。创作页问今天想做一条什么样的片子。",
        },
        {
          id: "study",
          name: "学习",
          line: "先选一条路，每一页只做一件事。计划、文库、课本和练习都在这台机器上，旁边有一位导师。",
          alt: "Yoyo 的学习。计划、阅读、课本和练习，旁边是导师。",
        },
        {
          id: "idol",
          name: "偶像",
          line: "选一位，或者新建。可以说话；你让她动手时，用的是这台电脑上的 Yoyo。",
          alt: "Yoyo 的偶像。左边选一位，中间可以和她说话。",
        },
      ],
    },
    local: {
      title: "换一面，还是它。",
      dek: "底栏换的是同一扇窗口里的一面，不是另一个应用。",
      items: [
        "今天负责这一天要你看的。已经在做的，留在任务里。记忆记下能核对的事实。",
        "视频、学习和偶像各有自己的桌子。动手的还是这一个助手。",
        "模型你自己接。密钥留在设置里，对话不托管到别人的云上。",
        "实验室里先改进、再检查。正在用的那一份，要你自己切换。",
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
      thisMachine: "这台电脑",
      note: "Windows / Linux 安装包未签名。SmartScreen 可能拦一下：更多信息 → 仍要运行。macOS 是 ad-hoc 签名，未经公证。源码 Apache-2.0，不舒服就自己编译。",
    },
    foot: { product: "产品", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
  },
  en: {
    meta: {
      title: "Yoyo — an assistant on this computer",
      description: "Today, tasks, skills, video, study, and a desktop companion in one window. You bring the model. Chats stay on this machine.",
    },
    nav: {
      home: "Today",
      rooms: "Places",
      local: "This machine",
      download: "Download",
      menu: "Menu",
      close: "Close menu",
      themeToDark: "Switch to dark",
      themeToLight: "Switch to light",
    },
    hero: {
      title: "Your day,\nin one window.",
      dek: "What needs you today, work you hand off, skills you can call, a film desk, study, and someone on the desktop. You bring the model.",
      download: "Download",
      github: "GitHub",
      builds: "All builds",
      chips: ["Windows · macOS · Linux", "Bring your own model", "Apache-2.0"],
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
          alt: "Yoyo’s Today. An evening brief on the left, the day’s conversation on the right.",
        },
        {
          id: "tasks",
          name: "Tasks",
          line: "Same room. A task is work the conversation handed off — finished, paused, or waiting for the next message.",
          alt: "Yoyo’s Tasks. Handed-off work listed on the left, the conversation still on the right.",
        },
        {
          id: "skills",
          name: "Skills",
          line: "Installed skills can be called with @ in the chat. Install from the catalog; they don’t walk into every turn on their own.",
          alt: "Yoyo’s skill catalog, with search and install.",
        },
        {
          id: "video",
          name: "Video",
          line: "Create, short film, drama, and canvas share one desk. Say what you want to make. A case only fills the box.",
          alt: "Yoyo’s video desk, asking what kind of film to make today.",
        },
        {
          id: "study",
          name: "Study",
          line: "Pick one path, and do one thing per page. Plan, library, textbook, and practice stay on this machine, with a mentor beside them.",
          alt: "Yoyo’s study plan, with reading, a textbook, practice, and a mentor.",
        },
        {
          id: "idol",
          name: "Idol",
          line: "Pick one, or make one. You can talk. When you ask her to do something, she uses the Yoyo on this computer.",
          alt: "Yoyo’s Idol. Choose someone on the left and talk with her.",
        },
      ],
    },
    local: {
      title: "Another face. Same Yoyo.",
      dek: "The dock changes the face of this window. It does not open another app.",
      items: [
        "Today is what needs you. Work already underway stays in Tasks. Memory keeps facts that can be checked.",
        "Video, Study, and Idol each have their own desk. The hands are still this one assistant.",
        "You connect the model. Keys stay in Settings. Chats are not hosted on someone else’s cloud.",
        "The lab improves, then checks. You switch on the one that’s in use.",
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
      thisMachine: "This machine",
      note: "Windows and Linux installers are unsigned. SmartScreen may pause on More info → Run anyway. macOS is ad-hoc signed and not notarized. The source is Apache-2.0 if you’d rather build it.",
    },
    foot: { product: "Product", issues: "Issues", license: "Apache-2.0", stack: "Go · Wails" },
  },
  ja: {
    meta: {
      title: "Yoyo — このパソコンのアシスタント",
      description: "今日、タスク、スキル、映像、学習、アイドルが一つのウィンドウにあります。モデルは自分でつなぎ、会話はこのマシンに残ります。",
    },
    nav: {
      home: "今日",
      rooms: "場所",
      local: "このマシン",
      download: "ダウンロード",
      menu: "メニュー",
      close: "メニューを閉じる",
      themeToDark: "ダークに切り替える",
      themeToLight: "ライトに切り替える",
    },
    hero: {
      title: "今日の用事は、\nこの一つのウィンドウで。",
      dek: "今日見てほしいこと、渡した仕事、呼び出せるスキル、映像、学習、デスクトップのアイドル。モデルは自分でつなぐ。",
      download: "ダウンロード",
      github: "GitHub",
      builds: "すべてのビルド",
      chips: ["Windows · macOS · Linux", "モデルは持参", "Apache-2.0"],
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
          alt: "Yoyo の今日。左に夜の報告、右にその日の会話。",
        },
        {
          id: "tasks",
          name: "タスク",
          line: "同じ部屋です。タスクは会話が手渡した一つの仕事。終わったもの、止めたもの、次のメッセージを待っているものが並びます。",
          alt: "Yoyo のタスク。左に手渡した仕事、右は同じ会話。",
        },
        {
          id: "skills",
          name: "スキル",
          line: "インストールしたスキルは、会話で @ から呼べます。カタログから入れるもので、勝手に毎ターン入ってはきません。",
          alt: "Yoyo のスキル。カタログを検索してインストールできる。",
        },
        {
          id: "video",
          name: "映像",
          line: "創作、短片、短劇、キャンバスは同じ作業台です。先に作りたいものを言う。事例は入力欄を埋めるだけです。",
          alt: "Yoyo の映像作業台。今日どんな映像を作るか聞いている。",
        },
        {
          id: "study",
          name: "学習",
          line: "一つの道を選び、一ページでは一つのことだけ。計画、文庫、教科書、練習はこのマシンにあり、隣にメンターがいます。",
          alt: "Yoyo の学習。計画、読書、教科書、練習と、隣のメンター。",
        },
        {
          id: "idol",
          name: "アイドル",
          line: "一人を選ぶか、新しく作る。話せます。何かを頼むと、使うのはこのパソコンの Yoyo です。",
          alt: "Yoyo のアイドル。左で一人を選び、話しかける。",
        },
      ],
    },
    local: {
      title: "面は変わる。Yoyo は同じ。",
      dek: "ドックが変えるのは、このウィンドウの面です。別のアプリは開きません。",
      items: [
        "今日は、今日見てほしいことを置きます。すでに進んでいる仕事はタスクに残ります。記憶は照合できる事実を残します。",
        "映像、学習、アイドルにはそれぞれの机があります。手を動かすのは、同じアシスタントです。",
        "モデルは自分でつなぐ。鍵は設定に残り、会話は他人のクラウドには置きません。",
        "ラボでは先に改善し、それから検査します。使っているものは、自分で切り替えます。",
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
      thisMachine: "このマシン",
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
