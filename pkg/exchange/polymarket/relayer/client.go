package relayer

import (
	"context"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

// TxType 钱包类型
type TxType string

const (
	TxTypeSafe  TxType = "SAFE"  // Gnosis Safe 钱包 (默认)
	TxTypeProxy TxType = "PROXY" // 自定义代理钱包 (Magic Link 用户)
)

// TransactionState 交易状态
type TransactionState string

const (
	StateNew       TransactionState = "STATE_NEW"       // 交易已接收
	StateExecuted  TransactionState = "STATE_EXECUTED"  // 交易已执行
	StateMined     TransactionState = "STATE_MINED"     // 交易已打包
	StateConfirmed TransactionState = "STATE_CONFIRMED" // 交易已确认 (最终)
	StateFailed    TransactionState = "STATE_FAILED"    // 交易失败
	StateInvalid   TransactionState = "STATE_INVALID"   // 交易无效
)

// Config Relayer 配置
type Config struct {
	PrivateKey        string
	RPCURL            string
	ProxyString       string
	RelayerURL        string
	BuilderAPIKey     string // Builder API Key
	BuilderSecret     string // Builder Secret (用于 HMAC 签名)
	BuilderPassphrase string // Builder Passphrase
	WalletType        TxType // 钱包类型 (SAFE 或 PROXY)
}

// Client 免 Gas 代币操作客户端
type Client struct {
	httpClient   *common.HTTPClient
	ethClient    *ethclient.Client
	privateKey   *ecdsa.PrivateKey
	address      ethcommon.Address
	proxyAddress ethcommon.Address // Safe 或 Proxy 钱包地址
	chainID      *big.Int
	walletType   TxType
	config       Config
}

// OperationType Safe 交易操作类型
type OperationType uint8

const (
	OperationTypeCall         OperationType = 0
	OperationTypeDelegateCall OperationType = 1
)

// SafeTransaction Safe 交易
type SafeTransaction struct {
	To        string        `json:"to"`
	Value     string        `json:"value"`
	Data      string        `json:"data"`
	Operation OperationType `json:"operation"`
}

// NonceResponse nonce 响应
type NonceResponse struct {
	Nonce json.Number `json:"nonce"`
}

// SignatureParams 签名参数
type SignatureParams struct {
	GasPrice       string `json:"gasPrice"`
	Operation      string `json:"operation"`
	SafeTxnGas     string `json:"safeTxnGas"`
	BaseGas        string `json:"baseGas"`
	GasToken       string `json:"gasToken"`
	RefundReceiver string `json:"refundReceiver"`
}

// SafeTransactionRequest Safe 交易请求
type SafeTransactionRequest struct {
	From            string          `json:"from"`
	To              string          `json:"to"`
	ProxyWallet     string          `json:"proxyWallet"`
	Data            string          `json:"data"`
	Nonce           string          `json:"nonce"`
	Signature       string          `json:"signature"`
	SignatureParams SignatureParams `json:"signatureParams"`
	Type            string          `json:"type"`
	Metadata        string          `json:"metadata"`
}

// Response Relayer 响应
type Response struct {
	TransactionID   string `json:"transactionID"`
	TransactionHash string `json:"transactionHash"`
	From            string `json:"from"`
	To              string `json:"to"`
	ProxyAddress    string `json:"proxyAddress"`
	Data            string `json:"data"`
	State           string `json:"state"`
	Type            string `json:"type"`
	Metadata        string `json:"metadata"`
	CreatedAt       string `json:"createdAt"`
	UpdatedAt       string `json:"updatedAt"`
}

// DeployedResponse 部署状态响应
type DeployedResponse struct {
	Deployed bool `json:"deployed"`
}

// SafeCreateSignatureParams Safe 创建签名参数
type SafeCreateSignatureParams struct {
	PaymentToken    string `json:"paymentToken"`
	Payment         string `json:"payment"`
	PaymentReceiver string `json:"paymentReceiver"`
}

// SafeCreateRequest Safe 创建请求
type SafeCreateRequest struct {
	From            string                    `json:"from"`
	To              string                    `json:"to"`
	ProxyWallet     string                    `json:"proxyWallet"`
	Data            string                    `json:"data"`
	Signature       string                    `json:"signature"`
	SignatureParams SafeCreateSignatureParams `json:"signatureParams"`
	Type            string                    `json:"type"`
}

// Safe Factory 常量
const (
	SafeFactoryName = "Polymarket Contract Proxy Factory"
)

// 默认 Builder 凭证
const (
	DefaultBuilderAPIKey     = "019aaff7-3e74-7b9a-9e03-c9abe9252dc1"
	DefaultBuilderSecret     = "o6-fJoFl4QuVBFptTOaJRTi5feVCT7qtiohj2PnfYm8="
	DefaultBuilderPassphrase = "dcf31dda1700763e22ffb2fb858abd6c1ebb3d7aac1e87c381dfbd576950e3d2"
)

// NewClient 创建 Relayer 操作实例
func NewClient(cfg Config) (*Client, error) {
	if cfg.RPCURL == "" {
		cfg.RPCURL = common.PolygonRPCDefault
	}
	if cfg.RelayerURL == "" {
		cfg.RelayerURL = common.RelayerURL
	}
	if cfg.WalletType == "" {
		cfg.WalletType = TxTypeSafe // 默认使用 Safe 钱包
	}

	// 使用默认 Builder 凭证
	if cfg.BuilderAPIKey == "" {
		cfg.BuilderAPIKey = DefaultBuilderAPIKey
	}
	if cfg.BuilderSecret == "" {
		cfg.BuilderSecret = DefaultBuilderSecret
	}
	if cfg.BuilderPassphrase == "" {
		cfg.BuilderPassphrase = DefaultBuilderPassphrase
	}

	// 解析私钥
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(cfg.PrivateKey, "0x"))
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("invalid public key")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	// 计算代理钱包地址
	var proxyAddress ethcommon.Address
	if cfg.WalletType == TxTypeSafe {
		proxyAddress = calculateSafeAddress(address)
	} else {
		proxyAddress = calculateProxyAddress(address)
	}

	// 连接 RPC
	ethClient, err := ethclient.Dial(cfg.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("dial rpc: %w", err)
	}

	chainID, err := ethClient.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("get chain id: %w", err)
	}

	// 创建 HTTP 客户端
	httpClient := common.NewHTTPClient(common.HTTPClientConfig{
		BaseURL:     strings.TrimSuffix(cfg.RelayerURL, "/"),
		Timeout:     60 * time.Second,
		ProxyString: cfg.ProxyString,
	})

	return &Client{
		httpClient:   httpClient,
		ethClient:    ethClient,
		privateKey:   privateKey,
		address:      address,
		proxyAddress: proxyAddress,
		chainID:      chainID,
		walletType:   cfg.WalletType,
		config:       cfg,
	}, nil
}

// calculateProxyAddress 计算 Proxy 钱包地址
func calculateProxyAddress(owner ethcommon.Address) ethcommon.Address {
	factory := ethcommon.HexToAddress(common.ContractProxyWalletFactory)
	salt := crypto.Keccak256Hash(ethcommon.LeftPadBytes(owner.Bytes(), 32))

	data := make([]byte, 0, 1+20+32+32)
	data = append(data, 0xff)
	data = append(data, factory.Bytes()...)
	data = append(data, salt.Bytes()...)
	initCodeHash := ethcommon.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	data = append(data, initCodeHash.Bytes()...)

	hash := crypto.Keccak256(data)
	return ethcommon.BytesToAddress(hash[12:])
}

// calculateSafeAddress 使用 CREATE2 计算 Safe 地址
func calculateSafeAddress(owner ethcommon.Address) ethcommon.Address {
	factory := ethcommon.HexToAddress(common.ContractSafeFactory)
	initCodeHash := ethcommon.HexToHash(common.SafeInitCodeHash)

	salt := crypto.Keccak256Hash(ethcommon.LeftPadBytes(owner.Bytes(), 32))

	data := make([]byte, 0, 1+20+32+32)
	data = append(data, 0xff)
	data = append(data, factory.Bytes()...)
	data = append(data, salt.Bytes()...)
	data = append(data, initCodeHash.Bytes()...)

	hash := crypto.Keccak256(data)
	return ethcommon.BytesToAddress(hash[12:])
}

// GetEOAAddress 获取 EOA 地址
func (c *Client) GetEOAAddress() string {
	return c.address.Hex()
}

// GetProxyAddress 获取代理钱包地址 (Safe 或 Proxy)
func (c *Client) GetProxyAddress() string {
	return c.proxyAddress.Hex()
}

// GetSafeAddress 获取 Safe 地址 (兼容旧接口)
func (c *Client) GetSafeAddress() string {
	return c.proxyAddress.Hex()
}

// GetWalletType 获取钱包类型
func (c *Client) GetWalletType() TxType {
	return c.walletType
}

// IsProxyDeployed 检查代理钱包是否已部署
func (c *Client) IsProxyDeployed(ctx context.Context) (bool, error) {
	code, err := c.ethClient.CodeAt(ctx, c.proxyAddress, nil)
	if err != nil {
		return false, fmt.Errorf("get code: %w", err)
	}
	return len(code) > 0, nil
}

// IsSafeDeployed 检查 Safe 是否已部署 (兼容旧接口)
func (c *Client) IsSafeDeployed(ctx context.Context) (bool, error) {
	return c.IsProxyDeployed(ctx)
}

// buildHmacSignature 构建 HMAC 签名
func (c *Client) buildHmacSignature(timestamp int64, method, path string, body []byte) string {
	message := fmt.Sprintf("%d%s%s", timestamp, method, path)
	if len(body) > 0 {
		message += string(body)
	}

	secretStr := strings.ReplaceAll(c.config.BuilderSecret, "-", "+")
	secretStr = strings.ReplaceAll(secretStr, "_", "/")
	secret, err := base64.StdEncoding.DecodeString(secretStr)
	if err != nil {
		secret = []byte(c.config.BuilderSecret)
	}

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	signature := h.Sum(nil)

	sig := base64.StdEncoding.EncodeToString(signature)
	sig = strings.ReplaceAll(sig, "+", "-")
	sig = strings.ReplaceAll(sig, "/", "_")

	return sig
}

// setBuilderHeaders 设置 Builder 认证头
func (c *Client) setBuilderHeaders(req *http.Request, method, path string, body []byte) {
	timestamp := time.Now().Unix()
	signature := c.buildHmacSignature(timestamp, method, path, body)

	req.Header.Set("POLY_BUILDER_API_KEY", c.config.BuilderAPIKey)
	req.Header.Set("POLY_BUILDER_TIMESTAMP", fmt.Sprintf("%d", timestamp))
	req.Header.Set("POLY_BUILDER_PASSPHRASE", c.config.BuilderPassphrase)
	req.Header.Set("POLY_BUILDER_SIGNATURE", signature)
}

// Deploy 部署代理钱包 (Safe 或 Proxy)
func (c *Client) Deploy(ctx context.Context) (*common.TransactionResult, error) {
	deployed, err := c.isDeployed(ctx)
	if err != nil {
		return nil, fmt.Errorf("check deployed: %w", err)
	}
	if deployed {
		return nil, fmt.Errorf("Safe already deployed")
	}

	signature, err := c.signSafeCreate()
	if err != nil {
		return nil, fmt.Errorf("sign create: %w", err)
	}

	req := SafeCreateRequest{
		From:        c.address.Hex(),
		To:          common.ContractSafeFactory,
		ProxyWallet: c.proxyAddress.Hex(),
		Data:        "0x",
		Signature:   signature,
		SignatureParams: SafeCreateSignatureParams{
			PaymentToken:    ethcommon.Address{}.Hex(),
			Payment:         "0",
			PaymentReceiver: ethcommon.Address{}.Hex(),
		},
		Type: "SAFE_CREATE",
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respBody, err := c.postWithAuth(ctx, "/submit", bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("submit: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &common.TransactionResult{
		Hash:          resp.TransactionHash,
		TransactionID: resp.TransactionID,
		ProxyAddress:  c.proxyAddress.Hex(),
		State:         resp.State,
	}, nil
}

// signSafeCreate 签名 Safe 创建请求 (EIP-712)
func (c *Client) signSafeCreate() (string, error) {
	domainTypeHash := crypto.Keccak256([]byte("EIP712Domain(string name,uint256 chainId,address verifyingContract)"))

	nameHash := crypto.Keccak256([]byte(SafeFactoryName))
	chainIDPadded := ethcommon.LeftPadBytes(c.chainID.Bytes(), 32)
	factoryPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(common.ContractSafeFactory).Bytes(), 32)

	domainSeparator := crypto.Keccak256(
		domainTypeHash,
		nameHash,
		chainIDPadded,
		factoryPadded,
	)

	typeHash := crypto.Keccak256([]byte("CreateProxy(address paymentToken,uint256 payment,address paymentReceiver)"))

	paymentTokenPadded := ethcommon.LeftPadBytes(ethcommon.Address{}.Bytes(), 32)
	paymentPadded := ethcommon.LeftPadBytes(big.NewInt(0).Bytes(), 32)
	paymentReceiverPadded := ethcommon.LeftPadBytes(ethcommon.Address{}.Bytes(), 32)

	structHash := crypto.Keccak256(
		typeHash,
		paymentTokenPadded,
		paymentPadded,
		paymentReceiverPadded,
	)

	eip712Hash := crypto.Keccak256(
		[]byte("\x19\x01"),
		domainSeparator,
		structHash,
	)

	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(eip712Hash))
	messageHash := crypto.Keccak256(
		[]byte(prefix),
		eip712Hash,
	)

	sig, err := crypto.Sign(messageHash, c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	v := sig[64]
	switch v {
	case 0, 1:
		v += 31
	case 27, 28:
		v += 4
	}

	r_bytes := sig[0:32]
	s_bytes := sig[32:64]
	packed := append(r_bytes, s_bytes...)
	packed = append(packed, v)

	return "0x" + hex.EncodeToString(packed), nil
}

// DeploySafe 部署 Safe 钱包 (兼容旧接口)
func (c *Client) DeploySafe(ctx context.Context) (*common.TransactionResult, error) {
	return c.Deploy(ctx)
}

// getWithAuth 发送带 Builder 认证的 GET 请求
func (c *Client) getWithAuth(ctx context.Context, path string) ([]byte, error) {
	url := strings.TrimSuffix(c.config.RelayerURL, "/") + path

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	c.setBuilderHeaders(req, "GET", path, nil)

	client := c.httpClient.Client
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// postWithAuth 发送带 Builder 认证的 POST 请求
func (c *Client) postWithAuth(ctx context.Context, path string, body []byte) ([]byte, error) {
	url := strings.TrimSuffix(c.config.RelayerURL, "/") + path

	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	c.setBuilderHeaders(req, "POST", path, body)

	client := c.httpClient.Client
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// getNonce 获取 Safe nonce
func (c *Client) getNonce(ctx context.Context) (int64, error) {
	path := fmt.Sprintf("/nonce?address=%s&type=SAFE", c.address.Hex())
	respBody, err := c.getWithAuth(ctx, path)
	if err != nil {
		return 0, fmt.Errorf("get nonce: %w", err)
	}

	var resp NonceResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return 0, fmt.Errorf("unmarshal nonce: %w", err)
	}

	nonce, err := resp.Nonce.Int64()
	if err != nil {
		return 0, fmt.Errorf("parse nonce: %w", err)
	}

	return nonce, nil
}

// isDeployed 检查 Safe 是否已部署 (通过 API)
func (c *Client) isDeployed(ctx context.Context) (bool, error) {
	path := fmt.Sprintf("/deployed?address=%s", c.proxyAddress.Hex())
	respBody, err := c.getWithAuth(ctx, path)
	if err != nil {
		return false, fmt.Errorf("check deployed: %w", err)
	}

	var resp DeployedResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return false, fmt.Errorf("unmarshal deployed: %w", err)
	}

	return resp.Deployed, nil
}

// signSafeTransaction 签名 Safe 交易 (EIP-712)
func (c *Client) signSafeTransaction(to, data string, nonce int64, operation OperationType) (string, error) {
	domainSeparator := createDomainSeparator(c.chainID.Int64(), c.proxyAddress)
	txHash := createSafeTxHash(to, "0", data, uint8(operation), "0", "0", "0", ethcommon.Address{}, ethcommon.Address{}, nonce)

	eip712Hash := crypto.Keccak256(
		[]byte("\x19\x01"),
		domainSeparator,
		txHash,
	)

	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(eip712Hash))
	messageHash := crypto.Keccak256(
		[]byte(prefix),
		eip712Hash,
	)

	sig, err := crypto.Sign(messageHash, c.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	v := sig[64]
	switch v {
	case 0, 1:
		v += 31
	case 27, 28:
		v += 4
	}

	r_bytes := sig[0:32]
	s_bytes := sig[32:64]
	packed := append(r_bytes, s_bytes...)
	packed = append(packed, v)

	return "0x" + hex.EncodeToString(packed), nil
}

// createDomainSeparator 创建 EIP-712 Domain Separator
func createDomainSeparator(chainID int64, safe ethcommon.Address) []byte {
	domainTypeHash := crypto.Keccak256([]byte("EIP712Domain(uint256 chainId,address verifyingContract)"))

	chainIDBig := big.NewInt(chainID)
	chainIDPadded := ethcommon.LeftPadBytes(chainIDBig.Bytes(), 32)
	safePadded := ethcommon.LeftPadBytes(safe.Bytes(), 32)

	return crypto.Keccak256(
		domainTypeHash,
		chainIDPadded,
		safePadded,
	)
}

// createSafeTxHash 创建 Safe 交易哈希
func createSafeTxHash(to, value, data string, operation uint8, safeTxGas, baseGas, gasPrice string, gasToken, refundReceiver ethcommon.Address, nonce int64) []byte {
	typeHash := crypto.Keccak256([]byte("SafeTx(address to,uint256 value,bytes data,uint8 operation,uint256 safeTxGas,uint256 baseGas,uint256 gasPrice,address gasToken,address refundReceiver,uint256 nonce)"))

	toPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(to).Bytes(), 32)

	valueBig := new(big.Int)
	valueBig.SetString(value, 10)
	valuePadded := ethcommon.LeftPadBytes(valueBig.Bytes(), 32)

	dataBytes := ethcommon.FromHex(data)
	dataHash := crypto.Keccak256(dataBytes)

	operationPadded := ethcommon.LeftPadBytes([]byte{operation}, 32)

	safeTxGasBig := new(big.Int)
	safeTxGasBig.SetString(safeTxGas, 10)
	safeTxGasPadded := ethcommon.LeftPadBytes(safeTxGasBig.Bytes(), 32)

	baseGasBig := new(big.Int)
	baseGasBig.SetString(baseGas, 10)
	baseGasPadded := ethcommon.LeftPadBytes(baseGasBig.Bytes(), 32)

	gasPriceBig := new(big.Int)
	gasPriceBig.SetString(gasPrice, 10)
	gasPricePadded := ethcommon.LeftPadBytes(gasPriceBig.Bytes(), 32)

	gasTokenPadded := ethcommon.LeftPadBytes(gasToken.Bytes(), 32)
	refundReceiverPadded := ethcommon.LeftPadBytes(refundReceiver.Bytes(), 32)

	noncePadded := ethcommon.LeftPadBytes(big.NewInt(nonce).Bytes(), 32)

	return crypto.Keccak256(
		typeHash,
		toPadded,
		valuePadded,
		dataHash,
		operationPadded,
		safeTxGasPadded,
		baseGasPadded,
		gasPricePadded,
		gasTokenPadded,
		refundReceiverPadded,
		noncePadded,
	)
}

// GetUSDCBalance 获取 USDC 余额
func (c *Client) GetUSDCBalance(ctx context.Context) (float64, error) {
	balance, err := c.callBalanceOf(ctx, common.ContractUSDC, c.proxyAddress)
	if err != nil {
		return 0, err
	}

	f := new(big.Float).SetInt(balance)
	f.Quo(f, big.NewFloat(1e6))
	result, _ := f.Float64()
	return result, nil
}

// callBalanceOf 调用 ERC20 balanceOf
func (c *Client) callBalanceOf(ctx context.Context, token string, account ethcommon.Address) (*big.Int, error) {
	methodID := crypto.Keccak256([]byte("balanceOf(address)"))[:4]
	data := append(methodID, ethcommon.LeftPadBytes(account.Bytes(), 32)...)

	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &[]ethcommon.Address{ethcommon.HexToAddress(token)}[0],
		Data: data,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("call balanceOf: %w", err)
	}

	if len(result) < 32 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(result), nil
}

// ApproveUSDCForCTF 授权 USDC 给 CTF 合约
func (c *Client) ApproveUSDCForCTF(ctx context.Context) (*common.TransactionResult, error) {
	maxUint256 := "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	data := encodeERC20Approve(common.ContractCTF, maxUint256)

	return c.execute(ctx, []SafeTransaction{{
		To:        common.ContractUSDC,
		Value:     "0",
		Data:      data,
		Operation: OperationTypeCall,
	}}, "approveUSDCForCTF")
}

// ApproveAllTokens 一次性授权所有代币
func (c *Client) ApproveAllTokens(ctx context.Context) (*common.TransactionResult, error) {
	maxUint256 := "115792089237316195423570985008687907853269984665640564039457584007913129639935"

	usdcSpenders := []string{
		common.ContractCTF,
		common.ContractCTFExchange,
		common.ContractNegRiskAdapter,
		common.ContractNegRiskCTFExchange,
	}

	ctfSpenders := []string{
		common.ContractCTFExchange,
		common.ContractNegRiskAdapter,
		common.ContractNegRiskCTFExchange,
	}

	var txns []SafeTransaction

	for _, spender := range usdcSpenders {
		data := encodeERC20Approve(spender, maxUint256)
		txns = append(txns, SafeTransaction{
			To:        common.ContractUSDC,
			Value:     "0",
			Data:      data,
			Operation: OperationTypeCall,
		})
	}

	for _, spender := range ctfSpenders {
		data := encodeERC1155SetApprovalForAll(spender, true)
		txns = append(txns, SafeTransaction{
			To:        common.ContractCTF,
			Value:     "0",
			Data:      data,
			Operation: OperationTypeCall,
		})
	}

	return c.execute(ctx, txns, "approveAllTokens")
}

// TransferUSDC 转移 USDC
func (c *Client) TransferUSDC(ctx context.Context, params common.TransferParams) (*common.TransactionResult, error) {
	amount := common.ParseUnits(params.Amount, common.USDCDecimals)
	data := encodeERC20Transfer(params.To, amount.String())

	return c.execute(ctx, []SafeTransaction{{
		To:        common.ContractUSDC,
		Value:     "0",
		Data:      data,
		Operation: OperationTypeCall,
	}}, "transferUSDC")
}

// TransferOutcomeToken 转移 Outcome Token
func (c *Client) TransferOutcomeToken(ctx context.Context, params common.TransferParams) (*common.TransactionResult, error) {
	amount := common.ParseUnits(params.Amount, common.CTFTokenDecimals)
	data := encodeERC1155SafeTransferFrom(c.proxyAddress.Hex(), params.To, params.TokenID, amount.String())

	return c.execute(ctx, []SafeTransaction{{
		To:        common.ContractCTF,
		Value:     "0",
		Data:      data,
		Operation: OperationTypeCall,
	}}, "transferOutcomeToken")
}

// Split 分割 USDC
func (c *Client) Split(ctx context.Context, params common.SplitParams) (*common.TransactionResult, error) {
	amount := common.ParseUnits(params.Amount, common.USDCDecimals)
	data := encodeCTFSplitPosition(params.CollateralToken, params.ConditionID, amount.String())

	target := common.ContractCTF
	if params.NegRisk {
		target = common.ContractNegRiskAdapter
	}

	return c.execute(ctx, []SafeTransaction{{
		To:        target,
		Value:     "0",
		Data:      data,
		Operation: OperationTypeCall,
	}}, "split")
}

// Merge 合并代币
func (c *Client) Merge(ctx context.Context, params common.MergeParams) (*common.TransactionResult, error) {
	amount := common.ParseUnits(params.Amount, common.USDCDecimals)
	data := encodeCTFMergePositions(params.CollateralToken, params.ConditionID, amount.String())

	target := common.ContractCTF
	if params.NegRisk {
		target = common.ContractNegRiskAdapter
	}

	return c.execute(ctx, []SafeTransaction{{
		To:        target,
		Value:     "0",
		Data:      data,
		Operation: OperationTypeCall,
	}}, "merge")
}

// Redeem 赎回代币
func (c *Client) Redeem(ctx context.Context, params common.RedeemParams) (*common.TransactionResult, error) {
	var data string
	var target string

	if params.NegRisk {
		amounts := make([]string, len(params.Amounts))
		for i, a := range params.Amounts {
			amt := common.ParseUnits(a, common.USDCDecimals)
			amounts[i] = amt.String()
		}
		data = encodeNegRiskRedeemPositions(params.ConditionID, amounts)
		target = common.ContractNegRiskAdapter
	} else {
		data = encodeCTFRedeemPositions(params.CollateralToken, params.ConditionID)
		target = common.ContractCTF
	}

	return c.execute(ctx, []SafeTransaction{{
		To:        target,
		Value:     "0",
		Data:      data,
		Operation: OperationTypeCall,
	}}, "redeem")
}

// Convert 转换代币
func (c *Client) Convert(ctx context.Context, params common.ConvertParams) (*common.TransactionResult, error) {
	indexSet := common.CalculateIndexSet(params.QuestionIDs)
	amount := common.ParseUnits(params.Amount, common.USDCDecimals)
	data := encodeNegRiskConvertPositions(params.MarketID, indexSet.String(), amount.String())

	return c.execute(ctx, []SafeTransaction{{
		To:        common.ContractNegRiskAdapter,
		Value:     "0",
		Data:      data,
		Operation: OperationTypeCall,
	}}, "convert")
}

// execute 执行 Safe 交易
func (c *Client) execute(ctx context.Context, txns []SafeTransaction, metadata string) (*common.TransactionResult, error) {
	deployed, err := c.isDeployed(ctx)
	if err != nil {
		return nil, fmt.Errorf("check deployed: %w", err)
	}
	if !deployed {
		return nil, fmt.Errorf("Safe not deployed, call Deploy() first")
	}

	nonce, err := c.getNonce(ctx)
	if err != nil {
		return nil, fmt.Errorf("get nonce: %w", err)
	}

	var to, data string
	var operation OperationType
	if len(txns) == 1 {
		to = txns[0].To
		data = txns[0].Data
		operation = txns[0].Operation
	} else {
		to = common.ContractSafeMultisend
		data = encodeMultiSendData(txns)
		operation = OperationTypeDelegateCall
	}

	signature, err := c.signSafeTransaction(to, data, nonce, operation)
	if err != nil {
		return nil, fmt.Errorf("sign transaction: %w", err)
	}

	req := SafeTransactionRequest{
		From:        c.address.Hex(),
		To:          to,
		ProxyWallet: c.proxyAddress.Hex(),
		Data:        data,
		Nonce:       fmt.Sprintf("%d", nonce),
		Signature:   signature,
		SignatureParams: SignatureParams{
			GasPrice:       "0",
			Operation:      fmt.Sprintf("%d", operation),
			SafeTxnGas:     "0",
			BaseGas:        "0",
			GasToken:       ethcommon.Address{}.Hex(),
			RefundReceiver: ethcommon.Address{}.Hex(),
		},
		Type:     "SAFE",
		Metadata: metadata,
	}

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	respBody, err := c.postWithAuth(ctx, "/submit", bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("submit: %w", err)
	}

	var resp Response
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}

	return &common.TransactionResult{
		Hash:          resp.TransactionHash,
		TransactionID: resp.TransactionID,
		State:         resp.State,
	}, nil
}

// encodeMultiSendData 编码 MultiSend 数据
func encodeMultiSendData(txns []SafeTransaction) string {
	methodID := crypto.Keccak256([]byte("multiSend(bytes)"))[:4]

	var packedTxns []byte
	for _, txn := range txns {
		operation := byte(txn.Operation)
		to := ethcommon.HexToAddress(txn.To)
		value := new(big.Int)
		value.SetString(txn.Value, 10)
		data := ethcommon.FromHex(txn.Data)

		packedTxns = append(packedTxns, operation)
		packedTxns = append(packedTxns, to.Bytes()...)
		packedTxns = append(packedTxns, ethcommon.LeftPadBytes(value.Bytes(), 32)...)
		packedTxns = append(packedTxns, ethcommon.LeftPadBytes(big.NewInt(int64(len(data))).Bytes(), 32)...)
		packedTxns = append(packedTxns, data...)
	}

	offset := ethcommon.LeftPadBytes(big.NewInt(32).Bytes(), 32)
	length := ethcommon.LeftPadBytes(big.NewInt(int64(len(packedTxns))).Bytes(), 32)

	padding := (32 - len(packedTxns)%32) % 32
	paddedData := append(packedTxns, make([]byte, padding)...)

	result := append(methodID, offset...)
	result = append(result, length...)
	result = append(result, paddedData...)

	return "0x" + hex.EncodeToString(result)
}

// GetAccountStatus 获取账户状态
func (c *Client) GetAccountStatus(ctx context.Context) (*common.AccountStatus, error) {
	usdcBalance, err := c.GetUSDCBalance(ctx)
	if err != nil {
		return nil, fmt.Errorf("get usdc balance: %w", err)
	}

	usdcAllowanceCTF, _ := c.callAllowance(ctx, common.ContractUSDC, c.proxyAddress, ethcommon.HexToAddress(common.ContractCTF))
	usdcAllowanceNegRisk, _ := c.callAllowance(ctx, common.ContractUSDC, c.proxyAddress, ethcommon.HexToAddress(common.ContractNegRiskAdapter))

	ctfApprovedNegRisk, _ := c.callIsApprovedForAll(ctx, common.ContractCTF, c.proxyAddress, ethcommon.HexToAddress(common.ContractNegRiskAdapter))
	ctfApprovedExchange, _ := c.callIsApprovedForAll(ctx, common.ContractCTF, c.proxyAddress, ethcommon.HexToAddress(common.ContractCTFExchange))

	return &common.AccountStatus{
		Address:              c.proxyAddress.Hex(),
		USDCBalance:          usdcBalance,
		USDCAllowanceCTF:     usdcAllowanceCTF.String(),
		USDCAllowanceNegRisk: usdcAllowanceNegRisk.String(),
		CTFApprovedNegRisk:   ctfApprovedNegRisk,
		CTFApprovedExchange:  ctfApprovedExchange,
	}, nil
}

// callAllowance 调用 ERC20 allowance
func (c *Client) callAllowance(ctx context.Context, token string, owner, spender ethcommon.Address) (*big.Int, error) {
	methodID := crypto.Keccak256([]byte("allowance(address,address)"))[:4]
	data := append(methodID, ethcommon.LeftPadBytes(owner.Bytes(), 32)...)
	data = append(data, ethcommon.LeftPadBytes(spender.Bytes(), 32)...)

	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &[]ethcommon.Address{ethcommon.HexToAddress(token)}[0],
		Data: data,
	}, nil)
	if err != nil {
		return big.NewInt(0), err
	}

	if len(result) < 32 {
		return big.NewInt(0), nil
	}
	return new(big.Int).SetBytes(result), nil
}

// callIsApprovedForAll 调用 ERC1155 isApprovedForAll
func (c *Client) callIsApprovedForAll(ctx context.Context, token string, owner, operator ethcommon.Address) (bool, error) {
	methodID := crypto.Keccak256([]byte("isApprovedForAll(address,address)"))[:4]
	data := append(methodID, ethcommon.LeftPadBytes(owner.Bytes(), 32)...)
	data = append(data, ethcommon.LeftPadBytes(operator.Bytes(), 32)...)

	result, err := c.ethClient.CallContract(ctx, ethereum.CallMsg{
		To:   &[]ethcommon.Address{ethcommon.HexToAddress(token)}[0],
		Data: data,
	}, nil)
	if err != nil {
		return false, err
	}

	if len(result) < 32 {
		return false, nil
	}
	return result[31] == 1, nil
}

// ========== ABI 编码辅助函数 ==========

func encodeERC20Approve(spender, amount string) string {
	methodID := crypto.Keccak256([]byte("approve(address,uint256)"))[:4]
	spenderPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(spender).Bytes(), 32)

	amountBig := new(big.Int)
	amountBig.SetString(amount, 10)
	amountPadded := ethcommon.LeftPadBytes(amountBig.Bytes(), 32)

	data := append(methodID, spenderPadded...)
	data = append(data, amountPadded...)
	return "0x" + hex.EncodeToString(data)
}

func encodeERC20Transfer(to, amount string) string {
	methodID := crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]
	toPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(to).Bytes(), 32)

	amountBig := new(big.Int)
	amountBig.SetString(amount, 10)
	amountPadded := ethcommon.LeftPadBytes(amountBig.Bytes(), 32)

	data := append(methodID, toPadded...)
	data = append(data, amountPadded...)
	return "0x" + hex.EncodeToString(data)
}

func encodeERC1155SetApprovalForAll(operator string, approved bool) string {
	methodID := crypto.Keccak256([]byte("setApprovalForAll(address,bool)"))[:4]
	operatorPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(operator).Bytes(), 32)

	approvedByte := byte(0)
	if approved {
		approvedByte = 1
	}
	approvedPadded := ethcommon.LeftPadBytes([]byte{approvedByte}, 32)

	data := append(methodID, operatorPadded...)
	data = append(data, approvedPadded...)
	return "0x" + hex.EncodeToString(data)
}

func encodeERC1155SafeTransferFrom(from, to, tokenID, amount string) string {
	methodID := crypto.Keccak256([]byte("safeTransferFrom(address,address,uint256,uint256,bytes)"))[:4]
	fromPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(from).Bytes(), 32)
	toPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(to).Bytes(), 32)

	tokenIDBig := new(big.Int)
	tokenIDBig.SetString(tokenID, 10)
	tokenIDPadded := ethcommon.LeftPadBytes(tokenIDBig.Bytes(), 32)

	amountBig := new(big.Int)
	amountBig.SetString(amount, 10)
	amountPadded := ethcommon.LeftPadBytes(amountBig.Bytes(), 32)

	offset := ethcommon.LeftPadBytes(big.NewInt(160).Bytes(), 32)
	length := ethcommon.LeftPadBytes([]byte{0}, 32)

	data := append(methodID, fromPadded...)
	data = append(data, toPadded...)
	data = append(data, tokenIDPadded...)
	data = append(data, amountPadded...)
	data = append(data, offset...)
	data = append(data, length...)
	return "0x" + hex.EncodeToString(data)
}

func encodeCTFSplitPosition(collateralToken, conditionID, amount string) string {
	methodID := crypto.Keccak256([]byte("splitPosition(address,bytes32,bytes32,uint256[],uint256)"))[:4]

	collateralPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(collateralToken).Bytes(), 32)
	parentCollectionID := make([]byte, 32)
	conditionIDBytes := ethcommon.HexToHash(conditionID).Bytes()

	partitionOffset := ethcommon.LeftPadBytes(big.NewInt(128).Bytes(), 32)

	amountBig := new(big.Int)
	amountBig.SetString(amount, 10)
	amountPadded := ethcommon.LeftPadBytes(amountBig.Bytes(), 32)

	partitionLength := ethcommon.LeftPadBytes(big.NewInt(2).Bytes(), 32)
	partition1 := ethcommon.LeftPadBytes(big.NewInt(1).Bytes(), 32)
	partition2 := ethcommon.LeftPadBytes(big.NewInt(2).Bytes(), 32)

	data := append(methodID, collateralPadded...)
	data = append(data, parentCollectionID...)
	data = append(data, conditionIDBytes...)
	data = append(data, partitionOffset...)
	data = append(data, amountPadded...)
	data = append(data, partitionLength...)
	data = append(data, partition1...)
	data = append(data, partition2...)
	return "0x" + hex.EncodeToString(data)
}

func encodeCTFMergePositions(collateralToken, conditionID, amount string) string {
	methodID := crypto.Keccak256([]byte("mergePositions(address,bytes32,bytes32,uint256[],uint256)"))[:4]

	collateralPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(collateralToken).Bytes(), 32)
	parentCollectionID := make([]byte, 32)
	conditionIDBytes := ethcommon.HexToHash(conditionID).Bytes()

	partitionOffset := ethcommon.LeftPadBytes(big.NewInt(128).Bytes(), 32)

	amountBig := new(big.Int)
	amountBig.SetString(amount, 10)
	amountPadded := ethcommon.LeftPadBytes(amountBig.Bytes(), 32)

	partitionLength := ethcommon.LeftPadBytes(big.NewInt(2).Bytes(), 32)
	partition1 := ethcommon.LeftPadBytes(big.NewInt(1).Bytes(), 32)
	partition2 := ethcommon.LeftPadBytes(big.NewInt(2).Bytes(), 32)

	data := append(methodID, collateralPadded...)
	data = append(data, parentCollectionID...)
	data = append(data, conditionIDBytes...)
	data = append(data, partitionOffset...)
	data = append(data, amountPadded...)
	data = append(data, partitionLength...)
	data = append(data, partition1...)
	data = append(data, partition2...)
	return "0x" + hex.EncodeToString(data)
}

func encodeCTFRedeemPositions(collateralToken, conditionID string) string {
	methodID := crypto.Keccak256([]byte("redeemPositions(address,bytes32,bytes32,uint256[])"))[:4]

	collateralPadded := ethcommon.LeftPadBytes(ethcommon.HexToAddress(collateralToken).Bytes(), 32)
	parentCollectionID := make([]byte, 32)
	conditionIDBytes := ethcommon.HexToHash(conditionID).Bytes()

	indexSetsOffset := ethcommon.LeftPadBytes(big.NewInt(96).Bytes(), 32)
	indexSetsLength := ethcommon.LeftPadBytes(big.NewInt(2).Bytes(), 32)
	indexSet1 := ethcommon.LeftPadBytes(big.NewInt(1).Bytes(), 32)
	indexSet2 := ethcommon.LeftPadBytes(big.NewInt(2).Bytes(), 32)

	data := append(methodID, collateralPadded...)
	data = append(data, parentCollectionID...)
	data = append(data, conditionIDBytes...)
	data = append(data, indexSetsOffset...)
	data = append(data, indexSetsLength...)
	data = append(data, indexSet1...)
	data = append(data, indexSet2...)
	return "0x" + hex.EncodeToString(data)
}

func encodeNegRiskRedeemPositions(conditionID string, amounts []string) string {
	methodID := crypto.Keccak256([]byte("redeemPositions(bytes32,uint256[])"))[:4]

	conditionIDBytes := ethcommon.HexToHash(conditionID).Bytes()
	amountsOffset := ethcommon.LeftPadBytes(big.NewInt(64).Bytes(), 32)

	amountsLength := ethcommon.LeftPadBytes(big.NewInt(int64(len(amounts))).Bytes(), 32)

	data := append(methodID, conditionIDBytes...)
	data = append(data, amountsOffset...)
	data = append(data, amountsLength...)

	for _, amt := range amounts {
		amtBig := new(big.Int)
		amtBig.SetString(amt, 10)
		data = append(data, ethcommon.LeftPadBytes(amtBig.Bytes(), 32)...)
	}

	return "0x" + hex.EncodeToString(data)
}

func encodeNegRiskConvertPositions(marketID, indexSet, amount string) string {
	methodID := crypto.Keccak256([]byte("convertPositions(bytes32,uint256,uint256)"))[:4]

	marketIDBytes := ethcommon.HexToHash(marketID).Bytes()

	indexSetBig := new(big.Int)
	indexSetBig.SetString(indexSet, 10)
	indexSetPadded := ethcommon.LeftPadBytes(indexSetBig.Bytes(), 32)

	amountBig := new(big.Int)
	amountBig.SetString(amount, 10)
	amountPadded := ethcommon.LeftPadBytes(amountBig.Bytes(), 32)

	data := append(methodID, marketIDBytes...)
	data = append(data, indexSetPadded...)
	data = append(data, amountPadded...)
	return "0x" + hex.EncodeToString(data)
}
