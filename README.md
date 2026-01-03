# Prediction Aggregator

预测市场聚合器，使用 Go 构建。

## 支持的交易所

### Polymarket
- **Gamma API** - 市场数据查询
- **CLOB API** - 订单簿交易
- **WebSocket** - 实时数据推送
- **Data API** - 用户持仓和统计
- **Relayer** - 免 Gas 链上操作
- **Bridge** - 跨链充值

## 快速开始

```bash
# 配置环境变量
cp .env.example .env
# 编辑 .env 填入私钥和代理

# 运行示例
go run ./examples/polymarket/gamma/      # 市场数据
go run ./examples/polymarket/clob/       # 订单交易
go run ./examples/polymarket/relayer/    # 链上操作
go run ./examples/polymarket/bridge/     # 跨链充值
go run ./examples/polymarket/data/       # 用户数据
go run ./examples/polymarket/wss_market/ # WebSocket 市场数据
go run ./examples/polymarket/wss_user/   # WebSocket 用户数据
```

## 项目结构

```
├── pkg/exchange/              # 交易所实现
│   └── polymarket/
│       ├── gamma/             # Gamma API
│       ├── clob/              # CLOB API
│       ├── wss/               # WebSocket
│       ├── data/              # Data API
│       ├── relayer/           # Relayer
│       ├── bridge/            # Bridge API
│       └── common/            # 共享工具
├── examples/                  # 示例代码
├── strategies/                # 策略脚本
│   ├── farm/                  # 撸空投脚本
│   └── trading/               # 量化交易策略
├── data/                      # 数据文件
└── docs/                      # 文档
```

## 策略

### 编译运行

```bash
# 编译
go build -o bin/polymarket_hedge ./strategies/farm/01_polymarket_hedge/

# 运行（配置文件放在 bin/ 目录下）
./bin/polymarket_hedge
```

## 配置

`.env` 文件：
```env
POLYMARKET_PRIVATE_KEY=你的私钥
POLYMARKET_PROXY_STRING=代理地址（可选）
```

## 许可证

MIT
