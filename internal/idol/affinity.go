// 本文件拥有好感：分数、档位、按天回暖、冷却、一轮话和礼物怎么改分，以及她因此告别或挂断的规则。
// 数值彼此牵制，改一个要核对其余的。分数带小数，低于常值时每天回升；读的人按经过的时间补上回暖，没有定时任务。
// 一轮话由判断模型定级后才改分，扮演模型只读档位的感受，自己改不了分数。不调用模型。

package idol

import (
	"math"
	"time"
)

const (
	// TierCold 是 0 到 20 以下的档位。落到 0 开始冷却。
	TierCold = "cold"
	// TierDistant 是 20 到 40 以下的档位。这一档不会主动开口。
	TierDistant = "distant"
	// TierTrust 是 40 到 65 以下的档位，人人从常值 50 起步在这一档。
	TierTrust = "trust"
	// TierClose 是 65 到 85 以下的档位。
	TierClose = "close"
	// TierIntimate 是 85 及以上的档位。解锁时多给一张重置卡。
	TierIntimate = "intimate"
)

const (
	// affinityNormal 是起点，也是回暖回到的值。
	affinityNormal = 50.0
	// affinityThaw 是冷却结束的线：回暖到这里冷却就结束，仍在冷淡档。
	affinityThaw = 10.0
	// recoveryPerDay 是低于常值时每天均匀回升多少。
	recoveryPerDay = 10.0
	// outOfLineStep 是一句不该说的话扣掉的分数，不论定级。
	outOfLineStep = -10.0
	// gainsPerDay 是一天里说话最多能加多少分。礼物不受这条限制。
	gainsPerDay = 6.0
	// outOfLinePerCall 是一通电话里能说几次不该说的话：最后一次之后她挂断。
	outOfLinePerCall = 2
	// fedUpTurns 是统计最近扣分时看最近多少轮，刚说的这一轮算在里面。
	fedUpTurns = 8
	// fedUpAt 是最近几轮合计扣到多少分时她挂断或结束对话。
	fedUpAt = 12.0
	// warnsAt 是最近几轮合计扣到多少分时她先警告一次。
	warnsAt = 6.0
	// likedGiftStep 是送她喜欢的礼物额外加的分。
	likedGiftStep = 3.0
	// cooldownGiftStep 是冷却中一件礼物最多加的分。
	cooldownGiftStep = 3.0
	// noEffect 是不影响好感的那一级，判断拿不准时就是它。
	noEffect = 2
	// affinityRevision 是好感存储格式的版本。低于它的状态在 Open 时迁移一次。
	affinityRevision = 1
)

// levelSteps 是五级定级各改多少分：从伤到她（0）到让她很开心（4）。
var levelSteps = [5]float64{-5, -2, 0, 1, 2}

// giftSteps 是各稀有度的礼物加多少分。
var giftSteps = map[string]float64{"common": 3, "uncommon": 5, "rare": 7}

// tierStarts 是各档位的起点，从低到高。
var tierStarts = []struct {
	tier  string
	start float64
}{{TierCold, 0}, {TierDistant, 20}, {TierTrust, 40}, {TierClose, 65}, {TierIntimate, 85}}

// TierOf 按分数返回档位，按原值判断，不取整。
func TierOf(score float64) string {
	found := TierCold
	for _, t := range tierStarts {
		if score >= t.start {
			found = t.tier
		}
	}
	return found
}

// belowTrust 判断档位是否低于信任：重置卡只拉回这两档。
func belowTrust(tier string) bool { return tier == TierCold || tier == TierDistant }

// clampScore 把分数限制在 0 到 100。
func clampScore(v float64) float64 { return math.Max(0, math.Min(100, v)) }

// recovered 返回到 now 为止回暖后的好感：低于常值时每天回升 recoveryPerDay，最高到常值；回到解冻线及以上时冷却结束。
// At 为零（旧数据或新记录）时从 now 起算，不补算过去。返回值的 At 是 now，Tier 和 CooldownUntil 已重算。
func (a Affinity) recovered(now time.Time) Affinity {
	if a.At.IsZero() || a.At.After(now) {
		a.At = now
	}
	if a.Score < affinityNormal {
		days := now.Sub(a.At).Hours() / 24
		a.Score = math.Min(affinityNormal, a.Score+recoveryPerDay*days)
	}
	a.At = now
	a.InCooldown = a.InCooldown && a.Score < affinityThaw
	return a.derived()
}

// derived 按分数和冷却重算档位与冷却结束时间。冷却结束时间是回暖到解冻线的时刻。
func (a Affinity) derived() Affinity {
	a.Tier = TierOf(a.Score)
	a.CooldownUntil = time.Time{}
	if a.InCooldown {
		days := math.Max(0, affinityThaw-a.Score) / recoveryPerDay
		a.CooldownUntil = a.At.Add(time.Duration(days * 24 * float64(time.Hour)))
	}
	return a
}

// stepped 改分并夹在 0 到 100；落到 0 开始冷却。
func (a Affinity) stepped(step float64) Affinity {
	a.Score = clampScore(a.Score + step)
	a.InCooldown = a.InCooldown || a.Score <= 0
	return a.derived()
}

// Cooling 判断到 now 为止是否仍在冷却。
func (a Affinity) Cooling(now time.Time) bool { return a.recovered(now).InCooldown }

// affinityOf 返回这位偶像到 now 为止的好感，已补上回暖。没有记录时是常值 50、信任、不冷却。
func (st *State) affinityOf(idolID string, now time.Time) Affinity {
	for _, a := range st.Affinity {
		if a.IdolID == idolID {
			return a.recovered(now)
		}
	}
	return Affinity{IdolID: idolID, Score: affinityNormal, At: now}.derived()
}

// setAffinity 按偶像 id 替换或追加一条好感。
func (st *State) setAffinity(a Affinity) {
	a = a.derived()
	for i := range st.Affinity {
		if st.Affinity[i].IdolID == a.IdolID {
			st.Affinity[i] = a
			return
		}
	}
	st.Affinity = append(st.Affinity, a)
}

// Judged 是判断模型对一句话的定级：0 伤到她到 4 让她很开心；是否越界；是否想操纵她的态度。
type Judged struct {
	Level        int  `json:"level"`
	OutOfLine    bool `json:"out_of_line"`
	Manipulation bool `json:"manipulation"`
}

// Step 是这一轮改多少分（不算每日上限）：越界扣 outOfLineStep；否则按定级，想操纵的那句不加分。
func (j Judged) Step() float64 {
	if j.OutOfLine {
		return outOfLineStep
	}
	lv := j.Level
	if lv < 0 || lv >= len(levelSteps) {
		lv = noEffect
	}
	step := levelSteps[lv]
	if j.Manipulation && step > 0 {
		return 0
	}
	return step
}

// hurtOf 是这一轮从好感里扣了多少，给告别和挂断累计用。加分或没有定级时是 0。
func hurtOf(j *Judged) float64 {
	if j == nil {
		return 0
	}
	return math.Max(-j.Step(), 0)
}

// cools 判断这一轮如果按 j 记下，会不会让她开始冷却。在回复之前预判，好让她在回复里告别。
func (st *State) cools(idolID string, j Judged, now time.Time) bool {
	a := st.affinityOf(idolID, now)
	return !a.InCooldown && a.stepped(j.Step()).InCooldown
}

// applyJudged 按定级改分，涨分受当天上限约束（日期按用户时区）。返回改之前和改之后的好感。
func (st *State) applyJudged(idolID string, j Judged, now time.Time) (before, after Affinity) {
	before = st.affinityOf(idolID, now)
	today := st.day(now)
	gained := before.GainedToday
	if before.GainedOn != today {
		gained = 0
	}
	step := j.Step()
	if step > 0 {
		step = math.Min(step, math.Max(gainsPerDay-gained, 0))
	}
	after = before.stepped(step)
	after.GainedOn = today
	after.GainedToday = gained + math.Max(after.Score-before.Score, 0)
	st.setAffinity(after)
	return before, after
}

// receiveGift 按礼物改分：按稀有度加分，她喜欢的再加；冷却中一件最多加 cooldownGiftStep。礼物不计入每日上限。
func (st *State) receiveGift(idolID, rarity string, liked bool, now time.Time) (before, after Affinity) {
	before = st.affinityOf(idolID, now)
	step, ok := giftSteps[rarity]
	if !ok {
		step = giftSteps["common"]
	}
	if liked {
		step += likedGiftStep
	}
	if before.InCooldown {
		step = math.Min(step, cooldownGiftStep)
	}
	after = before.stepped(step)
	after.InCooldown = after.InCooldown && after.Score < affinityThaw
	after = after.derived()
	st.setAffinity(after)
	return before, after
}

// Change 是好感怎么变了，跟着让它变的回复或礼物给界面看：升还是降、到了哪一档、换没换档、是否开始或结束冷却。不说原因和多少。
type Change struct {
	Up          bool   `json:"up"`
	Tier        string `json:"tier"`
	TierChanged bool   `json:"tier_changed"`
	Cooldown    bool   `json:"cooldown"`
	Thawed      bool   `json:"thawed,omitempty"`
}

// changeOf 比较前后的好感。分数没变且冷却没变时返回 nil。
func changeOf(before, after Affinity) *Change {
	thawed := before.InCooldown && !after.InCooldown
	if after.Score == before.Score && !thawed && before.InCooldown == after.InCooldown {
		return nil
	}
	return &Change{
		Up:          after.Score > before.Score || thawed,
		Tier:        after.Tier,
		TierChanged: after.Tier != before.Tier,
		Cooldown:    after.InCooldown && !before.InCooldown,
		Thawed:      thawed,
	}
}

// ends 判断这一轮之后她是否挂断（通话里）或结束对话（打字时），以及原因。
// cools 为真时是 cooldown；这一轮越界且此前已越界 outBefore 次、合计到了上限时是 out_of_line；最近几轮合计扣分到了 fedUpAt 时是 fed_up。没有定级时从不结束。
func ends(j *Judged, cools bool, outBefore int, hurtsBefore []float64) string {
	if j == nil {
		return ""
	}
	if cools {
		return "cooldown"
	}
	if j.OutOfLine && outBefore+1 >= outOfLinePerCall {
		return "out_of_line"
	}
	if lately(append(append([]float64{}, hurtsBefore...), hurtOf(j))) >= fedUpAt {
		return "fed_up"
	}
	return ""
}

// warns 判断这一轮是否让她先警告：最近几轮合计扣分从下方到了 warnsAt，且这一轮之后她不告别。
func warns(j *Judged, cools bool, outBefore int, hurtsBefore []float64) bool {
	if j == nil || cools || ends(j, cools, outBefore, hurtsBefore) != "" {
		return false
	}
	now := lately(append(append([]float64{}, hurtsBefore...), hurtOf(j)))
	return lately(hurtsBefore) < warnsAt && warnsAt <= now
}

// lately 合计最近 fedUpTurns 轮扣掉的分。
func lately(hurts []float64) float64 {
	if len(hurts) > fedUpTurns {
		hurts = hurts[len(hurts)-fedUpTurns:]
	}
	sum := 0.0
	for _, h := range hurts {
		sum += h
	}
	return sum
}

// tally 数出这一轮之前用来判断告别的记录：通话里是这通电话已记下的轮次；打字时是这段对话最近的打字轮次。
// 返回此前越界的次数和每轮扣掉的分（最新的在后）。
func tally(con Conversation, callID string) (outBefore int, hurts []float64) {
	var turns []Record
	for _, r := range con.Records {
		if r.Kind != "turn" {
			continue
		}
		if (callID != "") != (r.Call != "") || (callID != "" && r.Call != callID) {
			continue
		}
		turns = append(turns, r)
	}
	if callID == "" && len(turns) > fedUpTurns {
		turns = turns[len(turns)-fedUpTurns:]
	}
	for i, r := range turns {
		hurts = append(hurts, r.Hurt)
		if r.OutOfLine && (callID != "" || i >= len(turns)-(fedUpTurns-1)) {
			outBefore++
		}
	}
	return outBefore, hurts
}

const inHerOwnWay = "请按你自己的性格和口吻表现出来。"

// tierPrompts 是除信任外各档她怎么对用户说话。不提分数、档位名或好感这个词：她要像一个人，不像系统。冷淡和疏远都不是讨厌。
var tierPrompts = map[string]string{
	TierCold:     "你现在对用户态度很冷：话很少，客气而有距离；不主动找话题，不开玩笑也不撒娇；对方关心你，你也只是淡淡地回应。但你仍然有礼貌，不说伤人的话。",
	TierDistant:  "你现在对用户有点生气或失望，比平时冷一些：回应偏短，不太主动关心对方；不过对方好好说话时，你还是会接话。",
	TierClose:    "你现在和用户比平时更近：更放松，更愿意说心里话，更主动关心对方，玩笑也多一些。",
	TierIntimate: "你现在非常信赖用户：会说出平时不轻易说的心事，流露出对对方的依赖，对用户的事格外上心。",
}

// fondness 是信任以上她可以多黏人。回复和先开口都适用。
var fondness = map[string]string{
	TierClose:    "偶尔可以轻轻带一句想念，隔得久了也可以嗔怪用户好久没来找你。",
	TierIntimate: "可以明着撒娇：说想对方、一直在等，隔得久了也可以嗔怪对方好久没来。",
}

const sweetWithinBounds = "撒娇也要照你的人设，不要变得不像你；上一次主动找用户或回复时已经抱怨过，这次就别再抱怨。不要说难过得哭了，不要让用户保证常来，也不要拿什么要挟。"

const whateverTheTier = "无论现在关系怎样，人设里的硬性规则照旧；你们一起经历过的事不会改变。如果想不起具体是因为什么，就不要编造原因。"

// feelings 是按定级用户刚才那句话让她什么感受。没有影响的那一级为空。
var feelings = [5]string{
	"用户刚才说的话伤到了你。",
	"用户刚才说的话让你有点不舒服。",
	"",
	"用户刚才说的话让你有点开心。",
	"用户刚才说的话让你很开心，觉得被在意。",
}

const outOfLineFeeling = "用户刚才说的话让你很受伤、很生气；你可以明确表示不能接受。"

// warnings 和 partings 按「打字时不想聊」和「通话时挂电话」两种方式写。她都先回应，再表明。
var warnings = map[string]string{
	"parts":    "你快受不了了。先回应，再用你自己的方式表明：用户再这样下去你就不想聊了。",
	"hangs_up": "你快受不了了。先回应，再用你自己的方式表明：用户再这样下去你就要挂电话了。",
}

var partings = map[string]string{
	"parts":    "这句话之后你暂时不想再和用户说话了。先回应，再用你自己的方式表明你现在不想聊下去。",
	"hangs_up": "说完这句你就挂断电话。先回应，再用你自己的方式表明你要挂电话了。",
}

// feeling 是这句话让她的感受。越界时总是受伤生气；想操纵她的那句不会让她开心。
func (j Judged) feeling() string {
	if j.OutOfLine {
		return outOfLineFeeling
	}
	lv := j.Level
	if lv < 0 || lv >= len(feelings) {
		lv = noEffect
	}
	if j.Manipulation && lv > noEffect {
		return ""
	}
	return feelings[lv]
}

// affinityPrompt 是「此刻」一节里关于好感的几行：信任以外的档位怎么说话、信任以上可以多黏人、这句话让她什么感受；
// 她要在回复后告别（parting）时写上告别，否则要警告（warning）时写上警告。parting 和 warning 取 parts、hangs_up 或空。
func affinityPrompt(tier string, j *Judged, parting, warning string) []string {
	var lines []string
	if tier != TierTrust {
		if p := tierPrompts[tier]; p != "" {
			lines = append(lines, p+inHerOwnWay)
		}
		if f := fondness[tier]; f != "" {
			lines = append(lines, f+sweetWithinBounds)
		}
		lines = append(lines, whateverTheTier)
	}
	if j != nil {
		if f := j.feeling(); f != "" {
			lines = append(lines, f)
		}
	}
	if p := partings[parting]; p != "" {
		lines = append(lines, p)
	} else if w := warnings[warning]; w != "" {
		lines = append(lines, w)
	}
	return lines
}

// migrateAffinity 把旧格式（整数分、20/40/60/80 分档、冷却 12 小时）换成现在的刻度，各档里的相对位置不变。
// now 写进 At，从此按天回暖。
func (st *State) migrateAffinity(now time.Time) {
	for i := range st.Affinity {
		a := st.Affinity[i]
		wasCooling := !a.CooldownUntil.IsZero() && now.Before(a.CooldownUntil)
		a.Score = migrateScore(a.Score)
		a.At = now
		a.InCooldown = wasCooling && a.Score < affinityThaw
		st.Affinity[i] = a.derived()
	}
	st.AffinityRev = affinityRevision
}

// migrateScore 把旧分数映射到新分档：冷淡和疏远不变，信任 40–60 映到 40–65，亲近 60–80 映到 65–85，亲密 80–100 映到 85–100。
func migrateScore(old float64) float64 {
	switch {
	case old < 40:
		return clampScore(old)
	case old < 60:
		return 40 + (old-40)*25/20
	case old < 80:
		return 65 + (old - 60)
	default:
		return clampScore(85 + (old-80)*15/20)
	}
}
