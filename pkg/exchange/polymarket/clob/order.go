package clob

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	polycommon "github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/common"
)

// OrderDomain EIP-712 域
var OrderDomain = struct {
	Name    string
	Version string
}{
	Name:    "Polymarket CTF Exchange",
	Version: "1",
}

// OrderTypeHashStr Order 类型哈希字符串
const OrderTypeHashStr = "Order(uint256 salt,address maker,address signer,address taker,uint256 tokenId,uint256 makerAmount,uint256 takerAmount,uint256 expiration,uint256 nonce,uint256 feeRateBps,uint8 side,uint8 signatureType)"

// OrderBuilder 订单构建器
type OrderBuilder struct {
	privateKey    *ecdsa.PrivateKey
	chainID       int64
	signer        common.Address
	funder        common.Address
	signatureType SignatureType
}

// NewOrderBuilder 创建订单构建器
func NewOrderBuilder(privateKey *ecdsa.PrivateKey, chainID int64, signatureType SignatureType, funder string) *OrderBuilder {
	signer := crypto.PubkeyToAddress(privateKey.PublicKey)
	funderAddr := signer
	if funder != "" {
		funderAddr = common.HexToAddress(funder)
	}
	return &OrderBuilder{
		privateKey:    privateKey,
		chainID:       chainID,
		signer:        signer,
		funder:        funderAddr,
		signatureType: signatureType,
	}
}

// GetAddress 获取签名者地址
func (b *OrderBuilder) GetAddress() string {
	return b.signer.Hex()
}

// GetFunder 获取资金来源地址
func (b *OrderBuilder) GetFunder() string {
	return b.funder.Hex()
}

// BuildOrder 构建并签名订单
func (b *OrderBuilder) BuildOrder(order UserOrder, opts CreateOrderOptions) (*SignedOrder, error) {
	makerAmount, takerAmount := calculateOrderAmounts(order.Side, order.Size, order.Price, opts.TickSize)

	salt := generateSalt()

	expiration := order.Expiration
	if expiration == 0 {
		expiration = time.Now().Add(365 * 24 * time.Hour).Unix()
	}

	nonce := order.Nonce

	taker := order.Taker
	if taker == "" {
		taker = common.Address{}.Hex()
	}

	sideInt := 0
	if order.Side == SideSell {
		sideInt = 1
	}

	exchange := polycommon.ContractCTFExchange
	if opts.NegRisk {
		exchange = polycommon.ContractNegRiskCTFExchange
	}

	signedOrder := &SignedOrder{
		Salt:          salt,
		Maker:         b.funder.Hex(),
		Signer:        b.signer.Hex(),
		Taker:         taker,
		TokenID:       order.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          sideInt,
		Expiration:    strconv.FormatInt(expiration, 10),
		Nonce:         strconv.FormatInt(nonce, 10),
		FeeRateBps:    strconv.Itoa(order.FeeRateBps),
		SignatureType: int(b.signatureType),
	}

	signature, err := b.signOrder(signedOrder, exchange)
	if err != nil {
		return nil, fmt.Errorf("sign order: %w", err)
	}
	signedOrder.Signature = signature

	return signedOrder, nil
}

// BuildMarketOrder 构建市价单
func (b *OrderBuilder) BuildMarketOrder(order UserMarketOrder, opts CreateOrderOptions) (*SignedOrder, error) {
	price := order.Price
	if price == 0 {
		price = 1.0
	}

	makerAmount, takerAmount := calculateMarketOrderAmounts(order.Side, order.Amount, price, opts.TickSize)

	salt := generateSalt()

	taker := order.Taker
	if taker == "" {
		taker = common.Address{}.Hex()
	}

	sideInt := 0
	if order.Side == SideSell {
		sideInt = 1
	}

	exchange := polycommon.ContractCTFExchange
	if opts.NegRisk {
		exchange = polycommon.ContractNegRiskCTFExchange
	}

	signedOrder := &SignedOrder{
		Salt:          salt,
		Maker:         b.funder.Hex(),
		Signer:        b.signer.Hex(),
		Taker:         taker,
		TokenID:       order.TokenID,
		MakerAmount:   makerAmount.String(),
		TakerAmount:   takerAmount.String(),
		Side:          sideInt,
		Expiration:    "0",
		Nonce:         strconv.FormatInt(order.Nonce, 10),
		FeeRateBps:    strconv.Itoa(order.FeeRateBps),
		SignatureType: int(b.signatureType),
	}

	signature, err := b.signOrder(signedOrder, exchange)
	if err != nil {
		return nil, fmt.Errorf("sign order: %w", err)
	}
	signedOrder.Signature = signature

	return signedOrder, nil
}

// signOrder 签名订单 (EIP-712)
func (b *OrderBuilder) signOrder(order *SignedOrder, exchange string) (string, error) {
	domainSeparator := buildOrderDomainSeparator(b.chainID, exchange)
	structHash := buildOrderStructHash(order)

	messageHash := crypto.Keccak256([]byte("\x19\x01"), domainSeparator, structHash)

	sig, err := crypto.Sign(messageHash, b.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}

	if sig[64] < 27 {
		sig[64] += 27
	}

	return "0x" + hex.EncodeToString(sig), nil
}

// buildOrderDomainSeparator 构建订单 EIP-712 Domain Separator
func buildOrderDomainSeparator(chainID int64, exchange string) []byte {
	domainTypeHash := crypto.Keccak256([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))

	nameHash := crypto.Keccak256([]byte(OrderDomain.Name))
	versionHash := crypto.Keccak256([]byte(OrderDomain.Version))
	chainIDPadded := common.LeftPadBytes(big.NewInt(chainID).Bytes(), 32)
	exchangePadded := common.LeftPadBytes(common.HexToAddress(exchange).Bytes(), 32)

	return crypto.Keccak256(domainTypeHash, nameHash, versionHash, chainIDPadded, exchangePadded)
}

// buildOrderStructHash 构建订单结构哈希
func buildOrderStructHash(order *SignedOrder) []byte {
	typeHash := crypto.Keccak256([]byte(OrderTypeHashStr))

	salt := new(big.Int)
	salt.SetString(order.Salt, 10)
	saltPadded := common.LeftPadBytes(salt.Bytes(), 32)

	makerPadded := common.LeftPadBytes(common.HexToAddress(order.Maker).Bytes(), 32)
	signerPadded := common.LeftPadBytes(common.HexToAddress(order.Signer).Bytes(), 32)
	takerPadded := common.LeftPadBytes(common.HexToAddress(order.Taker).Bytes(), 32)

	tokenID := new(big.Int)
	tokenID.SetString(order.TokenID, 10)
	tokenIDPadded := common.LeftPadBytes(tokenID.Bytes(), 32)

	makerAmount := new(big.Int)
	makerAmount.SetString(order.MakerAmount, 10)
	makerAmountPadded := common.LeftPadBytes(makerAmount.Bytes(), 32)

	takerAmount := new(big.Int)
	takerAmount.SetString(order.TakerAmount, 10)
	takerAmountPadded := common.LeftPadBytes(takerAmount.Bytes(), 32)

	expiration := new(big.Int)
	expiration.SetString(order.Expiration, 10)
	expirationPadded := common.LeftPadBytes(expiration.Bytes(), 32)

	nonce := new(big.Int)
	nonce.SetString(order.Nonce, 10)
	noncePadded := common.LeftPadBytes(nonce.Bytes(), 32)

	feeRateBps := new(big.Int)
	feeRateBps.SetString(order.FeeRateBps, 10)
	feeRateBpsPadded := common.LeftPadBytes(feeRateBps.Bytes(), 32)

	sidePadded := common.LeftPadBytes([]byte{byte(order.Side)}, 32)
	signatureTypePadded := common.LeftPadBytes([]byte{byte(order.SignatureType)}, 32)

	return crypto.Keccak256(
		typeHash,
		saltPadded,
		makerPadded,
		signerPadded,
		takerPadded,
		tokenIDPadded,
		makerAmountPadded,
		takerAmountPadded,
		expirationPadded,
		noncePadded,
		feeRateBpsPadded,
		sidePadded,
		signatureTypePadded,
	)
}

// RoundConfig 舍入配置
type RoundConfig struct {
	Price  int
	Size   int
	Amount int
}

var roundingConfigs = map[TickSize]RoundConfig{
	TickSize01:    {Price: 1, Size: 2, Amount: 3},
	TickSize001:   {Price: 2, Size: 2, Amount: 4},
	TickSize0001:  {Price: 3, Size: 2, Amount: 5},
	TickSize00001: {Price: 4, Size: 2, Amount: 6},
}

// calculateOrderAmounts 计算订单金额
func calculateOrderAmounts(side Side, size, price float64, tickSize TickSize) (*big.Int, *big.Int) {
	config := roundingConfigs[tickSize]
	if config.Price == 0 {
		config = roundingConfigs[TickSize001]
	}

	rawPrice := roundNormal(price, config.Price)

	if side == SideBuy {
		rawTakerAmt := roundDown(size, config.Size)
		rawMakerAmt := rawTakerAmt * rawPrice

		if decimalPlaces(rawMakerAmt) > config.Amount {
			rawMakerAmt = roundUp(rawMakerAmt, config.Amount+4)
			if decimalPlaces(rawMakerAmt) > config.Amount {
				rawMakerAmt = roundDown(rawMakerAmt, config.Amount)
			}
		}

		return toUnits(rawMakerAmt), toUnits(rawTakerAmt)
	}

	rawMakerAmt := roundDown(size, config.Size)
	rawTakerAmt := rawMakerAmt * rawPrice

	if decimalPlaces(rawTakerAmt) > config.Amount {
		rawTakerAmt = roundUp(rawTakerAmt, config.Amount+4)
		if decimalPlaces(rawTakerAmt) > config.Amount {
			rawTakerAmt = roundDown(rawTakerAmt, config.Amount)
		}
	}

	return toUnits(rawMakerAmt), toUnits(rawTakerAmt)
}

// calculateMarketOrderAmounts 计算市价单金额
func calculateMarketOrderAmounts(side Side, amount, price float64, tickSize TickSize) (*big.Int, *big.Int) {
	config := roundingConfigs[tickSize]
	if config.Price == 0 {
		config = roundingConfigs[TickSize001]
	}

	rawPrice := roundDown(price, config.Price)

	if side == SideBuy {
		rawMakerAmt := roundDown(amount, config.Size)
		rawTakerAmt := rawMakerAmt / rawPrice

		if decimalPlaces(rawTakerAmt) > config.Amount {
			rawTakerAmt = roundUp(rawTakerAmt, config.Amount+4)
			if decimalPlaces(rawTakerAmt) > config.Amount {
				rawTakerAmt = roundDown(rawTakerAmt, config.Amount)
			}
		}

		return toUnits(rawMakerAmt), toUnits(rawTakerAmt)
	}

	rawMakerAmt := roundDown(amount, config.Size)
	rawTakerAmt := rawMakerAmt * rawPrice

	if decimalPlaces(rawTakerAmt) > config.Amount {
		rawTakerAmt = roundUp(rawTakerAmt, config.Amount+4)
		if decimalPlaces(rawTakerAmt) > config.Amount {
			rawTakerAmt = roundDown(rawTakerAmt, config.Amount)
		}
	}

	return toUnits(rawMakerAmt), toUnits(rawTakerAmt)
}

// toUnits 转换为链上单位
func toUnits(value float64) *big.Int {
	valueStr := fmt.Sprintf("%.6f", value)
	parts := strings.Split(valueStr, ".")
	intPart := parts[0]
	decPart := "000000"
	if len(parts) > 1 {
		decPart = parts[1]
		for len(decPart) < 6 {
			decPart += "0"
		}
		if len(decPart) > 6 {
			decPart = decPart[:6]
		}
	}
	result := new(big.Int)
	result.SetString(intPart+decPart, 10)
	return result
}

func roundNormal(value float64, decimals int) float64 {
	multiplier := pow10(decimals)
	return float64(int(value*multiplier+0.5)) / multiplier
}

func roundDown(value float64, decimals int) float64 {
	multiplier := pow10(decimals)
	return float64(int(value*multiplier)) / multiplier
}

func roundUp(value float64, decimals int) float64 {
	multiplier := pow10(decimals)
	return float64(int(value*multiplier)+1) / multiplier
}

func pow10(n int) float64 {
	result := 1.0
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}

func decimalPlaces(value float64) int {
	str := fmt.Sprintf("%.10f", value)
	parts := strings.Split(str, ".")
	if len(parts) < 2 {
		return 0
	}
	dec := strings.TrimRight(parts[1], "0")
	return len(dec)
}

func generateSalt() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	salt := new(big.Int).SetBytes(bytes)
	return salt.String()
}

// GetOrderHash 计算订单哈希
func GetOrderHash(order *SignedOrder, chainID int64, negRisk bool) string {
	exchange := polycommon.ContractCTFExchange
	if negRisk {
		exchange = polycommon.ContractNegRiskCTFExchange
	}

	domainSeparator := buildOrderDomainSeparator(chainID, exchange)
	structHash := buildOrderStructHash(order)

	hash := crypto.Keccak256([]byte("\x19\x01"), domainSeparator, structHash)

	return "0x" + hex.EncodeToString(hash)
}

// ParseSignedOrder 解析签名订单
func ParseSignedOrder(data map[string]interface{}) (*SignedOrder, error) {
	getString := func(key string) string {
		if v, ok := data[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	getInt := func(key string) int {
		if v, ok := data[key]; ok {
			switch val := v.(type) {
			case float64:
				return int(val)
			case int:
				return val
			case string:
				i, _ := strconv.Atoi(val)
				return i
			}
		}
		return 0
	}

	return &SignedOrder{
		Salt:          getString("salt"),
		Maker:         getString("maker"),
		Signer:        getString("signer"),
		Taker:         getString("taker"),
		TokenID:       getString("tokenId"),
		MakerAmount:   getString("makerAmount"),
		TakerAmount:   getString("takerAmount"),
		Side:          getInt("side"),
		Expiration:    getString("expiration"),
		Nonce:         getString("nonce"),
		FeeRateBps:    getString("feeRateBps"),
		SignatureType: getInt("signatureType"),
		Signature:     getString("signature"),
	}, nil
}

// ValidateOrder 验证订单基本参数
func ValidateOrder(order *SignedOrder) error {
	if order.Salt == "" {
		return fmt.Errorf("salt is required")
	}
	if order.Maker == "" || !strings.HasPrefix(order.Maker, "0x") {
		return fmt.Errorf("invalid maker address")
	}
	if order.Signer == "" || !strings.HasPrefix(order.Signer, "0x") {
		return fmt.Errorf("invalid signer address")
	}
	if order.TokenID == "" {
		return fmt.Errorf("tokenId is required")
	}
	if order.MakerAmount == "" || order.MakerAmount == "0" {
		return fmt.Errorf("makerAmount must be positive")
	}
	if order.TakerAmount == "" || order.TakerAmount == "0" {
		return fmt.Errorf("takerAmount must be positive")
	}
	if order.Side != 0 && order.Side != 1 {
		return fmt.Errorf("side must be 0 (BUY) or 1 (SELL)")
	}
	if order.SignatureType < 0 || order.SignatureType > 2 {
		return fmt.Errorf("signatureType must be 0, 1, or 2")
	}
	if order.Signature == "" || !strings.HasPrefix(order.Signature, "0x") {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

// GetPriceFromOrder 从订单中计算价格
func GetPriceFromOrder(order *SignedOrder) float64 {
	makerAmount := new(big.Float)
	makerAmount.SetString(order.MakerAmount)

	takerAmount := new(big.Float)
	takerAmount.SetString(order.TakerAmount)

	if order.Side == 0 {
		price := new(big.Float).Quo(makerAmount, takerAmount)
		result, _ := price.Float64()
		return result
	}

	price := new(big.Float).Quo(takerAmount, makerAmount)
	result, _ := price.Float64()
	return result
}

// GetSizeFromOrder 从订单中计算数量
func GetSizeFromOrder(order *SignedOrder) float64 {
	if order.Side == 0 {
		takerAmount := new(big.Float)
		takerAmount.SetString(order.TakerAmount)
		size := new(big.Float).Quo(takerAmount, big.NewFloat(1e6))
		result, _ := size.Float64()
		return result
	}

	makerAmount := new(big.Float)
	makerAmount.SetString(order.MakerAmount)
	size := new(big.Float).Quo(makerAmount, big.NewFloat(1e6))
	result, _ := size.Float64()
	return result
}
