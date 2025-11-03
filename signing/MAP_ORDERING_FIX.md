# Map 键顺序修复说明

## 问题

Go 的 `map[string]any` 在迭代时顺序是随机的，但 Python 3.7+ 的 `dict` 保持插入顺序。msgpack 编码时，键的顺序会影响编码结果，导致签名不一致。

## 解决方案

使用 `signing.NewOrderedMap()` 函数确保键的插入顺序与 Python SDK 完全一致。

## 使用方法

```go
import "github.com/dwdwow/hl-go/signing"

// ❌ 错误：键顺序可能随机
action := map[string]any{
    "type": "order",
    "orders": orders,
    "grouping": "na",
}

// ✅ 正确：确保键顺序与 Python SDK 一致
action := signing.NewOrderedMap(
    "type", "order",
    "orders", orders,
    "grouping", "na",
)
```

## 已修复的函数

### signing 包
1. `OrderWiresToOrderAction` - 订单 action
2. `ConstructPhantomAgent` - Phantom agent
3. `SignMultiSigAction` - Multi-sig envelope

### client/exchange.go 中的所有 action
所有用于签名的 action 都已修复，包括：
- `BulkCancel` / `BulkCancelByCloid` - 取消订单
- `BulkModifyOrders` - 修改订单
- `ScheduleCancel` - 计划取消
- `UpdateLeverage` - 更新杠杆
- `UpdateIsolatedMargin` - 更新隔离保证金
- `SetReferrer` - 设置推荐人
- `CreateSubAccount` - 创建子账户
- `USDTransfer` / `SpotTransfer` - 转账
- `USDClassTransfer` - USD 类别转账
- `SendAsset` - 发送资产
- `WithdrawFromBridge` - 提现
- `SubAccountTransfer` / `SubAccountSpotTransfer` - 子账户转账
- `VaultTransfer` - Vault 转账
- `TokenDelegate` - 代币委托
- `ApproveAgent` - 批准代理
- `ApproveBuilderFee` - 批准 builder 费用
- `Noop` - 空操作
- `UserDexAbstraction` - DEX 抽象
- `AgentEnableDexAbstraction` - Agent DEX 抽象
- `TWAPOrder` / `TWAPCancel` - TWAP 订单
- `UseBigBlocks` - 使用大块
- `ConvertToMultiSigUser` - 转换为多签账户
- Spot Deploy 相关操作（RegisterToken, UserGenesis, FreezeUser, Genesis, RegisterSpot, RegisterHyperliquidity, SetDeployerTradingFeeShare）
- Perp Deploy 相关操作（RegisterAsset, SetOracle）
- C-Signer 相关操作（UnjailSelf, JailSelf）
- C-Validator 相关操作（Register, ChangeProfile, Unregister）
- `MultiSig` - 多签操作

## 关键点

1. **查看 Python SDK 顺序**：每个 action 的键顺序必须与 Python SDK 完全一致
2. **嵌套 map**：嵌套的 map 也需要使用 `NewOrderedMap` 确保顺序
3. **排序要求**：某些情况下 Python SDK 会对 map 进行排序（如 `sorted(list(dict.items()))`），Go 代码也需要相应排序
4. **可选字段**：如果字段是可选添加的（如 builder），需要在创建 map 后添加，但应尽量避免这种情况

## 注意事项

- `postAction` 中的 `payload` map 不需要修复，因为它不用于签名，只是发送到 API
- 所有用于签名的 action 必须使用 `NewOrderedMap`
- 对于嵌套的 map，确保每层都使用 `NewOrderedMap`

## 验证

运行测试确保 msgpack 编码与 Python SDK 一致：
```bash
go test ./signing -v -run TestPhantomAgentHash
```

