package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/jmsperu/netscan/internal/scanner"
)

// Print renders hosts in the specified format.
func Print(hosts []scanner.Host, format string) {
	// Sort by IP
	sort.Slice(hosts, func(i, j int) bool {
		return compareIPs(hosts[i].IP, hosts[j].IP)
	})

	switch format {
	case "json":
		printJSON(hosts)
	case "csv":
		printCSV(hosts)
	case "wide":
		printWide(hosts)
	default:
		printTable(hosts)
	}
}

func printTable(hosts []scanner.Host) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "IP\tMAC\tVENDOR\tHOSTNAME\tPORTS\tLATENCY\n")
	fmt.Fprintf(w, "--\t---\t------\t--------\t-----\t-------\n")

	for _, h := range hosts {
		mac := h.MAC
		if mac == "" {
			mac = "-"
		}
		vendor := h.Vendor
		if vendor == "" {
			vendor = "-"
		}
		hostname := h.Hostname
		if hostname == "" {
			hostname = "-"
		}

		var ports []string
		for _, p := range h.Ports {
			if p.Service != "" {
				ports = append(ports, fmt.Sprintf("%d/%s", p.Number, p.Service))
			} else {
				ports = append(ports, fmt.Sprintf("%d", p.Number))
			}
		}
		portStr := "-"
		if len(ports) > 0 {
			portStr = strings.Join(ports, ",")
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%.1fms\n",
			h.IP, mac, vendor, hostname, portStr, h.Latency)
	}
	w.Flush()

	fmt.Printf("\n%d hosts found\n", len(hosts))
}

func printWide(hosts []scanner.Host) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "IP\tMAC\tVENDOR\tHOSTNAME\tPORT\tSERVICE\tBANNER\tLATENCY\n")
	fmt.Fprintf(w, "--\t---\t------\t--------\t----\t-------\t------\t-------\n")

	for _, h := range hosts {
		mac := h.MAC
		if mac == "" {
			mac = "-"
		}
		vendor := h.Vendor
		if vendor == "" {
			vendor = "-"
		}
		hostname := h.Hostname
		if hostname == "" {
			hostname = "-"
		}

		if len(h.Ports) == 0 {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t-\t-\t-\t%.1fms\n",
				h.IP, mac, vendor, hostname, h.Latency)
		} else {
			for i, p := range h.Ports {
				if i == 0 {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\t%.1fms\n",
						h.IP, mac, vendor, hostname, p.Number, p.Service, p.Banner, h.Latency)
				} else {
					fmt.Fprintf(w, "\t\t\t\t%d\t%s\t%s\t\n",
						p.Number, p.Service, p.Banner)
				}
			}
		}
	}
	w.Flush()

	fmt.Printf("\n%d hosts found\n", len(hosts))
}

func printJSON(hosts []scanner.Host) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(hosts)
}

func printCSV(hosts []scanner.Host) {
	fmt.Println("ip,mac,vendor,hostname,ports,latency_ms")
	for _, h := range hosts {
		var ports []string
		for _, p := range h.Ports {
			ports = append(ports, fmt.Sprintf("%d", p.Number))
		}
		fmt.Printf("%s,%s,%s,%s,%s,%.1f\n",
			h.IP, h.MAC, h.Vendor, h.Hostname, strings.Join(ports, ";"), h.Latency)
	}
}

func compareIPs(a, b string) bool {
	pa := strings.Split(a, ".")
	pb := strings.Split(b, ".")
	if len(pa) != 4 || len(pb) != 4 {
		return a < b
	}
	for i := 0; i < 4; i++ {
		ai, bi := 0, 0
		fmt.Sscanf(pa[i], "%d", &ai)
		fmt.Sscanf(pb[i], "%d", &bi)
		if ai != bi {
			return ai < bi
		}
	}
	return false
}
