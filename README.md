# loot-pity

抽卡池：基础概率、软保底曲线、硬保底与跨会话的保底进度。只用 Go 标准库。

```
go test ./...
go run ./cmd/loot-pity --rate-at 74
go run ./cmd/loot-pity --start-pity 89 --draws 3 --seed 1
go run ./cmd/loot-pity --start-pity 85 --resume --seed 1
```

## 口径（README 为准）

- **软保底从第 74 抽就开始**：第 n 抽（已垫 n-1 抽）的概率是
  `BaseRate + max(0, n-74+1) * SoftStep`，也就是「下一抽是第 74 抽」时已经 +6%。
  `--rate-at 74`（下一抽是第 74 抽）按口径是 0.0660，现在写成 `pity > 74`，给 0.0060。
- **出稀有必须清零保底**：`--start-pity 89 --draws 3`（第 90 抽必出）按口径是 `rares=1`
  （出完回到基础概率），现在忘了清零，`pity` 一路涨下去，三次全是必出，给 `rares=3`。
- **保底进度要跨会话保存**：导出/导入存档必须带上保底进度。
  `--start-pity 85 --resume` 按口径是 `pity_after_resume=85`，现在只保存总抽数、导入后 `Pity=0`，
  给 `pity_after_resume=0`（玩家会觉得「重登后垫的保底白垫了」）。
- 概率：第 1..73 抽是 0.006；从第 74 抽起每抽 +0.06；第 90 抽必出；概率永远落在 [0, 1]。
- 不变量：同一个 seed + 同样的起始保底，抽出的稀有位置完全一致（可复现）；稀有之间的最大间隔 ≤ 90；
  `pity` 只会随抽数增加，出稀有后归零；`Total` 单调递增。
- 规模：10^7 次抽取 2 秒内完成，内存 O(1)；不许每次抽卡都重算整条概率曲线或分配切片。

## 现在的行为

- `RateAt` 的条件是 `pity > SoftAt`：第 74 抽（pity=73）拿不到软保底提升。
- `Draw` 出稀有后忘记 `p.state.Pity = 0`：保底进度只增不减，出完稀有下一抽往往还接着出。
- `Export`/`Import` 只带 `total`：换会话后 `Pity` 被重置为 0。

## 计划新增的能力

```
--rate-at N       打印「下一抽是第 N 抽」时的概率
--start-pity N    开抽前的保底进度
--draws N         抽几次
--seed N          随机种子
--resume          抽之前导出再导入一次存档
```

## 输出

```
rate_at_74=0.0660
draws=3 rares=1 pity_end=1
state_pity=1 state_total=3
```

## 目录

```
gacha.go              概率曲线、抽卡与存档
cmd/loot-pity         命令行入口
gacha_test.go         go test 用例
```
