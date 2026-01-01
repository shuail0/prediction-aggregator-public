package clob

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

// Client CLOB API 客户端
type Client struct {
	httpClient    *common.HTTPClient
	baseURL       string
	chainID       int64
	privateKey    *ecdsa.PrivateKey
	address       string
	funder        string
	orderBuilder  *OrderBuilder
	apiCreds      *ApiKeyCreds
	signatureType SignatureType
}

// ClientConfig CLOB 客户端配置
type ClientConfig struct {
	BaseURL       string
	PrivateKey    string
	ChainID       int64
	Funder        string
	SignatureType SignatureType
	ApiCreds      *ApiKeyCreds
	ProxyString   string
	Timeout       time.Duration
}

// NewClient 创建 CLOB 客户端
func NewClient(cfg ClientConfig) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = BaseURL
	}
	if cfg.ChainID == 0 {
		cfg.ChainID = ChainIDPolygon
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()
	funder := cfg.Funder
	if funder == "" {
		funder = address
	}

	baseURL := strings.TrimSuffix(cfg.BaseURL, "/")

	httpClient := common.NewHTTPClient(common.HTTPClientConfig{
		BaseURL:     baseURL,
		Timeout:     cfg.Timeout,
		ProxyString: cfg.ProxyString,
	})

	orderBuilder := NewOrderBuilder(privateKey, cfg.ChainID, cfg.SignatureType, funder)

	// 使用默认 Builder 凭证
	apiCreds := cfg.ApiCreds
	if apiCreds == nil {
		apiCreds = &ApiKeyCreds{
			ApiKey:     DefaultBuilderAPIKey,
			Secret:     DefaultBuilderSecret,
			Passphrase: DefaultBuilderPassphrase,
		}
	}

	return &Client{
		httpClient:    httpClient,
		baseURL:       baseURL,
		chainID:       cfg.ChainID,
		privateKey:    privateKey,
		address:       address,
		funder:        funder,
		orderBuilder:  orderBuilder,
		apiCreds:      apiCreds,
		signatureType: cfg.SignatureType,
	}, nil
}

// GetAddress 获取签名者地址
func (c *Client) GetAddress() string { return c.address }

// GetFunder 获取资金来源地址
func (c *Client) GetFunder() string { return c.funder }

// SetApiCreds 设置 API 凭证
func (c *Client) SetApiCreds(creds *ApiKeyCreds) { c.apiCreds = creds }

// ========== Public 方法 ==========

// GetOk 健康检查
func (c *Client) GetOk(ctx context.Context) (bool, error) {
	var resp map[string]any
	if err := c.doGet(ctx, "/", nil, &resp); err != nil {
		return false, err
	}
	return true, nil
}

// GetServerTime 获取服务器时间
func (c *Client) GetServerTime(ctx context.Context) (int64, error) {
	var timestamp int64
	if err := c.doGet(ctx, "/time", nil, &timestamp); err != nil {
		return 0, err
	}
	return timestamp, nil
}

// GetTickSize 获取市场 tick size
func (c *Client) GetTickSize(ctx context.Context, tokenID string) (TickSize, error) {
	var resp TickSizeResponse
	if err := c.doGet(ctx, "/tick-size", url.Values{"token_id": {tokenID}}, &resp); err != nil {
		return "", err
	}
	return TickSize(resp.MinimumTickSize), nil
}

// GetNegRisk 获取市场 neg risk 状态
func (c *Client) GetNegRisk(ctx context.Context, tokenID string) (bool, error) {
	var resp NegRiskResponse
	if err := c.doGet(ctx, "/neg-risk", url.Values{"token_id": {tokenID}}, &resp); err != nil {
		return false, err
	}
	return resp.NegRisk, nil
}

// GetFeeRateBps 获取市场费率
func (c *Client) GetFeeRateBps(ctx context.Context, tokenID string) (float64, error) {
	var resp FeeRateResponse
	if err := c.doGet(ctx, "/fee-rate", url.Values{"token_id": {tokenID}}, &resp); err != nil {
		return 0, err
	}
	return resp.BaseFee, nil
}

// GetSamplingMarkets 获取采样市场列表 (分页)
func (c *Client) GetSamplingMarkets(ctx context.Context, nextCursor string) (*MarketsResponse, error) {
	params := url.Values{}
	if nextCursor != "" {
		params.Set("next_cursor", nextCursor)
	}

	var resp MarketsResponse
	if err := c.doGet(ctx, "/sampling-markets", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAllSamplingMarkets 获取所有采样市场 (自动分页)
func (c *Client) GetAllSamplingMarkets(ctx context.Context) ([]Market, error) {
	var results []Market
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		resp, err := c.GetSamplingMarkets(ctx, nextCursor)
		if err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
		if nextCursor == "" {
			break
		}
	}
	return results, nil
}

// GetSamplingSimplifiedMarkets 获取采样简化市场列表 (分页)
func (c *Client) GetSamplingSimplifiedMarkets(ctx context.Context, nextCursor string) (*SimplifiedMarketsResponse, error) {
	params := url.Values{}
	if nextCursor != "" {
		params.Set("next_cursor", nextCursor)
	}

	var resp SimplifiedMarketsResponse
	if err := c.doGet(ctx, "/sampling-simplified-markets", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMarketTradesEvents 获取市场交易事件
func (c *Client) GetMarketTradesEvents(ctx context.Context, conditionID string) ([]MarketTradeEvent, error) {
	var events []MarketTradeEvent
	if err := c.doGet(ctx, "/live-activity/events/"+conditionID, nil, &events); err != nil {
		return nil, err
	}
	return events, nil
}

// GetMarkets 获取市场列表 (分页)
func (c *Client) GetMarkets(ctx context.Context, nextCursor string) (*MarketsResponse, error) {
	params := url.Values{}
	if nextCursor != "" {
		params.Set("next_cursor", nextCursor)
	}

	var resp MarketsResponse
	if err := c.doGet(ctx, "/markets", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAllMarkets 获取所有市场 (自动分页)
func (c *Client) GetAllMarkets(ctx context.Context) ([]Market, error) {
	var results []Market
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		resp, err := c.GetMarkets(ctx, nextCursor)
		if err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
		if nextCursor == "" {
			break
		}
	}
	return results, nil
}

// GetMarket 获取单个市场
func (c *Client) GetMarket(ctx context.Context, conditionID string) (*Market, error) {
	var market Market
	if err := c.doGet(ctx, "/markets/"+conditionID, nil, &market); err != nil {
		return nil, err
	}
	return &market, nil
}

// GetSimplifiedMarkets 获取简化市场列表 (分页)
func (c *Client) GetSimplifiedMarkets(ctx context.Context, nextCursor string) (*SimplifiedMarketsResponse, error) {
	params := url.Values{}
	if nextCursor != "" {
		params.Set("next_cursor", nextCursor)
	}

	var resp SimplifiedMarketsResponse
	if err := c.doGet(ctx, "/simplified-markets", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAllSimplifiedMarkets 获取所有简化市场 (自动分页)
func (c *Client) GetAllSimplifiedMarkets(ctx context.Context) ([]SimplifiedMarket, error) {
	var results []SimplifiedMarket
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		resp, err := c.GetSimplifiedMarkets(ctx, nextCursor)
		if err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
		if nextCursor == "" {
			break
		}
	}
	return results, nil
}

// GetOrderBook 获取订单簿
func (c *Client) GetOrderBook(ctx context.Context, tokenID string) (*OrderBookSummary, error) {
	var book OrderBookSummary
	if err := c.doGet(ctx, "/book", url.Values{"token_id": {tokenID}}, &book); err != nil {
		return nil, err
	}
	return &book, nil
}

// GetOrderBooks 批量获取订单簿
func (c *Client) GetOrderBooks(ctx context.Context, tokenIDs []string) ([]OrderBookSummary, error) {
	var resp []OrderBookSummary
	body := map[string][]string{"token_ids": tokenIDs}
	if err := c.doPost(ctx, "/books", nil, body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPrice 获取价格
func (c *Client) GetPrice(ctx context.Context, tokenID string, side Side) (string, error) {
	var resp PriceResponse
	if err := c.doGet(ctx, "/price", url.Values{"token_id": {tokenID}, "side": {string(side)}}, &resp); err != nil {
		return "", err
	}
	return resp.Price, nil
}

// GetPrices 获取多个价格
func (c *Client) GetPrices(ctx context.Context, tokenIDs []string, side Side) (map[string]string, error) {
	var resp map[string]string
	body := map[string][]string{"token_ids": tokenIDs}
	if err := c.doPost(ctx, "/prices", url.Values{"side": {string(side)}}, body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetMidpoint 获取中间价
func (c *Client) GetMidpoint(ctx context.Context, tokenID string) (string, error) {
	var resp MidpointResponse
	if err := c.doGet(ctx, "/midpoint", url.Values{"token_id": {tokenID}}, &resp); err != nil {
		return "", err
	}
	return resp.Mid, nil
}

// GetMidpoints 获取多个中间价
func (c *Client) GetMidpoints(ctx context.Context, tokenIDs []string) (map[string]string, error) {
	var resp map[string]string
	body := map[string][]string{"token_ids": tokenIDs}
	if err := c.doPost(ctx, "/midpoints", nil, body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetSpread 获取价差
func (c *Client) GetSpread(ctx context.Context, tokenID string) (string, error) {
	var resp SpreadResponse
	if err := c.doGet(ctx, "/spread", url.Values{"token_id": {tokenID}}, &resp); err != nil {
		return "", err
	}
	return resp.Spread, nil
}

// GetSpreads 获取多个价差
func (c *Client) GetSpreads(ctx context.Context, tokenIDs []string) (map[string]string, error) {
	var resp map[string]string
	body := map[string][]string{"token_ids": tokenIDs}
	if err := c.doPost(ctx, "/spreads", nil, body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetLastTradePrice 获取最新成交价
func (c *Client) GetLastTradePrice(ctx context.Context, tokenID string) (*LastTradePrice, error) {
	var resp LastTradePrice
	if err := c.doGet(ctx, "/last-trade-price", url.Values{"token_id": {tokenID}}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetLastTradePrices 获取多个最新成交价
func (c *Client) GetLastTradePrices(ctx context.Context, tokenIDs []string) ([]LastTradePriceWithToken, error) {
	var resp []LastTradePriceWithToken
	body := map[string][]string{"token_ids": tokenIDs}
	if err := c.doPost(ctx, "/last-trades-prices", nil, body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetPriceHistory 获取价格历史
func (c *Client) GetPriceHistory(ctx context.Context, params PriceHistoryParams) ([]MarketPrice, error) {
	queryParams := url.Values{
		"market":   {params.Market},
		"interval": {string(params.Interval)},
	}
	if params.StartTs > 0 {
		queryParams.Set("startTs", strconv.FormatInt(params.StartTs, 10))
	}
	if params.EndTs > 0 {
		queryParams.Set("endTs", strconv.FormatInt(params.EndTs, 10))
	}
	if params.Fidelity > 0 {
		queryParams.Set("fidelity", strconv.Itoa(params.Fidelity))
	}

	var resp struct {
		History []MarketPrice `json:"history"`
	}
	if err := c.doGet(ctx, "/prices-history", queryParams, &resp); err != nil {
		return nil, err
	}
	return resp.History, nil
}

// ========== L1 方法 ==========

// CreateApiKey 创建 API Key
func (c *Client) CreateApiKey(ctx context.Context, nonce int64) (*ApiKeyCreds, error) {
	headers, err := buildL1AuthHeaders(c.privateKey, c.chainID, nonce)
	if err != nil {
		return nil, fmt.Errorf("build l1 auth headers: %w", err)
	}

	var creds ApiKeyCreds
	if err := c.doPostWithL1Auth(ctx, "/auth/api-key", headers, nil, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// DeriveApiKey 派生 API Key (使用 GET 请求)
func (c *Client) DeriveApiKey(ctx context.Context, nonce int64) (*ApiKeyCreds, error) {
	headers, err := buildL1AuthHeaders(c.privateKey, c.chainID, nonce)
	if err != nil {
		return nil, fmt.Errorf("build l1 auth headers: %w", err)
	}

	var creds ApiKeyCreds
	if err := c.doGetWithL1Auth(ctx, "/auth/derive-api-key", headers, nil, &creds); err != nil {
		return nil, err
	}
	return &creds, nil
}

// CreateOrDeriveApiKey 创建或派生 API Key
// 注意: nonce 必须为 0，这是官方 SDK 的默认行为
func (c *Client) CreateOrDeriveApiKey(ctx context.Context) (*ApiKeyCreds, error) {
	// 官方 SDK 默认 nonce = 0，API Key 的派生依赖于相同的 nonce
	var nonce int64 = 0

	// 先尝试创建，如果已存在则派生
	creds, err := c.CreateApiKey(ctx, nonce)
	if err == nil && creds.ApiKey != "" {
		return creds, nil
	}

	return c.DeriveApiKey(ctx, nonce)
}

// DeleteApiKey 删除 API Key
func (c *Client) DeleteApiKey(ctx context.Context, nonce int64) error {
	headers, err := buildL1AuthHeaders(c.privateKey, c.chainID, nonce)
	if err != nil {
		return fmt.Errorf("build l1 auth headers: %w", err)
	}
	return c.doDeleteWithL1Auth(ctx, "/auth/api-key", headers)
}

// GetApiKeys 获取所有 API Keys
func (c *Client) GetApiKeys(ctx context.Context, nonce int64) ([]string, error) {
	headers, err := buildL1AuthHeaders(c.privateKey, c.chainID, nonce)
	if err != nil {
		return nil, fmt.Errorf("build l1 auth headers: %w", err)
	}

	var keys []string
	if err := c.doGetWithL1Auth(ctx, "/auth/api-keys", headers, nil, &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

// CreateOrder 创建签名订单
func (c *Client) CreateOrder(order UserOrder, opts CreateOrderOptions) (*SignedOrder, error) {
	return c.orderBuilder.BuildOrder(order, opts)
}

// CreateMarketOrder 创建市价单
func (c *Client) CreateMarketOrder(order UserMarketOrder, opts CreateOrderOptions) (*SignedOrder, error) {
	return c.orderBuilder.BuildMarketOrder(order, opts)
}

// ========== L2 方法 ==========

// PostOrder 提交订单
func (c *Client) PostOrder(ctx context.Context, order *SignedOrder, orderType OrderType) (*OrderResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	body := postOrderRequest{
		Order:     *order,
		Owner:     c.apiCreds.ApiKey,
		OrderType: orderType,
	}

	var resp OrderResponse
	if err := c.doPostWithL2Auth(ctx, "/order", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// PostOrders 批量提交订单
func (c *Client) PostOrders(ctx context.Context, orders []PostOrdersArgs) ([]OrderResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var reqOrders []postOrderRequest
	for _, o := range orders {
		reqOrders = append(reqOrders, postOrderRequest{
			Order:     o.Order,
			Owner:     c.apiCreds.ApiKey,
			OrderType: o.OrderType,
		})
	}
	body := postOrdersRequest{Orders: reqOrders}

	var resp []OrderResponse
	if err := c.doPostWithL2Auth(ctx, "/orders", body, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// CancelOrder 取消单个订单
func (c *Client) CancelOrder(ctx context.Context, orderID string) (*CancelOrdersResponse, error) {
	return c.CancelOrders(ctx, []string{orderID})
}

// CancelOrders 取消多个订单
func (c *Client) CancelOrders(ctx context.Context, orderIDs []string) (*CancelOrdersResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	body := map[string][]string{"ids": orderIDs}

	var resp CancelOrdersResponse
	if err := c.doDeleteWithL2Auth(ctx, "/orders", body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelAll 取消所有订单
func (c *Client) CancelAll(ctx context.Context) (*CancelOrdersResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var resp CancelOrdersResponse
	if err := c.doDeleteWithL2Auth(ctx, "/orders/all", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelMarketOrders 取消指定市场的所有订单
func (c *Client) CancelMarketOrders(ctx context.Context, params OrderMarketCancelParams) (*CancelOrdersResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var resp CancelOrdersResponse
	if err := c.doDeleteWithL2Auth(ctx, "/orders/market", params, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetOpenOrdersPaginated 获取未结订单 (分页)
func (c *Client) GetOpenOrdersPaginated(ctx context.Context, params OpenOrderParams, nextCursor string) (*OpenOrdersResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	queryParams := url.Values{}
	if params.ID != "" {
		queryParams.Set("id", params.ID)
	}
	if params.Market != "" {
		queryParams.Set("market", params.Market)
	}
	if params.AssetID != "" {
		queryParams.Set("asset_id", params.AssetID)
	}
	if nextCursor != "" {
		queryParams.Set("next_cursor", nextCursor)
	}

	var resp OpenOrdersResponse
	if err := c.doGetWithL2Auth(ctx, "/data/orders", queryParams, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetOpenOrders 获取所有未结订单 (自动分页)
func (c *Client) GetOpenOrders(ctx context.Context, params OpenOrderParams) ([]OpenOrder, error) {
	var results []OpenOrder
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		resp, err := c.GetOpenOrdersPaginated(ctx, params, nextCursor)
		if err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
		if nextCursor == "" {
			break
		}
	}
	return results, nil
}

// GetOrder 获取单个订单
func (c *Client) GetOrder(ctx context.Context, orderID string) (*OpenOrder, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var order OpenOrder
	if err := c.doGetWithL2Auth(ctx, "/data/order/"+orderID, nil, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// GetTradesPaginated 获取交易记录 (分页)
func (c *Client) GetTradesPaginated(ctx context.Context, params TradeParams, nextCursor string) (*TradesResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	queryParams := url.Values{}
	if params.ID != "" {
		queryParams.Set("id", params.ID)
	}
	if params.MakerAddress != "" {
		queryParams.Set("maker_address", params.MakerAddress)
	}
	if params.Market != "" {
		queryParams.Set("market", params.Market)
	}
	if params.AssetID != "" {
		queryParams.Set("asset_id", params.AssetID)
	}
	if params.Before != "" {
		queryParams.Set("before", params.Before)
	}
	if params.After != "" {
		queryParams.Set("after", params.After)
	}
	if nextCursor != "" {
		queryParams.Set("next_cursor", nextCursor)
	}

	var resp TradesResponse
	if err := c.doGetWithL2Auth(ctx, "/data/trades", queryParams, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetTrades 获取所有交易记录 (自动分页)
func (c *Client) GetTrades(ctx context.Context, params TradeParams) ([]Trade, error) {
	var results []Trade
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		resp, err := c.GetTradesPaginated(ctx, params, nextCursor)
		if err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
		if nextCursor == "" {
			break
		}
	}
	return results, nil
}

// GetTradesFirstPage 只获取第一页交易记录
func (c *Client) GetTradesFirstPage(ctx context.Context, params TradeParams) (*TradesResponse, error) {
	return c.GetTradesPaginated(ctx, params, InitialCursor)
}

// GetBalanceAllowance 获取余额和授权
func (c *Client) GetBalanceAllowance(ctx context.Context, params BalanceAllowanceParams) (*BalanceAllowanceResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	queryParams := url.Values{"asset_type": {string(params.AssetType)}}
	if params.TokenID != "" {
		queryParams.Set("token_id", params.TokenID)
	}

	var resp BalanceAllowanceResponse
	if err := c.doGetWithL2Auth(ctx, "/balance-allowance", queryParams, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetNotifications 获取通知
func (c *Client) GetNotifications(ctx context.Context) ([]Notification, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var notifications []Notification
	if err := c.doGetWithL2Auth(ctx, "/notifications", nil, &notifications); err != nil {
		return nil, err
	}
	return notifications, nil
}

// DropNotifications 删除通知
func (c *Client) DropNotifications(ctx context.Context, ids []string) error {
	if c.apiCreds == nil {
		return fmt.Errorf("API credentials not set")
	}
	return c.doDeleteWithL2Auth(ctx, "/notifications", DropNotificationParams{IDs: ids}, nil)
}

// IsOrderScoring 检查订单是否参与评分
func (c *Client) IsOrderScoring(ctx context.Context, orderID string) (bool, error) {
	if c.apiCreds == nil {
		return false, fmt.Errorf("API credentials not set")
	}

	var resp OrderScoringResponse
	if err := c.doGetWithL2Auth(ctx, "/order-scoring", url.Values{"order_id": {orderID}}, &resp); err != nil {
		return false, err
	}
	return resp.Scoring, nil
}

// AreOrdersScoring 批量检查订单是否参与评分
func (c *Client) AreOrdersScoring(ctx context.Context, orderIDs []string) (map[string]bool, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var resp map[string]bool
	if err := c.doPostWithL2Auth(ctx, "/orders-scoring", OrdersScoringParams{OrderIDs: orderIDs}, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetClosedOnlyMode 获取封禁状态
func (c *Client) GetClosedOnlyMode(ctx context.Context) (*BanStatus, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var resp BanStatus
	if err := c.doGetWithL2Auth(ctx, "/auth/ban-status/closed-only", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateBalanceAllowance 更新余额授权
func (c *Client) UpdateBalanceAllowance(ctx context.Context, params BalanceAllowanceParams) error {
	if c.apiCreds == nil {
		return fmt.Errorf("API credentials not set")
	}

	queryParams := url.Values{
		"asset_type":     {string(params.AssetType)},
		"signature_type": {strconv.Itoa(int(c.signatureType))},
	}
	if params.TokenID != "" {
		queryParams.Set("token_id", params.TokenID)
	}

	return c.doGetWithL2Auth(ctx, "/balance-allowance/update", queryParams, nil)
}

// CreateAndPostOrder 创建并提交订单
func (c *Client) CreateAndPostOrder(ctx context.Context, userOrder UserOrder, opts CreateOrderOptions, orderType OrderType) (*OrderResponse, error) {
	order, err := c.CreateOrder(userOrder, opts)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}
	return c.PostOrder(ctx, order, orderType)
}

// CreateAndPostMarketOrder 创建并提交市价单
func (c *Client) CreateAndPostMarketOrder(ctx context.Context, userMarketOrder UserMarketOrder, opts CreateOrderOptions, orderType OrderType) (*OrderResponse, error) {
	order, err := c.CreateMarketOrder(userMarketOrder, opts)
	if err != nil {
		return nil, fmt.Errorf("create market order: %w", err)
	}
	return c.PostOrder(ctx, order, orderType)
}

// CalculateMarketPrice 计算市价单价格
func (c *Client) CalculateMarketPrice(ctx context.Context, tokenID string, side Side, amount float64, orderType OrderType) (float64, error) {
	book, err := c.GetOrderBook(ctx, tokenID)
	if err != nil {
		return 0, fmt.Errorf("get order book: %w", err)
	}

	if side == SideBuy {
		if len(book.Asks) == 0 {
			return 0, fmt.Errorf("no asks in orderbook")
		}
		return calculateBuyMarketPrice(book.Asks, amount, orderType)
	}

	if len(book.Bids) == 0 {
		return 0, fmt.Errorf("no bids in orderbook")
	}
	return calculateSellMarketPrice(book.Bids, amount, orderType)
}

// ========== Rewards 方法 ==========

// GetEarningsForUserForDay 获取用户某天的收益
func (c *Client) GetEarningsForUserForDay(ctx context.Context, date string) ([]UserEarning, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var results []UserEarning
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		queryParams := url.Values{
			"date":           {date},
			"signature_type": {strconv.Itoa(int(c.signatureType))},
			"next_cursor":    {nextCursor},
		}

		var resp struct {
			Data       []UserEarning `json:"data"`
			NextCursor string        `json:"next_cursor"`
		}
		if err := c.doGetWithL2Auth(ctx, "/rewards/user", queryParams, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
	}
	return results, nil
}

// GetTotalEarningsForUserForDay 获取用户某天的总收益
func (c *Client) GetTotalEarningsForUserForDay(ctx context.Context, date string) ([]TotalUserEarning, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	queryParams := url.Values{
		"date":           {date},
		"signature_type": {strconv.Itoa(int(c.signatureType))},
	}

	var resp []TotalUserEarning
	if err := c.doGetWithL2Auth(ctx, "/rewards/user/total", queryParams, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetRewardPercentages 获取奖励百分比
func (c *Client) GetRewardPercentages(ctx context.Context) (RewardsPercentages, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	queryParams := url.Values{"signature_type": {strconv.Itoa(int(c.signatureType))}}

	var resp RewardsPercentages
	if err := c.doGetWithL2Auth(ctx, "/rewards/user/percentages", queryParams, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetCurrentRewards 获取当前奖励
func (c *Client) GetCurrentRewards(ctx context.Context) ([]MarketReward, error) {
	var results []MarketReward
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		queryParams := url.Values{"next_cursor": {nextCursor}}

		var resp struct {
			Data       []MarketReward `json:"data"`
			NextCursor string         `json:"next_cursor"`
		}
		if err := c.doGet(ctx, "/rewards/markets/current", queryParams, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
	}
	return results, nil
}

// GetRawRewardsForMarket 获取市场原始奖励
func (c *Client) GetRawRewardsForMarket(ctx context.Context, conditionID string) ([]MarketReward, error) {
	var results []MarketReward
	nextCursor := InitialCursor

	for nextCursor != EndCursor {
		queryParams := url.Values{"next_cursor": {nextCursor}}

		var resp struct {
			Data       []MarketReward `json:"data"`
			NextCursor string         `json:"next_cursor"`
		}
		if err := c.doGet(ctx, "/rewards/markets/"+conditionID, queryParams, &resp); err != nil {
			return nil, err
		}
		results = append(results, resp.Data...)
		nextCursor = resp.NextCursor
	}
	return results, nil
}

// ========== Builder 方法 ==========

// CreateBuilderApiKey 创建 Builder API Key
func (c *Client) CreateBuilderApiKey(ctx context.Context) (*BuilderApiKey, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var resp BuilderApiKey
	if err := c.doPostWithL2Auth(ctx, "/auth/builder-api-key", nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetBuilderApiKeys 获取 Builder API Keys
func (c *Client) GetBuilderApiKeys(ctx context.Context) ([]BuilderApiKeyResponse, error) {
	if c.apiCreds == nil {
		return nil, fmt.Errorf("API credentials not set")
	}

	var resp []BuilderApiKeyResponse
	if err := c.doGetWithL2Auth(ctx, "/auth/builder-api-key", nil, &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// GetBuilderTrades 获取 Builder 交易
func (c *Client) GetBuilderTrades(ctx context.Context, params TradeParams, nextCursor string, builderCreds *ApiKeyCreds) ([]BuilderTrade, string, int, int, error) {
	if builderCreds == nil {
		return nil, "", 0, 0, fmt.Errorf("builder credentials not set")
	}

	queryParams := url.Values{}
	if params.ID != "" {
		queryParams.Set("id", params.ID)
	}
	if params.Market != "" {
		queryParams.Set("market", params.Market)
	}
	if params.AssetID != "" {
		queryParams.Set("asset_id", params.AssetID)
	}
	if nextCursor == "" {
		nextCursor = InitialCursor
	}
	queryParams.Set("next_cursor", nextCursor)

	var resp struct {
		Data       []BuilderTrade `json:"data"`
		NextCursor string         `json:"next_cursor"`
		Limit      int            `json:"limit"`
		Count      int            `json:"count"`
	}
	if err := c.doGetWithBuilderAuth(ctx, "/builder/trades", queryParams, builderCreds, &resp); err != nil {
		return nil, "", 0, 0, err
	}
	return resp.Data, resp.NextCursor, resp.Limit, resp.Count, nil
}

// ========== 辅助函数 ==========

func calculateBuyMarketPrice(asks []OrderSummary, amountToMatch float64, orderType OrderType) (float64, error) {
	if len(asks) == 0 {
		return 0, fmt.Errorf("no match")
	}

	var sum float64
	for i := len(asks) - 1; i >= 0; i-- {
		p := asks[i]
		price, _ := strconv.ParseFloat(p.Price, 64)
		size, _ := strconv.ParseFloat(p.Size, 64)
		sum += size * price
		if sum >= amountToMatch {
			return price, nil
		}
	}

	if orderType == OrderTypeFOK {
		return 0, fmt.Errorf("no match")
	}
	price, _ := strconv.ParseFloat(asks[0].Price, 64)
	return price, nil
}

func calculateSellMarketPrice(bids []OrderSummary, amountToMatch float64, orderType OrderType) (float64, error) {
	if len(bids) == 0 {
		return 0, fmt.Errorf("no match")
	}

	var sum float64
	for i := len(bids) - 1; i >= 0; i-- {
		p := bids[i]
		size, _ := strconv.ParseFloat(p.Size, 64)
		sum += size
		if sum >= amountToMatch {
			price, _ := strconv.ParseFloat(p.Price, 64)
			return price, nil
		}
	}

	if orderType == OrderTypeFOK {
		return 0, fmt.Errorf("no match")
	}
	price, _ := strconv.ParseFloat(bids[0].Price, 64)
	return price, nil
}

// ========== HTTP 请求方法 ==========

func (c *Client) doGet(ctx context.Context, path string, params url.Values, result interface{}) error {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	return c.doRequest(req, result)
}

func (c *Client) doPost(ctx context.Context, path string, params url.Values, body interface{}, result interface{}) error {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	return c.doRequest(req, result)
}

func (c *Client) doPostWithL1Auth(ctx context.Context, path string, headers *L1AuthHeaders, body interface{}, result interface{}) error {
	fullURL := c.baseURL + path

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("POLY_ADDRESS", headers.Address)
	req.Header.Set("POLY_SIGNATURE", headers.Signature)
	req.Header.Set("POLY_TIMESTAMP", headers.Timestamp)
	req.Header.Set("POLY_NONCE", strconv.FormatInt(headers.Nonce, 10))

	return c.doRequest(req, result)
}

func (c *Client) doGetWithL1Auth(ctx context.Context, path string, headers *L1AuthHeaders, params url.Values, result interface{}) error {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("POLY_ADDRESS", headers.Address)
	req.Header.Set("POLY_SIGNATURE", headers.Signature)
	req.Header.Set("POLY_TIMESTAMP", headers.Timestamp)
	req.Header.Set("POLY_NONCE", strconv.FormatInt(headers.Nonce, 10))

	return c.doRequest(req, result)
}

func (c *Client) doDeleteWithL1Auth(ctx context.Context, path string, headers *L1AuthHeaders) error {
	fullURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, "DELETE", fullURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("POLY_ADDRESS", headers.Address)
	req.Header.Set("POLY_SIGNATURE", headers.Signature)
	req.Header.Set("POLY_TIMESTAMP", headers.Timestamp)
	req.Header.Set("POLY_NONCE", strconv.FormatInt(headers.Nonce, 10))

	return c.doRequest(req, nil)
}

func (c *Client) doPostWithL2Auth(ctx context.Context, path string, body interface{}, result interface{}) error {
	fullURL := c.baseURL + path

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
	}

	headers, err := buildL2AuthHeaders(c.funder, c.apiCreds, "POST", path, bodyBytes)
	if err != nil {
		return fmt.Errorf("build l2 auth headers: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("POLY_ADDRESS", headers.Address)
	req.Header.Set("POLY_SIGNATURE", headers.Signature)
	req.Header.Set("POLY_TIMESTAMP", headers.Timestamp)
	req.Header.Set("POLY_API_KEY", headers.ApiKey)
	req.Header.Set("POLY_PASSPHRASE", headers.Passphrase)

	return c.doRequest(req, result)
}

func (c *Client) doGetWithL2Auth(ctx context.Context, path string, params url.Values, result interface{}) error {
	fullPath := path
	if len(params) > 0 {
		fullPath += "?" + params.Encode()
	}
	fullURL := c.baseURL + fullPath

	headers, err := buildL2AuthHeaders(c.funder, c.apiCreds, "GET", fullPath, nil)
	if err != nil {
		return fmt.Errorf("build l2 auth headers: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("POLY_ADDRESS", headers.Address)
	req.Header.Set("POLY_SIGNATURE", headers.Signature)
	req.Header.Set("POLY_TIMESTAMP", headers.Timestamp)
	req.Header.Set("POLY_API_KEY", headers.ApiKey)
	req.Header.Set("POLY_PASSPHRASE", headers.Passphrase)

	return c.doRequest(req, result)
}

func (c *Client) doDeleteWithL2Auth(ctx context.Context, path string, body interface{}, result interface{}) error {
	fullURL := c.baseURL + path

	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
	}

	headers, err := buildL2AuthHeaders(c.funder, c.apiCreds, "DELETE", path, bodyBytes)
	if err != nil {
		return fmt.Errorf("build l2 auth headers: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "DELETE", fullURL, strings.NewReader(string(bodyBytes)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("POLY_ADDRESS", headers.Address)
	req.Header.Set("POLY_SIGNATURE", headers.Signature)
	req.Header.Set("POLY_TIMESTAMP", headers.Timestamp)
	req.Header.Set("POLY_API_KEY", headers.ApiKey)
	req.Header.Set("POLY_PASSPHRASE", headers.Passphrase)

	return c.doRequest(req, result)
}

func (c *Client) doGetWithBuilderAuth(ctx context.Context, path string, params url.Values, builderCreds *ApiKeyCreds, result interface{}) error {
	fullPath := path
	if len(params) > 0 {
		fullPath += "?" + params.Encode()
	}
	fullURL := c.baseURL + fullPath

	headers, err := buildBuilderAuthHeaders(builderCreds, "GET", fullPath, nil)
	if err != nil {
		return fmt.Errorf("build builder auth headers: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("POLY_BUILDER_API_KEY", headers.ApiKey)
	req.Header.Set("POLY_BUILDER_TIMESTAMP", headers.Timestamp)
	req.Header.Set("POLY_BUILDER_PASSPHRASE", headers.Passphrase)
	req.Header.Set("POLY_BUILDER_SIGNATURE", headers.Signature)

	return c.doRequest(req, result)
}

func (c *Client) doRequest(req *http.Request, result interface{}) error {
	httpClient := c.httpClient.Client
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBody))
		}
	}

	return nil
}
