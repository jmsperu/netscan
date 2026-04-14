# netscan

Fast cross-platform network scanner. Discover hosts, scan ports, identify services, and resolve MAC vendors. Single binary, no dependencies.

Works on **Windows**, **macOS**, and **Linux**.

## Features

- **LAN discovery** -- find all devices on your network via ping sweep
- **Port scanning** -- scan specific ports, ranges, or all 65535
- **Service detection** -- identify services and grab banners
- **MAC vendor lookup** -- embedded OUI database (500+ vendors)
- **ARP table integration** -- reads system ARP cache for MAC addresses
- **Hostname resolution** -- reverse DNS lookup
- **Watch mode** -- continuous monitoring with new/gone device alerts
- **Wake-on-LAN** -- send magic packets to wake devices
- **mDNS/Bonjour discovery** -- find AppleTVs, Chromecasts, printers, HomeKit devices by service
- **SNMP v1/v2c/v3** -- query switches, routers, printers; walk OIDs; list interfaces
- **IPv6 neighbor discovery** -- list IPv6 neighbors from NDP cache
- **Prometheus metrics export** -- scrape scan results in watch mode for Grafana
- **Embedded web UI** -- clean dashboard at http://localhost:8080, no separate install
- **Multiple output formats** -- table, wide, JSON, CSV

## What Makes netscan Different

| Feature | nmap | Angry IP | Fing | netscan |
|---------|------|----------|------|---------|
| Single static binary (no deps) | ❌ | ❌ (JVM) | ❌ | ✅ |
| Cross-platform | ✅ | ✅ | Partial | ✅ |
| Size | 20+ MB | 120 MB JVM | App bundle | **~3.5 MB** |
| mDNS discovery built-in | ❌ | ❌ | ✅ paid | ✅ |
| SNMP walk built-in | ❌ | ❌ | ❌ | ✅ |
| Prometheus export | ❌ | ❌ | ❌ | ✅ |
| Web UI built-in | ❌ | ✅ (GUI) | ✅ (app) | ✅ (embedded) |
| Wake-on-LAN | ❌ | ❌ | ✅ paid | ✅ |
| JSON/CSV output | XML only | ✅ | ❌ | ✅ |
| curl-installable on any box | ❌ | ❌ | ❌ | ✅ |

## Install

### One-liner (Mac and Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/jmsperu/netscan/main/install.sh | sudo bash
```

The script auto-detects OS and architecture (amd64/arm64), downloads the correct binary from the latest release, and installs it to `/usr/local/bin/netscan`.

**Install a specific version:**

```bash
curl -fsSL https://raw.githubusercontent.com/jmsperu/netscan/main/install.sh | sudo bash -s -- --version v0.1.1
```

**Install to a custom directory (no sudo needed):**

```bash
curl -fsSL https://raw.githubusercontent.com/jmsperu/netscan/main/install.sh | INSTALL_DIR=~/.local/bin bash
```

### Manual download

Grab the right binary from [Releases](https://github.com/jmsperu/netscan/releases):

| Platform | File |
|----------|------|
| macOS (M-series) | `netscan-darwin-arm64` |
| macOS (Intel) | `netscan-darwin-amd64` |
| Linux (x86_64) | `netscan-linux-amd64` |
| Linux (ARM64) | `netscan-linux-arm64` |
| Windows (x64) | `netscan-windows-amd64.exe` |
| Windows (ARM64) | `netscan-windows-arm64.exe` |

```bash
# macOS M-series example
sudo curl -L https://github.com/jmsperu/netscan/releases/latest/download/netscan-darwin-arm64 -o /usr/local/bin/netscan
sudo chmod +x /usr/local/bin/netscan
sudo xattr -d com.apple.quarantine /usr/local/bin/netscan   # macOS Gatekeeper
```

### Go install

```bash
go install github.com/jmsperu/netscan@latest
```

### Build from source

```bash
git clone https://github.com/jmsperu/netscan.git
cd netscan
make build
```

Cross-compile for all platforms:

```bash
make build-all    # outputs to dist/
```

## Quick start

```bash
netscan                    # scan current subnet (auto-detect)
netscan 192.168.1.0/24     # scan a specific subnet
```

## Usage

### Default scan

```bash
netscan                              # auto-detect subnet, scan common ports
netscan 10.0.0.0/24                  # scan specific subnet
netscan -p 22,80,443                 # scan specific ports
netscan -p 1-1024                    # scan port range
netscan --all                        # scan all 65535 ports
netscan --fast                       # ping sweep only (no port scan)
```

### Output formats

```bash
netscan -o table                     # default table output
netscan -o wide                      # wide table with all fields
netscan -o json                      # JSON output
netscan -o csv                       # CSV output
```

### Deep port scan

```bash
netscan ports 192.168.1.1            # scan common ports on a single host
netscan ports 192.168.1.1 -p 1-65535 # scan all ports
netscan ports 192.168.1.1 --all      # same as above
netscan ports 192.168.1.1 -t 1000    # 1000ms timeout
netscan ports 192.168.1.1 -c 2000    # 2000 concurrent connections
```

### Watch mode

```bash
netscan watch                        # monitor current subnet (every 60s)
netscan watch 192.168.1.0/24         # monitor specific subnet
netscan watch -n 30                  # scan every 30 seconds
```

### Wake-on-LAN

```bash
netscan wake AA:BB:CC:DD:EE:FF              # send magic packet
netscan wake AA:BB:CC:DD:EE:FF -b 192.168.1.255   # specify broadcast address
```

### mDNS / Bonjour discovery

```bash
netscan mdns                              # browse common service types
netscan mdns --timeout 10                 # longer discovery window
netscan mdns --service _googlecast._tcp   # specific service
```

Finds AirPlay speakers, Chromecasts, HomeKit devices, network printers, SMB shares, and more.

### SNMP

```bash
netscan snmp 192.168.1.1                         # quick summary (v2c, "public")
netscan snmp 10.0.0.1 -c private                 # custom community
netscan snmp 192.168.1.1 --interfaces            # list interfaces
netscan snmp 192.168.1.1 --walk 1.3.6.1.2.1.1    # walk a subtree
netscan snmp 192.168.1.1 --oid 1.3.6.1.2.1.1.5.0 # single OID
netscan snmp 10.0.0.1 -v 3 -u admin --authkey PASS --privkey PRIV    # SNMPv3
```

### IPv6 neighbor discovery

```bash
netscan ipv6                    # list discovered IPv6 neighbors
netscan ipv6 --reachable        # only reachable neighbors
netscan ipv6 --refresh -i en0   # send multicast ping to refresh cache
```

### Prometheus metrics export

Expose scan results for Prometheus scraping (perfect for watch mode + Grafana):

```bash
netscan watch 10.0.0.0/24 --prometheus-port :9100
```

Then in Prometheus:

```yaml
scrape_configs:
  - job_name: netscan
    static_configs:
      - targets: ['netscan-host:9100']
```

Metrics exposed: `netscan_hosts_up`, `netscan_scan_count`, `netscan_scan_duration_ms`, `netscan_port_open{ip,port}`.

### Web UI

```bash
netscan web                       # http://localhost:8080
netscan web --addr 0.0.0.0:8080   # expose to LAN
```

Clean terminal-aesthetic dashboard — trigger scans, see hosts, ports, and vendor info. Metrics endpoint at `/metrics`.

## Flags reference

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--ports` | `-p` | `22,80,443,...` | Ports to scan (comma-separated or range) |
| `--fast` | `-f` | `false` | Ping sweep only, skip port scan |
| `--all` | | `false` | Scan all 65535 ports |
| `--output` | `-o` | `table` | Output format: table, json, csv, wide |
| `--timeout` | `-t` | `500` | Connection timeout in milliseconds |
| `--concurrency` | `-c` | `256` | Max concurrent connections |
| `--verbose` | `-v` | `false` | Show detailed output |
| `--no-resolve` | | `false` | Skip hostname resolution |
| `--no-vendor` | | `false` | Skip MAC vendor lookup |
| `--interface` | `-i` | | Network interface to use |

## License

[XcoBean Community License v1.0](LICENSE) — MIT-based with attribution.

Built on XcoBean open technology — https://xcobean.co.ke
