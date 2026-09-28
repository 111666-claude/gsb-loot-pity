package gacha

import "testing"

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
