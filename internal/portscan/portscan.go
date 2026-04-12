package portscan

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmsperu/netscan/internal/scanner"
)

// Common service names by port.
var commonServices = map[int]string{
	21: "ftp", 22: "ssh", 23: "telnet", 25: "smtp", 53: "dns",
	80: "http", 110: "pop3", 111: "rpcbind", 119: "nntp", 135: "msrpc",
	139: "netbios", 143: "imap", 161: "snmp", 389: "ldap", 443: "https",
	445: "smb", 465: "smtps", 514: "syslog", 587: "submission",
	636: "ldaps", 993: "imaps", 995: "pop3s", 1080: "socks",
	1433: "mssql", 1521: "oracle", 2049: "nfs", 2181: "zookeeper",
	3306: "mysql", 3389: "rdp", 5432: "postgres", 5672: "amqp",
	5900: "vnc", 5901: "vnc", 6379: "redis", 6443: "k8s-api",
	8080: "http-alt", 8443: "https-alt", 8888: "http-alt",
	9090: "prometheus", 9200: "elasticsearch", 9300: "elasticsearch",
	11211: "memcached", 27017: "mongodb", 50000: "sap",
}

// ParsePorts parses a port specification like "22,80,443" or "1-1024" or "22,80,100-200".
func ParsePorts(spec string) ([]int, error) {
	var ports []int
	seen := make(map[int]bool)

	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, "-") {
			rangeParts := strings.SplitN(part, "-", 2)
			start, err := strconv.Atoi(rangeParts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", rangeParts[0])
			}
			end, err := strconv.Atoi(rangeParts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", rangeParts[1])
			}
			if start > end || start < 1 || end > 65535 {
				return nil, fmt.Errorf("invalid port range: %d-%d", start, end)
			}
			for p := start; p <= end; p++ {
				if !seen[p] {
					ports = append(ports, p)
					seen[p] = true
				}
			}
		} else {
			p, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid port: %s", part)
			}
			if p < 1 || p > 65535 {
				return nil, fmt.Errorf("port out of range: %d", p)
			}
			if !seen[p] {
				ports = append(ports, p)
				seen[p] = true
			}
		}
	}

	return ports, nil
}

// ScanPorts scans the given ports on a host.
func ScanPorts(host *scanner.Host, ports []int, timeout time.Duration, concurrency int) {
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, port := range ports {
		wg.Add(1)
		sem <- struct{}{}
		go func(port int) {
			defer wg.Done()
			defer func() { <-sem }()

			addr := fmt.Sprintf("%s:%d", host.IP, port)
			conn, err := net.DialTimeout("tcp", addr, timeout)
			if err != nil {
				return
			}
			defer conn.Close()

			p := scanner.Port{
				Number: port,
				Proto:  "tcp",
				State:  "open",
			}

			// Service identification
			if svc, ok := commonServices[port]; ok {
				p.Service = svc
			}

			// Try to grab banner (quick read with short timeout)
			conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			reader := bufio.NewReader(conn)
			banner, err := reader.ReadString('\n')
			if err == nil {
				banner = strings.TrimSpace(banner)
				if len(banner) > 0 && len(banner) < 256 {
					p.Banner = banner
				}
			}

			mu.Lock()
			host.Ports = append(host.Ports, p)
			mu.Unlock()
		}(port)
	}

	wg.Wait()
}
