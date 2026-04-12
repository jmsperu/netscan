package scanner

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// GetARPTable reads the system ARP table and returns a map of IP -> MAC.
func GetARPTable() map[string]string {
	table := make(map[string]string)

	switch runtime.GOOS {
	case "linux":
		// Read /proc/net/arp
		f, err := os.Open("/proc/net/arp")
		if err != nil {
			return table
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Scan() // skip header
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) >= 4 {
				ip := fields[0]
				mac := fields[3]
				if mac != "00:00:00:00:00:00" {
					table[ip] = strings.ToUpper(mac)
				}
			}
		}

	case "darwin":
		// Run arp -a
		out, err := exec.Command("arp", "-a").Output()
		if err != nil {
			return table
		}
		for _, line := range strings.Split(string(out), "\n") {
			// format: host (ip) at mac on iface
			parts := strings.Fields(line)
			if len(parts) >= 4 && parts[1] != "" && parts[3] != "(incomplete)" {
				ip := strings.Trim(parts[1], "()")
				mac := strings.ToUpper(parts[3])
				if mac != "(INCOMPLETE)" && mac != "" {
					table[ip] = mac
				}
			}
		}

	case "windows":
		// Run arp -a
		out, err := exec.Command("arp", "-a").Output()
		if err != nil {
			return table
		}
		for _, line := range strings.Split(string(out), "\n") {
			parts := strings.Fields(strings.TrimSpace(line))
			if len(parts) >= 3 {
				ip := parts[0]
				mac := strings.ToUpper(strings.ReplaceAll(parts[1], "-", ":"))
				if isValidIP(ip) && isValidMAC(mac) {
					table[ip] = mac
				}
			}
		}
	}

	return table
}

func isValidIP(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 3 {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

func isValidMAC(s string) bool {
	parts := strings.Split(s, ":")
	return len(parts) == 6
}
