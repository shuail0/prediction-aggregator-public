package main

import "time"

// AccountPair 账户对（用于对刷交易）
type AccountPair struct {
	Index       int
	AddressA    string
	PrivateKeyA string
	ProxyA      string
	AddressB    string
	PrivateKeyB string
	ProxyB      string
}

// Config 策略配置（从 JSON 读取）
type Config struct {
	AccountsFile   string   `json:"accountsFile"`   // 账户配置文件路径
	MarketURLs     []string `json:"marketURLs"`     // 市场URL列表
	MaxTradeAmount float64  `json:"maxTradeAmount"` // 最大交易金额(USDC)
	MinSpreadTicks int      `json:"minSpreadTicks"` // 最小盘口间隔(tick数)
	MaxRetries     int      `json:"maxRetries"`     // 最大重试次数
	RetryDelaySec  int      `json:"retryDelaySec"`  // 重试间隔(秒)
}

// GetRetryDelay 获取重试间隔
func (c *Config) GetRetryDelay() time.Duration {
	if c.RetryDelaySec <= 0 {
		return 3 * time.Second
	}
	return time.Duration(c.RetryDelaySec) * time.Second
}

// Result 执行结果
type Result struct {
	Index    int
	Success  bool
	FilledA  string
	FilledB  string
	Error    string
	Duration time.Duration
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		AccountsFile:   "data/01_polymarket_hedge_accounts_example.csv",
		MaxTradeAmount: 10,
		MinSpreadTicks: 2,
		MaxRetries:     10,
		RetryDelaySec:  3,
	}
}
