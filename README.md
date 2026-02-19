# Raycoon

A modern CLI client for managing V2Ray/Xray proxy connections from your terminal.

```
Raycoon v1.1.0 — Your friendly neighborhood proxy manager
```

## Features

- **Multi-Protocol** — VMess, VLESS (+ Reality), Trojan, Shadowsocks
- **Xray-Core** — Full xray-core integration with detached process management
- **TUN Mode** — System-wide tunneling via virtual network device; routes all traffic without per-app proxy config (powered by [tun2socks](https://github.com/xjasonlyu/tun2socks))
- **Proxy Mode** — SOCKS5 + HTTP proxy for per-app configuration
- **Subscriptions** — Auto-updating subscription groups with a built-in scheduler
- **Latency Testing** — TCP & HTTP strategies with concurrent worker pool
- **Traffic Stats** — Real-time upload/download monitoring via Xray gRPC API
- **SQLite Storage** — Fast, reliable local database for all your configs

## Installation

### One-liner (recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/AmirTlbyn/raycoon/main/install.sh | bash
```

Installs `raycoon` and `xray-core` (with geo files) automatically. Supports macOS and Linux on amd64/arm64.

### Homebrew (macOS/Linux)

```bash
brew tap AmirTlbyn/raycoon
brew install raycoon
```

### Build from Source

```bash
git clone https://github.com/AmirTlbyn/raycoon.git
cd raycoon
make build
sudo cp bin/raycoon /usr/local/bin/raycoon
```

## Quick Start

```bash
# Create a group with a subscription link
raycoon group create myservers --subscription "https://your-sub-link.com/sub"

# Fetch configs from the subscription
raycoon sub update myservers

# Test all configs for latency
raycoon test --all

# Connect to the fastest server
raycoon connect --auto

# Connect with TUN mode (routes all system traffic)
sudo raycoon connect --auto --mode tun
```

## Usage

### Config Management

```bash
# Add a config from URI
raycoon config add 'vless://uuid@server:443?security=reality&...'
raycoon config add 'vmess://...' --name "US Fast" --group work
raycoon config add 'trojan://...' --tags "us,fast,streaming"

# List, filter, inspect, delete
raycoon config list
raycoon config list --group work --protocol vless
raycoon config show "US Fast"
raycoon config delete "US Fast" --force
```

### Group Management

```bash
# Create a group (plain or with subscription)
raycoon group create work --desc "Work servers"
raycoon group create personal \
  --subscription "https://example.com/sub" \
  --auto-update \
  --interval 86400

raycoon group list
raycoon group delete work --force
```

### Subscription Management

```bash
raycoon sub update personal        # Update one group
raycoon sub update --all           # Update all due groups
raycoon sub update personal --force
raycoon sub status
```

Example output:
```
GROUP      CONFIGS  AUTO-UPDATE  INTERVAL  LAST UPDATED  NEXT UPDATE  STATUS
-----      -------  -----------  --------  ------------  -----------  ------
personal   15       ✓            1d        2h ago        22h          OK
work       8        ✓            12h       30m ago       11h          OK
gaming     25       ✓            1h        55m ago       5m           Due
```

### Connecting

```bash
# Connect by ID or name
raycoon connect 1
raycoon connect "US Fast"

# Auto-select lowest latency
raycoon connect --auto
raycoon connect --auto --group work

# Choose mode
raycoon connect 1 --mode proxy      # SOCKS5 + HTTP (default, no sudo)
sudo raycoon connect 1 --mode tun   # TUN mode — all system traffic tunneled

# Check status and disconnect
raycoon status
raycoon disconnect
```

**Proxy mode** — configure apps manually or via environment:
```bash
curl --socks5 127.0.0.1:1080 https://ipinfo.io

export http_proxy=http://127.0.0.1:1081
export https_proxy=http://127.0.0.1:1081
```

**TUN mode** — creates a virtual network device and routes all system traffic through the proxy, including apps that don't support proxy settings. Requires root on first connect; disconnect does not require root.

```bash
sudo raycoon connect 1 --mode tun
raycoon disconnect
```

### Latency Testing

```bash
raycoon test 1
raycoon test --all
raycoon test --group work --strategy http --workers 20 --timeout 3000
raycoon test history "US Fast" --limit 50
```

Example output:
```
Testing 15 configs...

  [1/15]  US-Server-1          45 ms
  [2/15]  DE-Server-2          120 ms
  [3/15]  JP-Server-3          FAILED

Results (sorted by latency):
────────────────────────────────────────────────────────────────
#   NAME          ADDRESS          LATENCY  STATUS
1   US-Server-1   1.2.3.4:443      45 ms    OK
2   DE-Server-2   5.6.7.8:443      120 ms   OK

Summary: 15 tested, 12 succeeded, 3 failed (4.2s)
```

## Protocol Support

| Protocol     | Status | Notes                      |
|--------------|--------|----------------------------|
| VMess        | ✅     | Full support               |
| VLESS        | ✅     | Including XTLS Reality     |
| Trojan       | ✅     | Full support               |
| Shadowsocks  | ✅     | Full support               |
| Hysteria2    | 🔜     | Planned (sing-box)         |
| TUIC         | 🔜     | Planned (sing-box)         |

## Command Reference

```
raycoon
├── config                       Manage proxy configurations
│   ├── add <uri>                Add config from URI
│   ├── list                     List configs (filterable)
│   ├── show <id|name>           Show config details
│   └── delete <id|name>         Delete a config
├── group                        Manage config groups
│   ├── create <name>            Create a group (optional sub URL)
│   ├── list                     List all groups
│   └── delete <name>            Delete a group and its configs
├── sub                          Subscription management
│   ├── update [group|--all]     Fetch and sync configs
│   └── status                   Show update schedule
├── connect [id|name]            Connect to a proxy
│   ├── --auto                   Pick lowest-latency config
│   ├── --mode proxy|tun         VPN mode (default: proxy)
│   └── --group <name>           Limit auto-selection to a group
├── disconnect                   Disconnect and restore system state
├── status                       Show current connection and stats
├── test [id|name|--all]         Test latency
│   └── history <id|name>        Show latency history
└── version                      Print version info
```

## Architecture

```
raycoon/
├── cmd/raycoon/              Entry point
├── internal/
│   ├── app/                  Application context & initialization
│   ├── cli/                  Cobra CLI commands
│   ├── config/parser/        Protocol URI parsers (VMess/VLESS/Trojan/SS)
│   ├── core/                 Proxy core abstraction
│   │   ├── xray/             Xray-core wrapper & config builder
│   │   ├── tun/              TUN device management & daemon lifecycle (tun2socks)
│   │   ├── sysproxy/         System proxy settings (macOS/Linux)
│   │   └── types/            Shared types (VPNMode, CoreType)
│   ├── latency/              Latency testing (TCP/HTTP strategies)
│   ├── storage/              SQLite database layer
│   │   ├── models/           Data models
│   │   └── sqlite/           SQLite implementation
│   └── subscription/         Subscription fetcher & scheduler
└── pkg/errors/               Custom error types
```

## Data Locations

| Data         | Path                                |
|--------------|-------------------------------------|
| Database     | `~/.local/share/raycoon/raycoon.db` |
| Cache        | `~/.cache/raycoon/`                 |
| Xray PID     | `~/.cache/raycoon/xray.pid`         |
| TUN daemon   | `~/.cache/raycoon/tund.log`         |
| Xray binary  | `~/.local/bin/xray`                 |

## Development

```bash
make build      # Build binary to bin/raycoon
make test       # Run tests with race detection
make coverage   # Generate HTML coverage report
make fmt        # Format code
make lint       # Run golangci-lint
make deps       # Download & tidy dependencies
make help       # Show all targets
```

## Roadmap

### Released

| Version | Milestone |
|---------|-----------|
| v1.0.0  | Foundation — SQLite storage, URI parsers (VMess/VLESS/Trojan/SS), group & subscription management, latency testing, xray-core integration, system proxy (macOS + Linux) |
| v1.1.0  | **TUN mode** — system-wide tunneling via virtual network device with automatic route management, DNS override, and daemon lifecycle |

### Planned

| Version | Milestone |
|---------|-----------|
| v1.2.0  | Interactive TUI — BubbleTea-based terminal UI with tab navigation, live stats, and one-key connect/disconnect |
| v1.3.0  | Stats & monitoring — real-time traffic graphs, per-config bandwidth history |
| v1.4.0  | sing-box core — Hysteria2, TUIC, and additional modern protocols |
| v2.0.0  | Rule-based routing — split-tunnel, per-app proxy, domain/IP rule sets |

## Acknowledgments

- [Xray-core](https://github.com/XTLS/Xray-core) — High-performance proxy core
- [tun2socks](https://github.com/xjasonlyu/tun2socks) — TUN device to SOCKS5 engine powering TUN mode
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [BubbleTea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [go-sqlite3](https://github.com/mattn/go-sqlite3) — SQLite driver
- [gocron](https://github.com/go-co-op/gocron) — Job scheduler

## License

MIT — see [LICENSE](LICENSE)
