// FSRS-6 记忆模型：稳定度、难度和可提取率的纯计算。不读写库，也不决定什么时候复习。
// 公式和 21 个默认权重取自 open-spaced-repetition 的 FSRS-6。练习题和已掌握的知识点共用这一份。

package learn

import (
	"math"
	"time"
)

// fsrsWeights 是 FSRS-6 的默认权重。w0 到 w3 是四个评分的初始稳定度，w20 是遗忘曲线的衰减。
var fsrsWeights = [21]float64{
	0.212, 1.2931, 2.3065, 8.2956, 6.4133, 0.8334, 3.0194, 0.001, 1.8722, 0.1666, 0.796,
	1.4835, 0.0614, 0.2629, 1.6483, 0.6014, 1.8729, 0.5425, 0.0912, 0.0658, 0.1542,
}

const (
	fsrsSMin = 0.001
	fsrsSMax = 36500.0
	fsrsDMin = 1.0
	fsrsDMax = 10.0

	// DefaultRetention 是默认的目标记住率。到期那一刻预计还记得九成。
	DefaultRetention = 0.9
	// MinRetention 是目标记住率的下限。
	MinRetention = 0.7
	// MaxRetention 是目标记住率的上限。再高间隔会短到没有意义。
	MaxRetention = 0.97
)

// Rating 是一次复习的评分，1 到 4 依次是重来、吃力、记得、轻松。
type Rating int

const (
	// RatingAgain 表示没想起来。
	RatingAgain Rating = 1
	// RatingHard 表示想起来了但吃力。
	RatingHard Rating = 2
	// RatingGood 表示正常想起来。
	RatingGood Rating = 3
	// RatingEasy 表示毫不费力。
	RatingEasy Rating = 4
)

// Memory 是一条记忆的状态。Stability 是记住率降到九成所需的天数，0 表示还没复习过。Difficulty 在 1 到 10 之间。
type Memory struct {
	Stability  float64 `json:"stability"`
	Difficulty float64 `json:"difficulty"`
}

// Seen 在这条记忆已经有过一次评分时为真。
func (m Memory) Seen() bool { return m.Stability > 0 }

// ParseRating 把 again、hard、good、easy 收成评分。其余文本返回 false。
// s 按原样比较，调用方先转小写。
func ParseRating(s string) (Rating, bool) {
	switch s {
	case "again":
		return RatingAgain, true
	case "hard":
		return RatingHard, true
	case "good":
		return RatingGood, true
	case "easy":
		return RatingEasy, true
	}
	return 0, false
}

// String 返回评分的英文名。未知评分返回空串。
func (r Rating) String() string {
	switch r {
	case RatingAgain:
		return "again"
	case RatingHard:
		return "hard"
	case RatingGood:
		return "good"
	case RatingEasy:
		return "easy"
	}
	return ""
}

// ClampRetention 把目标记住率夹进 MinRetention 和 MaxRetention 之间。r 小于等于 0 或不是数时返回 DefaultRetention。
func ClampRetention(r float64) float64 {
	if math.IsNaN(r) || r <= 0 {
		return DefaultRetention
	}
	return math.Min(math.Max(r, MinRetention), MaxRetention)
}

// fsrsCurve 返回遗忘曲线的衰减和系数。系数的取法让间隔等于稳定度时记住率正好是九成。
func fsrsCurve() (decay, factor float64) {
	decay = -fsrsWeights[20]
	factor = math.Pow(0.9, 1/decay) - 1
	return decay, factor
}

func clampStability(s float64) float64 { return math.Min(math.Max(s, fsrsSMin), fsrsSMax) }

func clampDifficulty(d float64) float64 { return math.Min(math.Max(d, fsrsDMin), fsrsDMax) }

// Retrievability 返回隔了 elapsedDays 天之后还记得的概率。stability 先夹进合法范围。elapsedDays 小于 0 时按 0 算。
func Retrievability(elapsedDays, stability float64) float64 {
	if elapsedDays < 0 {
		elapsedDays = 0
	}
	decay, factor := fsrsCurve()
	return math.Pow(1+factor*elapsedDays/clampStability(stability), decay)
}

// FSRSInterval 返回记住率降到 retention 所需的天数，不取整。retention 先交给 ClampRetention。
// stability 先夹进合法范围。retention 是九成时结果等于稳定度。
func FSRSInterval(stability, retention float64) float64 {
	decay, factor := fsrsCurve()
	return clampStability(stability) / factor * (math.Pow(ClampRetention(retention), 1/decay) - 1)
}

// initDifficulty 是第一次评分为 r 时的难度，还没夹进范围。
func initDifficulty(r Rating) float64 {
	return fsrsWeights[4] - math.Exp(fsrsWeights[5]*float64(r-1)) + 1
}

// NextMemory 按一次评分推进记忆状态。cur 还没复习过时用初始值。
// elapsedDays 是距上次复习的整天数。为 0 时走当天复习的公式，记得以上的评分不会让稳定度变小。
// rating 不在 1 到 4 之间时按 RatingGood 处理。
func NextMemory(cur Memory, elapsedDays float64, rating Rating) Memory {
	if rating < RatingAgain || rating > RatingEasy {
		rating = RatingGood
	}
	w := fsrsWeights
	if !cur.Seen() {
		return Memory{
			Stability:  clampStability(math.Min(math.Max(w[rating-1], 0.1), fsrsSMax)),
			Difficulty: clampDifficulty(initDifficulty(rating)),
		}
	}
	d, s := cur.Difficulty, clampStability(cur.Stability)
	// 难度向「轻松」的初始难度回归一点，避免连续答错后再也降不下来。
	delta := -w[6] * float64(rating-3)
	nextD := d + (10-d)*delta/9
	nextD = clampDifficulty(w[7]*initDifficulty(RatingEasy) + (1-w[7])*nextD)

	var nextS float64
	switch {
	case elapsedDays <= 0:
		inc := math.Exp(w[17]*(float64(rating-3)+w[18])) * math.Pow(s, -w[19])
		if rating >= RatingHard && inc < 1 {
			inc = 1
		}
		nextS = s * inc
	case rating == RatingAgain:
		r := Retrievability(elapsedDays, s)
		forgot := w[11] * math.Pow(d, -w[12]) * (math.Pow(s+1, w[13]) - 1) * math.Exp((1-r)*w[14])
		// 忘掉之后的稳定度不能比原来还高。
		nextS = math.Min(forgot, s/math.Exp(w[17]*w[18]))
	default:
		r := Retrievability(elapsedDays, s)
		hard, easy := 1.0, 1.0
		if rating == RatingHard {
			hard = w[15]
		}
		if rating == RatingEasy {
			easy = w[16]
		}
		nextS = s * (1 + math.Exp(w[8])*(11-d)*math.Pow(s, -w[9])*(math.Exp((1-r)*w[10])-1)*hard*easy)
	}
	return Memory{Stability: clampStability(nextS), Difficulty: nextD}
}

// MemoryFromSM2 把旧的 SM-2 间隔和易度折成 FSRS 状态，迁移旧练习题用。
// intervalDays 小于等于 0 或 ease 小于等于 1 时返回零值，表示当成没复习过。
func MemoryFromSM2(ease, intervalDays float64) Memory {
	if intervalDays <= 0 || ease <= 1 || math.IsNaN(ease) || math.IsNaN(intervalDays) {
		return Memory{}
	}
	w := fsrsWeights
	s := clampStability(intervalDays)
	d := 11 - (ease-1)/(math.Exp(w[8])*math.Pow(s, -w[9])*math.Expm1((1-DefaultRetention)*w[10]))
	if math.IsNaN(d) || math.IsInf(d, 0) {
		return Memory{}
	}
	return Memory{Stability: s, Difficulty: clampDifficulty(d)}
}

// elapsedWholeDays 返回 from 到 to 之间的整天数。from 是零值或晚于 to 时返回 0。
func elapsedWholeDays(from, to time.Time) float64 {
	if from.IsZero() || !to.After(from) {
		return 0
	}
	return math.Floor(to.Sub(from).Hours() / 24)
}

// dueAfter 返回 at 之后按稳定度和目标记住率算出的到期时刻。间隔取整到天，至少 minDays 天，最多一百年。
// 取整之后按钮上写的「3 天」就是三个整天，每天在差不多的时刻复习的人不会总是晚一天才见到这道题。
func dueAfter(at time.Time, m Memory, retention, minDays float64) time.Time {
	days := math.Round(FSRSInterval(m.Stability, retention))
	if days < minDays {
		days = minDays
	}
	if days > fsrsSMax {
		days = fsrsSMax
	}
	return at.Add(time.Duration(days * 24 * float64(time.Hour)))
}
