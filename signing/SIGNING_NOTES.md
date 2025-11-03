# Hyperliquid Go SDK 签名实现关键要点

本文档总结了实现 Hyperliquid 签名时需要注意的所有关键问题，以确保与 Python SDK 完全兼容。

## 目录

1. [EIP-712 签名关键点](#eip-712-签名关键点)
2. [Msgpack 编码](#msgpack-编码)
3. [数据类型转换](#数据类型转换)
4. [地址处理](#地址处理)
5. [签名类型定义](#签名类型定义)
6. [常见陷阱](#常见陷阱)

---

## EIP-712 签名关键点

### 1. R 和 S 值必须去除前导零

Python SDK 会自动去除签名的 R 和 S 值中的前导零。Go 实现中必须显式处理：

```go
// signing.go:342-350
formatHex := func(b []byte) string {
    hexStr := common.Bytes2Hex(b)
    // Remove leading zeros but keep at least one digit
    trimmed := hexStr
    for len(trimmed) > 1 && trimmed[0] == '0' {
        trimmed = trimmed[1:]
    }
    return "0x" + trimmed
}

return &types.Signature{
    R: formatHex(r),  // 去除前导零
    S: formatHex(s),  // 去除前导零
    V: v,
}
```

**为什么重要**: 如果不去除前导零，签名验证会失败。

### 2. Hash 到 bytes32 的转换

在需要 `bytes32` 类型的地方（如 phantom agent 和 multi-sig），必须将 `[]byte` 转换为 `common.Hash`：

```go
// signing.go:124
func ConstructPhantomAgent(hash []byte, isMainnet bool) map[string]any {
    source := "b"
    if isMainnet {
        source = "a"
    }

    // 必须转换为 common.Hash 才能正确编码为 bytes32
    hash32 := common.BytesToHash(hash)

    return map[string]any{
        "source":       source,
        "connectionId": hash32,  // bytes32 类型
    }
}
```

```go
// signing.go:286-288
// Multi-sig action hash 也需要转换
multiSigActionHash := common.BytesToHash(multiSigActionHashBytes)

envelope := map[string]any{
    "multiSigActionHash": multiSigActionHash,  // bytes32 类型
    "nonce":              nonce,
}
```

**为什么重要**: EIP-712 编码 bytes32 时需要 32 字节的固定长度数组，`common.Hash` 确保了正确的编码格式。

---

## Msgpack 编码

### 1. Wire 类型必须有 msgpack 标签

Hyperliquid 使用 msgpack 序列化 action 数据后再计算 hash。所有 wire 类型必须添加 `msgpack` 标签：

```go
// types/types.go
type OrderWire struct {
    Asset      int           `json:"a" msgpack:"a"`      // 必须有 msgpack 标签
    IsBuy      bool          `json:"b" msgpack:"b"`
    LimitPx    string        `json:"p" msgpack:"p"`
    Sz         string        `json:"s" msgpack:"s"`
    ReduceOnly bool          `json:"r" msgpack:"r"`
    OrderType  OrderTypeWire `json:"t" msgpack:"t"`
    Cloid      *string       `json:"c,omitempty" msgpack:"c,omitempty"`
}

type LimitOrderType struct {
    Tif Tif `json:"tif" msgpack:"tif"`
}

type TriggerOrderTypeWire struct {
    TriggerPx string `json:"triggerPx" msgpack:"triggerPx"`
    IsMarket  bool   `json:"isMarket" msgpack:"isMarket"`
    Tpsl      Tpsl   `json:"tpsl" msgpack:"tpsl"`
}

type OrderTypeWire struct {
    Limit   *LimitOrderType       `json:"limit,omitempty" msgpack:"limit,omitempty"`
    Trigger *TriggerOrderTypeWire `json:"trigger,omitempty" msgpack:"trigger,omitempty"`
}

type ModifyWire struct {
    Oid   any       `json:"oid" msgpack:"oid"`
    Order OrderWire `json:"order" msgpack:"order"`
}
```

### 2. Msgpack 标签行为说明

`vmihailenco/msgpack/v5` 库的默认行为：

```go
// ❌ 错误：只有 json 标签，msgpack 不会自动使用
type BadWire struct {
    Asset int `json:"a"`  // msgpack.Marshal 会编码为 "Asset"，不是 "a"
}

// ✅ 正确：显式添加 msgpack 标签
type GoodWire struct {
    Asset int `json:"a" msgpack:"a"`  // msgpack.Marshal 会编码为 "a"
}

// 替代方案：使用 SetCustomStructTag（但需要修改 ActionHash 函数）
var buf bytes.Buffer
enc := msgpack.NewEncoder(&buf)
enc.SetCustomStructTag("json")  // 告诉 msgpack 使用 json 标签
enc.Encode(BadWire{Asset: 1})   // 现在会正确编码为 "a"
```

**为什么重要**: 如果字段名不匹配，生成的 hash 会不同，导致签名验证失败。

---

## 数据类型转换

### FloatToWire - 价格和数量转换

必须完全匹配 Python SDK 的实现：

```go
// utils/utils.go:14-38
func FloatToWire(x float64) (string, error) {
    // 1. 四舍五入到 8 位小数
    rounded := fmt.Sprintf("%.8f", x)

    // 2. 检查舍入误差
    parsedBack, err := strconv.ParseFloat(rounded, 64)
    if err != nil {
        return "", fmt.Errorf("failed to parse rounded value: %w", err)
    }

    if math.Abs(parsedBack-x) >= 1e-12 {
        return "", fmt.Errorf("float_to_wire causes rounding: %f", x)
    }

    // 3. 处理 -0 情况
    if rounded == "-0.00000000" {
        rounded = "0.00000000"
    }

    // 4. 标准化：去除尾随零和小数点
    normalized := strings.TrimRight(rounded, "0")
    normalized = strings.TrimRight(normalized, ".")

    return normalized, nil
}
```

**Python SDK 对比**:
```python
# hyperliquid-python-sdk/hyperliquid/utils/signing.py:455-462
def float_to_wire(x: float) -> str:
    rounded = f"{x:.8f}"
    if abs(float(rounded) - x) >= 1e-12:
        raise ValueError("float_to_wire causes rounding", x)
    if rounded == "-0":
        rounded = "0"
    normalized = Decimal(rounded).normalize()
    return f"{normalized:f}"
```

**为什么重要**: 价格和数量的字符串表示必须完全一致，否则 hash 不匹配。

---

## 地址处理

### 必须小写的地址

某些字段的地址必须转换为小写：

```go
// 1. Builder 地址
// exchange.go:311
if builder != nil {
    builder.B = strings.ToLower(builder.B)
}

// 2. User Genesis 用户地址
// exchange.go:1434
userWeiList[i] = []string{strings.ToLower(uw.User), uw.Wei}

// 3. Freeze User 地址
// exchange.go:1484
"user": strings.ToLower(user),

// 4. Oracle Updater 地址
// exchange.go:1716
oracleUpdater = strings.ToLower(*schema.OracleUpdater)

// 5. Multi-Sig 地址
// exchange.go:2029-2030
"multiSigUser": strings.ToLower(multiSigUser),
"outerSigner":  strings.ToLower(e.walletAddress),

// 6. User Dex Abstraction
// exchange.go:1168
"user": strings.ToLower(user),

// 7. WebSocket 订阅
// ws/websocket.go
return fmt.Sprintf("userFills:%s", strings.ToLower(*sub.User))
```

### 对应的 Python SDK

```python
# exchange.py:141
builder["b"] = builder["b"].lower()

# exchange.py:687
"userAndWei": [(user.lower(), wei) for (user, wei) in user_and_wei]

# exchange.py:714
"user": user.lower()

# exchange.py:878
"oracleUpdater": schema["oracleUpdater"].lower()

# exchange.py:1068, 1075
multi_sig_user = multi_sig_user.lower()
"outerSigner": self.wallet.address.lower()

# exchange.py:1137
"user": user.lower()
```

**为什么重要**: 地址的大小写不一致会导致签名验证失败或 API 调用失败。

---

## 签名类型定义

### 所有签名类型必须与 Python SDK 完全匹配

#### 字段顺序很重要

EIP-712 编码对字段顺序敏感。例如 `TokenDelegateSignTypes` 曾经的错误：

```go
// ❌ 错误的字段顺序
TokenDelegateSignTypes = []apitypes.Type{
    {Name: "hyperliquidChain", Type: "string"},
    {Name: "validator", Type: "address"},
    {Name: "isUndelegate", Type: "bool"},  // 错误：顺序不对
    {Name: "wei", Type: "uint256"},        // 错误：类型也不对
    {Name: "nonce", Type: "uint64"},
}

// ✅ 正确的字段顺序和类型
TokenDelegateSignTypes = []apitypes.Type{
    {Name: "hyperliquidChain", Type: "string"},
    {Name: "validator", Type: "address"},
    {Name: "wei", Type: "uint64"},         // 正确：uint64
    {Name: "isUndelegate", Type: "bool"},  // 正确：在 wei 之后
    {Name: "nonce", Type: "uint64"},
}
```

#### 完整的签名类型列表

参考 `signing/signing.go` 中的定义：

- `OrderSignTypes` - 下单
- `CancelOrderSignTypes` - 撤单
- `ModifyOrderSignTypes` - 改单
- `USDTransferSignTypes` - USDC 转账
- `SpotTransferSignTypes` - 现货转账
- `WithdrawSignTypes` - 提现
- `USDClassTransferSignTypes` - USD 类别转账（现货/合约互转）
- `SendAssetSignTypes` - 资产发送
- `TokenDelegateSignTypes` - 代币委托
- `ConvertToMultiSigUserSignTypes` - 转换为多签账户
- `MultiSigEnvelopeSignTypes` - 多签信封
- `UserDexAbstractionSignTypes` - DEX 抽象
- `ApproveAgentSignTypes` - 批准代理
- `ApproveBuilderFeeSignTypes` - 批准 builder 费用

所有定义必须与 Python SDK 的 `hyperliquid/utils/signing.py` 完全一致。

---

## 常见陷阱

### 1. ❌ 忘记去除 R/S 的前导零

```go
// 错误
return &types.Signature{
    R: "0x" + hex.EncodeToString(r),  // 可能有前导零
    S: "0x" + hex.EncodeToString(s),
    V: v,
}

// 正确
return &types.Signature{
    R: formatHex(r),  // 去除前导零
    S: formatHex(s),
    V: v,
}
```

### 2. ❌ Hash 没有转换为 common.Hash

```go
// 错误
return map[string]any{
    "source":       source,
    "connectionId": hash,  // []byte 不会被正确编码为 bytes32
}

// 正确
return map[string]any{
    "source":       source,
    "connectionId": common.BytesToHash(hash),  // common.Hash
}
```

### 3. ❌ Wire 类型缺少 msgpack 标签

```go
// 错误
type OrderWire struct {
    Asset int `json:"a"`  // 只有 json 标签
}

// 正确
type OrderWire struct {
    Asset int `json:"a" msgpack:"a"`  // 同时有 msgpack 标签
}
```

### 4. ❌ 地址没有转换为小写

```go
// 错误
action := map[string]any{
    "user": user,  // 可能是大小写混合的
}

// 正确
action := map[string]any{
    "user": strings.ToLower(user),
}
```

### 5. ❌ 签名类型字段顺序错误

```go
// 错误：顺序不对
{Name: "validator", Type: "address"},
{Name: "isUndelegate", Type: "bool"},
{Name: "wei", Type: "uint64"},

// 正确：必须匹配 Python SDK
{Name: "validator", Type: "address"},
{Name: "wei", Type: "uint64"},
{Name: "isUndelegate", Type: "bool"},
```

---

## 验证清单

在实现新的签名功能时，请检查：

- [ ] 所有 wire 类型是否有 `msgpack` 标签？
- [ ] 签名类型定义是否与 Python SDK 完全一致（字段名、类型、顺序）？
- [ ] 是否使用了 `formatHex` 去除 R/S 的前导零？
- [ ] bytes32 类型是否使用了 `common.BytesToHash()` 转换？
- [ ] 需要小写的地址是否调用了 `strings.ToLower()`？
- [ ] 浮点数转换是否使用了 `FloatToWire()`？
- [ ] Primary type 名称是否正确（如 "HyperliquidTransaction:SendMultiSig"）？

---

## 调试技巧

### 1. 对比 msgpack 编码结果

```go
data, _ := msgpack.Marshal(action)
fmt.Printf("Msgpack hex: %x\n", data)
fmt.Printf("Msgpack len: %d\n", len(data))

// 对比 Python SDK 的输出
```

### 2. 验证 action hash

```go
hash, _ := signing.ActionHash(action, nil, nonce, nil)
fmt.Printf("Action hash: %x\n", hash)

// 与 Python SDK 的 hash 对比
```

### 3. 检查 EIP-712 结构

```go
typedData := signing.UserSignedPayload(action, signTypes, primaryType)
fmt.Printf("TypedData: %+v\n", typedData)
```

---

## 参考资源

- **Python SDK**: `hyperliquid-python-sdk/hyperliquid/utils/signing.py`
- **EIP-712 规范**: https://eips.ethereum.org/EIPS/eip-712
- **Msgpack 文档**: https://github.com/vmihailenco/msgpack

---
