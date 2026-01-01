package data

import (
	"context"
	"fmt"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

// ClientConfig Data 客户端配置
type ClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	ProxyString string
	Debug       bool
}

// Client Data API 客户端
type Client struct {
	client *common.HTTPClient
}

// NewClient 创建 Data 客户端
func NewClient(cfg ClientConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = common.DataAPIBaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		client: common.NewHTTPClient(common.HTTPClientConfig{
			BaseURL:     cfg.BaseURL,
			Timeout:     cfg.Timeout,
			ProxyString: cfg.ProxyString,
			Debug:       cfg.Debug,
		}),
	}
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) (string, error) {
	body, err := c.client.Get(ctx, "/", nil)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// ========== Core API ==========

// GetPositions 获取用户持仓
func (c *Client) GetPositions(ctx context.Context, params *common.PositionQueryParams) ([]common.Position, error) {
	if params == nil || params.User == "" {
		return nil, fmt.Errorf("user is required")
	}

	var positions []common.Position
	if err := c.client.GetJSON(ctx, "/positions", params, &positions); err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}
	return positions, nil
}

// GetPositionsByMarket 获取用户在特定市场的持仓
func (c *Client) GetPositionsByMarket(ctx context.Context, user, marketID string) ([]common.Position, error) {
	params := struct {
		User string `url:"user"`
	}{User: user}

	var positions []common.Position
	if err := c.client.GetJSON(ctx, "/positions/"+marketID, &params, &positions); err != nil {
		return nil, fmt.Errorf("get positions by market: %w", err)
	}
	return positions, nil
}

// GetActivity 获取用户活动
func (c *Client) GetActivity(ctx context.Context, params *common.ActivityParams) ([]common.Activity, error) {
	if params == nil || params.User == "" {
		return nil, fmt.Errorf("user is required")
	}

	var activities []common.Activity
	if err := c.client.GetJSON(ctx, "/activity", params, &activities); err != nil {
		return nil, fmt.Errorf("get activity: %w", err)
	}
	return activities, nil
}

// GetTradeHistory 获取交易历史
func (c *Client) GetTradeHistory(ctx context.Context, params *common.TradeHistoryParams) ([]common.TradeHistory, error) {
	if params == nil || params.User == "" {
		return nil, fmt.Errorf("user is required")
	}

	var trades []common.TradeHistory
	if err := c.client.GetJSON(ctx, "/trades", params, &trades); err != nil {
		return nil, fmt.Errorf("get trade history: %w", err)
	}
	return trades, nil
}

// GetClosedPositions 获取已平仓持仓
func (c *Client) GetClosedPositions(ctx context.Context, params *common.ClosedPositionParams) ([]common.ClosedPosition, error) {
	if params == nil || params.User == "" {
		return nil, fmt.Errorf("user is required")
	}

	var positions []common.ClosedPosition
	if err := c.client.GetJSON(ctx, "/closed-positions", params, &positions); err != nil {
		return nil, fmt.Errorf("get closed positions: %w", err)
	}
	return positions, nil
}

// GetPortfolioValue 获取用户持仓总价值
func (c *Client) GetPortfolioValue(ctx context.Context, user string) ([]common.PortfolioValue, error) {
	if user == "" {
		return nil, fmt.Errorf("user is required")
	}

	params := struct {
		User string `url:"user"`
	}{User: user}

	var values []common.PortfolioValue
	if err := c.client.GetJSON(ctx, "/value", &params, &values); err != nil {
		return nil, fmt.Errorf("get portfolio value: %w", err)
	}
	return values, nil
}

// GetHolders 获取市场持有者
func (c *Client) GetHolders(ctx context.Context, params *common.HoldersParams) ([]common.MarketHolders, error) {
	if params == nil || params.Market == "" {
		return nil, fmt.Errorf("market (conditionId) is required")
	}

	var holders []common.MarketHolders
	if err := c.client.GetJSON(ctx, "/holders", params, &holders); err != nil {
		return nil, fmt.Errorf("get holders: %w", err)
	}
	return holders, nil
}

// ========== Misc API ==========

// GetOpenInterest 获取全局 Open Interest
func (c *Client) GetOpenInterest(ctx context.Context) ([]common.OpenInterest, error) {
	var oi []common.OpenInterest
	if err := c.client.GetJSON(ctx, "/oi", nil, &oi); err != nil {
		return nil, fmt.Errorf("get open interest: %w", err)
	}
	return oi, nil
}

// GetLiveVolume 获取事件实时交易量
func (c *Client) GetLiveVolume(ctx context.Context, eventID int) (*common.LiveVolume, error) {
	params := struct {
		ID int `url:"id"`
	}{ID: eventID}

	var volumes []common.LiveVolume
	if err := c.client.GetJSON(ctx, "/live-volume", &params, &volumes); err != nil {
		return nil, fmt.Errorf("get live volume: %w", err)
	}
	if len(volumes) == 0 {
		return &common.LiveVolume{}, nil
	}
	return &volumes[0], nil
}

// GetMarketsTraded 获取用户交易过的市场数量
func (c *Client) GetMarketsTraded(ctx context.Context, user string) (*common.MarketsTraded, error) {
	if user == "" {
		return nil, fmt.Errorf("user is required")
	}

	params := struct {
		User string `url:"user"`
	}{User: user}

	var result common.MarketsTraded
	if err := c.client.GetJSON(ctx, "/traded", &params, &result); err != nil {
		return nil, fmt.Errorf("get markets traded: %w", err)
	}
	return &result, nil
}

// GetLeaderboard 获取交易者排行榜
func (c *Client) GetLeaderboard(ctx context.Context, params *common.LeaderboardParams) ([]common.LeaderboardEntry, error) {
	if params == nil {
		params = &common.LeaderboardParams{}
	}
	// 设置默认值
	if params.Category == "" {
		params.Category = "OVERALL"
	}
	if params.TimePeriod == "" {
		params.TimePeriod = "DAY"
	}
	if params.OrderBy == "" {
		params.OrderBy = "PNL"
	}
	if params.Limit == 0 {
		params.Limit = 25
	}

	var entries []common.LeaderboardEntry
	if err := c.client.GetJSON(ctx, "/v1/leaderboard", params, &entries); err != nil {
		return nil, fmt.Errorf("get leaderboard: %w", err)
	}
	return entries, nil
}

// ========== Builders API ==========

// GetBuilderLeaderboard 获取 Builder 排行榜
func (c *Client) GetBuilderLeaderboard(ctx context.Context, params *common.BuilderLeaderboardParams) ([]common.BuilderLeaderboardEntry, error) {
	if params == nil {
		params = &common.BuilderLeaderboardParams{}
	}
	if params.Limit == 0 {
		params.Limit = 25
	}

	var entries []common.BuilderLeaderboardEntry
	if err := c.client.GetJSON(ctx, "/v1/builders/leaderboard", params, &entries); err != nil {
		return nil, fmt.Errorf("get builder leaderboard: %w", err)
	}
	return entries, nil
}

// GetBuilderVolume 获取 Builder 交易量时序数据
func (c *Client) GetBuilderVolume(ctx context.Context, params *common.BuilderVolumeParams) ([]common.BuilderVolumeEntry, error) {
	if params == nil {
		params = &common.BuilderVolumeParams{}
	}
	if params.Limit == 0 {
		params.Limit = 25
	}

	var entries []common.BuilderVolumeEntry
	if err := c.client.GetJSON(ctx, "/v1/builders/volume", params, &entries); err != nil {
		return nil, fmt.Errorf("get builder volume: %w", err)
	}
	return entries, nil
}
