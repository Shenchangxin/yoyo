// 本文件拥有礼物目录、抽卡、送礼和重置卡。Catalog 里的中文名是数据。
// 不调用模型。送礼改好感并写一条 kind 为 event 的记录，再记每日任务 gift。

package idol

import (
	"math/rand/v2"
	"strings"
	"time"
)

// Gift 是目录里的一件礼物。Rarity 取 common、uncommon、rare。
type Gift struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Rarity   string `json:"rarity"`
	Category string `json:"category"`
}

// Catalog 是可抽、可送的礼物。名字和稀有度是数据。
var Catalog = []Gift{
	{ID: "tea", Name: "热茶", Rarity: "common", Category: "drink"},
	{ID: "candy", Name: "糖", Rarity: "common", Category: "food"},
	{ID: "note", Name: "便笺", Rarity: "common", Category: "paper"},
	{ID: "ribbon", Name: "缎带", Rarity: "common", Category: "cloth"},
	{ID: "flower", Name: "一枝花", Rarity: "uncommon", Category: "flower"},
	{ID: "book", Name: "小书", Rarity: "uncommon", Category: "book"},
	{ID: "song", Name: "一首歌", Rarity: "uncommon", Category: "music"},
	{ID: "scarf", Name: "围巾", Rarity: "rare", Category: "cloth"},
	{ID: "star", Name: "玻璃星星", Rarity: "rare", Category: "charm"},
	{ID: "letter", Name: "一封信", Rarity: "rare", Category: "paper"},
}

// GiftByID 按 id 在 Catalog 里找礼物。没有则 ok 为 false。
func GiftByID(id string) (Gift, bool) {
	for _, g := range Catalog {
		if g.ID == id {
			return g, true
		}
	}
	return Gift{}, false
}

// stock 返回背包里这种礼物的数量，没有就是 0。
// id 与每条库存的 GiftID 比较，命中返回 Count，否则返回 0。
func (st *State) stock(id string) int {
	for _, s := range st.Inventory {
		if s.GiftID == id {
			return s.Count
		}
	}
	return 0
}

// addStock 增减背包数量。减到 0 以下就停在 0；没有这项且 n 大于 0 时追加一条。
// id 与每条库存的 GiftID 比较。没有这项且 n 大于 0 时，追加的 GiftID 就是它。
func (st *State) addStock(id string, n int) {
	for i := range st.Inventory {
		if st.Inventory[i].GiftID == id {
			st.Inventory[i].Count += n
			if st.Inventory[i].Count < 0 {
				st.Inventory[i].Count = 0
			}
			return
		}
	}
	if n > 0 {
		st.Inventory = append(st.Inventory, Stock{GiftID: id, Count: n})
	}
}

// DrawResult 是一次抽卡。ResetCard 为真时没有礼物，Left 是剩下的抽数。
type DrawResult struct {
	Gift      *Gift `json:"gift,omitempty"`
	ResetCard bool  `json:"reset_card"`
	Left      int   `json:"left"`
}

// Draw 消耗一次抽卡。先用 BonusDraws，没有再用 Draws。0 到 99 里小于 3 得到重置卡；否则 95 及以上 rare，70 及以上 uncommon，其余 common，再从该稀有度里随机一件放进背包。
// rng 调用 IntN(100) 决定重置卡和稀有度，再调用 IntN 从该稀有度里取一件。
func (st *State) Draw(rng *rand.Rand) (DrawResult, error) {
	left := st.Draws + st.BonusDraws
	if left <= 0 {
		return DrawResult{}, errf("no draws")
	}
	if st.BonusDraws > 0 {
		st.BonusDraws--
	} else {
		st.Draws--
	}
	if rng.IntN(100) < 3 {
		st.ResetCards++
		return DrawResult{ResetCard: true, Left: st.Draws + st.BonusDraws}, nil
	}
	roll := rng.IntN(100)
	rarity := "common"
	if roll >= 95 {
		rarity = "rare"
	} else if roll >= 70 {
		rarity = "uncommon"
	}
	var pool []Gift
	for _, g := range Catalog {
		if g.Rarity == rarity {
			pool = append(pool, g)
		}
	}
	if len(pool) == 0 {
		pool = Catalog
	}
	g := pool[rng.IntN(len(pool))]
	st.addStock(g.ID, 1)
	return DrawResult{Gift: &g, Left: st.Draws + st.BonusDraws}, nil
}

// giftDelta 计算送礼加的分数。common 是 2，uncommon 是 4，rare 是 7，喜欢的礼物再加 4。
// liked 为真时在稀有度分数上再加 4。
func giftDelta(g Gift, liked bool) int {
	n := 2
	switch g.Rarity {
	case "uncommon":
		n = 4
	case "rare":
		n = 7
	}
	if liked {
		n += 4
	}
	return n
}

// liked 看这位偶像的 LikedGifts 里有没有这件礼物。
// idol 的 LikedGifts 逐项与 giftID 比较。
// giftID 与 idol.LikedGifts 的每一项做相等比较。
func liked(idol Idol, giftID string) bool {
	for _, id := range idol.LikedGifts {
		if id == giftID {
			return true
		}
	}
	return false
}

// Give 从背包拿出一件礼物送给偶像，加上 giftDelta，并写一条 event 记录。当前对话已结束就另开一段。冷却中也继续送出。
// idolID 传给 idol、affinityOf、ApplyDelta、ensureCurrent 和 forceNew。
// giftID 传给 GiftByID、stock、addStock 和 liked，并写入记录的 GiftID。
// now 传给 Cooling、ApplyDelta、ensureCurrent、forceNew，写入记录的 At，并传给 markTask 和 unlock。
func (st *State) Give(idolID, giftID string, now time.Time) (Record, Affinity, error) {
	g, ok := GiftByID(giftID)
	if !ok {
		return Record{}, Affinity{}, errf("unknown gift")
	}
	if st.stock(giftID) <= 0 {
		return Record{}, Affinity{}, errf("not holding that gift")
	}
	idol, ok := st.idol(idolID)
	if !ok {
		return Record{}, Affinity{}, errf("idol not found")
	}
	if aff := st.affinityOf(idolID); aff.Cooling(now) {
		// 冷却期间礼物仍然送出，分数上来后可以解除冷却
	}
	st.addStock(giftID, -1)
	delta := giftDelta(g, liked(idol, giftID))
	aff := st.ApplyDelta(idolID, delta, now, 0)
	con := st.ensureCurrent(idolID, now)
	if con.Ended {
		con = st.forceNew(idolID, now)
	}
	rec := Record{
		ID: newID(), Kind: "event", At: now, GiftID: giftID, Delta: delta,
		Segments: []Segment{{Spoken: "收到了" + g.Name, Subtitle: "收到了" + g.Name}},
	}
	st.appendRecord(con.ID, rec)
	st.markTask(now, idolTZ(st.User), "gift")
	st.unlock(now)
	return rec, aff, nil
}

// UseReset 消耗一张重置卡，把分数低于 50 或仍在冷却的偶像设回 50 并清掉冷却。没有低于 trust 的人就返回错误，卡不消耗。返回值是改过的人数。
// now 传给 Cooling。
func (st *State) UseReset(now time.Time) (int, error) {
	if st.ResetCards <= 0 {
		return 0, errf("no reset card")
	}
	n := 0
	for _, idol := range st.Idols {
		a := st.affinityOf(idol.ID)
		if a.Score >= 50 && !a.Cooling(now) {
			continue
		}
		a.Score = 50
		a.CooldownUntil = time.Time{}
		st.setAffinity(a)
		n++
	}
	if n == 0 {
		return 0, errf("nobody is below trust")
	}
	st.ResetCards--
	return n, nil
}

// idolTZ 解析用户卡片上的时区。空的或 LoadLocation 失败时用 time.Local。
// u 的 Timezone 去掉空白后为空，或 LoadLocation 失败时返回 time.Local，否则返回加载到的时区。
func idolTZ(u UserCard) *time.Location {
	if strings.TrimSpace(u.Timezone) == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(strings.TrimSpace(u.Timezone))
	if err != nil {
		return time.Local
	}
	return loc
}
