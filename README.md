# netscan

Fast cross-platform network scanner. Discover hosts, scan ports, identify services, and resolve MAC vendors. Single binary, no dependencies.

Works on **Windows**, **macOS**, and **Linux**.

## Features

- **LAN discovery** — find all devices on your network
- **Port scanning** — scan specific ports or all 65535
- **Service detection** — identify services and grab banners
- **MAC vendor lookup** — embedded OUI database (500+ vendors)
- **ARP table integration** — reads system ARP cache for MAC addresses
- **Hostname resolution** — reverse DNS lookup
- **Watch mode** — continuous monitoring with new/gone alerts
- **Wake-on-LAN** — send magic packets
- **Multiple output formats** — table, wide, JSON, CSV

## Install

Download from [Releases](https://github.com/jmsperu/netscan/releases) or build from source:

```bash
go install github.com/jmsperu/netscan@latest
```

## Usage

```bash
netscan                          # scan current subnet
netscan 192.168.1.0/24           # scan specific subnet
netscan -p 22,80,443             # scan specific ports
netscan -p 1-1024                # scan port range
netscan --fast                   # ping sweep only
netscan --all                    # scan all 65535 ports
netscan -o json                  # JSON output
netscan ports 192.168.1.1        # deep scan single host
netscan watch                    # continuous monitoring
netscan wake AA:BB:CC:DD:EE:FF   # Wake-on-LAN
```

## License

MIT
