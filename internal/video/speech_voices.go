// speech_voices.go 是每家语音接口能选的声音，以及把工程里记的声音换成当前接口认得的 id。
// 工程换了语音账号后，原来记的声音在新接口里可能不存在，这时按性别挑新接口里相近的默认声音，而不是把旧 id 原样发过去报错。

package video

import "strings"

// speechVoice 是一家语音接口里能选的一个声音。Female 只用来在换接口时挑性别相近的声音。
type speechVoice struct {
	ID     string
	Name   string
	Female bool
}

// speechVoiceCatalog 按路由列出常用声音。每家先列女声再列男声，第一位女声和第一位男声是换接口时的默认声音。
var speechVoiceCatalog = map[string][]speechVoice{
	"openai": {
		{"alloy", "Alloy · 中性", true},
		{"nova", "Nova · 女声", true},
		{"shimmer", "Shimmer · 明亮女声", true},
		{"coral", "Coral · 温暖女声", true},
		{"sage", "Sage · 沉静女声", true},
		{"echo", "Echo · 男声", false},
		{"onyx", "Onyx · 低沉男声", false},
		{"fable", "Fable · 英式男声", false},
		{"ash", "Ash · 男声", false},
	},
	"minimax": {
		{"presenter_female", "女主持人", true},
		{"female-chengshu", "成熟女性", true},
		{"female-tianmei", "甜美女性", true},
		{"female-shaonv", "少女", true},
		{"female-yujie", "御姐", true},
		{"audiobook_female_1", "女声有声书", true},
		{"presenter_male", "男主持人", false},
		{"male-qn-jingying", "精英青年", false},
		{"male-qn-qingse", "青涩青年", false},
		{"male-qn-badao", "霸道青年", false},
		{"male-qn-daxuesheng", "青年大学生", false},
		{"audiobook_male_1", "男声有声书", false},
	},
	"volcengine": {
		{"zh_female_shuangkuaisisi_moon_bigtts", "爽快思思", true},
		{"zh_female_wanwanxiaohe_moon_bigtts", "湾湾小何", true},
		{"BV001_streaming", "通用女声", true},
		{"BV700_streaming", "灿灿", true},
		{"zh_male_wennuanahu_moon_bigtts", "温暖阿虎", false},
		{"zh_male_jingqiangkanye_moon_bigtts", "京腔侃爷", false},
		{"BV002_streaming", "通用男声", false},
		{"BV004_streaming", "开朗青年", false},
	},
	"dashscope": {
		{"Cherry", "芊悦 · 女声", true},
		{"Jennifer", "詹妮弗 · 女声", true},
		{"Sunny", "晴儿 · 四川话", true},
		{"Jada", "阿珍 · 上海话", true},
		{"Ethan", "晨煦 · 男声", false},
		{"Ryan", "甜茶 · 男声", false},
		{"Dylan", "晓东 · 北京话", false},
		{"Rocky", "阿强 · 粤语", false},
	},
	"cosyvoice": {
		{"longxiaochun_v2", "龙小淳 · 女声", true},
		{"longwan_v2", "龙婉 · 女声", true},
		{"longyue_v2", "龙悦 · 女声", true},
		{"longxiaoxia_v2", "龙小夏 · 女声", true},
		{"longcheng_v2", "龙橙 · 男声", false},
		{"longhua_v2", "龙华 · 男声", false},
		{"longshu_v2", "龙书 · 男声", false},
	},
	"tencent": {
		{"101001", "智瑜 · 情感女声", true},
		{"501001", "智兰 · 资讯女声", true},
		{"501004", "月华 · 聊天女声", true},
		{"502004", "智小满 · 营销女声", true},
		{"101013", "智辉 · 新闻男声", false},
		{"501000", "智斌 · 阅读男声", false},
		{"501005", "飞镜 · 聊天男声", false},
		{"502005", "智小解 · 解说男声", false},
	},
	"spark": {
		{"xiaoyan", "小燕 · 女声", true},
		{"aisjinger", "小婧 · 女声", true},
		{"aisxping", "小萍 · 女声", true},
		{"aisjiuxu", "许久 · 男声", false},
		{"aisbabyxu", "许小宝 · 童声", false},
	},
	"elevenlabs": {
		{"21m00Tcm4TlvDq8ikWAM", "Rachel · 女声", true},
		{"EXAVITQu4vr4xnSDxMaL", "Bella · 女声", true},
		{"MF3mGyEYCl7XYWbV9V6O", "Elli · 女声", true},
		{"pNInz6obpgDQGcFmaJgB", "Adam · 男声", false},
		{"ErXwobaYiN019PkySvjV", "Antoni · 男声", false},
		{"TxGEqnHWrfWFTfGW9XjX", "Josh · 男声", false},
	},
	"azure": {
		{"zh-CN-XiaoxiaoNeural", "晓晓 · 女声", true},
		{"zh-CN-XiaoyiNeural", "晓伊 · 女声", true},
		{"zh-CN-XiaochenNeural", "晓辰 · 女声", true},
		{"zh-CN-XiaohanNeural", "晓涵 · 女声", true},
		{"zh-CN-YunxiNeural", "云希 · 男声", false},
		{"zh-CN-YunyangNeural", "云扬 · 新闻男声", false},
		{"zh-CN-YunjianNeural", "云健 · 男声", false},
		{"zh-CN-YunzeNeural", "云泽 · 男声", false},
	},
	"edge": {
		{"zh-CN-XiaoxiaoNeural", "晓晓 · 女声", true},
		{"zh-CN-XiaoyiNeural", "晓伊 · 女声", true},
		{"zh-CN-YunxiNeural", "云希 · 男声", false},
		{"zh-CN-YunyangNeural", "云扬 · 新闻男声", false},
		{"zh-CN-YunjianNeural", "云健 · 男声", false},
		{"zh-CN-YunxiaNeural", "云夏 · 少年", false},
	},
}

// speechVendorNames 是路由对应的厂商名，写进工程和界面。
var speechVendorNames = map[string]string{
	"openai":     "OpenAI 兼容语音",
	"minimax":    "MiniMax",
	"volcengine": "火山引擎",
	"dashscope":  "通义千问",
	"cosyvoice":  "通义 CosyVoice",
	"tencent":    "腾讯云",
	"spark":      "讯飞",
	"elevenlabs": "ElevenLabs",
	"azure":      "Azure",
	"edge":       "本机 edge-tts",
}

// knownVoice 在所有目录里找这个 id，找到时返回声音和它所在的路由。
func knownVoice(id string) (speechVoice, string, bool) {
	id = strings.TrimSpace(id)
	for route, list := range speechVoiceCatalog {
		for _, v := range list {
			if v.ID == id {
				return v, route, true
			}
		}
	}
	return speechVoice{}, "", false
}

// voiceInRoute 判断这个 id 在这条路由的目录里。
func voiceInRoute(route, id string) (speechVoice, bool) {
	for _, v := range speechVoiceCatalog[route] {
		if v.ID == id {
			return v, true
		}
	}
	return speechVoice{}, false
}

// defaultVoice 返回这条路由里第一位女声或第一位男声。目录里没有这个性别就返回第一位。
func defaultVoice(route string, female bool) speechVoice {
	list := speechVoiceCatalog[route]
	for _, v := range list {
		if v.Female == female {
			return v
		}
	}
	if len(list) > 0 {
		return list[0]
	}
	return speechVoice{}
}

// resolveSpeechVoice 把工程记的声音换成这条路由认得的声音。
// 在这条路由目录里就原样用；是别家目录里的声音就按性别换成这家的默认声音；空着用这家的默认女声；
// 哪个目录都没有的 id（自己复刻的声音、网关自定义的声音）原样发过去。
// route 是 speechRoute 的结果。voice 是工程或界面里选的声音 id。
func resolveSpeechVoice(route, voice string) speechVoice {
	voice = strings.TrimSpace(voice)
	if voice == "" {
		return defaultVoice(route, true)
	}
	if v, ok := voiceInRoute(route, voice); ok {
		return v
	}
	if v, _, ok := knownVoice(voice); ok {
		return defaultVoice(route, v.Female)
	}
	return speechVoice{ID: voice, Name: voice, Female: true}
}

// speechVoiceRows 是给界面的声音列表：id 和中文名。
func speechVoiceRows(route string) []map[string]string {
	list := speechVoiceCatalog[route]
	out := make([]map[string]string, 0, len(list))
	for _, v := range list {
		out = append(out, map[string]string{"id": v.ID, "name": v.Name})
	}
	return out
}
