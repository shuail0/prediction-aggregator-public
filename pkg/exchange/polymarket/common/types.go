package common

import (
	"encoding/json"
)

// FlexString 可以从 JSON 字符串或数字解析的灵活类型
type FlexString string

func (f *FlexString) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*f = FlexString(s)
		return nil
	}
	var n json.Number
	if err := json.Unmarshal(data, &n); err == nil {
		*f = FlexString(n.String())
		return nil
	}
	*f = FlexString(string(data))
	return nil
}

// ========== Gamma API 类型 ==========

// Event 事件
type Event struct {
	ID               string     `json:"id"`
	Slug             string     `json:"slug"`
	Title            string     `json:"title"`
	Description      string     `json:"description"`
	StartDate        string     `json:"startDate"`
	EndDate          string     `json:"endDate"`
	CreationDate     string     `json:"creationDate"`
	Closed           bool       `json:"closed"`
	Active           bool       `json:"active"`
	Archived         bool       `json:"archived"`
	New              bool       `json:"new"`
	Featured         bool       `json:"featured"`
	Restricted       bool       `json:"restricted"`
	LiquidityClob    FlexString `json:"liquidityClob"`
	Volume           FlexString `json:"volume"`
	Volume24hr       FlexString `json:"volume24hr"`
	OpenInterest     FlexString `json:"openInterest"`
	CompetitorCount  int        `json:"competitorCount"`
	CommentCount     int        `json:"commentCount"`
	Markets          []Market   `json:"markets"`
	Tags             []Tag      `json:"tags"`
	NegRisk          bool       `json:"negRisk"`
	NegRiskMarketID  string     `json:"negRiskMarketId"`
	NegRiskRequestID string     `json:"negRiskRequestId"`
	EnableOrderBook  bool       `json:"enableOrderBook"`
	ClobRewards      []any      `json:"clobRewards"`
}

// Market 市场
type Market struct {
	ID                    string     `json:"id"`
	Slug                  string     `json:"slug"`
	Question              string     `json:"question"`
	ConditionID           string     `json:"conditionId"`
	Description           string     `json:"description"`
	EndDate               string     `json:"endDate"`
	StartDate             string     `json:"startDate"`
	CreatedAt             string     `json:"createdAt"`
	UpdatedAt             string     `json:"updatedAt"`
	Closed                bool       `json:"closed"`
	Active                bool       `json:"active"`
	Archived              bool       `json:"archived"`
	New                   bool       `json:"new"`
	Featured              bool       `json:"featured"`
	Restricted            bool       `json:"restricted"`
	GroupItemTitle        string     `json:"groupItemTitle"`
	GroupItemThreshold    FlexString `json:"groupItemThreshold"`
	Volume                FlexString `json:"volume"`
	Volume24hr            FlexString `json:"volume24hr"`
	Liquidity             FlexString `json:"liquidity"`
	LiquidityClob         FlexString `json:"liquidityClob"`
	OpenInterest          FlexString `json:"openInterest"`
	OutcomePrices         string     `json:"outcomePrices"`
	Outcomes              string     `json:"outcomes"`
	ClobTokenIds          string     `json:"clobTokenIds"`
	OrderPriceMinTickSize FlexString `json:"orderPriceMinTickSize"`
	RewardsMinSize        FlexString `json:"rewardsMinSize"`
	RewardsMaxSpread      FlexString `json:"rewardsMaxSpread"`
	Spread                FlexString `json:"spread"`
	NegRisk               bool       `json:"negRisk"`
	NegRiskMarketID       string     `json:"negRiskMarketId"`
	NegRiskRequestID      string     `json:"negRiskRequestId"`
	EnableOrderBook       bool       `json:"enableOrderBook"`
	AcceptingOrders       bool       `json:"acceptingOrders"`
	AcceptingOrderTs      string     `json:"acceptingOrderTimestamp"`
	Winner                string     `json:"winner"`
	Tags                  []Tag      `json:"tags"`
	OneDayPriceChange     FlexString `json:"oneDayPriceChange"`
	ClobRewards           []any      `json:"clobRewards"`
	EventSlug             string     `json:"eventSlug"`
}

// Tag 标签
type Tag struct {
	ID          string `json:"id"`
	Slug        string `json:"slug"`
	Label       string `json:"label"`
	ForceShow   bool   `json:"forceShow"`
	ForceHide   bool   `json:"forceHide"`
	IsCarousel  bool   `json:"isCarousel"`
	PublishedAt string `json:"publishedAt"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	EventCount  int    `json:"eventCount"`
}

// TagQueryParams 标签查询参数
type TagQueryParams struct {
	Limit           int    `url:"limit,omitempty"`
	Offset          int    `url:"offset,omitempty"`
	Order           string `url:"order,omitempty"`
	Ascending       bool   `url:"ascending,omitempty"`
	IncludeTemplate bool   `url:"include_template,omitempty"`
	IsCarousel      bool   `url:"is_carousel,omitempty"`
}

// RelatedTag 关联标签
type RelatedTag struct {
	ID           string `json:"id"`
	TagAID       string `json:"tagAId"`
	TagBID       string `json:"tagBId"`
	Relationship string `json:"relationship"`
}

// Series 系列
type Series struct {
	ID          string  `json:"id"`
	Ticker      string  `json:"ticker"`
	Slug        string  `json:"slug"`
	Title       string  `json:"title"`
	Subtitle    string  `json:"subtitle"`
	SeriesType  string  `json:"seriesType"`
	Recurrence  string  `json:"recurrence"`
	Description string  `json:"description"`
	Image       string  `json:"image"`
	Icon        string  `json:"icon"`
	Active      bool    `json:"active"`
	Closed      bool    `json:"closed"`
	Archived    bool    `json:"archived"`
	Volume      float64 `json:"volume"`
	Volume24hr  float64 `json:"volume24hr"`
	Liquidity   float64 `json:"liquidity"`
	Events      []Event `json:"events"`
	Tags        []Tag   `json:"tags"`
}

// SeriesQueryParams 系列查询参数
type SeriesQueryParams struct {
	Limit       int      `url:"limit,omitempty"`
	Offset      int      `url:"offset,omitempty"`
	Order       string   `url:"order,omitempty"`
	Ascending   bool     `url:"ascending,omitempty"`
	Slug        []string `url:"slug,omitempty"`
	Closed      *bool    `url:"closed,omitempty"`
	IncludeChat bool     `url:"include_chat,omitempty"`
	Recurrence  string   `url:"recurrence,omitempty"`
}

// Comment 评论
type Comment struct {
	ID               string         `json:"id"`
	Body             string         `json:"body"`
	ParentEntityType string         `json:"parentEntityType"`
	ParentEntityID   int            `json:"parentEntityID"`
	ParentCommentID  string         `json:"parentCommentID"`
	UserAddress      string         `json:"userAddress"`
	ReplyAddress     string         `json:"replyAddress"`
	CreatedAt        string         `json:"createdAt"`
	UpdatedAt        string         `json:"updatedAt"`
	Profile          *PublicProfile `json:"profile"`
	ReportCount      int            `json:"reportCount"`
	ReactionCount    int            `json:"reactionCount"`
}

// CommentQueryParams 评论查询参数
type CommentQueryParams struct {
	Limit            int    `url:"limit,omitempty"`
	Offset           int    `url:"offset,omitempty"`
	Order            string `url:"order,omitempty"`
	Ascending        bool   `url:"ascending,omitempty"`
	ParentEntityType string `url:"parent_entity_type,omitempty"`
	ParentEntityID   string `url:"parent_entity_id,omitempty"`
	GetPositions     bool   `url:"get_positions,omitempty"`
	HoldersOnly      bool   `url:"holders_only,omitempty"`
}

// PublicProfile 公开用户资料
type PublicProfile struct {
	Name                  string `json:"name"`
	Pseudonym             string `json:"pseudonym"`
	DisplayUsernamePublic bool   `json:"displayUsernamePublic"`
	Bio                   string `json:"bio"`
	ProxyWallet           string `json:"proxyWallet"`
	ProfileImage          string `json:"profileImage"`
	XUsername             string `json:"xUsername"`
	VerifiedBadge         bool   `json:"verifiedBadge"`
	CreatedAt             string `json:"createdAt"`
}

// SportsMarketTypes 体育市场类型
type SportsMarketTypes struct {
	MarketTypes []string `json:"marketTypes"`
}

// MarketQueryParams 市场查询参数
type MarketQueryParams struct {
	Limit           int    `url:"limit,omitempty"`
	Offset          int    `url:"offset,omitempty"`
	Order           string `url:"order,omitempty"`
	Ascending       bool   `url:"ascending,omitempty"`
	ID              int    `url:"id,omitempty"`
	Slug            string `url:"slug,omitempty"`
	Archived        *bool  `url:"archived,omitempty"`
	Active          *bool  `url:"active,omitempty"`
	Closed          *bool  `url:"closed,omitempty"`
	ClobTokenIDs    string `url:"clob_token_ids,omitempty"`
	ConditionIDs    string `url:"condition_ids,omitempty"`
	LiquidityNumMin int    `url:"liquidity_num_min,omitempty"`
	LiquidityNumMax int    `url:"liquidity_num_max,omitempty"`
	VolumeNumMin    int    `url:"volume_num_min,omitempty"`
	VolumeNumMax    int    `url:"volume_num_max,omitempty"`
	StartDateMin    string `url:"start_date_min,omitempty"`
	StartDateMax    string `url:"start_date_max,omitempty"`
	EndDateMin      string `url:"end_date_min,omitempty"`
	EndDateMax      string `url:"end_date_max,omitempty"`
	TagID           int    `url:"tag_id,omitempty"`
	RelatedTags     bool   `url:"related_tags,omitempty"`
}

// EventQueryParams 事件查询参数
type EventQueryParams struct {
	MarketQueryParams
	TagSlug string `url:"tag_slug,omitempty"`
}

// SearchParams 搜索参数
type SearchParams struct {
	Q                string `url:"q"`
	Cache            bool   `url:"cache,omitempty"`
	EventsStatus     string `url:"events-status,omitempty"`
	LimitPerType     int    `url:"limit-per-type,omitempty"`
	Page             int    `url:"page,omitempty"`
	EventsTag        string `url:"events-tag,omitempty"`
	KeepClosedMarket bool   `url:"keep-closed-markets,omitempty"`
	Sort             string `url:"sort,omitempty"`
	Ascending        bool   `url:"ascending,omitempty"`
	SearchTags       bool   `url:"search-tags,omitempty"`
	SearchProfiles   bool   `url:"search-profiles,omitempty"`
	Recurrence       string `url:"recurrence,omitempty"`
	ExcludeTagID     string `url:"exclude-tag-id,omitempty"`
	Optimized        bool   `url:"optimized,omitempty"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Events   []Event   `json:"events"`
	Markets  []Market  `json:"markets"`
	Profiles []Profile `json:"profiles"`
}

// Profile 用户档案
type Profile struct {
	Address   string `json:"address"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Bio       string `json:"bio"`
	ProfileURL string `json:"profileUrl"`
	AvatarURL  string `json:"avatarUrl"`
}

// ========== Data API 类型 ==========

// Position 持仓
type Position struct {
	ProxyWallet        string  `json:"proxyWallet"`
	Asset              string  `json:"asset"`
	ConditionID        string  `json:"conditionId"`
	Size               float64 `json:"size"`
	AveragePrice       float64 `json:"avgPrice"`
	InitialValue       float64 `json:"initialValue"`
	CurrentValue       float64 `json:"currentValue"`
	CashPnl            float64 `json:"cashPnl"`
	PercentPnl         float64 `json:"percentPnl"`
	TotalBought        float64 `json:"totalBought"`
	RealizedPnl        float64 `json:"realizedPnl"`
	PercentRealizedPnl float64 `json:"percentRealizedPnl"`
	CurrentPrice       float64 `json:"curPrice"`
	Redeemable         bool    `json:"redeemable"`
	Mergeable          bool    `json:"mergeable"`
	Title              string  `json:"title"`
	Slug               string  `json:"slug"`
	Icon               string  `json:"icon"`
	EventID            string  `json:"eventId"`
	EventSlug          string  `json:"eventSlug"`
	Outcome            string  `json:"outcome"`
	OutcomeIndex       int     `json:"outcomeIndex"`
	OppositeOutcome    string  `json:"oppositeOutcome"`
	OppositeAsset      string  `json:"oppositeAsset"`
	EndDate            string  `json:"endDate"`
	NegativeRisk       bool    `json:"negativeRisk"`
}

// PositionQueryParams 持仓查询参数
type PositionQueryParams struct {
	User          string `url:"user"`
	SizeThreshold string `url:"sizeThreshold,omitempty"`
	Limit         int    `url:"limit,omitempty"`
	Offset        int    `url:"offset,omitempty"`
	SortBy        string `url:"sortBy,omitempty"`
	SortDirection string `url:"sortDirection,omitempty"`
}

// UserStats 用户统计
type UserStats struct {
	TotalVolume    float64 `json:"totalVolume"`
	TotalPnl       float64 `json:"totalPnl"`
	TotalTrades    int     `json:"totalTrades"`
	WinRate        float64 `json:"winRate"`
	AvgReturn      float64 `json:"avgReturn"`
	BestTrade      float64 `json:"bestTrade"`
	WorstTrade     float64 `json:"worstTrade"`
	TotalPositions int     `json:"totalPositions"`
}

// TradeHistory 交易历史
type TradeHistory struct {
	ProxyWallet           string  `json:"proxyWallet"`
	Side                  string  `json:"side"`
	Asset                 string  `json:"asset"`
	ConditionID           string  `json:"conditionId"`
	Size                  float64 `json:"size"`
	Price                 float64 `json:"price"`
	Timestamp             int64   `json:"timestamp"`
	Title                 string  `json:"title"`
	Slug                  string  `json:"slug"`
	Icon                  string  `json:"icon"`
	EventSlug             string  `json:"eventSlug"`
	Outcome               string  `json:"outcome"`
	OutcomeIndex          int     `json:"outcomeIndex"`
	Name                  string  `json:"name"`
	Pseudonym             string  `json:"pseudonym"`
	Bio                   string  `json:"bio"`
	ProfileImage          string  `json:"profileImage"`
	ProfileImageOptimized string  `json:"profileImageOptimized"`
	TransactionHash       string  `json:"transactionHash"`
}

// TradeHistoryParams 交易历史查询参数
type TradeHistoryParams struct {
	User      string `url:"user"`
	MarketID  string `url:"marketId,omitempty"`
	Limit     int    `url:"limit,omitempty"`
	Offset    int    `url:"offset,omitempty"`
	StartDate string `url:"startDate,omitempty"`
	EndDate   string `url:"endDate,omitempty"`
}

// ActivityParams 活动查询参数
type ActivityParams struct {
	User          string `url:"user"`
	Type          string `url:"type,omitempty"`
	Side          string `url:"side,omitempty"`
	Start         int64  `url:"start,omitempty"`
	End           int64  `url:"end,omitempty"`
	SortBy        string `url:"sortBy,omitempty"`
	SortDirection string `url:"sortDirection,omitempty"`
	Limit         int    `url:"limit,omitempty"`
	Offset        int    `url:"offset,omitempty"`
}

// Activity 用户活动记录
type Activity struct {
	ProxyWallet           string  `json:"proxyWallet"`
	Timestamp             int64   `json:"timestamp"`
	ConditionID           string  `json:"conditionId"`
	Type                  string  `json:"type"`
	Size                  float64 `json:"size"`
	UsdcSize              float64 `json:"usdcSize"`
	TransactionHash       string  `json:"transactionHash"`
	Price                 float64 `json:"price"`
	Asset                 string  `json:"asset"`
	Side                  string  `json:"side"`
	OutcomeIndex          int     `json:"outcomeIndex"`
	Title                 string  `json:"title"`
	Slug                  string  `json:"slug"`
	Icon                  string  `json:"icon"`
	EventSlug             string  `json:"eventSlug"`
	Outcome               string  `json:"outcome"`
	Name                  string  `json:"name"`
	Pseudonym             string  `json:"pseudonym"`
	Bio                   string  `json:"bio"`
	ProfileImage          string  `json:"profileImage"`
	ProfileImageOptimized string  `json:"profileImageOptimized"`
}

// ClosedPositionParams 已平仓持仓查询参数
type ClosedPositionParams struct {
	User          string `url:"user"`
	Limit         int    `url:"limit,omitempty"`
	Offset        int    `url:"offset,omitempty"`
	SortBy        string `url:"sortBy,omitempty"`
	SortDirection string `url:"sortDirection,omitempty"`
}

// ClosedPosition 已平仓持仓
type ClosedPosition struct {
	ProxyWallet     string  `json:"proxyWallet"`
	Asset           string  `json:"asset"`
	ConditionID     string  `json:"conditionId"`
	AveragePrice    float64 `json:"avgPrice"`
	TotalBought     float64 `json:"totalBought"`
	RealizedPnl     float64 `json:"realizedPnl"`
	CurrentPrice    float64 `json:"curPrice"`
	Title           string  `json:"title"`
	Slug            string  `json:"slug"`
	Icon            string  `json:"icon"`
	EventSlug       string  `json:"eventSlug"`
	Outcome         string  `json:"outcome"`
	OutcomeIndex    int     `json:"outcomeIndex"`
	OppositeOutcome string  `json:"oppositeOutcome"`
	OppositeAsset   string  `json:"oppositeAsset"`
	EndDate         string  `json:"endDate"`
	Timestamp       int64   `json:"timestamp"`
}

// PortfolioValue 持仓总价值
type PortfolioValue struct {
	User  string  `json:"user"`
	Value float64 `json:"value"`
}

// HoldersParams 持有者查询参数
type HoldersParams struct {
	Market string `url:"market"`
	Limit  int    `url:"limit,omitempty"`
	Offset int    `url:"offset,omitempty"`
}

// Holder 持有者信息
type Holder struct {
	ProxyWallet           string  `json:"proxyWallet"`
	Bio                   string  `json:"bio"`
	Asset                 string  `json:"asset"`
	Pseudonym             string  `json:"pseudonym"`
	Amount                float64 `json:"amount"`
	DisplayUsernamePublic bool    `json:"displayUsernamePublic"`
	OutcomeIndex          int     `json:"outcomeIndex"`
	Name                  string  `json:"name"`
	ProfileImage          string  `json:"profileImage"`
	ProfileImageOptimized string  `json:"profileImageOptimized"`
	Verified              bool    `json:"verified"`
}

// MarketHolders 市场持有者响应
type MarketHolders struct {
	Token   string   `json:"token"`
	Holders []Holder `json:"holders"`
}

// OpenInterest Open Interest 数据
type OpenInterest struct {
	Market string  `json:"market"`
	Value  float64 `json:"value"`
}

// LiveVolumeParams 实时交易量查询参数
type LiveVolumeParams struct {
	ID int `url:"id"`
}

// LiveVolume 实时交易量
type LiveVolume struct {
	Total   float64       `json:"total"`
	Markets []MarketValue `json:"markets"`
}

// MarketValue 市场价值
type MarketValue struct {
	Market string  `json:"market"`
	Value  float64 `json:"value"`
}

// MarketsTraded 用户交易过的市场数量
type MarketsTraded struct {
	User   string `json:"user"`
	Traded int    `json:"traded"`
}

// LeaderboardParams 排行榜查询参数
type LeaderboardParams struct {
	Category   string `url:"category,omitempty"`
	TimePeriod string `url:"timePeriod,omitempty"`
	OrderBy    string `url:"orderBy,omitempty"`
	Limit      int    `url:"limit,omitempty"`
	Offset     int    `url:"offset,omitempty"`
}

// LeaderboardEntry 排行榜条目
type LeaderboardEntry struct {
	Rank          string  `json:"rank"`
	ProxyWallet   string  `json:"proxyWallet"`
	UserName      string  `json:"userName"`
	XUsername     string  `json:"xUsername"`
	VerifiedBadge bool    `json:"verifiedBadge"`
	Volume        float64 `json:"vol"`
	PnL           float64 `json:"pnl"`
	ProfileImage  string  `json:"profileImage"`
}

// BuilderLeaderboardParams Builder 排行榜查询参数
type BuilderLeaderboardParams struct {
	Limit  int `url:"limit,omitempty"`
	Offset int `url:"offset,omitempty"`
}

// BuilderLeaderboardEntry Builder 排行榜条目
type BuilderLeaderboardEntry struct {
	Rank        string  `json:"rank"`
	Builder     string  `json:"builder"`
	Volume      float64 `json:"volume"`
	ActiveUsers int     `json:"activeUsers"`
	Verified    bool    `json:"verified"`
	BuilderLogo string  `json:"builderLogo"`
}

// BuilderVolumeParams Builder 交易量时序查询参数
type BuilderVolumeParams struct {
	Limit  int `url:"limit,omitempty"`
	Offset int `url:"offset,omitempty"`
}

// BuilderVolumeEntry Builder 交易量时序条目
type BuilderVolumeEntry struct {
	Date        string  `json:"dt"`
	Builder     string  `json:"builder"`
	BuilderLogo string  `json:"builderLogo"`
	Verified    bool    `json:"verified"`
	Volume      float64 `json:"volume"`
	ActiveUsers int     `json:"activeUsers"`
	Rank        string  `json:"rank"`
}

// PnLData 盈亏数据
type PnLData struct {
	TotalPnl      float64 `json:"totalPnl"`
	RealizedPnl   float64 `json:"realizedPnl"`
	UnrealizedPnl float64 `json:"unrealizedPnl"`
	Timeframe     string  `json:"timeframe"`
}

// ========== WebSocket 类型 ==========

// OrderBookLevel 订单簿层级
type OrderBookLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// OrderBookSnapshot 订单簿快照
type OrderBookSnapshot struct {
	AssetID        string           `json:"asset_id"`
	Market         string           `json:"market"`
	Timestamp      string           `json:"timestamp"`
	Hash           string           `json:"hash"`
	Bids           []OrderBookLevel `json:"bids"`
	Asks           []OrderBookLevel `json:"asks"`
	LastTradePrice string           `json:"last_trade_price"`
}

// PriceChangeEvent 价格变化事件
type PriceChangeEvent struct {
	AssetID string `json:"asset_id"`
	Market  string `json:"market"`
	Price   string `json:"price"`
	Size    string `json:"size"`
	Side    string `json:"side"`
	Hash    string `json:"hash"`
	BestBid string `json:"best_bid"`
	BestAsk string `json:"best_ask"`
}

// LastTradePrice 最新成交价
type LastTradePrice struct {
	AssetID    string `json:"asset_id"`
	Market     string `json:"market"`
	Price      string `json:"price"`
	Side       string `json:"side"`
	Size       string `json:"size"`
	FeeRateBps string `json:"fee_rate_bps"`
	Timestamp  string `json:"timestamp"`
}

// TickSizeChange tick size 变化事件
type TickSizeChange struct {
	AssetID     string `json:"asset_id"`
	Market      string `json:"market"`
	OldTickSize string `json:"old_tick_size"`
	NewTickSize string `json:"new_tick_size"`
	Side        string `json:"side"`
	Timestamp   string `json:"timestamp"`
}

// OrderUpdate 订单更新
type OrderUpdate struct {
	ID              string   `json:"id"`
	Market          string   `json:"market"`
	AssetID         string   `json:"asset_id"`
	EventType       string   `json:"event_type"`
	Type            string   `json:"type"`
	Side            string   `json:"side"`
	Price           string   `json:"price"`
	Size            string   `json:"size"`
	SizeMatched     string   `json:"size_matched"`
	OriginalSize    string   `json:"original_size"`
	Outcome         string   `json:"outcome"`
	Owner           string   `json:"owner"`
	OrderOwner      string   `json:"order_owner"`
	AssociateTrades []string `json:"associate_trades"`
	Timestamp       string   `json:"timestamp"`
}

// TradeNotification 成交通知
type TradeNotification struct {
	ID           string       `json:"id"`
	EventType    string       `json:"event_type"`
	Market       string       `json:"market"`
	AssetID      string       `json:"asset_id"`
	TakerOrderID string       `json:"taker_order_id"`
	Side         string       `json:"side"`
	Price        string       `json:"price"`
	Size         string       `json:"size"`
	FeeRateBps   string       `json:"fee_rate_bps"`
	Status       string       `json:"status"`
	Outcome      string       `json:"outcome"`
	Owner        string       `json:"owner"`
	TradeOwner   string       `json:"trade_owner"`
	MatchTime    string       `json:"matchtime"`
	LastUpdate   string       `json:"last_update"`
	Timestamp    string       `json:"timestamp"`
	TradeID      string       `json:"trade_id"`
	TraderSide   string       `json:"trader_side"`
	Type         string       `json:"type"`
	MakerOrders  []MakerOrder `json:"maker_orders"`
}

// MakerOrder Maker订单详情
type MakerOrder struct {
	AssetID       string `json:"asset_id"`
	MatchedAmount string `json:"matched_amount"`
	OrderID       string `json:"order_id"`
	Outcome       string `json:"outcome"`
	Owner         string `json:"owner"`
	Price         string `json:"price"`
	Side          string `json:"side"`
}

// WssAuth WebSocket 认证信息
type WssAuth struct {
	APIKey     string `json:"apiKey"`
	Secret     string `json:"secret"`
	Passphrase string `json:"passphrase"`
}

// ========== 链上操作类型 ==========

// SplitParams Split 操作参数
type SplitParams struct {
	CollateralToken string
	ConditionID     string
	Amount          string
	NegRisk         bool
}

// MergeParams Merge 操作参数
type MergeParams struct {
	CollateralToken string
	ConditionID     string
	Amount          string
	NegRisk         bool
}

// ConvertParams Convert 操作参数
type ConvertParams struct {
	MarketID    string
	QuestionIDs []string
	Amount      string
}

// RedeemParams Redeem 操作参数
type RedeemParams struct {
	CollateralToken string
	ConditionID     string
	NegRisk         bool
	Amounts         []string
}

// ApproveParams 授权参数
type ApproveParams struct {
	TokenAddress string
	Spender      string
	Amount       string
}

// TransferParams 转账参数
type TransferParams struct {
	To      string
	Amount  string
	TokenID string
}

// TransactionResult 交易结果
type TransactionResult struct {
	Hash          string `json:"hash"`
	TransactionID string `json:"transactionId"`
	State         string `json:"state,omitempty"`
	ProxyAddress  string `json:"proxyAddress,omitempty"`
}

// AccountStatus 账户状态
type AccountStatus struct {
	Address              string  `json:"address"`
	USDCBalance          float64 `json:"usdcBalance"`
	USDCAllowanceCTF     string  `json:"usdcAllowanceCTF"`
	USDCAllowanceNegRisk string  `json:"usdcAllowanceNegRisk"`
	CTFApprovedNegRisk   bool    `json:"ctfApprovedNegRisk"`
	CTFApprovedExchange  bool    `json:"ctfApprovedExchange"`
}
