package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/data"
)

func main() {
	proxyString := "127.0.0.1:7897"
	userAddress := os.Getenv("DATA_USER")

	if userAddress == "" {
		userAddress = "0x7f69983eb28245bba0d5083502a78744a8f66162"
		fmt.Println("提示: 未设置 DATA_USER，使用默认测试地址")
	}

	client := data.NewClient(data.ClientConfig{
		Timeout:     30 * time.Second,
		ProxyString: proxyString,
		Debug:       os.Getenv("DEBUG") == "1",
	})

	ctx := context.Background()

	fmt.Println("=== Data API 示例 ===")
	fmt.Printf("用户地址: %s\n", userAddress)

	// 1. 健康检查
	fmt.Println("\n1. 健康检查")
	health, err := client.HealthCheck(ctx)
	if err != nil {
		fmt.Printf("健康检查失败: %v\n", err)
	} else {
		fmt.Printf("Data API 状态: %s\n", health)
	}

	// 2. 获取用户持仓
	fmt.Println("\n2. 获取用户持仓")
	positions, err := client.GetPositions(ctx, &common.PositionQueryParams{
		User:  userAddress,
		Limit: 10,
	})
	if err != nil {
		fmt.Printf("获取持仓失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个持仓:\n", len(positions))
		for i, p := range positions {
			if i >= 5 {
				fmt.Printf("  ... 还有 %d 个持仓\n", len(positions)-5)
				break
			}
			fmt.Printf("  %d. %s (%s)\n", i+1, p.Title, p.Outcome)
			fmt.Printf("     数量: %.2f, 均价: %.4f, 当前价: %.4f\n", p.Size, p.AveragePrice, p.CurrentPrice)
			fmt.Printf("     盈亏: %.2f (%.2f%%)\n", p.CashPnl, p.PercentPnl*100)
		}
	}

	// 3. 获取交易历史
	fmt.Println("\n3. 获取交易历史")
	trades, err := client.GetTradeHistory(ctx, &common.TradeHistoryParams{
		User:  userAddress,
		Limit: 5,
	})
	if err != nil {
		fmt.Printf("获取交易历史失败: %v\n", err)
	} else {
		fmt.Printf("最近 %d 笔交易:\n", len(trades))
		for i, t := range trades {
			fmt.Printf("  %d. %s %s @ %.4f (%.2f)\n", i+1, t.Side, t.Outcome, t.Price, t.Size)
		}
	}

	// 4. 获取用户活动
	fmt.Println("\n4. 获取用户活动")
	activities, err := client.GetActivity(ctx, &common.ActivityParams{
		User:  userAddress,
		Limit: 5,
	})
	if err != nil {
		fmt.Printf("获取活动失败: %v\n", err)
	} else {
		fmt.Printf("最近 %d 条活动:\n", len(activities))
		for i, a := range activities {
			fmt.Printf("  %d. [%s] %s - %s @ %.4f\n", i+1, a.Type, a.Title, a.Outcome, a.Price)
		}
	}

	// 5. 获取已平仓持仓
	fmt.Println("\n5. 获取已平仓持仓")
	closedPos, err := client.GetClosedPositions(ctx, &common.ClosedPositionParams{
		User:  userAddress,
		Limit: 5,
	})
	if err != nil {
		fmt.Printf("获取已平仓失败: %v\n", err)
	} else {
		fmt.Printf("最近 %d 个已平仓:\n", len(closedPos))
		for i, p := range closedPos {
			fmt.Printf("  %d. %s (%s) - 盈亏: $%.2f\n", i+1, p.Title, p.Outcome, p.RealizedPnl)
		}
	}

	// 6. 获取持仓总价值
	fmt.Println("\n6. 获取持仓总价值")
	values, err := client.GetPortfolioValue(ctx, userAddress)
	if err != nil {
		fmt.Printf("获取持仓价值失败: %v\n", err)
	} else if len(values) > 0 {
		fmt.Printf("持仓总价值: $%.2f\n", values[0].Value)
	}

	// 7. 获取全局 Open Interest
	fmt.Println("\n7. 获取全局 Open Interest")
	oi, err := client.GetOpenInterest(ctx)
	if err != nil {
		fmt.Printf("获取 OI 失败: %v\n", err)
	} else if len(oi) > 0 {
		fmt.Printf("全局 Open Interest: $%.2f\n", oi[0].Value)
	}

	// 8. 获取事件实时交易量
	fmt.Println("\n8. 获取事件实时交易量 (eventId=12345)")
	volume, err := client.GetLiveVolume(ctx, 12345)
	if err != nil {
		fmt.Printf("获取交易量失败: %v\n", err)
	} else {
		fmt.Printf("总交易量: $%.2f\n", volume.Total)
		if len(volume.Markets) > 0 {
			fmt.Printf("市场数量: %d\n", len(volume.Markets))
		}
	}

	// 9. 获取市场持有者 (需要 conditionId)
	fmt.Println("\n9. 获取市场持有者")
	// 使用一个已知的 conditionId
	conditionId := "0xe3b423dfad8c22ff75c9899c4e8176f628cf4ad4caa00481764d320e7415f7a9"
	holders, err := client.GetHolders(ctx, &common.HoldersParams{
		Market: conditionId,
		Limit:  3,
	})
	if err != nil {
		fmt.Printf("获取持有者失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个 token 的持有者:\n", len(holders))
		for _, h := range holders {
			fmt.Printf("  Token: %s...\n", h.Token[:20])
			for i, holder := range h.Holders {
				if i >= 3 {
					break
				}
				fmt.Printf("    %d. %s: %.2f\n", i+1, holder.Name, holder.Amount)
			}
		}
	}

	// 10. 获取用户交易过的市场数量
	fmt.Println("\n10. 获取用户交易过的市场数量")
	traded, err := client.GetMarketsTraded(ctx, userAddress)
	if err != nil {
		fmt.Printf("获取交易市场数失败: %v\n", err)
	} else {
		fmt.Printf("交易过的市场数: %d\n", traded.Traded)
	}

	// 11. 获取交易者排行榜
	fmt.Println("\n11. 获取交易者排行榜 (按 PnL)")
	leaderboard, err := client.GetLeaderboard(ctx, &common.LeaderboardParams{
		Category:   "OVERALL",
		TimePeriod: "DAY",
		OrderBy:    "PNL",
		Limit:      5,
	})
	if err != nil {
		fmt.Printf("获取排行榜失败: %v\n", err)
	} else {
		fmt.Printf("排行榜 Top %d:\n", len(leaderboard))
		for _, e := range leaderboard {
			fmt.Printf("  #%s %s - PnL: $%.2f, Volume: $%.2f\n",
				e.Rank, e.UserName, e.PnL, e.Volume)
		}
	}

	// 12. 获取 Builder 排行榜
	fmt.Println("\n12. 获取 Builder 排行榜")
	builders, err := client.GetBuilderLeaderboard(ctx, &common.BuilderLeaderboardParams{
		Limit: 5,
	})
	if err != nil {
		fmt.Printf("获取 Builder 排行榜失败: %v\n", err)
	} else {
		fmt.Printf("Builder 排行榜 Top %d:\n", len(builders))
		for _, b := range builders {
			fmt.Printf("  #%s %s - Volume: $%.2f, Users: %d\n",
				b.Rank, b.Builder, b.Volume, b.ActiveUsers)
		}
	}

	// 13. 获取 Builder 交易量时序
	fmt.Println("\n13. 获取 Builder 交易量时序")
	builderVolume, err := client.GetBuilderVolume(ctx, &common.BuilderVolumeParams{
		Limit: 3,
	})
	if err != nil {
		fmt.Printf("获取 Builder 交易量失败: %v\n", err)
	} else {
		fmt.Printf("Builder 交易量数据 (%d 条):\n", len(builderVolume))
		for i, v := range builderVolume {
			if i >= 3 {
				break
			}
			fmt.Printf("  %s: %s - $%.2f\n", v.Date[:10], v.Builder, v.Volume)
		}
	}

	fmt.Println("\n✅ Data API 示例完成")
}
