package clob

import (
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

// API 常量
const (
	BaseURL        = common.ClobAPIBaseURL
	ChainIDPolygon = common.PolygonChainID
)

// 默认 Builder 凭证 (公开可用)
const (
	DefaultBuilderAPIKey     = "019aaff7-3e74-7b9a-9e03-c9abe9252dc1"
	DefaultBuilderSecret     = "o6-fJoFl4QuVBFptTOaJRTi5feVCT7qtiohj2PnfYm8="
	DefaultBuilderPassphrase = "dcf31dda1700763e22ffb2fb858abd6c1ebb3d7aac1e87c381dfbd576950e3d2"
)

// Side 订单方向
type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

// OrderType 订单类型
type OrderType string

const (
	OrderTypeGTC OrderType = "GTC" // Good Till Cancelled
	OrderTypeFOK OrderType = "FOK" // Fill or Kill
	OrderTypeGTD OrderType = "GTD" // Good Till Date
	OrderTypeFAK OrderType = "FAK" // Fill and Kill
)

// SignatureType 签名类型
type SignatureType int

const (
	SignatureTypeEOA        SignatureType = 0
	SignatureTypePolyProxy  SignatureType = 1
	SignatureTypeGnosisSafe SignatureType = 2
)

// AssetType 资产类型
type AssetType string

const (
	AssetTypeCollateral  AssetType = "COLLATERAL"
	AssetTypeConditional AssetType = "CONDITIONAL"
)

// TickSize tick 大小
type TickSize string

const (
	TickSize01    TickSize = "0.1"
	TickSize001   TickSize = "0.01"
	TickSize0001  TickSize = "0.001"
	TickSize00001 TickSize = "0.0001"
)

// PriceHistoryInterval 价格历史间隔
type PriceHistoryInterval string

const (
	PriceHistoryMax     PriceHistoryInterval = "max"
	PriceHistoryOneWeek PriceHistoryInterval = "1w"
	PriceHistoryOneDay  PriceHistoryInterval = "1d"
	PriceHistory6Hours  PriceHistoryInterval = "6h"
	PriceHistoryOneHour PriceHistoryInterval = "1h"
)

// 分页常量
const (
	InitialCursor = "MA=="  // Base64("0")
	EndCursor     = "LTE="  // Base64("-1")
)

// PaginationParams 分页查询参数
type PaginationParams struct {
	NextCursor string `url:"next_cursor,omitempty"`
}

// PaginationResult 分页结果
type PaginationResult[T any] struct {
	Data       []T    `json:"data"`
	NextCursor string `json:"next_cursor"`
	Limit      int    `json:"limit,omitempty"`
	Count      int    `json:"count,omitempty"`
}

// HasMore 是否还有更多数据
func (p PaginationResult[T]) HasMore() bool {
	return p.NextCursor != "" && p.NextCursor != EndCursor
}

// BookParams 订单簿查询参数 (用于批量查询)
type BookParams struct {
	TokenID string `json:"token_id"`
	Side    Side   `json:"side,omitempty"`
}

// ApiKeyCreds API 凭证
type ApiKeyCreds struct {
	ApiKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// MarketToken 市场代币
type MarketToken struct {
	Outcome string  `json:"outcome"`
	Price   float64 `json:"price"`
	TokenID string  `json:"token_id"`
	Winner  bool    `json:"winner"`
}

// MarketRewards 市场奖励
type MarketRewards struct {
	MaxSpread float64 `json:"max_spread"`
	MinSize   float64 `json:"min_size"`
	Rates     any     `json:"rates"`
}

// Market CLOB 市场
type Market struct {
	AcceptingOrderTimestamp string        `json:"accepting_order_timestamp"`
	AcceptingOrders         bool          `json:"accepting_orders"`
	Active                  bool          `json:"active"`
	Archived                bool          `json:"archived"`
	Closed                  bool          `json:"closed"`
	ConditionID             string        `json:"condition_id"`
	Description             string        `json:"description"`
	EnableOrderBook         bool          `json:"enable_order_book"`
	EndDateISO              string        `json:"end_date_iso"`
	FPMM                    string        `json:"fpmm"`
	GameStartTime           string        `json:"game_start_time"`
	Icon                    string        `json:"icon"`
	Image                   string        `json:"image"`
	Is5050Outcome           bool          `json:"is_50_50_outcome"`
	MakerBaseFee            float64       `json:"maker_base_fee"`
	MarketSlug              string        `json:"market_slug"`
	MinimumOrderSize        float64       `json:"minimum_order_size"`
	MinimumTickSize         float64       `json:"minimum_tick_size"`
	NegRisk                 bool          `json:"neg_risk"`
	NegRiskMarketID         string        `json:"neg_risk_market_id"`
	NegRiskRequestID        string        `json:"neg_risk_request_id"`
	NotificationsEnabled    bool          `json:"notifications_enabled"`
	Question                string        `json:"question"`
	QuestionID              string        `json:"question_id"`
	Rewards                 MarketRewards `json:"rewards"`
	SecondsDelay            int           `json:"seconds_delay"`
	Tags                    []string      `json:"tags"`
	TakerBaseFee            float64       `json:"taker_base_fee"`
	Tokens                  []MarketToken `json:"tokens"`
}

// SimplifiedMarket 简化市场
type SimplifiedMarket struct {
	AcceptingOrders bool              `json:"accepting_orders"`
	Active          bool              `json:"active"`
	Archived        bool              `json:"archived"`
	Closed          bool              `json:"closed"`
	ConditionID     string            `json:"condition_id"`
	Rewards         MarketRewards     `json:"rewards"`
	Tokens          []SimplifiedToken `json:"tokens"`
}

// SimplifiedToken 简化代币
type SimplifiedToken struct {
	Outcome string  `json:"outcome"`
	Price   float64 `json:"price"`
	TokenID string  `json:"token_id"`
}

// MarketsResponse 市场列表响应
type MarketsResponse struct {
	Data       []Market `json:"data"`
	NextCursor string   `json:"next_cursor"`
	Limit      int      `json:"limit"`
	Count      int      `json:"count"`
}

// HasMore 是否还有更多数据
func (r MarketsResponse) HasMore() bool {
	return r.NextCursor != "" && r.NextCursor != EndCursor
}

// SimplifiedMarketsResponse 简化市场列表响应
type SimplifiedMarketsResponse struct {
	Data       []SimplifiedMarket `json:"data"`
	NextCursor string             `json:"next_cursor"`
	Limit      int                `json:"limit"`
	Count      int                `json:"count"`
}

// HasMore 是否还有更多数据
func (r SimplifiedMarketsResponse) HasMore() bool {
	return r.NextCursor != "" && r.NextCursor != EndCursor
}

// OrderSummary 订单摘要
type OrderSummary struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderBookSummary 订单簿摘要
type OrderBookSummary struct {
	Market       string         `json:"market"`
	AssetID      string         `json:"asset_id"`
	Timestamp    string         `json:"timestamp"`
	Bids         []OrderSummary `json:"bids"`
	Asks         []OrderSummary `json:"asks"`
	MinOrderSize string         `json:"min_order_size"`
	TickSize     string         `json:"tick_size"`
	NegRisk      bool           `json:"neg_risk"`
	Hash         string         `json:"hash"`
}

// PriceResponse 价格响应
type PriceResponse struct {
	Price string `json:"price"`
}

// MidpointResponse 中间价响应
type MidpointResponse struct {
	Mid string `json:"mid"`
}

// SpreadResponse 价差响应
type SpreadResponse struct {
	Spread string `json:"spread"`
}

// PriceHistoryParams 价格历史查询参数
type PriceHistoryParams struct {
	Market   string               `url:"market"`
	StartTs  int64                `url:"startTs,omitempty"`
	EndTs    int64                `url:"endTs,omitempty"`
	Fidelity int                  `url:"fidelity,omitempty"`
	Interval PriceHistoryInterval `url:"interval"`
}

// MarketPrice 市场价格
type MarketPrice struct {
	T int64   `json:"t"`
	P float64 `json:"p"`
}

// LastTradePrice 最新成交价
type LastTradePrice struct {
	Price string `json:"price"`
	Side  string `json:"side"`
}

// LastTradePriceWithToken 带代币的最新成交价
type LastTradePriceWithToken struct {
	Price   string `json:"price"`
	Side    string `json:"side"`
	TokenID string `json:"token_id"`
}

// MarketTradeEventMarket 市场交易事件市场信息
type MarketTradeEventMarket struct {
	ConditionID string `json:"condition_id"`
	AssetID     string `json:"asset_id"`
	Question    string `json:"question"`
	Icon        string `json:"icon"`
	Slug        string `json:"slug"`
}

// MarketTradeEventUser 市场交易事件用户信息
type MarketTradeEventUser struct {
	Address                 string `json:"address"`
	Username                string `json:"username"`
	ProfilePicture          string `json:"profile_picture"`
	OptimizedProfilePicture string `json:"optimized_profile_picture"`
	Pseudonym               string `json:"pseudonym"`
}

// MarketTradeEvent 市场交易事件
type MarketTradeEvent struct {
	EventType       string                 `json:"event_type"`
	Market          MarketTradeEventMarket `json:"market"`
	User            MarketTradeEventUser   `json:"user"`
	Side            Side                   `json:"side"`
	Size            string                 `json:"size"`
	FeeRateBps      string                 `json:"fee_rate_bps"`
	Price           string                 `json:"price"`
	Outcome         string                 `json:"outcome"`
	OutcomeIndex    int                    `json:"outcome_index"`
	TransactionHash string                 `json:"transaction_hash"`
	Timestamp       string                 `json:"timestamp"`
}

// UserOrder 用户订单
type UserOrder struct {
	TokenID    string  `json:"tokenID"`
	Price      float64 `json:"price"`
	Size       float64 `json:"size"`
	Side       Side    `json:"side"`
	FeeRateBps int     `json:"feeRateBps,omitempty"`
	Nonce      int64   `json:"nonce,omitempty"`
	Expiration int64   `json:"expiration,omitempty"`
	Taker      string  `json:"taker,omitempty"`
}

// UserMarketOrder 用户市价单
type UserMarketOrder struct {
	TokenID    string    `json:"tokenID"`
	Amount     float64   `json:"amount"`
	Side       Side      `json:"side"`
	Price      float64   `json:"price,omitempty"`
	FeeRateBps int       `json:"feeRateBps,omitempty"`
	Nonce      int64     `json:"nonce,omitempty"`
	Taker      string    `json:"taker,omitempty"`
	OrderType  OrderType `json:"orderType,omitempty"`
}

// CreateOrderOptions 创建订单选项
type CreateOrderOptions struct {
	TickSize TickSize `json:"tickSize"`
	NegRisk  bool     `json:"negRisk,omitempty"`
}

// SignedOrder 签名订单
type SignedOrder struct {
	Salt          string `json:"salt"`
	Maker         string `json:"maker"`
	Signer        string `json:"signer"`
	Taker         string `json:"taker"`
	TokenID       string `json:"tokenId"`
	MakerAmount   string `json:"makerAmount"`
	TakerAmount   string `json:"takerAmount"`
	Side          int    `json:"side"`
	Expiration    string `json:"expiration"`
	Nonce         string `json:"nonce"`
	FeeRateBps    string `json:"feeRateBps"`
	SignatureType int    `json:"signatureType"`
	Signature     string `json:"signature"`
}

// OrderResponse 订单响应
type OrderResponse struct {
	Success            bool     `json:"success"`
	ErrorMsg           string   `json:"errorMsg"`
	OrderID            string   `json:"orderID"`
	TransactionsHashes []string `json:"transactionsHashes"`
	Status             string   `json:"status"`
	TakingAmount       string   `json:"takingAmount"`
	MakingAmount       string   `json:"makingAmount"`
}

// PostOrdersArgs 批量提交订单参数
type PostOrdersArgs struct {
	Order     SignedOrder `json:"order"`
	OrderType OrderType   `json:"orderType"`
}

// CancelOrdersResponse 取消订单响应
type CancelOrdersResponse struct {
	Canceled    []string       `json:"canceled"`
	NotCanceled map[string]any `json:"not_canceled"`
}

// OrderMarketCancelParams 市场订单取消参数
type OrderMarketCancelParams struct {
	Market  string `json:"market,omitempty"`
	AssetID string `json:"asset_id,omitempty"`
}

// OpenOrder 未结订单
type OpenOrder struct {
	ID              string   `json:"id"`
	Status          string   `json:"status"`
	Owner           string   `json:"owner"`
	MakerAddress    string   `json:"maker_address"`
	Market          string   `json:"market"`
	AssetID         string   `json:"asset_id"`
	Side            string   `json:"side"`
	OriginalSize    string   `json:"original_size"`
	SizeMatched     string   `json:"size_matched"`
	Price           string   `json:"price"`
	AssociateTrades []string `json:"associate_trades"`
	Outcome         string   `json:"outcome"`
	CreatedAt       int64    `json:"created_at"`
	Expiration      string   `json:"expiration"`
	OrderType       string   `json:"order_type"`
}

// OpenOrderParams 未结订单查询参数
type OpenOrderParams struct {
	ID      string `url:"id,omitempty" json:"id,omitempty"`
	Market  string `url:"market,omitempty" json:"market,omitempty"`
	AssetID string `url:"asset_id,omitempty" json:"asset_id,omitempty"`
}

// OpenOrdersResponse 未结订单列表响应
type OpenOrdersResponse struct {
	Data       []OpenOrder `json:"data"`
	NextCursor string      `json:"next_cursor"`
	Limit      int         `json:"limit"`
	Count      int         `json:"count"`
}

// HasMore 是否还有更多数据
func (r OpenOrdersResponse) HasMore() bool {
	return r.NextCursor != "" && r.NextCursor != EndCursor
}

// MakerOrder Maker 订单详情
type MakerOrder struct {
	OrderID       string `json:"order_id"`
	Owner         string `json:"owner"`
	MakerAddress  string `json:"maker_address"`
	MatchedAmount string `json:"matched_amount"`
	Price         string `json:"price"`
	FeeRateBps    string `json:"fee_rate_bps"`
	AssetID       string `json:"asset_id"`
	Outcome       string `json:"outcome"`
	Side          Side   `json:"side"`
}

// Trade 交易记录
type Trade struct {
	ID              string       `json:"id"`
	TakerOrderID    string       `json:"taker_order_id"`
	Market          string       `json:"market"`
	AssetID         string       `json:"asset_id"`
	Side            Side         `json:"side"`
	Size            string       `json:"size"`
	FeeRateBps      string       `json:"fee_rate_bps"`
	Price           string       `json:"price"`
	Status          string       `json:"status"`
	MatchTime       string       `json:"match_time"`
	LastUpdate      string       `json:"last_update"`
	Outcome         string       `json:"outcome"`
	BucketIndex     int          `json:"bucket_index"`
	Owner           string       `json:"owner"`
	MakerAddress    string       `json:"maker_address"`
	MakerOrders     []MakerOrder `json:"maker_orders"`
	TransactionHash string       `json:"transaction_hash"`
	TraderSide      string       `json:"trader_side"`
}

// TradeParams 交易查询参数
type TradeParams struct {
	ID           string `url:"id,omitempty" json:"id,omitempty"`
	MakerAddress string `url:"maker_address,omitempty" json:"maker_address,omitempty"`
	Market       string `url:"market,omitempty" json:"market,omitempty"`
	AssetID      string `url:"asset_id,omitempty" json:"asset_id,omitempty"`
	Before       string `url:"before,omitempty" json:"before,omitempty"`
	After        string `url:"after,omitempty" json:"after,omitempty"`
}

// TradesResponse 交易列表响应
type TradesResponse struct {
	Data       []Trade `json:"data"`
	NextCursor string  `json:"next_cursor"`
	Limit      int     `json:"limit"`
	Count      int     `json:"count"`
}

// HasMore 是否还有更多数据
func (r TradesResponse) HasMore() bool {
	return r.NextCursor != "" && r.NextCursor != EndCursor
}

// BalanceAllowanceParams 余额授权查询参数
type BalanceAllowanceParams struct {
	AssetType AssetType `url:"asset_type"`
	TokenID   string    `url:"token_id,omitempty"`
}

// BalanceAllowanceResponse 余额授权响应
type BalanceAllowanceResponse struct {
	Balance   string `json:"balance"`
	Allowance string `json:"allowance"`
}

// NotificationType 通知类型
type NotificationType int

const (
	NotificationOrderCancellation NotificationType = 1
	NotificationOrderFill         NotificationType = 2
	NotificationMarketResolved    NotificationType = 4
)

// Notification 通知
type Notification struct {
	ID        int              `json:"id"`
	Owner     string           `json:"owner"`
	Payload   any              `json:"payload"`
	Timestamp int64            `json:"timestamp,omitempty"`
	Type      NotificationType `json:"type"`
}

// DropNotificationParams 删除通知参数
type DropNotificationParams struct {
	IDs []string `json:"ids"`
}

// OrderScoringResponse 订单评分响应
type OrderScoringResponse struct {
	Scoring bool `json:"scoring"`
}

// OrdersScoringParams 多订单评分查询参数
type OrdersScoringParams struct {
	OrderIDs []string `json:"orderIds"`
}

// BuilderTrade Builder 交易
type BuilderTrade struct {
	ID              string `json:"id"`
	TradeType       string `json:"tradeType"`
	TakerOrderHash  string `json:"takerOrderHash"`
	Builder         string `json:"builder"`
	Market          string `json:"market"`
	AssetID         string `json:"assetId"`
	Side            string `json:"side"`
	Size            string `json:"size"`
	SizeUsdc        string `json:"sizeUsdc"`
	Price           string `json:"price"`
	Status          string `json:"status"`
	Outcome         string `json:"outcome"`
	OutcomeIndex    int    `json:"outcomeIndex"`
	Owner           string `json:"owner"`
	Maker           string `json:"maker"`
	TransactionHash string `json:"transactionHash"`
	MatchTime       string `json:"matchTime"`
	BucketIndex     int    `json:"bucketIndex"`
	Fee             string `json:"fee"`
	FeeUsdc         string `json:"feeUsdc"`
	ErrMsg          string `json:"err_msg,omitempty"`
	CreatedAt       string `json:"createdAt,omitempty"`
	UpdatedAt       string `json:"updatedAt,omitempty"`
}

// RewardsToken 奖励代币
type RewardsToken struct {
	TokenID string  `json:"token_id"`
	Outcome string  `json:"outcome"`
	Price   float64 `json:"price"`
}

// RewardsConfig 奖励配置
type RewardsConfig struct {
	AssetAddress string  `json:"asset_address"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	RatePerDay   float64 `json:"rate_per_day"`
	TotalRewards float64 `json:"total_rewards"`
}

// MarketReward 市场奖励
type MarketReward struct {
	ConditionID       string          `json:"condition_id"`
	Question          string          `json:"question"`
	MarketSlug        string          `json:"market_slug"`
	EventSlug         string          `json:"event_slug"`
	Image             string          `json:"image"`
	RewardsMaxSpread  float64         `json:"rewards_max_spread"`
	RewardsMinSize    float64         `json:"rewards_min_size"`
	Tokens            []RewardsToken  `json:"tokens"`
	RewardsConfigList []RewardsConfig `json:"rewards_config"`
}

// UserEarning 用户收益
type UserEarning struct {
	Date         string  `json:"date"`
	ConditionID  string  `json:"condition_id"`
	AssetAddress string  `json:"asset_address"`
	MakerAddress string  `json:"maker_address"`
	Earnings     float64 `json:"earnings"`
	AssetRate    float64 `json:"asset_rate"`
}

// TotalUserEarning 用户总收益
type TotalUserEarning struct {
	Date         string  `json:"date"`
	AssetAddress string  `json:"asset_address"`
	MakerAddress string  `json:"maker_address"`
	Earnings     float64 `json:"earnings"`
	AssetRate    float64 `json:"asset_rate"`
}

// RewardsPercentages 奖励百分比
type RewardsPercentages map[string]float64

// BuilderApiKey Builder API 凭证
type BuilderApiKey struct {
	Key        string `json:"key"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// BuilderApiKeyResponse Builder API Key 响应
type BuilderApiKeyResponse struct {
	Key       string `json:"key"`
	CreatedAt string `json:"createdAt,omitempty"`
	RevokedAt string `json:"revokedAt,omitempty"`
}

// BanStatus 封禁状态
type BanStatus struct {
	ClosedOnly bool `json:"closed_only"`
}

// TickSizeResponse TickSize 响应
type TickSizeResponse struct {
	MinimumTickSize string `json:"minimum_tick_size"`
}

// NegRiskResponse NegRisk 响应
type NegRiskResponse struct {
	NegRisk bool `json:"neg_risk"`
}

// FeeRateResponse FeeRate 响应
type FeeRateResponse struct {
	BaseFee float64 `json:"base_fee"`
}

// 内部请求类型
type postOrderRequest struct {
	Order     SignedOrder `json:"order"`
	Owner     string      `json:"owner"`
	OrderType OrderType   `json:"orderType"`
	DeferExec bool        `json:"deferExec,omitempty"`
}

type postOrdersRequest struct {
	Orders []postOrderRequest `json:"orders"`
}
