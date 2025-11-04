# WebSocket 使用问题总结

## 检查日期
2024年全面检查

## 问题总结

### 🔴 严重问题（影响签名和功能）

#### 1. `postAction` 中 payload 的键顺序问题
**位置**: `client/exchange.go:203-212`

**问题描述**:
```go
payload := map[string]any{
    "action":       action,
    "nonce":        nonce,
    "signature":    signature,
    "vaultAddress": vaultAddr,
}

if e.expiresAfter != nil {
    payload["expiresAfter"] = *e.expiresAfter
}
```

**问题**:
- 使用普通 `map[string]any` 创建 payload，Go map 的迭代顺序是随机的
- 虽然 payload 不直接用于签名（签名使用的是 action），但 payload 的 JSON 序列化顺序可能影响服务器端的处理
- 如果服务器端对 payload 的键顺序有要求，可能会导致问题

**影响**: 
- 中等（payload 不用于签名，但可能影响服务器端处理）
- JSON 编码时键顺序不确定

**Python SDK 参考**:
- Python dict 保持插入顺序（Python 3.7+）
- 需要确认 Python SDK websocket 发送的 payload 键顺序

#### 2. WebSocket 消息构建中的键顺序问题
**位置**: `ws/ws_post.go:82-89`

**问题描述**:
```go
msg := map[string]any{
    "method": "post",
    "id":     c.id,
    "request": map[string]any{
        "type":    magType,
        "payload": payload,
    },
}
```

**问题**:
- 外层 `msg` 使用普通 `map[string]any`，键顺序不确定
- 内层 `request` 也使用普通 `map[string]any`，键顺序不确定
- 虽然 WebSocket 消息的键顺序通常不影响功能，但为了与 Python SDK 保持一致，应该使用有序 map

**影响**: 
- 低（WebSocket 消息格式通常不依赖键顺序）

### 🟡 代码质量问题

#### 3. `newAPIUsingWs` 中重复调用 `Start()`
**位置**: `client/api.go:65-85`

**问题描述**:
```go
func newAPIUsingWs(baseURL string, timeout time.Duration) (*API, error) {
    // ...
    w := ws.NewPostOnlyClient()
    if err := w.Start(); err != nil {  // 第一次调用
        return nil, fmt.Errorf("failed to start WebSocket client: %w", err)
    }
    err := w.Start()  // 第二次调用 - 重复！
    if err != nil {
        return nil, fmt.Errorf("failed to start WebSocket client: %w", err)
    }
    // ...
}
```

**问题**:
- `w.Start()` 被调用了两次
- 第一次调用如果成功，第二次调用可能会失败或导致资源泄漏
- 第一次调用后已经连接，第二次调用可能重复连接

**影响**: 
- 中等（可能导致连接问题或资源泄漏）

#### 4. `PostOnlyClient` 缺少 `respWaiters` 初始化
**位置**: `ws/ws_post.go:67-72`

**问题描述**:
```go
func NewPostOnlyClient() *PostOnlyClient {
    return &PostOnlyClient{
        url:          MainnetWsURL,
        pingInterval: 40 * time.Second,
        // respWaiters 没有被初始化！
    }
}
```

**问题**:
- `respWaiters` 字段没有被初始化，在 `Request` 方法中会被使用
- 如果 `respWaiters` 是 `nil`，会导致 panic

**影响**: 
- 高（会导致 panic）

**检查**:
- 需要确认 `respWaiters` 的类型定义，如果是指针类型，可能不是问题
- 如果是 `map[int64]PostOnlyRespWaiter`，需要初始化

### 🟢 潜在问题（需要验证）

#### 5. WebSocket 消息的键顺序是否重要
**问题**:
- 需要确认 Hyperliquid WebSocket API 是否对消息的键顺序有要求
- 需要对比 Python SDK 的 WebSocket 消息格式

#### 6. Payload 中 expiresAfter 的添加时机
**位置**: `client/exchange.go:210-212`

**问题**:
- `expiresAfter` 是在创建 payload 之后添加的
- 这会导致键顺序与 Python SDK 可能不同（如果 Python SDK 在创建时就包含该字段）

**影响**: 
- 低（如果服务器不依赖键顺序）

## 与 HTTP 方法的对比

### HTTP 方法
- ✅ 所有 action 创建都使用 `NewOrderedMap`
- ✅ 签名机制正确
- ✅ 测试通过

### WebSocket 方法
- ❌ `postAction` 中的 payload 未使用 `NewOrderedMap`
- ❌ WebSocket 消息构建未使用 `NewOrderedMap`
- ❌ 有代码质量问题（重复调用 Start，可能的 nil map）

## 建议修复优先级

1. **高优先级**:
   - 修复 `respWaiters` 初始化问题（如果确实存在问题）
   - 修复 `newAPIUsingWs` 中重复调用 `Start()` 的问题

2. **中优先级**:
   - 修复 `postAction` 中 payload 的键顺序（使用 `NewOrderedMap`）
   - 确保与 Python SDK 的键顺序一致

3. **低优先级**:
   - 修复 WebSocket 消息构建的键顺序（如果确实需要）

## 注意事项

1. **Payload 不用于签名**: `postAction` 中的 payload 包含 `action`、`signature` 等，这些是已经签名后的数据，所以 payload 本身的键顺序不影响签名正确性。

2. **JSON 序列化**: Go 的 `json.Marshal` 和 `encoding/json` 对 map 的键顺序是随机的（但 Go 1.12+ 为了测试稳定性，使用了某种排序）。如果需要完全确定顺序，应该使用 `NewOrderedMap`。

3. **WebSocket 协议**: WebSocket 消息是 JSON 格式，理论上键顺序不应该影响功能，但为了与 Python SDK 保持一致，最好还是使用有序 map。

