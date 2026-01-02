package bridge

import (
	"context"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

const (
	// BaseURL Bridge API 基础地址
	BaseURL = "https://bridge.polymarket.com"
)

// ClientConfig Bridge 客户端配置
type ClientConfig struct {
	BaseURL     string
	Timeout     time.Duration
	ProxyString string
}

// Client Bridge API 客户端
type Client struct {
	client *common.HTTPClient
}

// NewClient 创建 Bridge 客户端
func NewClient(cfg ClientConfig) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = BaseURL
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		client: common.NewHTTPClient(common.HTTPClientConfig{
			BaseURL:     cfg.BaseURL,
			Timeout:     cfg.Timeout,
			ProxyString: cfg.ProxyString,
		}),
	}
}

// GetSupportedAssets 获取支持的资产列表
// 返回所有支持跨链充值的链和代币信息
func (c *Client) GetSupportedAssets(ctx context.Context) ([]SupportedAsset, error) {
	var resp SupportedAssetsResponse
	if err := c.client.GetJSON(ctx, "/supported-assets", nil, &resp); err != nil {
		return nil, err
	}
	return resp.SupportedAssets, nil
}

// CreateDepositAddresses 创建充值地址
// address: Polymarket 钱包地址 (通常是 Safe 地址)
// 返回 EVM、Solana、Bitcoin 三种类型的充值地址
func (c *Client) CreateDepositAddresses(ctx context.Context, address string) (*DepositResponse, error) {
	req := DepositRequest{Address: address}
	var resp DepositResponse
	if err := c.client.PostJSON(ctx, "/deposit", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
