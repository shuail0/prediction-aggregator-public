package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LoadAccounts 从 CSV 文件加载账户对
// CSV 格式: index,evmAddressA,evmPrivateKeyA,proxyAddressA,evmAddressB,evmPrivateKeyB,proxyAddressB
func LoadAccounts(path string) ([]AccountPair, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开文件失败: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("读取CSV失败: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV文件为空或只有表头")
	}

	// 解析表头，获取列索引
	header := records[0]
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.TrimSpace(strings.ToLower(col))] = i
	}

	// 必需的列
	requiredCols := []string{"evmprivatekeya", "evmprivatekeyb"}
	for _, col := range requiredCols {
		if _, ok := colIndex[col]; !ok {
			return nil, fmt.Errorf("缺少必需列: %s", col)
		}
	}

	var accounts []AccountPair
	for i, row := range records[1:] {
		if len(row) < len(header) {
			continue
		}

		pair := AccountPair{Index: i + 1}

		// 解析索引
		if idx, ok := colIndex["index"]; ok && idx < len(row) {
			if n, err := strconv.Atoi(strings.TrimSpace(row[idx])); err == nil {
				pair.Index = n
			}
		}

		// 解析账户A
		if idx, ok := colIndex["evmaddressa"]; ok && idx < len(row) {
			pair.AddressA = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["evmprivatekeya"]; ok && idx < len(row) {
			pair.PrivateKeyA = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["proxyaddressa"]; ok && idx < len(row) {
			pair.ProxyA = strings.TrimSpace(row[idx])
		}

		// 解析账户B
		if idx, ok := colIndex["evmaddressb"]; ok && idx < len(row) {
			pair.AddressB = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["evmprivatekeyb"]; ok && idx < len(row) {
			pair.PrivateKeyB = strings.TrimSpace(row[idx])
		}
		if idx, ok := colIndex["proxyaddressb"]; ok && idx < len(row) {
			pair.ProxyB = strings.TrimSpace(row[idx])
		}

		// 验证必需字段
		if pair.PrivateKeyA == "" || pair.PrivateKeyB == "" {
			continue
		}

		accounts = append(accounts, pair)
	}

	if len(accounts) == 0 {
		return nil, fmt.Errorf("未找到有效账户")
	}

	return accounts, nil
}
