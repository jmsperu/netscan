package scanner

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"
)

// Host represents a discovered network host.
type Host struct {
	IP       string   `json:"ip"`
	MAC      string   `json:"mac,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
	Vendor   string   `json:"vendor,omitempty"`
	Ports    []Port   `json:"ports,omitempty"`
	Latency  float64  `json:"latency_ms"`
	Alive    bool     `json:"alive"`
}

// Port represents an open port on a host.
type Port struct {
	Number  int    `json:"port"`
	Proto   string `json:"proto"`
	State   string `json:"state"`
	Service string `json:"service,omitempty"`
	Banner  string `json:"banner,omitempty"`
}

// ScanConfig holds scan parameters.
type ScanConfig struct {
	Subnet      string
	Ports       []int
	Timeout     time.Duration
	Concurrency int
	NoResolve   bool
	NoVendor    bool
	Interface   string
}

// GetLocalSubnet auto-detects the local subnet from the specified interface or default route.
func GetLocalSubnet(ifaceName string) (string, error) {
	var ifaces []net.Interface
	var err error

	if ifaceName != "" {
		iface, err := net.InterfaceByName(ifaceName)
		if err != nil {
			return "", fmt.Errorf("interface %q not found: %w", ifaceName, err)
		}
		ifaces = []net.Interface{*iface}
	} else {
		ifaces, err = net.Interfaces()
		if err != nil {
			return "", fmt.Errorf("listing interfaces: %w", err)
		}
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil {
				continue
			}
			// Skip link-local
			if ip[0] == 169 && ip[1] == 254 {
				continue
			}
			ones, bits := ipNet.Mask.Size()
			network := ip.Mask(ipNet.Mask)
			return fmt.Sprintf("%s/%d", network.String(), ones), nil
			_ = bits
		}
	}

	return "", fmt.Errorf("no suitable network interface found")
}

// ExpandSubnet generates all host IPs in a CIDR range.
func ExpandSubnet(cidr string) ([]string, error) {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	var ips []string
	addr := prefix.Addr()

	// Calculate network size
	bits := prefix.Bits()
	hostBits := 32 - bits
	if hostBits > 20 {
		return nil, fmt.Errorf("subnet too large (/%d), max is /12", bits)
	}

	totalHosts := 1 << hostBits
	for i := 1; i < totalHosts-1; i++ { // skip network and broadcast
		raw := addr.As4()
		ipInt := binary.BigEndian.Uint32(raw[:])
		ipInt += uint32(i)
		var newIP [4]byte
		binary.BigEndian.PutUint32(newIP[:], ipInt)
		ips = append(ips, netip.AddrFrom4(newIP).String())
	}

	return ips, nil
}

// PingSweep checks which hosts are alive using TCP connect to common ports.
func PingSweep(ips []string, timeout time.Duration, concurrency int) []Host {
	var mu sync.Mutex
	var hosts []Host
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	probePorts := []int{22, 80, 443, 445, 3389, 8080}

	for _, ip := range ips {
		wg.Add(1)
		sem <- struct{}{}
		go func(ip string) {
			defer wg.Done()
			defer func() { <-sem }()

			start := time.Now()
			alive := false

			for _, port := range probePorts {
				addr := fmt.Sprintf("%s:%d", ip, port)
				conn, err := net.DialTimeout("tcp", addr, timeout)
				if err == nil {
					conn.Close()
					alive = true
					break
				}
			}

			// Also try ICMP-like check (UDP to unlikely port, looking for ICMP unreachable)
			if !alive {
				conn, err := net.DialTimeout("udp", fmt.Sprintf("%s:33434", ip), timeout)
				if err == nil {
					conn.Close()
					// UDP "connect" succeeds — doesn't mean alive, but we tried
				}
			}

			latency := float64(time.Since(start).Microseconds()) / 1000.0

			if alive {
				host := Host{
					IP:      ip,
					Latency: latency,
					Alive:   true,
				}

				mu.Lock()
				hosts = append(hosts, host)
				mu.Unlock()
			}
		}(ip)
	}

	wg.Wait()
	return hosts
}

// ResolveHostnames does reverse DNS lookups for discovered hosts.
func ResolveHostnames(hosts []Host) {
	var wg sync.WaitGroup
	for i := range hosts {
		wg.Add(1)
		go func(h *Host) {
			defer wg.Done()
			names, err := net.LookupAddr(h.IP)
			if err == nil && len(names) > 0 {
				h.Hostname = names[0]
				// Remove trailing dot
				if len(h.Hostname) > 0 && h.Hostname[len(h.Hostname)-1] == '.' {
					h.Hostname = h.Hostname[:len(h.Hostname)-1]
				}
			}
		}(&hosts[i])
	}
	wg.Wait()
}
