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
- **Multiple output formats** -- table, wide, JSON, CSV

## Install

### Binary download

Grab the latest binary from [Releases](https://github.com/jmsperu/netscan/releases) and place it in your `PATH`.

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
