package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/bridge"
	"github.com/shuail0/prediction-aggregator/pkg/exchange/polymarket/relayer"
)

// ==================== 配置区域 ====================
var (
	proxyString = "127.0.0.1:7897"
)

// ==================== 配置区域结束 ====================

func init() {
	if f, err := os.Open(".env"); err == nil {
		defer f.Close()
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.Index(line, "="); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				val = strings.Trim(val, "'\"")
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
			}
		}
	}
}

func main() {
	ctx := context.Background()

	fmt.Println("=== Bridge API 示例 ===")
	fmt.Println("Bridge API 用于跨链充值到 Polymarket\n")

	// 创建 Bridge 客户端
	client := bridge.NewClient(bridge.ClientConfig{
		Timeout:     30 * time.Second,
		ProxyString: proxyString,
	})

	// 1. 获取支持的资产列表
	fmt.Println("1. 获取支持的资产列表")
	assets, err := client.GetSupportedAssets(ctx)
	if err != nil {
		fmt.Printf("   获取失败: %v\n", err)
	} else {
		fmt.Printf("   共支持 %d 种资产:\n", len(assets))
		for _, asset := range assets {
			fmt.Printf("   - %s (%s): %s 最小充值 $%.0f\n",
				asset.ChainName, asset.ChainID, asset.Token.Symbol, asset.MinCheckoutUsd)
		}
	}

	// 2. 创建充值地址 (需要私钥获取 Safe 地址)
	privateKey := os.Getenv("POLYMARKET_PRIVATE_KEY")
	if privateKey != "" {
		fmt.Println("\n2. 创建充值地址")

		// 获取 Safe 地址
		relayerClient, err := relayer.NewClient(relayer.Config{
			PrivateKey:  privateKey,
			ProxyString: proxyString,
		})
		if err != nil {
			fmt.Printf("   创建 Relayer 失败: %v\n", err)
			return
		}

		safeAddress := relayerClient.GetProxyAddress()
		fmt.Printf("   Safe 地址: %s\n", safeAddress)

		// 创建充值地址
		deposit, err := client.CreateDepositAddresses(ctx, safeAddress)
		if err != nil {
			fmt.Printf("   创建失败: %v\n", err)
		} else {
			fmt.Println("   充值地址:")
			fmt.Printf("   - EVM (Ethereum/Arbitrum/Base): %s\n", deposit.Address.EVM)
			fmt.Printf("   - Solana: %s\n", deposit.Address.SVM)
			fmt.Printf("   - Bitcoin: %s\n", deposit.Address.BTC)
			fmt.Printf("   备注: %s\n", deposit.Note)
		}
	} else {
		fmt.Println("\n2. 跳过创建充值地址 (未设置 POLYMARKET_PRIVATE_KEY)")
	}

	fmt.Println("\n=== Bridge API 示例完成 ===")
}
