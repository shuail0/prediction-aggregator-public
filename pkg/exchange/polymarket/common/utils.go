package common

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
)

// ParseMarketSlug 从 URL 解析市场 slug
func ParseMarketSlug(marketURL string) (string, error) {
	u, err := url.Parse(marketURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "event" {
		if len(parts) >= 3 {
			return parts[2], nil
		}
		return parts[1], nil
	}
	return "", fmt.Errorf("invalid market URL format: %s", marketURL)
}

// ParseEventSlug 从 URL 解析事件 slug
func ParseEventSlug(eventURL string) (string, error) {
	u, err := url.Parse(eventURL)
	if err != nil {
		return "", fmt.Errorf("parse url: %w", err)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 2 && parts[0] == "event" {
		return parts[1], nil
	}
	return "", fmt.Errorf("invalid event URL format: %s", eventURL)
}

// ParseTokenIDs 解析 clobTokenIds JSON 字符串
func ParseTokenIDs(clobTokenIds string) ([]string, error) {
	if clobTokenIds == "" {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(clobTokenIds), &ids); err != nil {
		return nil, fmt.Errorf("parse token ids: %w", err)
	}
	return ids, nil
}

// GetYesTokenID 获取 YES token ID
func GetYesTokenID(market *Market) (string, error) {
	ids, err := ParseTokenIDs(market.ClobTokenIds)
	if err != nil {
		return "", err
	}
	if len(ids) < 1 {
		return "", fmt.Errorf("no token ids found")
	}
	return ids[0], nil
}

// GetNoTokenID 获取 NO token ID
func GetNoTokenID(market *Market) (string, error) {
	ids, err := ParseTokenIDs(market.ClobTokenIds)
	if err != nil {
		return "", err
	}
	if len(ids) < 2 {
		return "", fmt.Errorf("no token id not found")
	}
	return ids[1], nil
}

// GetTokenID 根据类型获取 token ID
func GetTokenID(market *Market, tokenType string) (string, error) {
	if strings.ToUpper(tokenType) == "YES" {
		return GetYesTokenID(market)
	}
	return GetNoTokenID(market)
}

// GetTickSize 获取价格精度
func GetTickSize(market *Market) float64 {
	if market.OrderPriceMinTickSize == "" {
		return 0.01
	}
	tick, err := strconv.ParseFloat(string(market.OrderPriceMinTickSize), 64)
	if err != nil {
		return 0.01
	}
	return tick
}

// ParseOutcomePrices 解析价格 JSON
func ParseOutcomePrices(outcomePrices string) ([]float64, error) {
	if outcomePrices == "" {
		return nil, nil
	}
	var prices []float64
	if err := json.Unmarshal([]byte(outcomePrices), &prices); err != nil {
		var priceStrings []string
		if err := json.Unmarshal([]byte(outcomePrices), &priceStrings); err != nil {
			return nil, fmt.Errorf("parse outcome prices: %w", err)
		}
		for _, s := range priceStrings {
			p, _ := strconv.ParseFloat(s, 64)
			prices = append(prices, p)
		}
	}
	return prices, nil
}

// ParseOutcomes 解析结果名称
func ParseOutcomes(outcomes string) ([]string, error) {
	if outcomes == "" {
		return nil, nil
	}
	var names []string
	if err := json.Unmarshal([]byte(outcomes), &names); err != nil {
		return nil, fmt.Errorf("parse outcomes: %w", err)
	}
	return names, nil
}

// AlignPrice 价格对齐
func AlignPrice(price, tickSize float64, side string) float64 {
	if tickSize <= 0 {
		tickSize = 0.01
	}
	ticks := price / tickSize
	if strings.ToUpper(side) == "BUY" {
		return float64(int(ticks)) * tickSize
	}
	return float64(int(ticks+0.9999)) * tickSize
}

// ClampPrice 限制价格范围
func ClampPrice(price, tickSize float64) float64 {
	min, max := tickSize, 1.0-tickSize
	if price < min {
		return min
	}
	if price > max {
		return max
	}
	return price
}

// AlignAmount 数量对齐
func AlignAmount(amount, tickSize float64) float64 {
	if tickSize <= 0 {
		tickSize = 0.01
	}
	return float64(int(amount/tickSize)) * tickSize
}

// Pow10 计算 10^n
func Pow10(n int) int64 {
	result := int64(1)
	for i := 0; i < n; i++ {
		result *= 10
	}
	return result
}

// ParseUnits 解析金额字符串为 BigInt
func ParseUnits(amount string, decimals int) *big.Int {
	f, _ := strconv.ParseFloat(amount, 64)
	multiplier := new(big.Float).SetFloat64(float64(Pow10(decimals)))
	result := new(big.Float).Mul(big.NewFloat(f), multiplier)
	intResult, _ := result.Int(nil)
	return intResult
}

// FormatUnits 格式化 BigInt 为字符串
func FormatUnits(amount *big.Int, decimals int) string {
	if amount == nil {
		return "0"
	}
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	whole := new(big.Int).Div(amount, divisor)
	frac := new(big.Int).Mod(amount, divisor)
	if frac.Sign() == 0 {
		return whole.String()
	}
	fracStr := fmt.Sprintf("%0*d", decimals, frac)
	fracStr = strings.TrimRight(fracStr, "0")
	return fmt.Sprintf("%s.%s", whole.String(), fracStr)
}

// CalculateIndexSet 从 questionIDs 计算 indexSet
func CalculateIndexSet(questionIDs []string) *big.Int {
	indexSet := big.NewInt(0)
	seen := make(map[int]bool)
	for _, id := range questionIDs {
		if len(id) < 2 {
			continue
		}
		hexStr := id[len(id)-2:]
		index, err := strconv.ParseInt(hexStr, 16, 64)
		if err != nil {
			continue
		}
		if !seen[int(index)] {
			seen[int(index)] = true
			bit := new(big.Int).Lsh(big.NewInt(1), uint(index))
			indexSet.Or(indexSet, bit)
		}
	}
	return indexSet
}
