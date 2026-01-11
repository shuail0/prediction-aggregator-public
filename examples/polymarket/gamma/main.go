package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/gamma"
)

func main() {
	// 代理配置
	proxyString := "127.0.0.1:7897"

	// 创建 Gamma 客户端
	client := gamma.NewClient(gamma.ClientConfig{
		Timeout:     30 * time.Second,
		ProxyString: proxyString,
		Debug:       os.Getenv("DEBUG") == "1",
	})

	ctx := context.Background()

	fmt.Println("=== Gamma API 示例 ===")

	// 1. 健康检查
	fmt.Println("\n1. 健康检查")
	health, err := client.HealthCheck(ctx)
	if err != nil {
		fmt.Printf("健康检查失败: %v\n", err)
	} else {
		fmt.Printf("健康检查: %v\n", health)
	}

	// 2. 查询活跃市场
	fmt.Println("\n2. 查询活跃市场")
	active := true
	markets, err := client.ListMarkets(ctx, &common.MarketQueryParams{
		Limit:  5,
		Active: &active,
	})
	if err != nil {
		fmt.Printf("查询市场失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个市场:\n", len(markets))
		for i, m := range markets {
			fmt.Printf("  %d. %s (Volume: %s)\n", i+1, m.Question, m.Volume)
		}
	}

	// 3. 查询活跃事件
	fmt.Println("\n3. 查询活跃事件")
	events, err := client.ListEvents(ctx, &common.EventQueryParams{
		MarketQueryParams: common.MarketQueryParams{
			Limit:  5,
			Active: &active,
		},
	})
	if err != nil {
		fmt.Printf("查询事件失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个事件:\n", len(events))
		for i, e := range events {
			fmt.Printf("  %d. %s (%d 个市场)\n", i+1, e.Title, len(e.Markets))
		}
	}

	// 4. 搜索功能
	searchQuery := os.Getenv("SEARCH_QUERY")
	if searchQuery == "" {
		searchQuery = "Bitcoin" // 默认搜索词
	}
	fmt.Printf("\n4. 搜索: %s\n", searchQuery)
	result, err := client.SearchMarketsEventsAndProfiles(ctx, &common.SearchParams{
		Q:            searchQuery,
		LimitPerType: 3,
	})
	if err != nil {
		fmt.Printf("搜索失败: %v\n", err)
	} else {
		fmt.Printf("搜索结果: %d 事件, %d 市场, %d 用户\n",
			len(result.Events), len(result.Markets), len(result.Profiles))
	}

	// 5. 根据 URL 获取市场
	testURLs := []string{
		"https://polymarket.com/event/btc-updown-15m-1767462300?tid=1767461441245",
	}
	marketURL := os.Getenv("MARKET_URL")
	if marketURL != "" {
		testURLs = []string{marketURL}
	}

	fmt.Println("\n5. 根据 URL 获取市场 (GetMarketByURL)")
	for _, testURL := range testURLs {
		fmt.Printf("\n  URL: %s\n", testURL)
		market, err := client.GetMarketByURL(ctx, testURL)
		if err != nil {
			fmt.Printf("  获取失败: %v\n", err)
			continue
		}
		fmt.Printf("  市场: %s\n", market.Question)
		fmt.Printf("  条件ID: %s\n", market.ConditionID)
		fmt.Printf("  Token IDs: %s\n", market.ClobTokenIds)
		fmt.Printf("  Tick Size: %s\n", market.OrderPriceMinTickSize)
		fmt.Printf("  活跃: %v, 已结算: %v\n", market.Active, market.Closed)

		// 解析价格
		prices, _ := common.ParseOutcomePrices(market.OutcomePrices)
		outcomes, _ := common.ParseOutcomes(market.Outcomes)
		if len(prices) > 0 && len(outcomes) > 0 {
			fmt.Printf("  价格:\n")
			for i, o := range outcomes {
				if i < len(prices) {
					fmt.Printf("    %s: %.4f\n", o, prices[i])
				}
			}
		}
	}

	// 6. 根据 URL 获取事件
	fmt.Println("\n6. 根据 URL 获取事件 (GetEventByURL)")
	eventURL := "https://polymarket.com/event/nba-det-bos-2025-11-26"
	if envURL := os.Getenv("EVENT_URL"); envURL != "" {
		eventURL = envURL
	}
	fmt.Printf("  URL: %s\n", eventURL)
	event, err := client.GetEventByURL(ctx, eventURL)
	if err != nil {
		fmt.Printf("  获取失败: %v\n", err)
	} else {
		fmt.Printf("  事件: %s\n", event.Title)
		fmt.Printf("  包含 %d 个市场\n", len(event.Markets))
		for i, m := range event.Markets {
			fmt.Printf("    %d. %s\n", i+1, m.Question)
		}

		// 输出完整 JSON
		if os.Getenv("DEBUG") == "1" {
			data, _ := json.MarshalIndent(event, "", "  ")
			fmt.Printf("\n完整数据:\n%s\n", string(data))
		}
	}

	// 7. 列出标签
	fmt.Println("\n7. 列出标签 (ListTags)")
	tags, err := client.ListTags(ctx, &common.TagQueryParams{Limit: 5})
	if err != nil {
		fmt.Printf("获取标签失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个标签:\n", len(tags))
		for i, t := range tags {
			fmt.Printf("  %d. %s (slug: %s)\n", i+1, t.Label, t.Slug)
		}
	}

	// 8. 获取标签详情
	if len(tags) > 0 {
		fmt.Println("\n8. 获取标签详情 (GetTagByID)")
		tag, err := client.GetTagByID(ctx, tags[0].ID)
		if err != nil {
			fmt.Printf("获取标签详情失败: %v\n", err)
		} else {
			fmt.Printf("  标签: %s\n", tag.Label)
			fmt.Printf("  Slug: %s\n", tag.Slug)
			fmt.Printf("  ForceShow: %v, IsCarousel: %v\n", tag.ForceShow, tag.IsCarousel)
		}
	}

	// 9. 列出系列
	fmt.Println("\n9. 列出系列 (ListSeries)")
	series, err := client.ListSeries(ctx, &common.SeriesQueryParams{Limit: 5})
	if err != nil {
		fmt.Printf("获取系列失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个系列:\n", len(series))
		for i, s := range series {
			fmt.Printf("  %d. %s (slug: %s)\n", i+1, s.Title, s.Slug)
		}
	}

	// 10. 获取系列详情
	if len(series) > 0 {
		fmt.Println("\n10. 获取系列详情 (GetSeriesByID)")
		s, err := client.GetSeriesByID(ctx, series[0].ID)
		if err != nil {
			fmt.Printf("获取系列详情失败: %v\n", err)
		} else {
			fmt.Printf("  系列: %s\n", s.Title)
			fmt.Printf("  类型: %s\n", s.SeriesType)
			fmt.Printf("  包含 %d 个事件\n", len(s.Events))
		}
	}

	// 11. 列出评论 (需要 parent_entity_id 和 parent_entity_type)
	fmt.Println("\n11. 列出评论 (ListComments)")
	// 使用前面获取的第一个事件的 ID
	if len(events) > 0 {
		comments, err := client.ListComments(ctx, &common.CommentQueryParams{
			Limit:            3,
			ParentEntityType: "Event", // API 需要首字母大写
			ParentEntityID:   events[0].ID,
		})
		if err != nil {
			fmt.Printf("获取评论失败: %v\n", err)
		} else {
			fmt.Printf("找到 %d 条评论:\n", len(comments))
			for i, c := range comments {
				body := c.Body
				if len(body) > 50 {
					body = body[:50] + "..."
				}
				if len(c.UserAddress) > 10 {
					fmt.Printf("  %d. %s (by: %s...)\n", i+1, body, c.UserAddress[:10])
				} else {
					fmt.Printf("  %d. %s (by: %s)\n", i+1, body, c.UserAddress)
				}
			}
		}
	} else {
		fmt.Println("  跳过: 没有可用的事件 ID")
	}

	// 12. 获取用户资料
	fmt.Println("\n12. 获取用户资料 (GetPublicProfile)")
	// 尝试从搜索结果中获取有效的用户地址
	var testAddress string
	if result != nil && len(result.Profiles) > 0 {
		testAddress = result.Profiles[0].Address
	} else {
		// 使用一个知名的 Polymarket 用户地址
		testAddress = "0x1234567890123456789012345678901234567890"
	}
	fmt.Printf("  测试地址: %s\n", testAddress)
	profile, err := client.GetPublicProfile(ctx, testAddress)
	if err != nil {
		fmt.Printf("  获取失败 (可能用户不存在): %v\n", err)
	} else {
		name := profile.Name
		if name == "" {
			name = profile.Pseudonym
		}
		if name == "" {
			name = "(未设置)"
		}
		fmt.Printf("  用户: %s\n", name)
		fmt.Printf("  ProxyWallet: %s\n", profile.ProxyWallet)
		fmt.Printf("  VerifiedBadge: %v\n", profile.VerifiedBadge)
	}

	// 13. 获取体育市场类型
	fmt.Println("\n13. 获取体育市场类型 (GetValidSportsMarketTypes)")
	sportsTypes, err := client.GetValidSportsMarketTypes(ctx)
	if err != nil {
		fmt.Printf("获取体育市场类型失败: %v\n", err)
	} else {
		fmt.Printf("支持 %d 种体育市场类型:\n", len(sportsTypes.MarketTypes))
		for i, t := range sportsTypes.MarketTypes {
			if i >= 10 {
				fmt.Printf("  ... 还有 %d 种\n", len(sportsTypes.MarketTypes)-10)
				break
			}
			fmt.Printf("  - %s\n", t)
		}
	}

	// 14. 获取体育元数据
	fmt.Println("\n14. 获取体育元数据 (GetSportsMetadata)")
	sportsMetadata, err := client.GetSportsMetadata(ctx)
	if err != nil {
		fmt.Printf("获取体育元数据失败: %v\n", err)
	} else {
		fmt.Printf("体育元数据: %T\n", sportsMetadata)
	}

	// 15. 列出团队
	fmt.Println("\n15. 列出团队 (ListTeams)")
	teams, err := client.ListTeams(ctx, &gamma.ListTeamsParams{Limit: 5})
	if err != nil {
		fmt.Printf("获取团队失败: %v\n", err)
	} else {
		fmt.Printf("找到 %d 个团队:\n", len(teams))
		for i, t := range teams {
			fmt.Printf("  %d. %s (%s) - %s\n", i+1, t.Name, t.Abbreviation, t.League)
		}
	}

	fmt.Println("\n✅ Gamma API 示例完成")
}
