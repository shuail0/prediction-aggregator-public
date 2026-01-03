package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	// 命令行参数
	configFile := flag.String("config", "config.json", "配置文件路径(JSON)")
	flag.Parse()

	// 加载配置
	cfg, err := loadConfig(*configFile)
	if err != nil {
		fmt.Printf("加载配置失败: %v\n", err)
		os.Exit(1)
	}

	if len(cfg.MarketURLs) == 0 {
		fmt.Println("错误: 配置文件中 marketURLs 不能为空")
		os.Exit(1)
	}

	fmt.Printf("配置文件: %s\n", *configFile)
	fmt.Printf("账户文件: %s\n", cfg.AccountsFile)
	fmt.Printf("市场数量: %d\n", len(cfg.MarketURLs))
	fmt.Printf("最大交易金额: %.2f USDC\n", cfg.MaxTradeAmount)
	fmt.Printf("最小盘口间隔: %d tick\n", cfg.MinSpreadTicks)

	// 加载账户
	accounts, err := LoadAccounts(cfg.AccountsFile)
	if err != nil {
		fmt.Printf("加载账户失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("已加载 %d 对账户\n\n", len(accounts))

	// 创建策略
	strategy := NewStrategy(*cfg)

	// 执行结果统计
	var successCount, failCount int
	var results []*Result

	ctx := context.Background()
	startTime := time.Now()

	// 串行执行每对账户
	for i, pair := range accounts {
		fmt.Printf("\n========== 账户对 %d/%d (Index: %d) ==========\n", i+1, len(accounts), pair.Index)

		result := strategy.Execute(ctx, pair)
		results = append(results, result)

		if result.Success {
			successCount++
			fmt.Printf("[%d] 成功: A成交=%s, B成交=%s, 耗时=%v\n", pair.Index, result.FilledA, result.FilledB, result.Duration)
		} else {
			failCount++
			fmt.Printf("[%d] 失败: %s, 耗时=%v\n", pair.Index, result.Error, result.Duration)
		}
	}

	// 输出汇总
	fmt.Printf("\n========== 执行汇总 ==========\n")
	fmt.Printf("总账户对: %d\n", len(accounts))
	fmt.Printf("成功: %d\n", successCount)
	fmt.Printf("失败: %d\n", failCount)
	fmt.Printf("总耗时: %v\n", time.Since(startTime))

	if failCount > 0 {
		fmt.Println("\n失败详情:")
		for _, r := range results {
			if !r.Success {
				fmt.Printf("  [%d] %s\n", r.Index, r.Error)
			}
		}
	}
}

// loadConfig 从 JSON 文件加载配置
// 如果是相对路径，优先在可执行文件所在目录查找
func loadConfig(path string) (*Config, error) {
	// 如果是相对路径，尝试在可执行文件目录查找
	if !filepath.IsAbs(path) {
		if exePath, err := os.Executable(); err == nil {
			exeDir := filepath.Dir(exePath)
			absPath := filepath.Join(exeDir, path)
			if _, err := os.Stat(absPath); err == nil {
				path = absPath
			}
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	return &cfg, nil
}
