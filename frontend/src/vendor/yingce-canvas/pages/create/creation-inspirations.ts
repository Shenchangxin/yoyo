// @ts-nocheck
import type { CreationMode } from "./creation-types";

export type CreationCaseLane = "studio" | "drama" | "canvas" | "chat";

/** One shipped still in frontend/public/short-drama-styles. `file` is the public filename; `id` is the style preset id when `preset` is true. */
export type StyleCoverRecord = {
    id: string;
    title: string;
    category: string;
    file: string;
    preset: boolean;
};

/** Every jpg under public/short-drama-styles. Titles match the canvas style presets, except warm-interior, which is only a still. */
export const styleCoverCatalog: StyleCoverRecord[] = [
    { id: "cyberpunk-neon", title: "赛博霓虹", category: "科幻实拍", file: "cyberpunk-neon.jpg", preset: true },
    { id: "retro-hong-kong", title: "复古港风", category: "年代实拍", file: "retro-hong-kong.jpg", preset: true },
    { id: "suspense-noir", title: "悬疑犯罪夜景", category: "真人实拍", file: "suspense-noir.jpg", preset: true },
    { id: "fantasy-3d", title: "国风 3D 玄幻", category: "三维动画", file: "fantasy-3d.jpg", preset: true },
    { id: "warm-interior", title: "暖室内景", category: "真人实拍", file: "warm-interior.jpg", preset: false },
    { id: "storybook-fantasy", title: "绘本童话", category: "插画动画", file: "storybook-fantasy.jpg", preset: true },
    { id: "urban-live-action", title: "都市真人短剧", category: "真人实拍", file: "urban-live-action.jpg", preset: true },
    { id: "future-tech", title: "未来科技", category: "科幻实拍", file: "future-tech.jpg", preset: true },
    { id: "nature-healing", title: "自然疗愈", category: "真人实拍", file: "nature-healing.jpg", preset: true },
    { id: "real-life-documentary", title: "现实生活纪实", category: "真人实拍", file: "real-life.jpg", preset: true },
    { id: "black-white-noir", title: "黑白默片", category: "风格化实拍", file: "black-white-noir.jpg", preset: true },
    { id: "period-live-action", title: "古装 / 年代真人", category: "真人实拍", file: "period-live-action.jpg", preset: true },
    { id: "space-opera", title: "太空歌剧", category: "科幻动画", file: "space-opera.jpg", preset: true },
    { id: "clay-stop-motion", title: "黏土定格", category: "定格动画", file: "clay-stop-motion.jpg", preset: true },
    { id: "ink-narrative", title: "水墨叙事", category: "风格化动画", file: "ink-narrative.jpg", preset: true },
    { id: "chinese-2d", title: "国漫 2D", category: "二维动画", file: "chinese-2d.jpg", preset: true },
    { id: "surreal-dream", title: "超现实梦境", category: "风格化实拍", file: "surreal-dream.jpg", preset: true },
    { id: "comic-pop", title: "漫画分镜", category: "风格化动画", file: "comic-pop.jpg", preset: true },
    { id: "three-d-cartoon", title: "3D 卡通短剧", category: "三维动画", file: "three-d-cartoon.jpg", preset: true },
];

export type CreationInspiration = {
    id: string;
    title: string;
    description: string;
    image: string;
    mode: CreationMode;
    prompt: string;
    featured?: boolean;
    styleId: string;
    lane: CreationCaseLane;
};

const cover = (file: string) => `/short-drama-styles/${file}`;

export const creationFeaturedWorks: CreationInspiration[] = [
    {
        id: "rain-clinic",
        title: "雨港记忆诊所",
        description: "湿路面反光 · 从人群远景推到侧脸",
        image: cover("cyberpunk-neon.jpg"),
        mode: "video",
        styleId: "cyberpunk-neon",
        lane: "studio",
        prompt: "雨夜东亚都市十字路口，青洋红霓虹倒映在湿沥青上。一位穿深色风衣的年轻人撑透明伞走入人群，镜头从宽银幕远景缓慢推进到侧脸，浅景深、变形宽银幕光斑，电影感，无字幕无标识。",
        featured: true,
    },
    {
        id: "rooftop-line",
        title: "天台晾衣绳",
        description: "九十年代港风 · 黄昏逆光对望",
        image: cover("retro-hong-kong.jpg"),
        mode: "video",
        styleId: "retro-hong-kong",
        lane: "studio",
        prompt: "九十年代华语都市天台，白衬衫晾在绳上，黄昏尘雾。两位多年未见的旧友隔着晾衣绳对望，风吹动衣角，轻微手持并缓慢靠近，胶片颗粒，无字幕。",
    },
    {
        id: "alley-witness",
        title: "巷口证人",
        description: "黑白高反差 · 低机位跟拍加速",
        image: cover("suspense-noir.jpg"),
        mode: "video",
        styleId: "suspense-noir",
        lane: "studio",
        prompt: "黑白悬疑电影，侦探穿长风衣跑进狭窄砖巷，路灯切出硬光带，低机位跟拍，湿石板反光，节奏逐渐加快，无彩色无字幕。",
    },
    {
        id: "mist-sword",
        title: "雾山女剑",
        description: "国风 3D 角色 · 材质与披风层次",
        image: cover("fantasy-3d.jpg"),
        mode: "image",
        styleId: "fantasy-3d",
        lane: "canvas",
        prompt: "原创东方幻想女剑客概念图，层叠轻甲与朱红披风，银白发饰，山雾背景，半写实三维材质，电影级侧光，不模仿已有角色，无文字。",
    },
    {
        id: "barn-afternoon",
        title: "谷仓午后",
        description: "暖室内戏 · 用动作写告别",
        image: cover("warm-interior.jpg"),
        mode: "text",
        styleId: "warm-interior",
        lane: "drama",
        prompt: "写一场发生在改造谷仓工作室里的母女对话：女儿准备离开家乡，母亲表面平静，一直在整理旧胶片和木桌上的笔记。用动作、停顿和潜台词推进情绪，不要直接说出舍不得。",
    },
    {
        id: "first-snow",
        title: "第一场雪之前",
        description: "绘本叙事 · 愿望、阻碍、可见行动",
        image: cover("storybook-fantasy.jpg"),
        mode: "text",
        styleId: "storybook-fantasy",
        lane: "chat",
        prompt: "为亲子观众写一个森林童话短篇。主角是一只替动物送信的小狐狸，必须在第一场雪之前送出一封没有地址的信。先给出愿望与阻碍，再设计三个递进事件，最后用一个可见的行动完成情绪转折。避免说教。",
    },
    {
        id: "last-bus",
        title: "末班车笔记",
        description: "单场景短剧 · 对白里藏目的",
        image: cover("urban-live-action.jpg"),
        mode: "text",
        styleId: "urban-live-action",
        lane: "drama",
        prompt: "写一段发生在末班公交车上的三分钟短剧。两位陌生人都声称自己丢失了一本笔记，但只有一人说真话。先定义动机、场景限制和各自隐瞒的信息，再写动作、对白与反转结尾。全剧限两个角色、一个场景，适合低成本实拍。",
    },
    {
        id: "color-city",
        title: "颜色被删掉的城市",
        description: "近未来大纲 · 规则与代价",
        image: cover("future-tech.jpg"),
        mode: "text",
        styleId: "future-tech",
        lane: "chat",
        prompt: "构思一部关于记忆修补师的近未来都市短篇。居民能删除痛苦记忆，但主角发现每次删除都会遗失一种颜色。建立三条世界规则、主人公的核心缺陷、五个转折和付出代价的结局。先交付大纲，不铺写长篇。",
    },
    {
        id: "rain-farewell",
        title: "把告别写成一场雨",
        description: "诗性旁白 · 气味、触感、声音、光线",
        image: cover("nature-healing.jpg"),
        mode: "text",
        styleId: "nature-healing",
        lane: "chat",
        prompt: "为一段雨中离乡的影像写四组中文旁白，每组不超过四十字。分别用气味、触感、声音与光线表达告别，不要直接使用悲伤、不舍或想念。语言朴素可朗读，每组留一个停顿。",
    },
    {
        id: "commute-soda",
        title: "下班后的一口气泡",
        description: "十五秒广告 · 人群、主张、三镜头",
        image: cover("real-life.jpg"),
        mode: "text",
        styleId: "real-life-documentary",
        lane: "chat",
        prompt: "为面向年轻通勤者的无糖气泡饮料设计十五秒广告。定义目标人群、核心场景和一句传播主张，再给出三个镜头的画面、时长、音效与字幕。不要虚构营养功效，用下班后一口清爽的体验讲故事。",
    },
    {
        id: "alley-coffee",
        title: "巷口手作咖啡",
        description: "品牌视觉 · 三种方向同一气质",
        image: cover("retro-hong-kong.jpg"),
        mode: "text",
        styleId: "retro-hong-kong",
        lane: "chat",
        prompt: "为社区里一家安静的独立咖啡店提出三种视觉方向。品牌重视慢节奏、邻里关系和手作。每种方向包含色彩、摄影氛围、文字语气、三条社交内容主题和差异化理由。不要模仿连锁品牌标识。",
    },
    {
        id: "script-notes",
        title: "给剧本做一次镜头诊断",
        description: "创作复盘 · 引用片段再给改法",
        image: cover("black-white-noir.jpg"),
        mode: "text",
        styleId: "black-white-noir",
        lane: "chat",
        prompt: "以电影评论者的视角审阅我接下来提供的剧本或分镜，从叙事、人物、构图、运镜和声音五个维度提出反馈。每条意见都引用我提供的片段，并给出一个能实际拍摄的修改方案。还没有收到材料时，先向我索取，不要假装看过。",
    },
    {
        id: "moon-gate",
        title: "月门下的黄昏",
        description: "年代场景 · 史实、推测、虚构分开",
        image: cover("period-live-action.jpg"),
        mode: "text",
        styleId: "period-live-action",
        lane: "chat",
        prompt: "协助我设计一处晚清到民初过渡的宅院影视场景。先问清具体年份、地点和人物阶层，再列出服饰、交通、灯具、建筑和日常用语的考据清单。明确区分有来源的史实、合理推测与戏剧性虚构，不编造文献。",
    },
    {
        id: "lunar-antenna",
        title: "月背天线",
        description: "从极远景推进到头盔反光",
        image: cover("space-opera.jpg"),
        mode: "video",
        styleId: "space-opera",
        lane: "studio",
        prompt: "月球背面荒原，一位头盔飞行员独自走向废弃天线。镜头从极远景缓慢推进，低角度阳光在月尘上拉出长影，步伐沉重，面罩反射星云，克制冷色，宽银幕，无字幕无界面文字。",
    },
    {
        id: "rain-mouse",
        title: "雨村小老鼠",
        description: "黏土定格 · 指纹与窗灯光",
        image: cover("clay-stop-motion.jpg"),
        mode: "video",
        styleId: "clay-stop-motion",
        lane: "studio",
        prompt: "黏土定格动画，一只穿雨衣的小老鼠站在雨后鹅卵石巷口，窗里有暖黄灯。手作指纹质感、微缩布景、柔和体积光，固定中景，动作短而连续，可爱但不夸张，无文字。",
    },
    {
        id: "leaf-boat",
        title: "一叶舟",
        description: "水墨留白 · 横向移镜",
        image: cover("ink-narrative.jpg"),
        mode: "video",
        styleId: "ink-narrative",
        lane: "studio",
        prompt: "中国水墨动画，一叶小舟缓缓穿过层叠远山，薄雾留白，舟上人抬手触碰落下的花瓣。镜头平稳横移，墨色自然晕染，少量朱红点缀，节奏舒缓，无印章文字。",
    },
    {
        id: "glass-greenhouse",
        title: "雨后玻璃花房",
        description: "前景水滴 · 侧逆光浅景深",
        image: cover("nature-healing.jpg"),
        mode: "video",
        styleId: "nature-healing",
        lane: "studio",
        prompt: "清晨雨后的玻璃花房，镜头透过挂着水滴的叶片缓缓推向盛开的白花，阳光从侧后方进入，浅景深，真实植物纹理，空气中薄薄水汽，无人物无文字。",
    },
    {
        id: "willow-scholar",
        title: "柳园学者",
        description: "国漫 2D · 正侧背统一",
        image: cover("chinese-2d.jpg"),
        mode: "image",
        styleId: "chinese-2d",
        lane: "canvas",
        prompt: "原创二维动画古风学者角色设定，青衫、书匣与柳园黄昏，正面、侧面与背面三视图，线稿粗细和身高比例一致，赛璐璐平涂，暖灰背景，不添加文字，不模仿已有角色。",
    },
    {
        id: "lake-stairs",
        title: "湖上的木质楼梯",
        description: "超现实静帧 · 尺度错位",
        image: cover("surreal-dream.jpg"),
        mode: "image",
        styleId: "surreal-dream",
        lane: "canvas",
        prompt: "超现实电影静帧，旧木楼梯从平静湖面升入暮色天空，钟表的指针融化进水纹。低饱和奶油与雾蓝，柔和漫反射，精确透视，细腻材质，无人物无文字。",
    },
    {
        id: "two-teas",
        title: "两杯没喝完的茶",
        description: "海边旅店 · 用道具留下人",
        image: cover("urban-live-action.jpg"),
        mode: "image",
        styleId: "urban-live-action",
        lane: "canvas",
        prompt: "沿海小城旧旅店房间的电影置景，窗外黄昏海面，风吹动米白窗帘，桌上有两杯没喝完的茶和一本翻开的书。暖橙与浅蓝对比，胶片颗粒，眼平中广角，没有人物和可读文字。",
    },
    {
        id: "roof-courier",
        title: "屋顶信使",
        description: "漫画块面 · 低角度跃出",
        image: cover("comic-pop.jpg"),
        mode: "image",
        styleId: "comic-pop",
        lane: "canvas",
        prompt: "原创漫画封面插画，一位城市信使背着邮包在屋顶之间跃起，戏剧性低角度，青蓝、珊瑚红与暖黄大色块，半调网点，清晰黑轮廓，背景简洁，为标题留白但不生成文字。",
    },
    {
        id: "canyon-kid",
        title: "峡谷里的小探险家",
        description: "三维卡通 · 柔材质与魔法时分",
        image: cover("three-d-cartoon.jpg"),
        mode: "image",
        styleId: "three-d-cartoon",
        lane: "canvas",
        prompt: "原创三维卡通角色，圆脸儿童探险家背着双肩包站在彩色峡谷，魔法时分侧光，柔软材质与清晰剪影，亲切但不幼态恐怖谷，不模仿已有动画工作室角色，无文字。",
    },
];

export const caseCatalogSource = {
    id: "yoyo.creation-cases",
    covers: "/short-drama-styles",
    notice: "提示词写在工作室案例目录里。封面是 public/short-drama-styles 的画风参考静帧，文件名对应画风 id，不是成片，也不是用户生成结果。",
};

function resolveCreationCases(rows: CreationInspiration[]) {
    const covers = new Map(styleCoverCatalog.map((row) => [row.id, row]));
    const files = new Set<string>();
    for (const entry of styleCoverCatalog) {
        if (files.has(entry.file)) throw new Error(`duplicate style cover file ${entry.file}`);
        files.add(entry.file);
    }
    const ids = new Set<string>();
    const titlesByStyle = new Map<string, string[]>();
    for (const row of rows) {
        if (!row.id || ids.has(row.id)) throw new Error(`creation case id missing or duplicated: ${row.id}`);
        ids.add(row.id);
        const entry = covers.get(row.styleId);
        if (!entry) throw new Error(`creation case ${row.id} style ${row.styleId} is not in the cover catalog`);
        const image = cover(entry.file);
        if (row.image !== image) throw new Error(`creation case ${row.id} image ${row.image} does not match ${image}`);
        if (row.lane !== "studio" && row.lane !== "drama" && row.lane !== "canvas" && row.lane !== "chat") {
            throw new Error(`creation case ${row.id} has no lane`);
        }
        const titles = titlesByStyle.get(row.styleId) || [];
        titles.push(row.title);
        titlesByStyle.set(row.styleId, titles);
    }
    const unused = styleCoverCatalog.filter((entry) => !titlesByStyle.has(entry.id));
    if (unused.length) throw new Error(`style covers without a case: ${unused.map((entry) => entry.id).join(", ")}`);
    return rows.map((row) => {
        const entry = covers.get(row.styleId);
        return {
            ...row,
            styleTitle: entry.title,
            styleCategory: entry.category,
            coverFile: entry.file,
            preset: entry.preset,
            sharedWith: (titlesByStyle.get(row.styleId) || []).filter((title) => title !== row.title),
        };
    });
}

export type CreationCase = ReturnType<typeof resolveCreationCases>[number];

export const creationCases: CreationCase[] = resolveCreationCases(creationFeaturedWorks);

export function caseCredit(item: { id?: string }): string {
    const row = creationCases.find((candidate) => candidate.id === item.id);
    if (!row) return "";
    const shared = row.sharedWith.length ? ` · 封面与「${row.sharedWith[0]}」共用` : "";
    const preset = row.preset ? "" : " · 静帧未进画风预设";
    return `${row.styleCategory} · ${row.styleTitle} · ${row.coverFile}${shared}${preset}`;
}

export const inspirationSource = {
    notice: caseCatalogSource.notice,
};
