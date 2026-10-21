package idol

import (
	"math/rand/v2"
	"strings"
	"time"
)

type Gift struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Rarity   string `json:"rarity"`
	Category string `json:"category"`
}

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

func GiftByID(id string) (Gift, bool) {
	for _, g := range Catalog {
		if g.ID == id {
			return g, true
		}
	}
	return Gift{}, false
}

func (st *State) stock(id string) int {
	for _, s := range st.Inventory {
		if s.GiftID == id {
			return s.Count
		}
	}
	return 0
}

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

type DrawResult struct {
	Gift      *Gift `json:"gift,omitempty"`
	ResetCard bool  `json:"reset_card"`
	Left      int   `json:"left"`
}

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

func liked(idol Idol, giftID string) bool {
	for _, id := range idol.LikedGifts {
		if id == giftID {
			return true
		}
	}
	return false
}

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
		// gifts still land during a cooldown and can lift it
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
