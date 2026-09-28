// Package gacha 是抽卡池：基础概率、软保底曲线、硬保底与跨会话的保底进度。
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

// RateAt 返回第 pity 抽（已经垫了 pity-1 抽）的稀有概率。
// 第 1..73 抽是基础概率，第 74 抽起每抽 +SoftStep，第 90 抽必出。
func (p *Pool) RateAt(pity int) float64 {
	if pity >= HardAt {
		return 1
	}
	if pity >= SoftAt {
		rate := BaseRate + float64(pity-SoftAt+1)*SoftStep
		if rate > 1 {
			return 1
		}
		return rate
	}
	return BaseRate
}

// Draw 抽一次，返回是否出稀有与本抽的保底计数（出稀有前的垫刀数）。
// 出稀有后保底计数清零，Total 只增不减。
func (p *Pool) Draw() (bool, int) {
	p.state.Pity++
	p.state.Total++
	pity := p.state.Pity
	if p.rng.Float64() < p.RateAt(pity) {
		p.state.Pity = 0
		return true, pity
	}
	return false, pity
}

// Export 导出保底存档（含保底进度与总抽数）。
func (p *Pool) Export() string {
	return fmt.Sprintf("total=%d;pity=%d", p.state.Total, p.state.Pity)
}

// Import 恢复保底存档，保底进度与总抽数一起还原；数据非法时报 ErrState。
func (p *Pool) Import(data string) error {
	state := State{}
	for _, field := range strings.Split(data, ";") {
		key, value, found := strings.Cut(strings.TrimSpace(field), "=")
		if !found || (key != "total" && key != "pity") {
			continue
		}
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return ErrState
		}
		switch key {
		case "total":
			state.Total = parsed
		case "pity":
			state.Pity = parsed
		}
	}
	if state.Total < 0 || state.Pity < 0 {
		return ErrState
	}
	p.state = state
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
