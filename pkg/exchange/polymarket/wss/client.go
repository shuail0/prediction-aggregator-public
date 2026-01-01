package wss

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

// ClientConfig WebSocket 客户端配置
type ClientConfig struct {
	BaseURL              string
	PingInterval         time.Duration
	ReconnectDelay       time.Duration
	MaxReconnectAttempts int
	Debug                bool
	ProxyString          string
}

// ChannelType 频道类型
type ChannelType string

const (
	ChannelMarket ChannelType = "market"
	ChannelUser   ChannelType = "user"
)

// Client WebSocket 客户端
type Client struct {
	config ClientConfig
}

// NewClient 创建 WebSocket 客户端
func NewClient(cfg ClientConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = common.WssBaseURL
	}
	if cfg.PingInterval == 0 {
		cfg.PingInterval = 10 * time.Second
	}
	if cfg.ReconnectDelay == 0 {
		cfg.ReconnectDelay = 5 * time.Second
	}
	if cfg.MaxReconnectAttempts == 0 {
		cfg.MaxReconnectAttempts = 10
	}

	return &Client{config: cfg}
}

// CreateMarketConnection 创建市场频道连接
func (c *Client) CreateMarketConnection(assetIDs []string) *Connection {
	if len(assetIDs) == 0 {
		return nil
	}

	payload := map[string]interface{}{
		"assets_ids": assetIDs,
		"type":       "market",
	}

	return NewConnection(ChannelMarket, c.config, payload)
}

// CreateUserConnection 创建用户频道连接
func (c *Client) CreateUserConnection(auth common.WssAuth, markets []string) *Connection {
	payload := map[string]interface{}{
		"type": "user",
		"auth": map[string]string{
			"apiKey":     auth.APIKey,
			"secret":     auth.Secret,
			"passphrase": auth.Passphrase,
		},
	}
	if len(markets) > 0 {
		payload["markets"] = markets
	}

	return NewConnection(ChannelUser, c.config, payload)
}

// Connection WebSocket 连接
type Connection struct {
	channel            ChannelType
	config             ClientConfig
	subscribePayload   map[string]interface{}
	conn               *websocket.Conn
	mu                 sync.RWMutex
	isConnected        bool
	isIntentionalClose bool
	reconnectAttempts  int
	pingTimer          *time.Ticker
	reconnectTimer     *time.Timer
	stopCh             chan struct{}
	processedTrades    sync.Map // 成交去重

	// 回调函数
	onConnected     func()
	onDisconnected  func(code int, reason string)
	onError         func(err error)
	onReconnecting  func(attempt int, delay time.Duration)
	onReconnectFail func(attempts int)

	// Market 频道回调
	onBook           func(*common.OrderBookSnapshot)
	onPriceChange    func(*common.PriceChangeEvent)
	onLastTradePrice func(*common.LastTradePrice)
	onTickSizeChange func(*common.TickSizeChange)

	// User 频道回调
	onOrder func(*common.OrderUpdate)
	onTrade func(*common.TradeNotification)

	// 通用消息回调
	onMessage func(channel ChannelType, data []byte)
}

// NewConnection 创建 WebSocket 连接
func NewConnection(channel ChannelType, config ClientConfig, payload map[string]interface{}) *Connection {
	return &Connection{
		channel:          channel,
		config:           config,
		subscribePayload: payload,
		stopCh:           make(chan struct{}),
	}
}

// OnConnected 设置连接成功回调
func (c *Connection) OnConnected(fn func()) { c.onConnected = fn }

// OnDisconnected 设置断开连接回调
func (c *Connection) OnDisconnected(fn func(code int, reason string)) { c.onDisconnected = fn }

// OnError 设置错误回调
func (c *Connection) OnError(fn func(err error)) { c.onError = fn }

// OnReconnecting 设置重连中回调
func (c *Connection) OnReconnecting(fn func(attempt int, delay time.Duration)) {
	c.onReconnecting = fn
}

// OnReconnectFail 设置重连失败回调
func (c *Connection) OnReconnectFail(fn func(attempts int)) { c.onReconnectFail = fn }

// OnBook 设置订单簿快照回调
func (c *Connection) OnBook(fn func(*common.OrderBookSnapshot)) { c.onBook = fn }

// OnPriceChange 设置价格变化回调
func (c *Connection) OnPriceChange(fn func(*common.PriceChangeEvent)) { c.onPriceChange = fn }

// OnLastTradePrice 设置最新成交价回调
func (c *Connection) OnLastTradePrice(fn func(*common.LastTradePrice)) { c.onLastTradePrice = fn }

// OnTickSizeChange 设置 tick size 变化回调
func (c *Connection) OnTickSizeChange(fn func(*common.TickSizeChange)) { c.onTickSizeChange = fn }

// OnOrder 设置订单更新回调
func (c *Connection) OnOrder(fn func(*common.OrderUpdate)) { c.onOrder = fn }

// OnTrade 设置成交通知回调
func (c *Connection) OnTrade(fn func(*common.TradeNotification)) { c.onTrade = fn }

// OnMessage 设置原始消息回调
func (c *Connection) OnMessage(fn func(channel ChannelType, data []byte)) { c.onMessage = fn }

// Connect 连接
func (c *Connection) Connect() error {
	c.mu.Lock()
	if c.isConnected {
		c.mu.Unlock()
		return nil
	}
	c.isIntentionalClose = false
	c.mu.Unlock()

	wsURL := fmt.Sprintf("%s/ws/%s", c.config.BaseURL, c.channel)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// 配置代理
	if c.config.ProxyString != "" {
		proxyCfg := common.ParseProxyString(c.config.ProxyString)
		if proxyCfg != nil {
			if proxyCfg.IsSocks() {
				// SOCKS5 代理
				proxyDialer, err := common.CreateProxyDialer(c.config.ProxyString)
				if err == nil && proxyDialer != nil {
					dialer.NetDial = proxyDialer.Dial
				}
			} else {
				// HTTP 代理
				dialer.Proxy = http.ProxyURL(proxyCfg.GetProxyURL())
			}
		}
	}

	conn, _, err := dialer.Dial(wsURL, http.Header{})
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.mu.Lock()
	c.conn = conn
	c.isConnected = true
	c.reconnectAttempts = 0
	c.mu.Unlock()

	// 发送订阅消息
	if err := c.subscribe(); err != nil {
		c.Close()
		return fmt.Errorf("subscribe: %w", err)
	}

	// 启动心跳
	c.startPing()

	// 启动消息读取
	go c.readLoop()

	if c.onConnected != nil {
		c.onConnected()
	}

	return nil
}

// Close 关闭连接
func (c *Connection) Close() {
	c.mu.Lock()
	c.isIntentionalClose = true
	c.mu.Unlock()

	c.stopPing()
	c.stopReconnect()

	c.mu.Lock()
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.isConnected = false
	c.mu.Unlock()

	close(c.stopCh)
}

// IsConnected 检查连接状态
func (c *Connection) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected
}

// GetStatus 获取状态
func (c *Connection) GetStatus() (connected bool, attempts int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isConnected, c.reconnectAttempts
}

// Send 发送消息
func (c *Connection) Send(data interface{}) error {
	c.mu.RLock()
	conn := c.conn
	connected := c.isConnected
	c.mu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected")
	}

	var msg []byte
	switch v := data.(type) {
	case string:
		msg = []byte(v)
	case []byte:
		msg = v
	default:
		var err error
		msg, err = json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
	}

	return conn.WriteMessage(websocket.TextMessage, msg)
}

// subscribe 发送订阅消息
func (c *Connection) subscribe() error {
	return c.Send(c.subscribePayload)
}

// startPing 启动心跳
func (c *Connection) startPing() {
	c.stopPing()
	c.pingTimer = time.NewTicker(c.config.PingInterval)

	go func() {
		for {
			select {
			case <-c.pingTimer.C:
				if c.IsConnected() {
					c.Send("PING")
				}
			case <-c.stopCh:
				return
			}
		}
	}()
}

// stopPing 停止心跳
func (c *Connection) stopPing() {
	if c.pingTimer != nil {
		c.pingTimer.Stop()
		c.pingTimer = nil
	}
}

// stopReconnect 停止重连
func (c *Connection) stopReconnect() {
	if c.reconnectTimer != nil {
		c.reconnectTimer.Stop()
		c.reconnectTimer = nil
	}
}

// readLoop 消息读取循环
func (c *Connection) readLoop() {
	for {
		c.mu.RLock()
		conn := c.conn
		c.mu.RUnlock()

		if conn == nil {
			return
		}

		_, msg, err := conn.ReadMessage()
		if err != nil {
			c.handleClose(websocket.CloseAbnormalClosure, err.Error())
			return
		}

		c.handleMessage(msg)
	}
}

// handleMessage 处理消息
func (c *Connection) handleMessage(msg []byte) {
	text := string(msg)

	// 心跳响应
	if text == "PING" {
		c.Send("PONG")
		return
	}
	if text == "PONG" {
		return
	}

	// 原始消息回调
	if c.onMessage != nil {
		c.onMessage(c.channel, msg)
	}

	// 解析 JSON
	var data interface{}
	if err := json.Unmarshal(msg, &data); err != nil {
		return
	}

	// 分发消息
	if c.channel == ChannelMarket {
		c.handleMarketMessage(data)
	} else if c.channel == ChannelUser {
		c.handleUserMessage(data)
	}
}

// handleMarketMessage 处理市场频道消息
func (c *Connection) handleMarketMessage(data interface{}) {
	// Market 频道消息可能是数组
	var messages []map[string]interface{}
	switch v := data.(type) {
	case []interface{}:
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				messages = append(messages, m)
			}
		}
	case map[string]interface{}:
		messages = []map[string]interface{}{v}
	default:
		return
	}

	for _, msg := range messages {
		eventType, _ := msg["event_type"].(string)

		switch eventType {
		case "book":
			if c.onBook != nil {
				var book common.OrderBookSnapshot
				if b, err := json.Marshal(msg); err == nil {
					if json.Unmarshal(b, &book) == nil {
						c.onBook(&book)
					}
				}
			}
		case "price_change":
			// price_change 事件包含 price_changes 数组
			if c.onPriceChange != nil {
				if changes, ok := msg["price_changes"].([]interface{}); ok {
					for _, change := range changes {
						if changeMap, ok := change.(map[string]interface{}); ok {
							var event common.PriceChangeEvent
							if b, err := json.Marshal(changeMap); err == nil {
								if json.Unmarshal(b, &event) == nil {
									c.onPriceChange(&event)
								}
							}
						}
					}
				}
			}
		case "last_trade_price":
			if c.onLastTradePrice != nil {
				var event common.LastTradePrice
				if b, err := json.Marshal(msg); err == nil {
					if json.Unmarshal(b, &event) == nil {
						c.onLastTradePrice(&event)
					}
				}
			}
		case "tick_size_change":
			if c.onTickSizeChange != nil {
				var event common.TickSizeChange
				if b, err := json.Marshal(msg); err == nil {
					if json.Unmarshal(b, &event) == nil {
						c.onTickSizeChange(&event)
					}
				}
			}
		}
	}
}

// handleUserMessage 处理用户频道消息
func (c *Connection) handleUserMessage(data interface{}) {
	msg, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	eventType, _ := msg["event_type"].(string)

	switch eventType {
	case "order":
		if c.onOrder != nil {
			var order common.OrderUpdate
			if b, err := json.Marshal(msg); err == nil {
				if json.Unmarshal(b, &order) == nil {
					c.onOrder(&order)
				}
			}
		}
	case "trade":
		if c.onTrade != nil {
			var trade common.TradeNotification
			if b, err := json.Marshal(msg); err == nil {
				if json.Unmarshal(b, &trade) == nil {
					// 去重处理
					tradeID := trade.ID
					if tradeID == "" {
						tradeID = trade.TradeID
					}
					if tradeID != "" {
						if _, loaded := c.processedTrades.LoadOrStore(tradeID, true); loaded {
							return // 已处理过
						}
					}
					c.onTrade(&trade)
				}
			}
		}
	}
}

// handleClose 处理连接关闭
func (c *Connection) handleClose(code int, reason string) {
	c.mu.Lock()
	c.isConnected = false
	c.stopPing()
	intentional := c.isIntentionalClose
	c.mu.Unlock()

	if c.onDisconnected != nil {
		c.onDisconnected(code, reason)
	}

	// 非主动关闭时尝试重连
	if !intentional && c.config.MaxReconnectAttempts > 0 {
		c.tryReconnect()
	}
}

// tryReconnect 尝试重连
func (c *Connection) tryReconnect() {
	c.mu.Lock()
	if c.reconnectAttempts >= c.config.MaxReconnectAttempts {
		c.mu.Unlock()
		if c.onReconnectFail != nil {
			c.onReconnectFail(c.reconnectAttempts)
		}
		return
	}

	c.reconnectAttempts++
	attempt := c.reconnectAttempts
	delay := c.config.ReconnectDelay * time.Duration(attempt)
	c.mu.Unlock()

	if c.onReconnecting != nil {
		c.onReconnecting(attempt, delay)
	}

	c.reconnectTimer = time.AfterFunc(delay, func() {
		c.mu.RLock()
		intentional := c.isIntentionalClose
		c.mu.RUnlock()

		if !intentional {
			if err := c.Connect(); err != nil {
				if c.onError != nil {
					c.onError(err)
				}
			}
		}
	})
}

// ClearProcessedTrades 清除已处理的成交记录（用于内存管理）
func (c *Connection) ClearProcessedTrades() {
	c.processedTrades = sync.Map{}
}

// Subscribe 动态订阅更多 assets（仅 Market 频道）
func (c *Connection) Subscribe(assetIDs []string) error {
	if c.channel != ChannelMarket {
		return fmt.Errorf("subscribe only supported for market channel")
	}
	return c.Send(map[string]interface{}{
		"assets_ids": assetIDs,
		"operation":  "subscribe",
	})
}

// Unsubscribe 取消订阅 assets（仅 Market 频道）
func (c *Connection) Unsubscribe(assetIDs []string) error {
	if c.channel != ChannelMarket {
		return fmt.Errorf("unsubscribe only supported for market channel")
	}
	return c.Send(map[string]interface{}{
		"assets_ids": assetIDs,
		"operation":  "unsubscribe",
	})
}
