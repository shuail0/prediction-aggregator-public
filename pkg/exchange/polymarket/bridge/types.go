package bridge

// SupportedAsset 支持的资产
type SupportedAsset struct {
	ChainID       string `json:"chainId"`       // 链 ID
	ChainName     string `json:"chainName"`     // 链名称
	Token         Token  `json:"token"`         // 代币信息
	MinCheckoutUsd float64 `json:"minCheckoutUsd"` // 最小充值金额(USD)
}

// Token 代币信息
type Token struct {
	Name     string `json:"name"`     // 代币名称
	Symbol   string `json:"symbol"`   // 代币符号
	Address  string `json:"address"`  // 合约地址
	Decimals int    `json:"decimals"` // 精度
}

// SupportedAssetsResponse 支持资产响应
type SupportedAssetsResponse struct {
	SupportedAssets []SupportedAsset `json:"supportedAssets"`
}

// DepositAddresses 充值地址
type DepositAddresses struct {
	EVM string `json:"evm"` // EVM 链充值地址 (Ethereum, Arbitrum, Base 等)
	SVM string `json:"svm"` // Solana 充值地址
	BTC string `json:"btc"` // Bitcoin 充值地址
}

// DepositResponse 创建充值地址响应
type DepositResponse struct {
	Address DepositAddresses `json:"address"` // 充值地址
	Note    string           `json:"note"`    // 备注信息
}

// DepositRequest 创建充值地址请求
type DepositRequest struct {
	Address string `json:"address"` // Polymarket 钱包地址
}
