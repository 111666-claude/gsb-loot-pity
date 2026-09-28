package gacha

import (
	"math"
	"testing"
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
	pool := NewPool(5)
	cases := []struct {
		draw int // 第几抽
		want float64
	}{
		{1, 0.006},
		{73, 0.006},
		{74, 0.066},
		{75, 0.126},
		{89, 0.966},
		{90, 1},
	}
	for _, c := range cases {
		if got := pool.RateAt(c.draw - 1); math.Abs(got-c.want) > 1e-9 {
			t.Fatalf("第 %d 抽概率应为 %v，实际 %v", c.draw, c.want, got)
		}
	}
}

func TestRareResetsPity(t *testing.T) {
	pool := NewPool(6)
	if err := pool.SetPity(HardAt - 1); err != nil {
		t.Fatalf("SetPity 出错：%v", err)
	}
	rare, _ := pool.Draw()
	if !rare {
		t.Fatal("第 90 抽必须必出")
	}
	if pool.State().Pity != 0 {
		t.Fatalf("出稀有后保底必须清零，实际 pity=%d", pool.State().Pity)
	}
	// 清零后下一抽回到基础概率，不该接着必出。
	rare, _ = pool.Draw()
	if rare {
		t.Fatal("出稀有后下一抽不该必出")
	}
	if pool.State().Total != 2 {
		t.Fatalf("Total 只增不减，实际 %d", pool.State().Total)
	}
}

func TestExportImportRoundTrip(t *testing.T) {
	pool := NewPool(7)
	for i := 0; i < 50; i++ {
		pool.Draw()
	}
	data := pool.Export()

	restored := NewPool(8)
	if err := restored.Import(data); err != nil {
		t.Fatalf("导入合法存档出错：%v", err)
	}
	want := pool.State()
	if got := restored.State(); got != want {
		t.Fatalf("存档往返后状态不一致：got %+v want %+v", got, want)
	}
}

func TestImportRejectsInvalidSaves(t *testing.T) {
	pool := NewPool(9)
	for _, data := range []string{"", "pity=-1;total=10", "pity=1", "total=10", "pity=1;total=2;foo=3", "pity;x"} {
		if err := pool.Import(data); err != ErrState {
			t.Fatalf("非法存档 %q 应该报 ErrState，实际 %v", data, err)
		}
	}
}

func TestDeterministicRares(t *testing.T) {
	positions := func() []int {
		pool := NewPool(42)
		if err := pool.SetPity(60); err != nil {
			t.Fatalf("SetPity 出错：%v", err)
		}
		var got []int
		for i := 0; i < 1000; i++ {
			if rare, _ := pool.Draw(); rare {
				got = append(got, i)
			}
		}
		return got
	}
	first, second := positions(), positions()
	if len(first) == 0 {
		t.Fatal("1000 抽一个稀有都没有，不对劲")
	}
	if len(first) != len(second) {
		t.Fatalf("同一 seed 同一初始保底，稀有位置不一致：%v vs %v", first, second)
	}
	for i := range first {
		if first[i] != second[i] {
			t.Fatalf("同一 seed 同一初始保底，稀有位置不一致：%v vs %v", first, second)
		}
	}
}

func TestMaxGapBetweenRares(t *testing.T) {
	pool := NewPool(10)
	gap := 0
	for i := 0; i < 100000; i++ {
		gap++
		if rare, _ := pool.Draw(); rare {
			if gap > HardAt {
				t.Fatalf("两次稀有间隔 %d 超过硬保底 %d", gap, HardAt)
			}
			gap = 0
		}
	}
}

func TestRareRateMatchesTheory(t *testing.T) {
	// 理论期望：一个保底周期的期望长度 = sum P(连续 n 抽不出稀有)，n=0..89。
	survival, expectedCycle := 1.0, 0.0
	for draw := 0; draw < HardAt; draw++ {
		expectedCycle += survival
		rate := BaseRate + math.Max(0, float64(draw+1-SoftAt+1))*SoftStep
		if rate > 1 {
			rate = 1
		}
		survival *= 1 - rate
	}

	const draws = 100000
	pool := NewPool(11)
	rares := 0
	for i := 0; i < draws; i++ {
		if rare, _ := pool.Draw(); rare {
			rares++
		}
	}
	expected := draws / expectedCycle
	if diff := math.Abs(float64(rares)-expected) / expected; diff > 0.02 {
		t.Fatalf("稀有次数偏离理论期望超过 2%%：got %d want %.1f (%.2f%%)", rares, expected, diff*100)
	}
}

func BenchmarkDraw(b *testing.B) {
	pool := NewPool(12)
	for i := 0; i < b.N; i++ {
		pool.Draw()
	}
}
