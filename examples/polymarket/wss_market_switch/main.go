package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/gamma"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/wss"
)

// ==================== 配置 ====================

var (
	proxyString = "127.0.0.1:7897"
	symbol      = "btc"  // btc, eth, sol, xrp
	period      = "15m"  // 15m, 1h, 4h
	preSubSec   = 30     // 提前多少秒预订阅下一轮
)

var symbolFullName = map[string]string{
	"btc": "bitcoin", "eth": "ethereum", "sol": "solana", "xrp": "xrp",
}

// ==================== OrderBook ====================

type OrderBook struct {
	mu      sync.RWMutex
	AssetID string
	Side    string // "UP" or "DOWN"
	Bids    map[string]string
	Asks    map[string]string
}

func NewOrderBook(assetID, side string) *OrderBook {
	return &OrderBook{
		AssetID: assetID,
		Side:    side,
		Bids:    make(map[string]string),
		Asks:    make(map[string]string),
	}
}

func (ob *OrderBook) ApplySnapshot(snapshot *common.OrderBookSnapshot) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	ob.Bids = make(map[string]string)
	ob.Asks = make(map[string]string)
	for _, bid := range snapshot.Bids {
		if bid.Size != "0" {
			ob.Bids[bid.Price] = bid.Size
		}
	}
	for _, ask := range snapshot.Asks {
		if ask.Size != "0" {
			ob.Asks[ask.Price] = ask.Size
		}
	}
}

func (ob *OrderBook) ApplyPriceChange(event *common.PriceChangeEvent) {
	ob.mu.Lock()
	defer ob.mu.Unlock()
	if event.Side == "BUY" {
		if event.Size == "0" {
			delete(ob.Bids, event.Price)
		} else {
			ob.Bids[event.Price] = event.Size
		}
	} else {
		if event.Size == "0" {
			delete(ob.Asks, event.Price)
		} else {
			ob.Asks[event.Price] = event.Size
		}
	}
}

func (ob *OrderBook) GetBestBid() (price float64, size float64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	for p, s := range ob.Bids {
		if pf, _ := strconv.ParseFloat(p, 64); pf > price {
			price = pf
			size, _ = strconv.ParseFloat(s, 64)
		}
	}
	return
}

func (ob *OrderBook) GetBestAsk() (price float64, size float64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	var best float64 = 999
	for p, s := range ob.Asks {
		if pf, _ := strconv.ParseFloat(p, 64); pf < best {
			best = pf
			size, _ = strconv.ParseFloat(s, 64)
		}
	}
	if best == 999 {
		return 0, 0
	}
	return best, size
}

// ==================== Round ====================

type Round struct {
	Slug        string
	UpTokenID   string
	DownTokenID string
	StartTime   time.Time
	EndTime     time.Time
}

// ==================== MarketSwitcher ====================

type MarketSwitcher struct {
	mu          sync.RWMutex
	gammaClient *gamma.Client
	wssClient   *wss.Client
	conn        *wss.Connection

	current   *Round
	next      *Round
	upBook    *OrderBook
	downBook  *OrderBook
	stopChan  chan struct{}
}

func NewMarketSwitcher() *MarketSwitcher {
	return &MarketSwitcher{
		gammaClient: gamma.NewClient(gamma.ClientConfig{
			Timeout:     30 * time.Second,
			ProxyString: proxyString,
		}),
		wssClient: wss.NewClient(wss.ClientConfig{ProxyString: proxyString}),
		stopChan:  make(chan struct{}),
	}
}

func (m *MarketSwitcher) Stop() {
	close(m.stopChan)
	if m.conn != nil {
		m.conn.Close()
	}
}

// getSlug 根据时间戳生成 slug
func getSlug(timestamp int64) string {
	if period == "daily" {
		t := time.Unix(timestamp, 0).UTC()
		return fmt.Sprintf("%s-up-or-down-on-%s-%d", symbolFullName[symbol], strings.ToLower(t.Month().String()), t.Day())
	}
	return fmt.Sprintf("%s-updown-%s-%d", symbol, period, timestamp)
}

// getPeriodDuration 获取周期时长
func getPeriodDuration() time.Duration {
	switch period {
	case "15m":
		return 15 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "daily":
		return 24 * time.Hour
	}
	return 15 * time.Minute
}

// calcCurrentRoundStart 计算当前轮次开始时间
func calcCurrentRoundStart() time.Time {
	now := time.Now().UTC()
	duration := getPeriodDuration()

	if period == "daily" {
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}

	periodSec := int(duration.Seconds())
	currentSec := now.Hour()*3600 + now.Minute()*60 + now.Second()
	alignedSec := (currentSec / periodSec) * periodSec
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, alignedSec, 0, time.UTC)
}

// parseTokenIDs 解析 token IDs
func parseTokenIDs(s string) []string {
	s = strings.Trim(s, "[]")
	parts := strings.Split(s, ",")
	var ids []string
	for _, p := range parts {
		if id := strings.Trim(strings.TrimSpace(p), "\""); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// fetchRound 获取指定时间戳的轮次信息
func (m *MarketSwitcher) fetchRound(ctx context.Context, startTime time.Time) (*Round, error) {
	slug := getSlug(startTime.Unix())
	event, err := m.gammaClient.GetEventBySlug(ctx, slug)
	if err != nil {
		return nil, fmt.Errorf("获取市场失败 [%s]: %w", slug, err)
	}

	if len(event.Markets) == 0 {
		return nil, fmt.Errorf("市场无数据: %s", slug)
	}

	ids := parseTokenIDs(event.Markets[0].ClobTokenIds)
	if len(ids) < 2 {
		return nil, fmt.Errorf("token IDs 不足: %s", slug)
	}

	endTime, _ := time.Parse(time.RFC3339, event.EndDate)

	return &Round{
		Slug:        slug,
		UpTokenID:   ids[0],
		DownTokenID: ids[1],
		StartTime:   startTime,
		EndTime:     endTime,
	}, nil
}

// subscribe 订阅当前轮次
func (m *MarketSwitcher) subscribe(ctx context.Context) error {
	m.upBook = NewOrderBook(m.current.UpTokenID, "UP")
	m.downBook = NewOrderBook(m.current.DownTokenID, "DOWN")

	m.conn = m.wssClient.CreateMarketConnection([]string{m.current.UpTokenID, m.current.DownTokenID})

	m.conn.OnConnected(func() {
		fmt.Println("[WSS] 已连接")
	})
	m.conn.OnDisconnected(func(code int, reason string) {
		fmt.Printf("[WSS] 断开: %d %s\n", code, reason)
	})
	m.conn.OnError(func(err error) {
		fmt.Printf("[WSS] 错误: %v\n", err)
	})

	return m.conn.Connect()
}

// preSubscribeNext 预订阅下一轮
func (m *MarketSwitcher) preSubscribeNext(ctx context.Context) error {
	if m.next != nil {
		return nil // 已预订阅
	}

	nextStart := m.current.EndTime
	round, err := m.fetchRound(ctx, nextStart)
	if err != nil {
		return err
	}

	m.next = round

	// 订阅下一轮的 token
	if err := m.conn.Subscribe([]string{round.UpTokenID, round.DownTokenID}); err != nil {
		return fmt.Errorf("订阅下一轮失败: %w", err)
	}

	fmt.Printf("[预订阅] %s\n", round.Slug)
	return nil
}

// switchToNext 切换到下一轮
func (m *MarketSwitcher) switchToNext() {
	if m.next == nil {
		return
	}

	// 取消旧订阅
	m.conn.Unsubscribe([]string{m.current.UpTokenID, m.current.DownTokenID})

	// 切换
	m.current = m.next
	m.next = nil

	// 重置订单簿
	m.upBook = NewOrderBook(m.current.UpTokenID, "UP")
	m.downBook = NewOrderBook(m.current.DownTokenID, "DOWN")

	fmt.Printf("\n[切换] %s\n", m.current.Slug)
}

// handleBook 处理订单簿快照
func (m *MarketSwitcher) handleBook(snapshot *common.OrderBookSnapshot) {
	if snapshot.AssetID == m.current.UpTokenID {
		m.upBook.ApplySnapshot(snapshot)
	} else if snapshot.AssetID == m.current.DownTokenID {
		m.downBook.ApplySnapshot(snapshot)
	} else if m.next != nil {
		// 可能是下一轮的数据，忽略或处理
		return
	}
	m.display()
}

// handlePriceChange 处理价格变化
func (m *MarketSwitcher) handlePriceChange(event *common.PriceChangeEvent) {
	if event.AssetID == m.current.UpTokenID {
		m.upBook.ApplyPriceChange(event)
	} else if event.AssetID == m.current.DownTokenID {
		m.downBook.ApplyPriceChange(event)
	} else {
		return
	}
	m.display()
}

// display 显示订单簿
func (m *MarketSwitcher) display() {
	upBid, upBidAmt := m.upBook.GetBestBid()
	upAsk, upAskAmt := m.upBook.GetBestAsk()
	downBid, downBidAmt := m.downBook.GetBestBid()
	downAsk, downAskAmt := m.downBook.GetBestAsk()

	if upAsk == 0 || downAsk == 0 {
		return
	}

	sum := upAsk + downAsk
	spread := (1 - sum) * 100

	remaining := time.Until(m.current.EndTime)
	var status string
	if remaining > 0 {
		status = fmt.Sprintf("剩余=%v", remaining.Round(time.Second))
	} else {
		status = "已结束"
	}

	fmt.Printf("[%s] UP bid=%.2f(%.0f) ask=%.2f(%.0f) | DOWN bid=%.2f(%.0f) ask=%.2f(%.0f) | Sum=%.4f Spread=%.2f%% | %s\n",
		m.current.Slug, upBid, upBidAmt, upAsk, upAskAmt, downBid, downBidAmt, downAsk, downAskAmt, sum, spread, status)
}

// Run 运行主循环
func (m *MarketSwitcher) Run(ctx context.Context) error {
	// 1. 计算当前轮次
	startTime := calcCurrentRoundStart()
	elapsed := time.Since(startTime)

	// 如果当前轮次已开始超过10秒，跳到下一轮
	if elapsed > 10*time.Second {
		fmt.Printf("当前轮次已开始 %v，跳到下一轮\n", elapsed.Round(time.Second))
		startTime = startTime.Add(getPeriodDuration())
	}

	// 2. 获取轮次信息
	round, err := m.fetchRound(ctx, startTime)
	if err != nil {
		return err
	}
	m.current = round
	fmt.Printf("[当前轮次] %s, 结束于 %s\n", round.Slug, round.EndTime.Format("15:04:05"))

	// 3. 订阅 WebSocket
	if err := m.subscribe(ctx); err != nil {
		return fmt.Errorf("订阅失败: %w", err)
	}

	// 4. 等待市场开始
	if wait := time.Until(m.current.StartTime); wait > 0 {
		fmt.Printf("等待市场开始，剩余 %v\n", wait.Round(time.Second))
		select {
		case <-time.After(wait):
			fmt.Println("市场开始!")
		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopChan:
			return nil
		}
	}

	// 5. 启动消息处理
	go m.messageLoop(ctx)

	// 6. 主循环：检测轮次切换
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			remaining := time.Until(m.current.EndTime)

			// 提前预订阅下一轮
			if remaining > 0 && remaining < time.Duration(preSubSec)*time.Second {
				if err := m.preSubscribeNext(ctx); err != nil {
					fmt.Printf("预订阅失败: %v\n", err)
				}
			}

			// 轮次结束，切换
			if remaining <= 0 {
				if m.next != nil {
					m.switchToNext()
				} else {
					// 备用：重新获取
					fmt.Println("预订阅未成功，重新获取...")
					round, err := m.fetchRound(ctx, m.current.EndTime)
					if err != nil {
						fmt.Printf("获取失败: %v\n", err)
						time.Sleep(time.Second)
						continue
					}
					m.current = round
					m.conn.Close()
					if err := m.subscribe(ctx); err != nil {
						fmt.Printf("重新订阅失败: %v\n", err)
						time.Sleep(time.Second)
						continue
					}
					go m.messageLoop(ctx)
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		case <-m.stopChan:
			return nil
		}
	}
}

// messageLoop 消息处理循环
func (m *MarketSwitcher) messageLoop(ctx context.Context) {
	for {
		select {
		case book := <-m.conn.BookCh():
			m.handleBook(book)
		case event := <-m.conn.PriceChangeCh():
			m.handlePriceChange(event)
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		}
	}
}

// ==================== Main ====================

func main() {
	fmt.Println("=== Up/Down 市场自动切换示例 ===")
	fmt.Printf("Symbol: %s, Period: %s\n\n", symbol, period)

	ctx, cancel := context.WithCancel(context.Background())
	switcher := NewMarketSwitcher()

	// 优雅退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n收到退出信号...")
		switcher.Stop()
		cancel()
	}()

	if err := switcher.Run(ctx); err != nil && err != context.Canceled {
		fmt.Printf("运行错误: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("示例结束")
}
