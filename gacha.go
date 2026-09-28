// Package gacha 是抽卡池：基础概率、软保底曲线、硬保底与跨会话的保底进度。
// 缺陷：软保底从第 75 抽才开始（差一）、出稀有后不清零保底、跨会话只保存总抽数丢掉保底进度。
package gacha

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
)

// ErrState 表示存档数据不合法。
var ErrState = errors.New("gacha: 保底存档不合法")

// 池子参数：基础 0.6%，第 74 抽起每次 +6%，第 90 抽必出。
const (
	BaseRate = 0.006
	SoftAt   = 74
	SoftStep = 0.06
	HardAt   = 90
)

// State 是保底进度。
type State struct {
	Pity  int // 距离上次稀有的抽数
	Total int // 累计抽数
}

// Pool 是一个卡池。
type Pool struct {
	state State
	rng   *rand.Rand
}

// NewPool 建池。
func NewPool(seed int64) *Pool {
	return &Pool{rng: rand.New(rand.NewSource(seed))}
}

// State 返回当前保底进度。
func (p *Pool) State() State {
	return p.state
}

// RateAt 返回「已经垫了 pity 抽之后，下一抽」的稀有概率。
// 缺陷：软保底写成 pity > SoftAt，第 74 抽享受不到提升。
func (p *Pool) RateAt(pity int) float64 {
	if pity >= HardAt {
		return 1
	}
	if pity > SoftAt {
		rate := BaseRate + float64(pity-SoftAt)*SoftStep
		if rate > 1 {
			return 1
		}
		return rate
	}
	return BaseRate
}

// Draw 抽一次，返回是否出稀有与本抽的保底计数。
// 缺陷：出稀有后忘了把 Pity 清零。
func (p *Pool) Draw() (bool, int) {
	p.state.Pity++
	p.state.Total++
	if p.rng.Float64() < p.RateAt(p.state.Pity) {
		pity := p.state.Pity
		return true, pity
	}
	return false, p.state.Pity
}

// Export 导出保底存档。
// 缺陷：只写了总抽数，保底进度丢掉。
func (p *Pool) Export() string {
	return fmt.Sprintf("total=%d", p.state.Total)
}

// Import 恢复保底存档。
// 缺陷：只恢复总抽数，Pity 置 0（换会话/换池后保底归零）。
func (p *Pool) Import(data string) error {
	total := 0
	for _, field := range strings.Split(data, ";") {
		key, value, found := strings.Cut(strings.TrimSpace(field), "=")
		if !found || key != "total" {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return ErrState
		}
		total = parsed
	}
	p.state = State{Total: total}
	return nil
}

// SetPity 直接设置保底进度（模拟「玩家已经垫了 85 抽」）。
func (p *Pool) SetPity(pity int) error {
	if pity < 0 {
		return ErrState
	}
	p.state.Pity = pity
	return nil
}
