package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/gamma"
)

var proxyString = "127.0.0.1:7897"

// PeriodTagIDs 周期对应的tag_id
var PeriodTagIDs = map[string]int{
	"15m":   102467,
	"1h":    102175,
	"4h":    102531,
	"daily": 102281,
}

func main() {
	ctx := context.Background()

	client := gamma.NewClient(gamma.ClientConfig{
		Timeout:     30 * time.Second,
		ProxyString: proxyString,
	})

	// 测试1: 验证 Daily 市场 slug 格式
	fmt.Println("=== 验证 Daily 市场 Slug 格式 ===\n")
	testDailySlugFormat(ctx, client)

	// 测试2: 搜索当前 15m 市场 (通过时间戳推算)
	fmt.Println("\n=== 搜索当前时间段的 15m 市场 ===\n")
	testFindCurrent15mMarket(ctx, client)
}

// testListEventsWithTagID 使用tag_id获取指定周期的市场
func testListEventsWithTagID(ctx context.Context, client *gamma.Client, period, symbol string) {
	start := time.Now()

	tagID, ok := PeriodTagIDs[period]
	if !ok {
		fmt.Printf("未知周期: %s\n", period)
		return
	}

	fmt.Printf("查询参数: tag_id=%d, closed=false, limit=100\n", tagID)
	fmt.Printf("筛选条件: slug包含 %q\n\n", symbol)

	closed := false
	events, err := client.ListEvents(ctx, &common.EventQueryParams{
		MarketQueryParams: common.MarketQueryParams{
			TagID:  tagID,
			Closed: &closed,
			Limit:  100,
		},
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Printf("获取到 %d 个活跃事件\n\n", len(events))

	// 筛选指定币种的事件
	var matchedEvents []common.Event
	symbolLower := strings.ToLower(symbol)
	for _, e := range events {
		slugLower := strings.ToLower(e.Slug)
		if strings.Contains(slugLower, symbolLower) {
			matchedEvents = append(matchedEvents, e)
		}
	}

	fmt.Printf("筛选出 %d 个 %s 相关事件\n\n", len(matchedEvents), strings.ToUpper(symbol))

	// 显示事件详情
	for i, e := range matchedEvents {
		if i >= 5 {
			fmt.Printf("... 还有 %d 个事件\n", len(matchedEvents)-5)
			break
		}

		fmt.Printf("[事件 %d]\n", i+1)
		fmt.Printf("  Title: %s\n", e.Title)
		fmt.Printf("  Slug: %s\n", e.Slug)
		fmt.Printf("  EndDate: %s\n", e.EndDate)
		fmt.Printf("  Active: %v\n", e.Active)
		fmt.Printf("  Closed: %v\n", e.Closed)
		fmt.Printf("  市场数: %d\n", len(e.Markets))

		// 解析EndDate并计算距离结束时间
		if e.EndDate != "" {
			endTime, err := time.Parse(time.RFC3339, e.EndDate)
			if err == nil {
				remaining := time.Until(endTime)
				if remaining > 0 {
					fmt.Printf("  距结束: %v\n", remaining.Round(time.Second))
				} else {
					fmt.Printf("  已结束: %v 前\n", (-remaining).Round(time.Second))
				}
			}
		}

		// 显示市场详情
		for j, m := range e.Markets {
			if j >= 2 {
				fmt.Printf("    ... 还有 %d 个市场\n", len(e.Markets)-2)
				break
			}
			fmt.Printf("  [市场 %d] %s\n", j+1, m.Question)
			fmt.Printf("    OutcomeTokens: %s\n", m.Outcomes)
			fmt.Printf("    OutcomePrices: %s\n", m.OutcomePrices)
			fmt.Printf("    ClobTokenIds: %s\n", m.ClobTokenIds)
			fmt.Printf("    TickSize: %s\n", m.OrderPriceMinTickSize)

			// 解析价格
			prices := strings.Trim(m.OutcomePrices, "[]")
			priceList := strings.Split(prices, ",")
			if len(priceList) >= 2 {
				fmt.Printf("    价格解析: UP=%.4s, DOWN=%.4s\n",
					strings.TrimSpace(priceList[0]),
					strings.TrimSpace(priceList[1]))

				// 计算价格之和
				var upPrice, downPrice float64
				fmt.Sscanf(strings.TrimSpace(priceList[0]), "%f", &upPrice)
				fmt.Sscanf(strings.TrimSpace(priceList[1]), "%f", &downPrice)
				sum := upPrice + downPrice
				fmt.Printf("    价格之和: %.4f (套利空间: %.2f%%)\n", sum, (1-sum)*100)
			}
		}
		fmt.Println()
	}

	fmt.Printf("耗时: %v\n", time.Since(start))
}

// testShowAllEvents 显示指定tag_id的所有事件
func testShowAllEvents(ctx context.Context, client *gamma.Client, tagID int) {
	closed := false
	events, err := client.ListEvents(ctx, &common.EventQueryParams{
		MarketQueryParams: common.MarketQueryParams{
			TagID:  tagID,
			Closed: &closed,
			Limit:  20,
		},
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Printf("获取到 %d 个事件\n\n", len(events))

	for i, e := range events {
		if i >= 10 {
			fmt.Printf("... 还有 %d 个事件\n", len(events)-10)
			break
		}
		fmt.Printf("[%d] %s\n", i+1, e.Title)
		fmt.Printf("    Slug: %s\n", e.Slug)
		fmt.Printf("    EndDate: %s, Active: %v, Closed: %v\n", e.EndDate, e.Active, e.Closed)
	}
}

// testSearchUpdown 搜索 updown 相关市场
func testSearchUpdown(ctx context.Context, client *gamma.Client) {
	keywords := []string{
		"Up or Down 15 minutes",
		"BTC Up or Down",
		"Bitcoin Up or Down",
		"crypto 15 minute",
	}

	for _, kw := range keywords {
		result, err := client.SearchMarketsEventsAndProfiles(ctx, &common.SearchParams{
			Q:            kw,
			LimitPerType: 5,
		})
		if err != nil {
			fmt.Printf("搜索 %q 错误: %v\n", kw, err)
			continue
		}

		fmt.Printf("搜索 %q: 找到 %d 个事件, %d 个市场\n", kw, len(result.Events), len(result.Markets))

		for i, e := range result.Events {
			if i >= 3 {
				break
			}
			fmt.Printf("  [事件] %s (Slug: %s, Closed: %v)\n", e.Title, e.Slug, e.Closed)
		}

		for i, m := range result.Markets {
			if i >= 3 {
				break
			}
			fmt.Printf("  [市场] %s (Slug: %s, Closed: %v)\n", m.Question, m.Slug, m.Closed)
		}
		fmt.Println()
	}
}

// testDailySlugFormat 验证 Daily 市场 slug 格式
func testDailySlugFormat(ctx context.Context, client *gamma.Client) {
	now := time.Now().UTC()

	// 尝试不同的 slug 格式
	slugFormats := []string{
		// bitcoin 格式
		fmt.Sprintf("bitcoin-up-or-down-on-january-%d", now.Day()),
		fmt.Sprintf("bitcoin-up-or-down-on-january-%d", now.Day()-1),
		// btc 格式
		fmt.Sprintf("btc-up-or-down-on-january-%d", now.Day()),
		// ethereum 格式
		fmt.Sprintf("ethereum-up-or-down-on-january-%d", now.Day()),
	}

	for _, slug := range slugFormats {
		fmt.Printf("尝试 slug: %s\n", slug)
		e, err := client.GetEventBySlug(ctx, slug)
		if err != nil {
			fmt.Printf("  未找到\n")
			continue
		}
		fmt.Printf("  找到! Title: %s\n", e.Title)
		fmt.Printf("  EndDate: %s, Closed: %v\n", e.EndDate, e.Closed)
		if len(e.Markets) > 0 {
			m := e.Markets[0]
			fmt.Printf("  Outcomes: %s\n", m.Outcomes)
			fmt.Printf("  OutcomePrices: %s\n", m.OutcomePrices)
		}
		fmt.Println()
	}
}

// testFindEventBySlug 按slug查找事件
func testFindEventBySlug(ctx context.Context, client *gamma.Client, slug string) {
	event, err := client.GetEventBySlug(ctx, slug)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	e := event
	fmt.Printf("事件: %s\n", e.Title)
	fmt.Printf("  Slug: %s\n", e.Slug)
	fmt.Printf("  EndDate: %s\n", e.EndDate)
	fmt.Printf("  Active: %v, Closed: %v\n", e.Active, e.Closed)
	fmt.Printf("  市场数: %d\n\n", len(e.Markets))

	for i, m := range e.Markets {
		fmt.Printf("  [市场 %d] %s\n", i+1, m.Question)
		fmt.Printf("    Outcomes: %s\n", m.Outcomes)
		fmt.Printf("    OutcomePrices: %s\n", m.OutcomePrices)
		fmt.Printf("    ClobTokenIds: %s\n", m.ClobTokenIds)
		fmt.Printf("    Active: %v, Closed: %v\n", m.Active, m.Closed)

		// 解析价格
		prices := strings.Trim(m.OutcomePrices, "[]")
		priceList := strings.Split(prices, ",")
		if len(priceList) >= 2 {
			var upPrice, downPrice float64
			fmt.Sscanf(strings.TrimSpace(priceList[0]), "%f", &upPrice)
			fmt.Sscanf(strings.TrimSpace(priceList[1]), "%f", &downPrice)
			sum := upPrice + downPrice
			fmt.Printf("    价格: UP=%.4f, DOWN=%.4f, Sum=%.4f\n", upPrice, downPrice, sum)
		}
		fmt.Println()
	}
}

// testFindCurrent15mMarket 查找当前时间段的 15m 市场
func testFindCurrent15mMarket(ctx context.Context, client *gamma.Client) {
	// 使用 UTC 时间计算
	now := time.Now().UTC()
	// 对齐到15分钟边界 - slug 使用的是周期开始时间
	minutes := now.Minute()
	alignedMinute := (minutes / 15) * 15 // 当前15分钟周期的开始
	startTime := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), alignedMinute, 0, 0, time.UTC)
	timestamp := startTime.Unix()

	fmt.Printf("当前 UTC 时间: %s\n", now.Format(time.RFC3339))
	fmt.Printf("当前周期开始: %s (timestamp: %d)\n", startTime.Format(time.RFC3339), timestamp)
	fmt.Printf("当前周期结束: %s\n\n", startTime.Add(15*time.Minute).Format(time.RFC3339))

	// 测试多个时间段的 slug
	symbols := []string{"btc", "eth", "sol", "xrp"}
	offsets := []int{0, -900, 900, 1800} // 当前、上一个、下一个、下下个

	for _, symbol := range symbols {
		fmt.Printf("=== %s 15m 市场 ===\n", strings.ToUpper(symbol))
		for _, offset := range offsets {
			ts := timestamp + int64(offset)
			slug := fmt.Sprintf("%s-updown-15m-%d", symbol, ts)
			periodStart := time.Unix(ts, 0).UTC()

			e, err := client.GetEventBySlug(ctx, slug)
			if err != nil {
				continue
			}

			// 计算是否为当前活跃市场
			endTime, _ := time.Parse(time.RFC3339, e.EndDate)
			isActive := now.Before(endTime) && !e.Closed

			status := "已过期"
			if isActive {
				status = "活跃中"
				remaining := time.Until(endTime)
				status = fmt.Sprintf("活跃中 (剩余 %v)", remaining.Round(time.Second))
			}

			fmt.Printf("  [%s] %s\n", status, periodStart.Format("15:04"))
			fmt.Printf("    Slug: %s\n", slug)
			fmt.Printf("    EndDate: %s\n", e.EndDate)
			if len(e.Markets) > 0 {
				m := e.Markets[0]
				prices := strings.Trim(m.OutcomePrices, "[]")
				priceList := strings.Split(prices, ",")
				if len(priceList) >= 2 {
					var upPrice, downPrice float64
					fmt.Sscanf(strings.TrimSpace(priceList[0]), "%f", &upPrice)
					fmt.Sscanf(strings.TrimSpace(priceList[1]), "%f", &downPrice)
					fmt.Printf("    UP=%.4f, DOWN=%.4f, Sum=%.4f\n", upPrice, downPrice, upPrice+downPrice)
				}
				fmt.Printf("    ClobTokenIds: %s\n", m.ClobTokenIds)
			}
		}
		fmt.Println()
	}
}

// testListUpdownMarkets 使用 ListMarkets 获取 updown 市场
func testListUpdownMarkets(ctx context.Context, client *gamma.Client) {
	closed := false
	markets, err := client.ListMarkets(ctx, &common.MarketQueryParams{
		Closed: &closed,
		Limit:  200,
	})
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	fmt.Printf("获取到 %d 个未关闭市场\n\n", len(markets))

	// 筛选 updown 相关市场
	var updownMarkets []common.Market
	for _, m := range markets {
		slugLower := strings.ToLower(m.Slug)
		if strings.Contains(slugLower, "updown") ||
			strings.Contains(strings.ToLower(m.Question), "up or down") {
			updownMarkets = append(updownMarkets, m)
		}
	}

	fmt.Printf("筛选出 %d 个 Up/Down 市场:\n\n", len(updownMarkets))

	for i, m := range updownMarkets {
		if i >= 10 {
			fmt.Printf("... 还有 %d 个市场\n", len(updownMarkets)-10)
			break
		}
		fmt.Printf("[%d] %s\n", i+1, m.Question)
		fmt.Printf("    Slug: %s\n", m.Slug)
		fmt.Printf("    EndDate: %s\n", m.EndDate)
		fmt.Printf("    OutcomePrices: %s\n", m.OutcomePrices)
		fmt.Printf("    ClobTokenIds: %s\n", m.ClobTokenIds)

		// 计算距离结束时间
		if m.EndDate != "" {
			if endTime, err := time.Parse(time.RFC3339, m.EndDate); err == nil {
				remaining := time.Until(endTime)
				if remaining > 0 {
					fmt.Printf("    距结束: %v\n", remaining.Round(time.Second))
				}
			}
		}
		fmt.Println()
	}
}

func init() {
	if p := os.Getenv("PROXY"); p != "" {
		proxyString = p
	}
}
