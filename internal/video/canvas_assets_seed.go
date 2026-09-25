package video

import (
	"fmt"
	"strings"
)

func (e *Engine) seedStudioAssets() error {
	now := Now()
	existing := map[string]bool{}
	if rows, err := e.DB.Query(`SELECT id FROM canvas_assets WHERE deleted_at = ''`); err == nil {
		defer rows.Close()
		for rows.Next() {
			var id string
			if rows.Scan(&id) == nil {
				existing[id] = true
			}
		}
	}
	folders := []struct {
		id, name string
		pos      int
	}{
		{"studio-folder-styles", "画风参考", 0},
		{"studio-folder-sets", "场景置景", 1},
		{"studio-folder-characters", "角色资产", 2},
		{"studio-folder-prompts", "提示词库", 3},
		{"studio-folder-boards", "镜头板", 4},
	}
	for _, f := range folders {
		_, err := e.DB.Exec(`INSERT OR IGNORE INTO canvas_asset_folders(id, name, position, created_at, updated_at) VALUES(?,?,?,?,?)`,
			f.id, f.name, f.pos, now, now)
		if err != nil {
			return err
		}
	}
	type still struct {
		file, title, note, category, folder string
		tags                                []string
		bytes                               int
	}
	stills := []still{
		{"cyberpunk-neon.jpg", "雨港记忆诊所", "赛博夜景：湿沥青、青洋红霓虹、透明伞。", "environment", "studio-folder-sets", []string{"夜景", "霓虹", "开场"}, 334440},
		{"retro-hong-kong.jpg", "天台晾衣绳", "九十年代港风天台，黄昏尘雾。", "environment", "studio-folder-sets", []string{"港风", "年代", "天台"}, 339172},
		{"suspense-noir.jpg", "巷口证人", "黑白高反差暗巷追逐。", "environment", "studio-folder-sets", []string{"悬疑", "黑白", "追逐"}, 230328},
		{"fantasy-3d.jpg", "雾山女剑", "国风 3D 剑客定妆参考。", "character", "studio-folder-characters", []string{"角色", "玄幻", "定妆"}, 268767},
		{"storybook-fantasy.jpg", "第一场雪之前", "绘本狐狸送信。", "character", "studio-folder-characters", []string{"绘本", "童话"}, 596903},
		{"urban-live-action.jpg", "两杯没喝完的茶", "海边旅店黄昏置景。", "environment", "studio-folder-sets", []string{"实拍", "旅店", "空镜"}, 292073},
		{"future-tech.jpg", "记忆修补诊所", "近未来城市夜店面。", "environment", "studio-folder-sets", []string{"科幻", "近未来"}, 258228},
		{"nature-healing.jpg", "雨后玻璃花房", "前景水滴与逆光植物。", "environment", "studio-folder-sets", []string{"治愈", "空镜"}, 356118},
		{"real-life.jpg", "下班后的一口气泡", "通勤纪实口感。", "material", "studio-folder-styles", []string{"纪实", "广告"}, 237912},
		{"black-white-noir.jpg", "默片酒吧", "硬侧光与灰阶层次。", "material", "studio-folder-styles", []string{"黑白", "布光"}, 192713},
		{"period-live-action.jpg", "月门黄昏", "宅院灯笼与真丝绸缎。", "environment", "studio-folder-sets", []string{"年代", "古装"}, 276308},
		{"space-opera.jpg", "月背天线", "星际远景推进参考。", "environment", "studio-folder-sets", []string{"太空", "史诗"}, 288686},
		{"clay-stop-motion.jpg", "雨村小老鼠", "黏土定格材质参考。", "character", "studio-folder-characters", []string{"定格", "黏土"}, 332734},
		{"ink-narrative.jpg", "一叶舟", "水墨留白与横移。", "material", "studio-folder-styles", []string{"水墨", "东方"}, 332270},
		{"chinese-2d.jpg", "柳园学者", "国漫 2D 角色转面参考。", "character", "studio-folder-characters", []string{"国漫", "二维"}, 394068},
		{"surreal-dream.jpg", "湖上的木质楼梯", "超现实尺度错位。", "environment", "studio-folder-sets", []string{"梦境", "超现实"}, 296981},
		{"comic-pop.jpg", "屋顶信使", "漫画块面与低角度。", "character", "studio-folder-characters", []string{"漫画", "动作"}, 756004},
		{"three-d-cartoon.jpg", "峡谷小探险家", "三维卡通角色。", "character", "studio-folder-characters", []string{"卡通", "亲子"}, 383405},
		{"warm-interior.jpg", "谷仓午后", "暖室内改造工作室。", "environment", "studio-folder-sets", []string{"室内", "生活"}, 384035},
		{"cyberpunk-neon.jpg", "开场板·湿沥青倒影", "同一夜景的机位板：低机位看路面反光。", "material", "studio-folder-boards", []string{"镜头板", "开场"}, 334440},
		{"suspense-noir.jpg", "覆盖板·巷口过肩", "暗巷对手戏的过肩参考。", "material", "studio-folder-boards", []string{"镜头板", "覆盖"}, 230328},
		{"retro-hong-kong.jpg", "空镜板·晾衣绳", "天台风动衣角的空镜。", "material", "studio-folder-boards", []string{"镜头板", "空镜"}, 339172},
		{"black-white-noir.jpg", "布光板·硬侧光", "酒吧硬侧光与烟灰层次。", "material", "studio-folder-boards", []string{"镜头板", "布光"}, 192713},
		{"nature-healing.jpg", "空镜板·花房水滴", "前景水滴与逆光植物。", "material", "studio-folder-boards", []string{"镜头板", "治愈"}, 356118},
		{"warm-interior.jpg", "室内板·窗光", "谷仓窗光切在木桌笔记上。", "material", "studio-folder-boards", []string{"镜头板", "室内"}, 384035},
	}
	put := func(id string, record map[string]any) {
		if existing[id] {
			return
		}
		e.upsertAsset(record)
		existing[id] = true
	}
	for i, s := range stills {
		src := "/short-drama-styles/" + s.file
		id := "studio-still-" + strings.TrimSuffix(s.file, ".jpg")
		if i >= 19 {
			id = "studio-board-" + strings.TrimSuffix(s.file, ".jpg") + "-" + fmt.Sprint(i)
		}
		put(id, map[string]any{
			"id": id, "kind": "image", "title": s.title, "coverUrl": src, "tags": s.tags,
			"folderId": s.folder, "category": s.category, "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室", "note": s.note,
			"data": map[string]any{"dataUrl": src, "width": 1920, "height": 1080, "bytes": s.bytes, "mimeType": "image/jpeg"},
			"sort": i,
		})
	}
	texts := []struct {
		id, title, folder, category, cover, content string
		tags                                        []string
	}{
		{"studio-text-shot-grammar", "镜头提示词骨架", "studio-folder-prompts", "other", "/short-drama-styles/cyberpunk-neon.jpg",
			"主体 + 动作 + 景别 + 机位 + 运镜 + 光线来源 + 材质 + 禁用（字幕/标识/水印）。\n例：雨夜十字路口，年轻人撑透明伞走入人群，远景缓慢推进到侧脸，霓虹倒映湿沥青，无字幕。",
			[]string{"提示词", "分镜"}},
		{"studio-text-character-card", "角色卡：林晚", "studio-folder-prompts", "character", "/short-drama-styles/urban-live-action.jpg",
			"林晚，28 岁，东亚女性，短发，深色风衣，左腕细银链。职业：记忆诊所夜班接待。气质克制，不笑。衣橱编号 A1 风衣 / A2 灰针织。禁用网红脸与换发型。",
			[]string{"角色", "一致性"}},
		{"studio-text-15s-ad", "十五秒广告结构", "studio-folder-prompts", "other", "/short-drama-styles/real-life.jpg",
			"0–3s 钩子（可见动作或提问）\n3–8s 产品进入使用场景\n8–12s 一个可感知结果\n12–15s 主张字幕（交给后期，不让模型写字）",
			[]string{"电商", "广告"}},
		{"studio-text-color-script", "雨港色彩脚本", "studio-folder-prompts", "material", "/short-drama-styles/future-tech.jpg",
			"全片 60% 沥青黑与湿水泥，30% 冷青与脏黄钠灯，10% 洋红只给诊所灯箱与关键道具。肤色保持自然暖中性。禁用全片彩虹霓虹。",
			[]string{"色彩", "项目规范"}},
		{"studio-text-dialogue", "末班车三分钟对白骨架", "studio-folder-prompts", "other", "/short-drama-styles/black-white-noir.jpg",
			"两人争一本笔记。限制：同一车厢、无闪回。每人隐瞒一件事。转折必须是可见动作（打开笔记、把伞让出）。不要互相解释主题。",
			[]string{"编剧", "对白"}},
		{"studio-text-coverage", "对手戏覆盖清单", "studio-folder-prompts", "other", "/short-drama-styles/retro-hong-kong.jpg",
			"主镜头（空间关系）→ A 过肩 → B 过肩 → 手部插入 → 反应特写。轴线：窗在画面左侧。越轴必须经中性镜头。",
			[]string{"覆盖", "轴线"}},
		{"studio-text-world-rule", "记忆删除三条规则", "studio-folder-prompts", "other", "/short-drama-styles/future-tech.jpg",
			"1. 每次删除痛苦记忆会失去一种颜色。\n2. 颜色不可买回，只能通过留下新记忆缓慢恢复。\n3. 诊所灯箱的洋红是全城最后的红。视觉证据必须出现在场景里。",
			[]string{"设定", "科幻"}},
		{"studio-text-safety", "生成安全底线", "studio-folder-prompts", "other", "/short-drama-styles/nature-healing.jpg",
			"不生成未成年人性化内容；不写真实商标与在世明星脸；暴力保持可读但不猎奇；提示词要求无水印无乱码文字。",
			[]string{"安全", "规范"}},
		{"studio-text-lighting", "夜戏室内布光", "studio-folder-prompts", "material", "/short-drama-styles/black-white-noir.jpg",
			"主光必须来自可见灯具；窗外只做轮廓；肤色保持可读；强调色只给一个道具。禁用无来源霓虹和全黑死区。",
			[]string{"布光", "夜戏"}},
		{"studio-text-export", "竖屏导出规格", "studio-folder-prompts", "other", "/short-drama-styles/real-life.jpg",
			"画幅 9:16；时长 15 / 30 / 60；字幕安全区距边 8%；不要让模型烧录文字；音量对白优先于音乐。",
			[]string{"导出", "规格"}},
		{"studio-text-hook", "三秒钩子变体", "studio-folder-prompts", "other", "/short-drama-styles/comic-pop.jpg",
			"A. 反常动作：她把伞递给陌生人却自己淋雨。\nB. 提问：如果你能删除一段记忆，最先删哪一天？\nC. 物件：一本被胶带缠死的笔记在座位上。",
			[]string{"钩子", "短视频"}},
		{"studio-text-qc", "成片质检十条", "studio-folder-prompts", "other", "/short-drama-styles/suspense-noir.jpg",
			"换脸、跳轴、乱码字、水印、商标、未成年、声画错位、衣橱漂移、道具焕新、无来源光线。命中即返工。",
			[]string{"质检", "成片"}},
		{"studio-text-naming", "镜头命名", "studio-folder-prompts", "other", "/short-drama-styles/ink-narrative.jpg",
			"EP集-S场-SH镜-v版本，例如 EP01-S03-SH12-v3。角色用资产 id，不用“最终”“最新”。",
			[]string{"命名", "归档"}},
		{"studio-text-sfx", "雨港声音分层", "studio-folder-prompts", "other", "/short-drama-styles/cyberpunk-neon.jpg",
			"对白干声；环境层：雨 + 远处车辙；拟音：伞骨、鞋底湿沥青；音乐只在诊所灯箱亮起时进入。",
			[]string{"声音", "分层"}},
		{"studio-text-ad-claim", "广告主张边界", "studio-folder-prompts", "other", "/short-drama-styles/real-life.jpg",
			"只说可感知体验（一口气、清爽、下班后），不说疗效、认证数据和绝对化。字幕不超过 8 字/行。",
			[]string{"电商", "合规"}},
	}
	for _, t := range texts {
		put(t.id, map[string]any{
			"id": t.id, "kind": "text", "title": t.title, "coverUrl": t.cover, "tags": t.tags,
			"folderId": t.folder, "category": t.category, "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室",
			"data": map[string]any{"content": t.content},
		})
	}
	entities := []struct {
		id, title, cover, folder, category string
		def                                map[string]any
	}{
		{"studio-entity-linwan", "角色·林晚", "/short-drama-styles/urban-live-action.jpg", "studio-folder-characters", "character", map[string]any{
			"type": "character", "name": "林晚", "age": 28, "wardrobe": []string{"A1 深色风衣", "A2 灰针织"}, "marker": "左腕细银链",
		}},
		{"studio-entity-qiao", "角色·阿乔", "/short-drama-styles/retro-hong-kong.jpg", "studio-folder-characters", "character", map[string]any{
			"type": "character", "name": "阿乔", "age": 31, "wardrobe": []string{"宽肩西装", "白衬衫"}, "marker": "旧钢表",
		}},
		{"studio-entity-clinic", "场景·雨港诊所", "/short-drama-styles/cyberpunk-neon.jpg", "studio-folder-sets", "environment", map[string]any{
			"type": "location", "name": "雨港记忆诊所", "time": "夜/雨", "light": "洋红灯箱 + 室内暖钨丝",
		}},
		{"studio-entity-umbrella", "道具·透明伞", "/short-drama-styles/cyberpunk-neon.jpg", "studio-folder-characters", "prop", map[string]any{
			"type": "prop", "name": "透明伞", "duty": "开场识别物", "wear": "伞骨有一处胶布",
		}},
		{"studio-entity-notebook", "道具·胶带笔记", "/short-drama-styles/black-white-noir.jpg", "studio-folder-characters", "prop", map[string]any{
			"type": "prop", "name": "胶带缠死的笔记", "duty": "末班车冲突核心", "wear": "封面缺一角",
		}},
		{"studio-entity-chain", "道具·细银链", "/short-drama-styles/urban-live-action.jpg", "studio-folder-characters", "prop", map[string]any{
			"type": "prop", "name": "左腕细银链", "duty": "林晚识别物", "wear": "扣环发暗",
		}},
		{"studio-entity-rooftop", "场景·港风天台", "/short-drama-styles/retro-hong-kong.jpg", "studio-folder-sets", "environment", map[string]any{
			"type": "location", "name": "天台晾衣绳", "time": "黄昏", "light": "逆光尘雾 + 钠灯",
		}},
		{"studio-entity-bus", "场景·末班公交", "/short-drama-styles/urban-live-action.jpg", "studio-folder-sets", "environment", map[string]any{
			"type": "location", "name": "末班公交车厢", "time": "夜", "light": "车厢暖荧光 + 窗外霓虹扫过",
		}},
		{"studio-entity-barn", "场景·谷仓工作室", "/short-drama-styles/warm-interior.jpg", "studio-folder-sets", "environment", map[string]any{
			"type": "location", "name": "改造谷仓", "time": "午后", "light": "窗光切在木桌",
		}},
		{"studio-entity-mother", "角色·陈秋", "/short-drama-styles/warm-interior.jpg", "studio-folder-characters", "character", map[string]any{
			"type": "character", "name": "陈秋", "age": 54, "wardrobe": []string{"洗旧卡其外套", "细金耳钉"}, "marker": "右手关节有胶片刮痕",
		}},
		{"studio-entity-fox", "角色·送信狐狸", "/short-drama-styles/storybook-fantasy.jpg", "studio-folder-characters", "character", map[string]any{
			"type": "character", "name": "阿禾", "age": "幼年拟人动物", "wardrobe": []string{"补过的绿围巾"}, "marker": "左耳缺口",
		}},
	}
	for _, ent := range entities {
		put(ent.id, map[string]any{
			"id": ent.id, "kind": "entity", "title": ent.title, "coverUrl": ent.cover, "tags": []string{"实体", "可复用"},
			"folderId": ent.folder, "category": ent.category, "status": "confirmed",
			"createdAt": now, "updatedAt": now, "source": "Yoyo 工作室",
			"data": map[string]any{"definition": ent.def},
		})
	}
	return nil
}
