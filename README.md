# 🦝 Raycoon

A modern, powerful CLI client for managing V2Ray/Xray proxy connections from your terminal.

```
🦝 Raycoon v1.0.0 — Your friendly neighborhood proxy manager
```

## Features

- **Multi-Protocol** — VMess, VLESS (+ Reality), Trojan, Shadowsocks
- **Xray-Core** — Full xray-core integration with detached process management
- **Subscriptions** — Auto-updating subscription groups with scheduler
- **Latency Testing** — TCP & HTTP strategies with concurrent worker pool
- **VPN Modes** — Proxy mode (SOCKS5/HTTP) or TUN mode (system-wide tunneling via TUN device)
- **TUN Mode** — macOS and Linux support — routes all traffic through xray without manual proxy config
- **Traffic Stats** — Real-time upload/download monitoring via Xray gRPC API
- **SQLite Storage** — Fast, reliable local database for all your configs

## Quick Start

```bash
# 1. Build & install
make build && cp bin/raycoon ~/.local/bin/

# 2. Create a group with a subscription link
raycoon group create myservers --subscription "https://your-sub-link.com/sub"

# 3. Fetch configs from subscription
raycoon sub update myservers

# 4. Test all configs for latency
raycoon test --all

# 5. Connect to the fastest server
raycoon connect --auto

# 6. Check status
raycoon status
```

## Installation

### Prerequisites

- **Go 1.22+**
- **Xray-core** binary at `~/.local/bin/xray` (with `geoip.dat` and `geosite.dat` alongside)
- **SQLite3** (usually pre-installed on macOS/Linux)

### Build from Source

```bash
git clone https://github.com/your-username/raycoon.git
cd raycoon
make build
```

The binary will be at `bin/raycoon`. Optionally install system-wide:

```bash
make install   # installs to $GOPATH/bin
```

### Xray-Core Setup

Download xray-core and place it where raycoon expects it:

```bash
mkdir -p ~/.local/bin
# Download latest xray-core release for your platform
# Extract xray, geoip.dat, geosite.dat to ~/.local/bin/
```

## Usage

### 🦝 Config Management

```bash
# Add a config from URI
raycoon config add 'vless://uuid@server:443?security=reality&...'

# Add with custom name and group
raycoon config add 'vmess://...' --name "US Fast" --group work

# Add with tags
raycoon config add 'trojan://...' --tags "us,fast,streaming"

# List all configs
raycoon config list

# Filter by group or protocol
raycoon config list --group work
raycoon config list --protocol vless
raycoon config list --enabled

# Show full details
raycoon config show 1
raycoon config show "US Fast"

# Delete a config
raycoon config delete 1
raycoon config delete "US Fast" --force
```

### 🦝 Group Management

Groups organize configs and can have subscription links for automatic updates.

```bash
# Create a simple group
raycoon group create work --desc "Work servers"

# Create with subscription (auto-updates daily by default)
raycoon group create personal \
  --subscription "https://example.com/sub" \
  --auto-update \
  --interval 86400

# List all groups
raycoon group list

# Delete a group and all its configs
raycoon group delete work
raycoon group delete work --force
```

### 🦝 Subscription Management

```bash
# Update a specific group
raycoon sub update personal

# Update all groups due for update
raycoon sub update --all

# Force update regardless of schedule
raycoon sub update personal --force

# Check subscription status
raycoon sub status
```

Example `sub status` output:
```
GROUP      CONFIGS  AUTO-UPDATE  INTERVAL  LAST UPDATED  NEXT UPDATE  STATUS
-----      -------  -----------  --------  ------------  -----------  ------
personal   15       ✓            1d        2h ago        22h          OK
work       8        ✓            12h       30m ago       11h          OK
gaming     25       ✓            1h        55m ago       5m           ⚠ Due
```

### 🦝 Connecting

```bash
# Connect by ID or name
raycoon connect 1
raycoon connect "US Fast"

# Auto-select lowest latency config
raycoon connect --auto
raycoon connect --auto --group work

# Specify ports
raycoon connect 1 --port 1080 --http-port 1081

# Choose VPN mode
raycoon connect 1 --mode proxy    # SOCKS5 + HTTP proxy (default)
sudo raycoon connect 1 --mode tun # TUN mode — all system traffic tunneled (requires root)

# Test latency before connecting
raycoon connect 1 --test

# Check connection status
raycoon status

# Disconnect
raycoon disconnect
```

**Proxy mode** — Configure your apps manually:
```bash
# SOCKS5 proxy
curl --socks5 127.0.0.1:1080 https://ipinfo.io

# HTTP proxy
curl --proxy http://127.0.0.1:1081 https://ipinfo.io

# Environment variables
export http_proxy=http://127.0.0.1:1081
export https_proxy=http://127.0.0.1:1081
```

**TUN mode** — Creates a virtual network device and routes ALL system traffic through the xray proxy, including apps that don't support proxy settings. Requires root (`sudo`). Disconnect can be done without sudo.

```bash
# Connect with TUN mode
sudo raycoon connect 1 --mode tun

# Disconnect (no sudo needed)
raycoon disconnect
```

### 🦝 Latency Testing

```bash
# Test a single config
raycoon test 1
raycoon test "US Fast"

# Test all enabled configs
raycoon test --all

# Test a specific group
raycoon test --group work

# Use HTTP strategy (slower but validates full proxy chain)
raycoon test --all --strategy http

# Adjust concurrency and timeout
raycoon test --all --workers 20 --timeout 3000

# View latency history
raycoon test history 1
raycoon test history "US Fast" --limit 50
```

Example batch test output:
```
Testing 15 configs...

  [1/15]  US-Server-1                              45 ms
  [2/15]  DE-Server-2                              120 ms
  [3/15]  JP-Server-3                              FAILED
  ...

Results (sorted by latency):
───────────────────────────────────────────────────────────────────────────
#   NAME                     ADDRESS              LATENCY  STATUS
-   ----                     -------              -------  ------
1   US-Server-1              1.2.3.4:443          45 ms    OK
2   DE-Server-2              5.6.7.8:443          120 ms   OK
...

Summary: 15 tested, 12 succeeded, 3 failed (4.2s)
```

## Protocol Support

| Protocol     | Status | Notes                          |
|-------------|--------|--------------------------------|
| VMess       | ✅     | Full support                   |
| VLESS       | ✅     | Including XTLS Reality         |
| Trojan      | ✅     | Full support                   |
| Shadowsocks | ✅     | Full support                   |
| Hysteria2   | 🔜     | Parser ready, core pending     |
| TUIC        | 🔜     | Parser ready, core pending     |
| WireGuard   | 🔜     | Parser ready, core pending     |

## Command Reference

```
raycoon                          🦝 Root command — shows help
├── config                       Manage proxy configurations
│   ├── add <uri>                Add config from URI
│   ├── list                     List all configs
│   ├── show <id|name>           Show config details
│   └── delete <id|name>         Delete a config
├── group                        Manage config groups
│   ├── create <name>            Create a new group
│   ├── list                     List all groups
│   └── delete <name>            Delete a group
├── sub (subscription)           Manage subscriptions
│   ├── update [group]           Update subscription(s)
│   └── status                   Show subscription status
├── connect [id|name]            Connect to a proxy
├── disconnect                   Disconnect current connection
├── status                       Show connection status
├── test [id|name]               Test proxy latency
│   └── history <id|name>        Show latency history
└── version                      Print version info
```

## Architecture

```
raycoon/
├── cmd/raycoon/              # Entry point
├── internal/
│   ├── app/                  # Application context & initialization
│   ├── cli/                  # Cobra CLI commands
│   ├── config/parser/        # Protocol URI parsers
│   ├── core/                 # Proxy core abstraction
│   │   ├── xray/             # Xray-core wrapper & config builder
│   │   ├── sysproxy/         # System proxy (macOS/Linux)
│   │   └── types/            # Shared types (VPNMode, CoreType)
│   ├── latency/              # Latency testing (TCP/HTTP strategies)
│   ├── storage/              # SQLite database layer
│   │   ├── models/           # Data models
│   │   └── sqlite/           # SQLite implementation
│   ├── subscription/         # Subscription fetcher & scheduler
│   └── tui/                  # Terminal UI (planned)
└── pkg/errors/               # Custom error types
```

## Data Storage

Raycoon uses SQLite. Default locations:

| Data     | Path                                |
|----------|-------------------------------------|
| Database | `~/.local/share/raycoon/raycoon.db` |
| Cache    | `~/.cache/raycoon/`                 |
| PID file | `~/.cache/raycoon/xray.pid`         |
| Xray bin | `~/.local/bin/xray`                 |

## Development

```bash
make build      # Build binary to bin/raycoon
make test       # Run tests with race detection
make coverage   # Generate HTML coverage report
make fmt        # Format code
make lint       # Run golangci-lint
make deps       # Download & tidy dependencies
make run        # Run from source
make help       # Show all targets
```

## Roadmap

- [x] Phase 1: Foundation (storage, parsers, CLI)
- [x] Phase 2: Core Integration (xray wrapper, connect/disconnect)
- [x] Phase 3: Subscription Management (fetcher, decoder, scheduler)
- [x] Phase 4: Latency Testing (TCP/HTTP strategies, worker pool)
- [ ] Phase 5: Interactive TUI (BubbleTea)
- [ ] Phase 6: Stats & Monitoring dashboard
- [ ] Phase 7: Polish & Release

## Acknowledgments

- [Xray-core](https://github.com/XTLS/Xray-core) — High-performance proxy core
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [BubbleTea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [go-sqlite3](https://github.com/mattn/go-sqlite3) — SQLite driver

## License

MIT
