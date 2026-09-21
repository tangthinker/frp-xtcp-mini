# frp-xtcp-mini

[README](README.md) | [中文文档](README_zh.md)

基于 [frp](https://github.com/fatedier/frp) **v0.71.0** 的精简版本，只保留 **XTCP NAT 打洞**。

`frps` 作为公网协调节点。两台 `frpc` 完成打洞后，业务流量在客户端之间 **P2P 直连**，不再经过服务端转发。

## 做什么

```
  visitor frpc  ── 登录 / 打洞信令 ──►  frps（公网 IP）
  proxy   frpc  ── 登录 / 打洞信令 ──►  frps
       │                                  │
       └──── P2P（KCP / QUIC）────────────┘
            打洞成功后的数据通道
```

| 角色 | 程序 | 职责 |
|------|------|------|
| 协调端 | `frps` | 接受登录，交换 NAT 打洞信息 |
| XTCP 服务端 | `frpc` 配置 `[[proxies]]` `type = "xtcp"` | 暴露本机 TCP 服务 |
| XTCP 访问端 | `frpc` 配置 `[[visitors]]` `type = "xtcp"` | 在本机监听，并向 proxy 打洞 |

与 `frps` 的控制连接是 **TCP**（默认 TLS + yamux）。打洞成功后的 P2P 数据通道使用 **QUIC**（默认）或 **KCP**。

## 运行条件

- 一台有 **公网 IP** 的机器跑 `frps`
- 两台内网机器分别跑 `frpc`（proxy + visitor）
- 打洞 **不保证** 在所有 NAT 下都能成功。两边都是对称 NAT 时通常会失败。连不上时可以用 `frpc nathole discover` 看 NAT 类型。

## 快速开始

编译：

```bash
make build
```

产物在 `bin/frps` 和 `bin/frpc`。

### 1. 公网服务器（`frps`）

```toml
# frps.toml
bindPort = 7000
auth.method = "token"
auth.token = "12345678"
```

```bash
./frps -c ./frps.toml
```

### 2. 机器 B — 暴露本机 SSH（`frpc` proxy）

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000
auth.method = "token"
auth.token = "12345678"

[[proxies]]
name = "ssh"
type = "xtcp"
secretKey = "abcdefg"
localIP = "127.0.0.1"
localPort = 22
```

```bash
./frpc -c ./frpc.toml
```

### 3. 机器 C — 访问该 SSH（`frpc` visitor）

```toml
# frpc.toml
serverAddr = "x.x.x.x"
serverPort = 7000
auth.method = "token"
auth.token = "12345678"

[[visitors]]
name = "ssh-visitor"
type = "xtcp"
serverName = "ssh"
secretKey = "abcdefg"
bindAddr = "127.0.0.1"
bindPort = 6000
keepTunnelOpen = true
```

`serverName` 必须和 proxy 的 `name` 一致，两边 `secretKey` 必须相同。

```bash
./frpc -c ./frpc.toml
ssh -oPort=6000 127.0.0.1
```

示例配置：

- [conf/frps.toml](./conf/frps.toml) / [conf/frps_full_example.toml](./conf/frps_full_example.toml)
- [conf/frpc.toml](./conf/frpc.toml) / [conf/frpc_full_example.toml](./conf/frpc_full_example.toml)

## 配置说明

支持 TOML、YAML、JSON，不再支持 INI。

认证只保留 token：`frps` 和每台 `frpc` 的 `auth.token` 必须相同。也可以用 `auth.tokenSource` 从文件读取 token。

常用字段：

| 字段 | 位置 | 含义 |
|------|------|------|
| `natHoleStunServer` | `frpc` | 用于探测公网地址 / NAT 类型的 STUN。默认 `stun.easyvoip.com:3478` |
| `keepTunnelOpen` | visitor | 持续打洞，保持隧道就绪 |
| `protocol` | visitor | P2P 传输：`quic`（默认）或 `kcp` |
| `fallbackTo` | visitor | 当前 visitor 失败时，回退到另一个 **xtcp** visitor |
| `transport.useEncryption` / `useCompression` | proxy / visitor | 隧道额外加密 / 压缩 |
| `natholeAnalysisDataReserveHours` | `frps` | 打洞分析数据保留时长（默认 168 小时） |

配置里可以用环境变量 `{{ .Envs.NAME }}`，也可以用 `includes` 拆分文件。

探测 NAT：

```bash
./frpc nathole discover
```

## 已移除的功能

本仓库只保留 XTCP。相对上游 frp，以下能力均已去掉：

TCP / UDP / HTTP / HTTPS / STCP / SUDP / TCPMUX 代理、插件、VirtualNet、SSH Tunnel Gateway、Dashboard、OIDC、Prometheus、vhost、负载均衡、健康检查、带宽限制、Proxy Protocol、INI 配置，以及用 KCP / QUIC / websocket 作为连 `frps` 的 **控制通道**。

## 开发

```bash
make build      # 编译 frps + frpc
make libfrpc    # 可嵌入的 C 共享库（见 libfrpc/README.md）
make test       # 单元测试
make e2e        # XTCP 端到端测试（Ginkgo）
make alltest    # vet + 单元测试 + e2e
```

上游项目：[fatedier/frp](https://github.com/fatedier/frp)。许可证：[Apache 2.0](./LICENSE)。
