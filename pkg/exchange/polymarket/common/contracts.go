package common

// Polygon Mainnet 合约地址
const (
	// USDC 代币合约
	ContractUSDC = "0x2791Bca1f2de4661ED88A30C99A7a9449Aa84174"

	// Conditional Tokens Framework
	ContractCTF = "0x4D97DCd97eC945f40cF65F87097ACe5EA0476045"

	// CTF Exchange
	ContractCTFExchange = "0x4bFb41d5B3570DeFd03C39a9A4D8dE6Bd8B8982E"

	// Neg Risk Adapter
	ContractNegRiskAdapter = "0xd91E80cF2E7be2e162c6513ceD06f1dD0dA35296"

	// Neg Risk CTF Exchange
	ContractNegRiskCTFExchange = "0xC5d563A36AE78145C45a50134d48A1215220f80a"

	// Safe Factory
	ContractSafeFactory = "0xaacFeEa03eb1561C4e67d661e40682Bd20E3541b"

	// Safe Multisend
	ContractSafeMultisend = "0x40A2aCCbd92BCA938b02010E17A5b8929b49130D"

	// Proxy Wallet Factory
	ContractProxyWalletFactory = "0xaB45c5A4B0c941a2F231C04C3f49182e1A254052"
)

// 代币精度
const (
	USDCDecimals     = 6
	CTFTokenDecimals = 6
)

// API 端点
const (
	GammaAPIBaseURL   = "https://gamma-api.polymarket.com"
	DataAPIBaseURL    = "https://data-api.polymarket.com"
	ClobAPIBaseURL    = "https://clob.polymarket.com"
	WssBaseURL        = "wss://ws-subscriptions-clob.polymarket.com"
	RelayerURL        = "https://relayer-v2.polymarket.com/"
	PolygonRPCDefault = "https://polygon-rpc.com"
)

// Chain ID
const (
	PolygonChainID = 137
)

// Safe 部署常量
const (
	SafeInitCodeHash = "0x2bce2127ff07fb632d16c8347c4ebf501f4841168bed00d9e6ef715ddb6fcecf"
)

// CTF 操作常量
var (
	ParentCollectionID = [32]byte{} // 0x0000...
	BinaryPartition    = []int{1, 2}
)

// ABI 定义
const (
	ERC20ABI = `[
		{"constant":false,"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],"name":"transfer","outputs":[{"name":"","type":"bool"}],"type":"function"},
		{"constant":false,"inputs":[{"name":"spender","type":"address"},{"name":"amount","type":"uint256"}],"name":"approve","outputs":[{"name":"","type":"bool"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"owner","type":"address"},{"name":"spender","type":"address"}],"name":"allowance","outputs":[{"name":"","type":"uint256"}],"type":"function"}
	]`

	ERC1155ABI = `[
		{"inputs":[{"name":"from","type":"address"},{"name":"to","type":"address"},{"name":"id","type":"uint256"},{"name":"amount","type":"uint256"},{"name":"data","type":"bytes"}],"name":"safeTransferFrom","outputs":[],"type":"function"},
		{"inputs":[{"name":"operator","type":"address"},{"name":"approved","type":"bool"}],"name":"setApprovalForAll","outputs":[],"type":"function"},
		{"constant":true,"inputs":[{"name":"account","type":"address"},{"name":"operator","type":"address"}],"name":"isApprovedForAll","outputs":[{"name":"","type":"bool"}],"type":"function"},
		{"constant":true,"inputs":[{"name":"account","type":"address"},{"name":"id","type":"uint256"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"}
	]`

	CTFABI = `[
		{"inputs":[{"name":"collateralToken","type":"address"},{"name":"parentCollectionId","type":"bytes32"},{"name":"conditionId","type":"bytes32"},{"name":"partition","type":"uint256[]"},{"name":"amount","type":"uint256"}],"name":"splitPosition","outputs":[],"type":"function"},
		{"inputs":[{"name":"collateralToken","type":"address"},{"name":"parentCollectionId","type":"bytes32"},{"name":"conditionId","type":"bytes32"},{"name":"partition","type":"uint256[]"},{"name":"amount","type":"uint256"}],"name":"mergePositions","outputs":[],"type":"function"},
		{"inputs":[{"name":"collateralToken","type":"address"},{"name":"parentCollectionId","type":"bytes32"},{"name":"conditionId","type":"bytes32"},{"name":"indexSets","type":"uint256[]"}],"name":"redeemPositions","outputs":[],"type":"function"}
	]`

	NegRiskAdapterABI = `[
		{"inputs":[{"name":"fixedProductMarketMaker","type":"bytes32"},{"name":"indexSet","type":"uint256"},{"name":"amount","type":"uint256"}],"name":"convertPositions","outputs":[],"type":"function"},
		{"inputs":[{"name":"conditionId","type":"bytes32"},{"name":"amounts","type":"uint256[]"}],"name":"redeemPositions","outputs":[],"type":"function"}
	]`

	GnosisSafeABI = `[
		{"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"operation","type":"uint8"},{"name":"safeTxGas","type":"uint256"},{"name":"baseGas","type":"uint256"},{"name":"gasPrice","type":"uint256"},{"name":"gasToken","type":"address"},{"name":"refundReceiver","type":"address"},{"name":"signatures","type":"bytes"}],"name":"execTransaction","outputs":[{"name":"success","type":"bool"}],"type":"function"},
		{"inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"},{"name":"data","type":"bytes"},{"name":"operation","type":"uint8"},{"name":"safeTxGas","type":"uint256"},{"name":"baseGas","type":"uint256"},{"name":"gasPrice","type":"uint256"},{"name":"gasToken","type":"address"},{"name":"refundReceiver","type":"address"},{"name":"nonce","type":"uint256"}],"name":"getTransactionHash","outputs":[{"name":"","type":"bytes32"}],"type":"function"},
		{"constant":true,"inputs":[],"name":"nonce","outputs":[{"name":"","type":"uint256"}],"type":"function"}
	]`
)
