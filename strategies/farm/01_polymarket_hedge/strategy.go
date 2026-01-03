package main

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/clob"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/gamma"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/relayer"
)

// Strategy 对刷策略
type Strategy struct {
	config Config
}

// NewStrategy 创建策略实例
func NewStrategy(cfg Config) *Strategy {
	return &Strategy{config: cfg}
}

// Execute 执行单个账户对的对刷任务
func (s *Strategy) Execute(ctx context.Context, pair AccountPair) *Result {
	start := time.Now()
	result := &Result{Index: pair.Index}

	// 使用账户自己的代理（空则不使用代理）
	proxyA, proxyB := pair.ProxyA, pair.ProxyB

	// 1. 初始化两个账户的 Relayer
	fmt.Printf("[%d] 初始化 Relayer...\n", pair.Index)
	relayerA, err := relayer.NewClient(relayer.Config{PrivateKey: pair.PrivateKeyA, ProxyString: proxyA})
	if err != nil {
		result.Error = fmt.Sprintf("创建RelayerA失败: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	relayerB, err := relayer.NewClient(relayer.Config{PrivateKey: pair.PrivateKeyB, ProxyString: proxyB})
	if err != nil {
		result.Error = fmt.Sprintf("创建RelayerB失败: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	safeA, safeB := relayerA.GetProxyAddress(), relayerB.GetProxyAddress()
	fmt.Printf("[%d] Safe地址: A=%s, B=%s\n", pair.Index, safeA[:10]+"...", safeB[:10]+"...")

	// 2. 创建/派生 API Key
	fmt.Printf("[%d] 创建 API Key...\n", pair.Index)
	tempClientA, _ := clob.NewClient(clob.ClientConfig{PrivateKey: pair.PrivateKeyA, ChainID: clob.ChainIDPolygon, ProxyString: proxyA})
	tempClientB, _ := clob.NewClient(clob.ClientConfig{PrivateKey: pair.PrivateKeyB, ChainID: clob.ChainIDPolygon, ProxyString: proxyB})

	credsA, err := tempClientA.CreateOrDeriveApiKey(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("创建ApiKeyA失败: %v", err)
		result.Duration = time.Since(start)
		return result
	}
	credsB, err := tempClientB.CreateOrDeriveApiKey(ctx)
	if err != nil {
		result.Error = fmt.Sprintf("创建ApiKeyB失败: %v", err)
		result.Duration = time.Since(start)
		return result
	}

	// 3. 创建 CLOB 客户端
	clientA, _ := clob.NewClient(clob.ClientConfig{
		PrivateKey: pair.PrivateKeyA, ChainID: clob.ChainIDPolygon, ProxyString: proxyA,
		Funder: safeA, SignatureType: clob.SignatureTypeGnosisSafe, ApiCreds: credsA,
	})
	clientB, _ := clob.NewClient(clob.ClientConfig{
		PrivateKey: pair.PrivateKeyB, ChainID: clob.ChainIDPolygon, ProxyString: proxyB,
		Funder: safeB, SignatureType: clob.SignatureTypeGnosisSafe, ApiCreds: credsB,
	})

	// 4. 查询余额并检查授权
	fmt.Printf("[%d] 检查余额和授权...\n", pair.Index)
	balanceA, _ := relayerA.GetUSDCBalance(ctx)
	balanceB, _ := relayerB.GetUSDCBalance(ctx)
	fmt.Printf("[%d] 余额: A=%.2f USDC, B=%.2f USDC\n", pair.Index, balanceA, balanceB)

	// 检查并执行授权
	statusA, _ := relayerA.GetAccountStatus(ctx)
	statusB, _ := relayerB.GetAccountStatus(ctx)
	if !statusA.CTFApprovedExchange || statusA.USDCAllowanceCTF == "0" {
		fmt.Printf("[%d] 账户A未授权，执行授权...\n", pair.Index)
		relayerA.ApproveAllTokens(ctx)
	}
	if !statusB.CTFApprovedExchange || statusB.USDCAllowanceCTF == "0" {
		fmt.Printf("[%d] 账户B未授权，执行授权...\n", pair.Index)
		relayerB.ApproveAllTokens(ctx)
	}

	// 5. 选择市场
	fmt.Printf("[%d] 选择市场...\n", pair.Index)
	gammaClient := gamma.NewClient(gamma.ClientConfig{ProxyString: proxyA, Timeout: 30 * time.Second})

	var market *common.Market
	var yesTokenID, noTokenID string
	var tickSize, bestBid, bestAsk float64

	urls := make([]string, len(s.config.MarketURLs))
	copy(urls, s.config.MarketURLs)
	rand.Shuffle(len(urls), func(i, j int) { urls[i], urls[j] = urls[j], urls[i] })

	for _, url := range urls {
		fmt.Printf("[%d] 尝试市场: %s\n", pair.Index, url[:minInt(60, len(url))]+"...")
		m, err := gammaClient.GetMarketByURL(ctx, url)
		if err != nil {
			continue
		}

		ids, _ := common.ParseTokenIDs(m.ClobTokenIds)
		if len(ids) < 2 {
			continue
		}
		yesTokenID, noTokenID = ids[0], ids[1]

		// 获取订单簿
		book, err := clientA.GetOrderBook(ctx, yesTokenID)
		if err != nil {
			continue
		}

		tickSize, _ = strconv.ParseFloat(book.TickSize, 64)
		if tickSize <= 0 {
			tickSize = 0.01
		}

		// 解析盘口
		bestBid, bestAsk = parseBestPrices(book)
		if bestBid == 0 || bestAsk == 1 {
			fmt.Printf("[%d] 盘口数据不完整\n", pair.Index)
			continue
		}

		spreadTicks := int((bestAsk - bestBid) / tickSize)
		fmt.Printf("[%d] 市场: %s | 盘口: %.4f/%.4f, 间隔=%d tick\n", pair.Index, m.Question[:minInt(30, len(m.Question))], bestBid, bestAsk, spreadTicks)

		if spreadTicks >= s.config.MinSpreadTicks {
			market = m
			break
		}
	}

	if market == nil {
		result.Error = "所有市场盘口条件都不满足"
		result.Duration = time.Since(start)
		return result
	}

	// 6. 下单循环
	for attempt := 1; attempt <= s.config.MaxRetries; attempt++ {
		if attempt > 1 {
			fmt.Printf("[%d] 第 %d 次重试...\n", pair.Index, attempt)
			time.Sleep(s.config.GetRetryDelay())

			// 重新获取盘口
			book, err := clientA.GetOrderBook(ctx, yesTokenID)
			if err != nil {
				continue
			}
			bestBid, bestAsk = parseBestPrices(book)
			spreadTicks := int((bestAsk - bestBid) / tickSize)
			if spreadTicks < s.config.MinSpreadTicks {
				fmt.Printf("[%d] 盘口间隔不足，等待...\n", pair.Index)
				continue
			}
		}

		// 计算自成交价格
		yesBuyPrice := common.AlignPrice(bestBid+tickSize, tickSize, "BUY")
		noBuyPrice := roundTo(1.0-yesBuyPrice, 4)

		// 计算交易数量
		maxAmount := s.config.MaxTradeAmount
		maxFromA := balanceA / yesBuyPrice
		maxFromB := balanceB / noBuyPrice
		tradeAmount := common.AlignAmount(minFloat(maxAmount, maxFromA, maxFromB), tickSize)
		if tradeAmount < tickSize {
			result.Error = "余额不足"
			result.Duration = time.Since(start)
			return result
		}

		fmt.Printf("[%d] 下单: YES@%.4f, NO@%.4f, 数量=%.2f\n", pair.Index, yesBuyPrice, noBuyPrice, tradeAmount)

		// 获取市场参数
		tickSizeA, _ := clientA.GetTickSize(ctx, yesTokenID)
		negRiskA, _ := clientA.GetNegRisk(ctx, yesTokenID)
		tickSizeB, _ := clientB.GetTickSize(ctx, noTokenID)
		negRiskB, _ := clientB.GetNegRisk(ctx, noTokenID)

		// 并行下单
		var orderA, orderB *clob.OrderResponse
		var errA, errB error
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			orderA, errA = clientA.CreateAndPostOrder(ctx, clob.UserOrder{
				TokenID: yesTokenID, Side: clob.SideBuy, Price: yesBuyPrice, Size: tradeAmount, FeeRateBps: 0,
			}, clob.CreateOrderOptions{TickSize: tickSizeA, NegRisk: negRiskA}, clob.OrderTypeGTC)
		}()
		go func() {
			defer wg.Done()
			orderB, errB = clientB.CreateAndPostOrder(ctx, clob.UserOrder{
				TokenID: noTokenID, Side: clob.SideBuy, Price: noBuyPrice, Size: tradeAmount, FeeRateBps: 0,
			}, clob.CreateOrderOptions{TickSize: tickSizeB, NegRisk: negRiskB}, clob.OrderTypeGTC)
		}()
		wg.Wait()

		if errA != nil || errB != nil {
			fmt.Printf("[%d] 下单失败: A=%v, B=%v\n", pair.Index, errA, errB)
			continue
		}
		fmt.Printf("[%d] 订单已提交: A=%s, B=%s\n", pair.Index, orderA.OrderID[:16], orderB.OrderID[:16])

		// 等待成交
		time.Sleep(3 * time.Second)

		// 检查成交状态
		statusA, _ := clientA.GetOrder(ctx, orderA.OrderID)
		statusB, _ := clientB.GetOrder(ctx, orderB.OrderID)

		filledA := parseFloat(statusA.SizeMatched)
		filledB := parseFloat(statusB.SizeMatched)

		if filledA > 0 || filledB > 0 {
			result.Success = true
			result.FilledA = statusA.SizeMatched
			result.FilledB = statusB.SizeMatched
			result.Duration = time.Since(start)
			fmt.Printf("[%d] 对刷成功! A成交: %s, B成交: %s\n", pair.Index, result.FilledA, result.FilledB)
			return result
		}

		// 取消未成交订单
		fmt.Printf("[%d] 订单未成交，取消订单...\n", pair.Index)
		clientA.CancelOrder(ctx, orderA.OrderID)
		clientB.CancelOrder(ctx, orderB.OrderID)
	}

	result.Error = fmt.Sprintf("达到最大重试次数 %d，未能成交", s.config.MaxRetries)
	result.Duration = time.Since(start)
	return result
}

// parseBestPrices 解析最优买卖价
func parseBestPrices(book *clob.OrderBookSummary) (bestBid, bestAsk float64) {
	if len(book.Bids) > 0 {
		// 按价格降序排序，取最高价
		bids := make([]clob.OrderSummary, len(book.Bids))
		copy(bids, book.Bids)
		sort.Slice(bids, func(i, j int) bool {
			pi, _ := strconv.ParseFloat(bids[i].Price, 64)
			pj, _ := strconv.ParseFloat(bids[j].Price, 64)
			return pi > pj
		})
		bestBid, _ = strconv.ParseFloat(bids[0].Price, 64)
	}

	if len(book.Asks) > 0 {
		// 按价格升序排序，取最低价
		asks := make([]clob.OrderSummary, len(book.Asks))
		copy(asks, book.Asks)
		sort.Slice(asks, func(i, j int) bool {
			pi, _ := strconv.ParseFloat(asks[i].Price, 64)
			pj, _ := strconv.ParseFloat(asks[j].Price, 64)
			return pi < pj
		})
		bestAsk, _ = strconv.ParseFloat(asks[0].Price, 64)
	} else {
		bestAsk = 1
	}

	return
}

func parseFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func roundTo(v float64, decimals int) float64 {
	p := 1.0
	for i := 0; i < decimals; i++ {
		p *= 10
	}
	return float64(int(v*p+0.5)) / p
}

func minFloat(vals ...float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	m := vals[0]
	for _, v := range vals[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
