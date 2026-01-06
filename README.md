# Prediction Aggregator

DEV：<a href="https://x.com/ShuaiWeb3"><img src="https://abs.twimg.com/responsive-web/client-web/icon-ios.b1fc727a.png" alt="X" width="18" height="18" style="vertical-align:middle;"> @ShuaiWeb3</a>

预测市场聚合器，使用 Go 构建。

## 支持的交易所

### Polymarket
- **Gamma API** - 市场数据查询
- **CLOB API** - 订单簿交易
- **WebSocket** - 实时数据推送
- **Data API** - 用户持仓和统计
- **Relayer** - 免 Gas 链上操作
- **Bridge** - 跨链充值

### Opinion
- **CLOB API** - 市场查询、订单管理、交易
- **WebSocket** - 订单簿、价格、用户订单实时推送
- **Chain** - 链上操作 (Split/Merge/Redeem)

## 快速开始

```bash
# 配置环境变量
cp .env.example .env
# 编辑 .env 填入私钥和代理

# Polymarket 示例
go run ./examples/polymarket/gamma/      # 市场数据
go run ./examples/polymarket/clob/       # 订单交易
go run ./examples/polymarket/relayer/    # 链上操作
go run ./examples/polymarket/bridge/     # 跨链充值
go run ./examples/polymarket/data/       # 用户数据
go run ./examples/polymarket/wss_market/ # WebSocket 市场数据
go run ./examples/polymarket/wss_user/   # WebSocket 用户数据

# Opinion 示例
go run ./examples/opinion/clob/              # CLOB 综合示例 (市场数据 + 交易)
go run ./examples/opinion/chain/             # 链上操作 (Split/Merge/Redeem)
go run ./examples/opinion/wss/               # WebSocket 基础示例
go run ./examples/opinion/wss_orderbook/     # 订单簿维护
go run ./examples/opinion/wss_market/        # 市场价格订阅
go run ./examples/opinion/wss_user/          # 用户订单/成交订阅
```

## 项目结构

```
├── pkg/exchange/              # 交易所实现
│   ├── polymarket/
│   │   ├── gamma/             # Gamma API
│   │   ├── clob/              # CLOB API
│   │   ├── wss/               # WebSocket
│   │   ├── data/              # Data API
│   │   ├── relayer/           # Relayer
│   │   ├── bridge/            # Bridge API
│   │   └── common/            # 共享工具
│   └── opinion/
│       ├── clob/              # CLOB API (市场查询 + 订单交易)
│       ├── wss/               # WebSocket 实时数据
│       ├── chain/             # 链上操作
│       └── common/            # 共享工具
├── examples/                  # 示例代码
├── strategies/                # 策略脚本
│   ├── farm/                  # 撸空投脚本
│   └── trading/               # 量化交易策略
├── data/                      # 数据文件
└── docs/                      # 文档
```

## 配置

`.env` 文件：
```env
# Polymarket
POLYMARKET_PRIVATE_KEY=你的私钥
POLYMARKET_PROXY_STRING=代理地址（可选）

# Opinion
OPINION_API_KEY=你的API Key
OPINION_PRIVATE_KEY=你的私钥（交易需要）
OPINION_MULTI_SIG=Gnosis Safe 多签地址（交易需要）
OPINION_PROXY=代理地址（可选）
```

## 许可证

MIT
