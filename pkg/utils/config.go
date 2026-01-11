package utils

import (
	"encoding/json"
	"os"
	"strings"
)

// LoadJSON 从文件加载 JSON 配置到指定结构体
func LoadJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// LoadEnvFile 加载 .env 文件到环境变量（不覆盖已存在的）
func LoadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if idx := strings.Index(line, "="); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.Trim(strings.TrimSpace(line[idx+1:]), `"'`)
			if os.Getenv(key) == "" {
				os.Setenv(key, val)
			}
		}
	}
}

// ParseTokenIDs 解析 JSON 数组格式的 token IDs（如 "[\"id1\",\"id2\"]"）
func ParseTokenIDs(s string) []string {
	s = strings.Trim(s, "[]")
	var ids []string
	for _, p := range strings.Split(s, ",") {
		if id := strings.Trim(strings.TrimSpace(p), "\""); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
