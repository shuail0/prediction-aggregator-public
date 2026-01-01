package gamma

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

// ClientConfig Gamma 客户端配置
type ClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	ProxyString string
	Debug       bool
}

// Client Gamma API 客户端
type Client struct {
	client *common.HTTPClient
}

// NewClient 创建 Gamma 客户端
func NewClient(cfg ClientConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = common.GammaAPIBaseURL
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
func (c *Client) HealthCheck(ctx context.Context) (interface{}, error) {
	body, err := c.client.Get(ctx, "/status", nil)
	if err != nil {
		return nil, err
	}

	var result interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return string(body), nil
	}
	return result, nil
}

// ListEvents 查询事件列表
func (c *Client) ListEvents(ctx context.Context, params *common.EventQueryParams) ([]common.Event, error) {
	var events []common.Event
	if err := c.client.GetJSON(ctx, "/events", params, &events); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return events, nil
}

// GetEventByID 根据 ID 获取事件
func (c *Client) GetEventByID(ctx context.Context, id string) (*common.Event, error) {
	var event common.Event
	if err := c.client.GetJSON(ctx, "/events/"+id, nil, &event); err != nil {
		return nil, fmt.Errorf("get event by id: %w", err)
	}
	return &event, nil
}

// GetEventBySlug 根据 Slug 获取事件
func (c *Client) GetEventBySlug(ctx context.Context, slug string) (*common.Event, error) {
	var event common.Event
	if err := c.client.GetJSON(ctx, "/events/slug/"+slug, nil, &event); err != nil {
		return nil, fmt.Errorf("get event by slug: %w", err)
	}
	return &event, nil
}

// GetEventTags 获取事件标签
func (c *Client) GetEventTags(ctx context.Context, eventID string) ([]common.Tag, error) {
	var tags []common.Tag
	if err := c.client.GetJSON(ctx, "/events/"+eventID+"/tags", nil, &tags); err != nil {
		return nil, fmt.Errorf("get event tags: %w", err)
	}
	return tags, nil
}

// ListMarkets 查询市场列表
func (c *Client) ListMarkets(ctx context.Context, params *common.MarketQueryParams) ([]common.Market, error) {
	var markets []common.Market
	if err := c.client.GetJSON(ctx, "/markets", params, &markets); err != nil {
		return nil, fmt.Errorf("list markets: %w", err)
	}
	return markets, nil
}

// GetMarketByID 根据 ID 获取市场
func (c *Client) GetMarketByID(ctx context.Context, id string) (*common.Market, error) {
	var market common.Market
	if err := c.client.GetJSON(ctx, "/markets/"+id, nil, &market); err != nil {
		return nil, fmt.Errorf("get market by id: %w", err)
	}
	return &market, nil
}

// GetMarketBySlug 根据 Slug 获取市场
func (c *Client) GetMarketBySlug(ctx context.Context, slug string) (*common.Market, error) {
	var market common.Market
	if err := c.client.GetJSON(ctx, "/markets/slug/"+slug, nil, &market); err != nil {
		return nil, fmt.Errorf("get market by slug: %w", err)
	}
	return &market, nil
}

// GetMarketTagsByID 获取市场标签
func (c *Client) GetMarketTagsByID(ctx context.Context, id string) ([]common.Tag, error) {
	var tags []common.Tag
	if err := c.client.GetJSON(ctx, "/markets/"+id+"/tags", nil, &tags); err != nil {
		return nil, fmt.Errorf("get market tags: %w", err)
	}
	return tags, nil
}

// GetMarketStats 获取市场统计
func (c *Client) GetMarketStats(ctx context.Context, marketID string) (interface{}, error) {
	var stats interface{}
	if err := c.client.GetJSON(ctx, "/markets/"+marketID+"/stats", nil, &stats); err != nil {
		return nil, fmt.Errorf("get market stats: %w", err)
	}
	return stats, nil
}

// SearchMarketsEventsAndProfiles 搜索市场、事件和用户
func (c *Client) SearchMarketsEventsAndProfiles(ctx context.Context, params *common.SearchParams) (*common.SearchResult, error) {
	if params == nil || params.Q == "" {
		return nil, fmt.Errorf("q parameter is required")
	}

	var result common.SearchResult
	if err := c.client.GetJSON(ctx, "/public-search", params, &result); err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	return &result, nil
}

// ListTeamsParams 团队列表查询参数
type ListTeamsParams struct {
	Limit        int    `url:"limit,omitempty"`
	Offset       int    `url:"offset,omitempty"`
	Order        string `url:"order,omitempty"`
	Ascending    bool   `url:"ascending,omitempty"`
	League       string `url:"league,omitempty"`
	Name         string `url:"name,omitempty"`
	Abbreviation string `url:"abbreviation,omitempty"`
}

// Team 团队信息
type Team struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Abbreviation string `json:"abbreviation"`
	League       string `json:"league"`
	LogoURL      string `json:"logoUrl"`
}

// ListTeams 列出团队
func (c *Client) ListTeams(ctx context.Context, params *ListTeamsParams) ([]Team, error) {
	var teams []Team
	if err := c.client.GetJSON(ctx, "/teams", params, &teams); err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}
	return teams, nil
}

// GetSportsMetadata 获取体育元数据
func (c *Client) GetSportsMetadata(ctx context.Context) (interface{}, error) {
	var result interface{}
	if err := c.client.GetJSON(ctx, "/sports", nil, &result); err != nil {
		return nil, fmt.Errorf("get sports metadata: %w", err)
	}
	return result, nil
}

// GetMarketByURL 根据 URL 获取市场（便捷方法）
func (c *Client) GetMarketByURL(ctx context.Context, marketURL string) (*common.Market, error) {
	slug, err := common.ParseMarketSlug(marketURL)
	if err != nil {
		return nil, err
	}
	return c.GetMarketBySlug(ctx, slug)
}

// GetEventByURL 根据 URL 获取事件（便捷方法）
func (c *Client) GetEventByURL(ctx context.Context, eventURL string) (*common.Event, error) {
	slug, err := common.ParseEventSlug(eventURL)
	if err != nil {
		return nil, err
	}
	return c.GetEventBySlug(ctx, slug)
}

// ========== Tags API ==========

// ListTags 列出标签
func (c *Client) ListTags(ctx context.Context, params *common.TagQueryParams) ([]common.Tag, error) {
	var tags []common.Tag
	if err := c.client.GetJSON(ctx, "/tags", params, &tags); err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return tags, nil
}

// GetTagByID 根据 ID 获取标签
func (c *Client) GetTagByID(ctx context.Context, id string) (*common.Tag, error) {
	var tag common.Tag
	if err := c.client.GetJSON(ctx, "/tags/"+id, nil, &tag); err != nil {
		return nil, fmt.Errorf("get tag by id: %w", err)
	}
	return &tag, nil
}

// GetTagBySlug 根据 Slug 获取标签
func (c *Client) GetTagBySlug(ctx context.Context, slug string) (*common.Tag, error) {
	var tag common.Tag
	if err := c.client.GetJSON(ctx, "/tags/slug/"+slug, nil, &tag); err != nil {
		return nil, fmt.Errorf("get tag by slug: %w", err)
	}
	return &tag, nil
}

// GetRelatedTagsByID 获取标签关联关系（按 ID）
func (c *Client) GetRelatedTagsByID(ctx context.Context, id string) ([]common.RelatedTag, error) {
	var tags []common.RelatedTag
	if err := c.client.GetJSON(ctx, "/tags/"+id+"/relationships", nil, &tags); err != nil {
		return nil, fmt.Errorf("get related tags: %w", err)
	}
	return tags, nil
}

// GetRelatedTagsBySlug 获取标签关联关系（按 Slug）
func (c *Client) GetRelatedTagsBySlug(ctx context.Context, slug string) ([]common.RelatedTag, error) {
	var tags []common.RelatedTag
	if err := c.client.GetJSON(ctx, "/tags/slug/"+slug+"/relationships", nil, &tags); err != nil {
		return nil, fmt.Errorf("get related tags: %w", err)
	}
	return tags, nil
}

// GetTagsRelatedToID 获取与指定标签相关的标签（按 ID）
func (c *Client) GetTagsRelatedToID(ctx context.Context, id string) ([]common.Tag, error) {
	var tags []common.Tag
	if err := c.client.GetJSON(ctx, "/tags/"+id+"/related", nil, &tags); err != nil {
		return nil, fmt.Errorf("get tags related to: %w", err)
	}
	return tags, nil
}

// GetTagsRelatedToSlug 获取与指定标签相关的标签（按 Slug）
func (c *Client) GetTagsRelatedToSlug(ctx context.Context, slug string) ([]common.Tag, error) {
	var tags []common.Tag
	if err := c.client.GetJSON(ctx, "/tags/slug/"+slug+"/related", nil, &tags); err != nil {
		return nil, fmt.Errorf("get tags related to: %w", err)
	}
	return tags, nil
}

// ========== Series API ==========

// ListSeries 列出系列
func (c *Client) ListSeries(ctx context.Context, params *common.SeriesQueryParams) ([]common.Series, error) {
	var series []common.Series
	if err := c.client.GetJSON(ctx, "/series", params, &series); err != nil {
		return nil, fmt.Errorf("list series: %w", err)
	}
	return series, nil
}

// GetSeriesByID 根据 ID 获取系列
func (c *Client) GetSeriesByID(ctx context.Context, id string) (*common.Series, error) {
	var series common.Series
	if err := c.client.GetJSON(ctx, "/series/"+id, nil, &series); err != nil {
		return nil, fmt.Errorf("get series by id: %w", err)
	}
	return &series, nil
}

// ========== Comments API ==========

// ListComments 列出评论
func (c *Client) ListComments(ctx context.Context, params *common.CommentQueryParams) ([]common.Comment, error) {
	var comments []common.Comment
	if err := c.client.GetJSON(ctx, "/comments", params, &comments); err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	return comments, nil
}

// GetCommentByID 根据 ID 获取评论
func (c *Client) GetCommentByID(ctx context.Context, id string) (*common.Comment, error) {
	var comment common.Comment
	if err := c.client.GetJSON(ctx, "/comments/"+id, nil, &comment); err != nil {
		return nil, fmt.Errorf("get comment by id: %w", err)
	}
	return &comment, nil
}

// GetCommentsByUser 获取用户的评论
func (c *Client) GetCommentsByUser(ctx context.Context, userAddress string, params *common.CommentQueryParams) ([]common.Comment, error) {
	if params == nil {
		params = &common.CommentQueryParams{}
	}
	var comments []common.Comment
	path := "/comments/user/" + userAddress
	if err := c.client.GetJSON(ctx, path, params, &comments); err != nil {
		return nil, fmt.Errorf("get comments by user: %w", err)
	}
	return comments, nil
}

// ========== Profiles API ==========

// GetPublicProfile 获取公开用户资料
func (c *Client) GetPublicProfile(ctx context.Context, address string) (*common.PublicProfile, error) {
	params := struct {
		Address string `url:"address"`
	}{Address: address}

	var profile common.PublicProfile
	if err := c.client.GetJSON(ctx, "/public-profile", &params, &profile); err != nil {
		return nil, fmt.Errorf("get public profile: %w", err)
	}
	return &profile, nil
}

// ========== Sports API ==========

// GetValidSportsMarketTypes 获取有效的体育市场类型
func (c *Client) GetValidSportsMarketTypes(ctx context.Context) (*common.SportsMarketTypes, error) {
	var result common.SportsMarketTypes
	if err := c.client.GetJSON(ctx, "/sports/market-types", nil, &result); err != nil {
		return nil, fmt.Errorf("get sports market types: %w", err)
	}
	return &result, nil
}
