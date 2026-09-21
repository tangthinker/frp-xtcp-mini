# frp-xtcp-mini

[README](README.md) | [中文文档](README_zh.md)

A slim fork of [frp](https://github.com/fatedier/frp) **v0.71.0** that keeps only **XTCP NAT hole punching**.

`frps` is a public coordinator. Two `frpc` instances punch a P2P tunnel; after that, traffic goes **directly between clients**, not through the server.

## What it does

```
  visitor frpc  ── login / nathole ──►  frps (public IP)
  proxy   frpc  ── login / nathole ──►  frps
       │                                  │
       └──── P2P (KCP / QUIC) ────────────┘
            after hole punch succeeds
```

| Role | Binary | Job |
|------|--------|-----|
| Coordinator | `frps` | Accept logins, exchange NAT hole-punching info |
| XTCP server | `frpc` with `[[proxies]]` `type = "xtcp"` | Expose a local TCP service |
| XTCP visitor | `frpc` with `[[visitors]]` `type = "xtcp"` | Listen locally and punch toward the proxy |

Control plane to `frps` is **TCP** (TLS + yamux by default). The P2P data tunnel uses **QUIC** (default) or **KCP**.

## Requirements

- A machine with a **public IP** to run `frps`
- Two machines behind NAT to run `frpc` (proxy + visitor)
- Hole punching does **not** work with every NAT. Symmetric NAT on both sides is the usual failure case. Test with `frpc nathole discover` if a tunnel never comes up.

## Quick start

Build:

```bash
make build
```

Binaries land in `bin/frps` and `bin/frpc`.

### 1. Public server (`frps`)

```toml
# frps.toml
bindPort = 7000
auth.method = "token"
auth.token = "12345678"
```

```bash
./frps -c ./frps.toml
```

### 2. Machine B — expose local SSH (`frpc` proxy)

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

### 3. Machine C — visit that SSH (`frpc` visitor)

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

`serverName` must match the proxy `name`. `secretKey` must match on both sides.

```bash
./frpc -c ./frpc.toml
ssh -oPort=6000 127.0.0.1
```

Example configs:

- [conf/frps.toml](./conf/frps.toml) / [conf/frps_full_example.toml](./conf/frps_full_example.toml)
- [conf/frpc.toml](./conf/frpc.toml) / [conf/frpc_full_example.toml](./conf/frpc_full_example.toml)

## Configuration

TOML, YAML, and JSON are supported. INI is not.

Token auth is required: set the same `auth.token` on `frps` and every `frpc`. A file-backed `auth.tokenSource` is also supported.

Useful fields:

| Field | Where | Meaning |
|-------|--------|---------|
| `natHoleStunServer` | `frpc` | STUN server used to learn public addr / NAT type. Default: `stun.easyvoip.com:3478` |
| `keepTunnelOpen` | visitor | Keep punching so the tunnel stays ready |
| `protocol` | visitor | P2P transport: `quic` (default) or `kcp` |
| `fallbackTo` | visitor | Fall back to another **xtcp** visitor if this one fails |
| `transport.useEncryption` / `useCompression` | proxy / visitor | Optional extra crypto / compression on the tunnel |
| `natholeAnalysisDataReserveHours` | `frps` | How long to keep nathole analysis data (default 168h) |

Environment variables can be referenced in config (`{{ .Envs.NAME }}`). Multiple files can be pulled in with `includes`.

Discover NAT mapping:

```bash
./frpc nathole discover
```

## What was removed

This fork is XTCP-only. Compared with upstream frp it does **not** include:

TCP / UDP / HTTP / HTTPS / STCP / SUDP / TCPMUX proxies, plugins, VirtualNet, SSH tunnel gateway, dashboards, OIDC, Prometheus, vhost, load balancing, health checks, bandwidth limits, proxy protocol, INI config, or KCP / QUIC / websocket as the **control** transport to `frps`.

## Development

```bash
make build      # frps + frpc
make test       # unit tests
make e2e        # XTCP e2e (Ginkgo)
make alltest    # vet + unit + e2e
```

Upstream project: [fatedier/frp](https://github.com/fatedier/frp). Licensed under [Apache 2.0](./LICENSE).
