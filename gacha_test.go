package gacha

import (
	"math"
	"testing"
	"time"
)

func TestBaseRate(t *testing.T) {
	pool := NewPool(1)
	if pool.RateAt(0) != BaseRate {
		t.Fatalf("第 1 抽应该是基础概率：%v", pool.RateAt(0))
	}
}

func TestHardPity(t *testing.T) {
	pool := NewPool(1)
	if pool.RateAt(HardAt) != 1 {
		t.Fatalf("第 %d 抽必须必出：%v", HardAt, pool.RateAt(HardAt))
	}
}

func TestRateNeverExceedsOne(t *testing.T) {
	pool := NewPool(1)
	for pity := 0; pity <= HardAt; pity++ {
		if rate := pool.RateAt(pity); rate < 0 || rate > 1 {
			t.Fatalf("概率越界：pity=%d rate=%v", pity, rate)
		}
	}
}

func TestDrawIncrementsTotal(t *testing.T) {
	pool := NewPool(2)
	pool.Draw()
	pool.Draw()
	if pool.State().Total != 2 {
		t.Fatalf("总抽数不对：%d", pool.State().Total)
	}
}

func TestSetPity(t *testing.T) {
	pool := NewPool(3)
	if err := pool.SetPity(85); err != nil {
		t.Fatalf("SetPity 出错：%v", err)
	}
	if pool.State().Pity != 85 {
		t.Fatalf("保底进度不对：%d", pool.State().Pity)
	}
	if err := pool.SetPity(-1); err != ErrState {
		t.Fatalf("负数应该报错：%v", err)
	}
}

func TestImportBadData(t *testing.T) {
	pool := NewPool(4)
	if err := pool.Import("total=abc"); err != ErrState {
		t.Fatalf("非法存档应该报错：%v", err)
	}
}

func TestSoftPityBoundary(t *testing.T) {
	pool := NewPool(1)
	cases := []struct {
		pity int
		want float64
	}{
		{73, BaseRate}, // 第 73 抽仍是基础概率
		{74, 0.066},    // 第 74 抽起 +0.06
		{75, 0.126},    // 每抽再 +0.06
		{89, 0.966},    // 第 89 抽
		{HardAt, 1},    // 第 90 抽必出
	}
	for _, c := range cases {
		if got := pool.RateAt(c.pity); math.Abs(got-c.want) > 1e-9 {
			t.Fatalf("第 %d 抽概率应为 %v，实际 %v", c.pity, c.want, got)
		}
	}
}

func TestDrawResetsPityOnRare(t *testing.T) {
	pool := NewPool(1)
	if err := pool.SetPity(HardAt - 1); err != nil {
		t.Fatalf("SetPity 出错：%v", err)
	}
	rare, _ := pool.Draw() // 第 90 抽必出
	if !rare {
		t.Fatal("第 90 抽必须出稀有")
	}
	if pool.State().Pity != 0 {
		t.Fatalf("出稀有后保底必须清零，实际 %d", pool.State().Pity)
	}
	if pool.State().Total != 1 {
		t.Fatalf("Total 只增不减，实际 %d", pool.State().Total)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	pool := NewPool(1)
	if err := pool.SetPity(85); err != nil {
		t.Fatalf("SetPity 出错：%v", err)
	}
	pool.Draw()
	pool.Draw()
	data := pool.Export()
	restored := NewPool(2)
	if err := restored.Import(data); err != nil {
		t.Fatalf("Import 出错：%v", err)
	}
	if restored.State() != pool.State() {
		t.Fatalf("往返后状态不一致：got %+v want %+v", restored.State(), pool.State())
	}
	if err := restored.Import("total=10;pity=-1"); err != ErrState {
		t.Fatalf("负数保底应该报错：%v", err)
	}
}

func TestDeterministicSameSeed(t *testing.T) {
	a, b := NewPool(7), NewPool(7)
	if err := a.SetPity(10); err != nil {
		t.Fatal(err)
	}
	if err := b.SetPity(10); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10000; i++ {
		ra, _ := a.Draw()
		rb, _ := b.Draw()
		if ra != rb {
			t.Fatalf("第 %d 抽结果不一致：%v vs %v", i+1, ra, rb)
		}
	}
}

func TestRareGapNeverExceedsHardPity(t *testing.T) {
	pool := NewPool(42)
	gap := 0
	for i := 0; i < 200000; i++ {
		gap++
		if rare, _ := pool.Draw(); rare {
			if gap > HardAt {
				t.Fatalf("稀有间隔 %d 超过硬保底 %d", gap, HardAt)
			}
			gap = 0
		}
	}
}

// 理论期望：稀有间隔 T 满足 E[T] = Σ_{k=1..HardAt} Π_{j<k}(1-rate_j)，
// 每抽稀有数期望为 1/E[T]。
func expectedRaresPerDraw(pool *Pool) float64 {
	expectation := 0.0
	survive := 1.0
	for k := 1; k <= HardAt; k++ {
		expectation += survive
		survive *= 1 - pool.RateAt(k)
	}
	return 1 / expectation
}

func TestRareRateMatchesTheory(t *testing.T) {
	pool := NewPool(2026)
	const draws = 100000
	rares := 0
	for i := 0; i < draws; i++ {
		if rare, _ := pool.Draw(); rare {
			rares++
		}
	}
	want := expectedRaresPerDraw(pool) * draws
	if diff := math.Abs(float64(rares)-want) / want; diff > 0.02 {
		t.Fatalf("稀有次数偏离理论期望超过 2%%：got %d want %.1f", rares, want)
	}
}

func TestDrawPerformance(t *testing.T) {
	pool := NewPool(99)
	start := time.Now()
	const draws = 10000000
	rares := 0
	for i := 0; i < draws; i++ {
		if rare, _ := pool.Draw(); rare {
			rares++
		}
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("10^7 次抽取耗时 %v，超过 2 秒", elapsed)
	}
	if rares == 0 {
		t.Fatal("10^7 次抽取不可能没有稀有")
	}
}
