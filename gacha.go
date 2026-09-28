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

// RateAt 返回「已经垫了 pity 抽之后，下一抽」的稀有概率。
// 下一抽是第 pity+1 抽：第 1..73 抽是基础概率，第 74 抽起每抽 +SoftStep，
// 第 90 抽必出，概率恒在 [0, 1]。
func (p *Pool) RateAt(pity int) float64 {
	if pity >= HardAt-1 {
		return 1
	}
	if pity >= SoftAt-1 {
		rate := BaseRate + float64(pity-SoftAt+2)*SoftStep
		if rate > 1 {
			return 1
		}
		return rate
	}
	return BaseRate
}

// Draw 抽一次，返回是否出稀有与本抽的保底计数。
// 出稀有后保底计数清零，Total 只增不减。
func (p *Pool) Draw() (bool, int) {
	pity := p.state.Pity + 1
	p.state.Total++
	if p.rng.Float64() < p.RateAt(p.state.Pity) {
		p.state.Pity = 0
		return true, pity
	}
	p.state.Pity = pity
	return false, pity
}

// Export 导出保底存档。
func (p *Pool) Export() string {
	return fmt.Sprintf("pity=%d;total=%d", p.state.Pity, p.state.Total)
}

// Import 恢复保底存档。
func (p *Pool) Import(data string) error {
	var state State
	seenPity, seenTotal := false, false
	for _, field := range strings.Split(data, ";") {
		key, value, found := strings.Cut(strings.TrimSpace(field), "=")
		if !found {
			return ErrState
		}
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 0 {
			return ErrState
		}
		switch key {
		case "pity":
			state.Pity = parsed
			seenPity = true
		case "total":
			state.Total = parsed
			seenTotal = true
		default:
			return ErrState
		}
	}
	if !seenPity || !seenTotal {
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
