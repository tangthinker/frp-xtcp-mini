# libfrpc

把 `frpc` 嵌进宿主进程（Flutter / iOS / Android / 桌面）的共享库。配置、状态、事件都走 **JSON**；C ABI 定义在 [`frpc.h`](./frpc.h)。

实现分两层：

| 层 | 位置 | 用途 |
|----|------|------|
| Go API | `libfrpc` 本包 | 逻辑与测试的真实来源 |
| C ABI | `cmd/libfrpc` + `frpc.h` | `c-shared` / `c-archive`，给 FFI 用 |

Flutter 用 `dart:ffi` 调 C ABI 即可。打洞成功后，visitor 在本机监听 TCP，宿主把 HTTP `baseUrl` 指到 `status.visitors[0].url`。

## 生命周期

```
Start(config)
    │
    ▼
 starting ──► connecting ──► connected
                                │
                     控制连接断开 / Reconnect()
                                │
                                ▼
                          reconnecting ──► connected
                                │
 Stop() / 首次登录失败且 loginFailExit=true
                                │
                                ▼
                     stopping / failed / stopped
```

`Start` **只校验配置并拉起后台 goroutine**，不等待登录。要用隧道前先：

1. `WaitConnected(timeout)`，或
2. 轮询 `GetStatus` 直到 `state == "connected"`，或
3. 订阅事件，等到 `state == "connected"`。

默认 `loginFailExit = false`（与命令行 `frpc` 相反）：控制连接掉了会一直重连。这是给移动端准备的。

## 配置

字符串可以是 **JSON / TOML / YAML**，字段与普通 `frpc` 客户端配置相同。嵌入时多用 JSON：

```json
{
  "serverAddr": "x.x.x.x",
  "serverPort": 7000,
  "auth": { "method": "token", "token": "frps-token" },
  "visitors": [
    {
      "name": "nas-visitor",
      "type": "xtcp",
      "serverName": "nas",
      "secretKey": "per-household-key",
      "bindAddr": "127.0.0.1",
      "bindPort": 0,
      "keepTunnelOpen": true
    }
  ]
}
```

`bindPort = 0` 会分配一个空闲本机端口，并写进 status。Flutter 应使用返回的端口，不要自己猜。

家中 NAS 仍跑独立的 `frpc` proxy（或以后也用本库），`name` 必须等于 visitor 的 `serverName`，`secretKey` 必须一致。

## Status JSON

`StatusJSON` / `frpc_status` 返回：

```json
{
  "apiVersion": 1,
  "handle": 1,
  "state": "connected",
  "runId": "....",
  "reconnectCount": 0,
  "lastError": "",
  "serverAddr": "x.x.x.x",
  "serverPort": 7000,
  "visitors": [
    {
      "name": "nas-visitor",
      "type": "xtcp",
      "serverName": "nas",
      "bindAddr": "127.0.0.1",
      "bindPort": 61234,
      "url": "http://127.0.0.1:61234"
    }
  ],
  "proxies": []
}
```

`apiVersion` 只在 JSON 不兼容时才会加。当前为 `1`。

`url` 已经把 `0.0.0.0` / `::` 折成 `127.0.0.1`，可直接当 HTTP `baseUrl`。

## 事件 JSON

回调参数与 status 重叠，便于只订阅、不轮询：

```json
{
  "apiVersion": 1,
  "handle": 1,
  "state": "reconnecting",
  "runId": "....",
  "reconnectCount": 1,
  "error": "control connection lost",
  "ts": 1710000000000
}
```

回调跑在 **Go 后台线程**。必须很快返回，且 **禁止在回调里对同一个 handle 调 `Stop`**（`Stop` 会等会话退出）。

## C ABI

见 [`frpc.h`](./frpc.h)。约定：

| 函数 | 行为 |
|------|------|
| `frpc_start(config, &handle)` | 异步启动，成功时写入 handle |
| `frpc_wait_connected(handle, timeout_ms)` | 阻塞到 connected / 失败 / 超时 |
| `frpc_status(handle, &json)` | 堆上 JSON，调用方 `frpc_free` |
| `frpc_reconnect(handle)` | 主动掐断控制连接，触发重连 |
| `frpc_stop(handle)` | 停掉会话并释放 handle |
| `frpc_set_event_callback` | `cb == NULL` 取消订阅 |
| `frpc_version` / `frpc_strerror` | 静态字符串，不要 `free` |

错误码（与 Go `libfrpc.Code` 一致）：

| 码 | 含义 |
|----|------|
| `0` | 成功 |
| `-1` | 空指针 / 空配置 |
| `-2` | JSON/TOML/YAML 解析失败 |
| `-3` | 配置语义非法 |
| `-4` | handle 不存在（已 stop 或从未 start） |
| `-5` | 尚未 connected 就 `Reconnect` |
| `-6` | `WaitConnected` 超时 |
| `-7` | 等待过程中被 stop |
| `-8` | 会话进入 `failed`（例如 `loginFailExit=true` 且首次登录失败） |
| `-9` | 内部错误 |

可同时存在多个 handle（测试或 proxy + visitor 分两个会话）。

## 重连

底层 `client.Service` 在控制连接断开后会自动再登录。库把它暴露为：

- **被动**：frps 重启、网络切换 → `state=reconnecting`，成功后回到 `connected`，`reconnectCount` 加一。
- **主动**：`Reconnect()` 掐掉当前控制连接，走同一条路径。visitor 的 `bindPort` 保持不变。

第一次登录成功前调用 `Reconnect` 会返回 `-5`。

## 编译

```bash
make libfrpc
```

产物：

- macOS: `bin/libfrpc.dylib` + `bin/frpc.h`
- Linux: `bin/libfrpc.so`
- iOS: `go build -buildmode=c-archive -o libfrpc.a ./cmd/libfrpc`

示例：

```bash
cc -o /tmp/libfrpc-example libfrpc/example/example.c -I libfrpc bin/libfrpc.dylib -Wl,-rpath,$PWD/bin
```

## 测试

```bash
go test ./libfrpc/...
```

覆盖配置解析、登录、status 协议、事件、强制重连、frps 重启后重连、重连过程中的并发 status、stop。

## 线程与内存

- Go API 线程安全；C ABI 也是。
- `frpc_status` / `frpc_last_error` 返回的 `char*` 必须 `frpc_free`。
- 事件 JSON 只在回调期间有效，需要自己拷贝。
- 一个进程里 Go runtime 只初始化一次，适合长期嵌在 App 里。
