package cmd

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/jmsperu/netscan/internal/oui"
	"github.com/jmsperu/netscan/internal/portscan"
	"github.com/jmsperu/netscan/internal/scanner"
	"github.com/spf13/cobra"
)

//go:embed webui
var webFS embed.FS

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Launch a local web UI for network scanning",
	Long: `Runs a local HTTP server with an embedded web dashboard at http://localhost:8080

The web UI lets you trigger scans, view discovered hosts, and see open ports
in a clean terminal-style interface — no installation of a separate dashboard required.

Examples:
  netscan web                       # default :8080
  netscan web --addr :3000          # custom port
  netscan web --addr 0.0.0.0:8080   # expose to LAN`,
	RunE: runWeb,
}

func init() {
	webCmd.Flags().String("addr", ":8080", "Listen address")
	rootCmd.AddCommand(webCmd)
}

type scanRequest struct {
	Subnet string `json:"subnet"`
	Mode   string `json:"mode"`
}

type hostResult struct {
	IP       string `json:"ip"`
	MAC      string `json:"mac"`
	Vendor   string `json:"vendor"`
	Hostname string `json:"hostname"`
	Ports    []int  `json:"ports"`
}

type scanResponse struct {
	Hosts []hostResult `json:"hosts"`
	Error string       `json:"error,omitempty"`
}

type statusResponse struct {
	Subnet string `json:"subnet"`
	Iface  string `json:"interface"`
}

func runWeb(cmd *cobra.Command, args []string) error {
	addr, _ := cmd.Flags().GetString("addr")

	sub, _ := fs.Sub(webFS, "webui")

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(sub)))
	mux.Handle("/metrics", globalMetrics.Handler())

	mux.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		subnet, _ := scanner.GetLocalSubnet("")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(statusResponse{Subnet: subnet})
	})

	mux.HandleFunc("/api/scan", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req scanRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, scanResponse{Error: "invalid request: " + err.Error()})
			return
		}

		subnet := req.Subnet
		if subnet == "" {
			var err error
			subnet, err = scanner.GetLocalSubnet("")
			if err != nil {
				writeJSON(w, scanResponse{Error: "auto-detect subnet: " + err.Error()})
				return
			}
		}

		ips, err := scanner.ExpandSubnet(subnet)
		if err != nil {
			writeJSON(w, scanResponse{Error: err.Error()})
			return
		}

		globalMetrics.ScanStarted()
		hosts := scanner.PingSweep(ips, 500*time.Millisecond, 256)

		arpTable := scanner.GetARPTable()
		for i := range hosts {
			if mac, ok := arpTable[hosts[i].IP]; ok {
				hosts[i].MAC = mac
				hosts[i].Vendor = oui.Lookup(mac)
			}
		}
		scanner.ResolveHostnames(hosts)

		// Port scan if requested
		ports := []int{}
		if req.Mode == "full" {
			ports = []int{22, 80, 443, 8080, 3389, 5900, 8443, 3306, 5432, 6379, 27017, 161, 5060, 5080}
		} else if req.Mode == "all" {
			for p := 1; p <= 65535; p++ {
				ports = append(ports, p)
			}
		}

		results := make([]hostResult, 0, len(hosts))
		for i := range hosts {
			h := &hosts[i]
			if len(ports) > 0 {
				portscan.ScanPorts(h, ports, 500*time.Millisecond, 128)
			}
			hr := hostResult{
				IP:       h.IP,
				MAC:      h.MAC,
				Vendor:   h.Vendor,
				Hostname: h.Hostname,
			}
			for _, p := range h.Ports {
				hr.Ports = append(hr.Ports, p.Number)
				globalMetrics.AddOpenPort(h.IP, p.Number)
			}
			results = append(results, hr)
		}

		globalMetrics.SetHostsUp(len(results))
		globalMetrics.ScanFinished()

		writeJSON(w, scanResponse{Hosts: results})
	})

	fmt.Printf("netscan web UI: http://localhost%s\n", addr)
	fmt.Printf("Prometheus metrics: http://localhost%s/metrics\n", addr)
	return http.ListenAndServe(addr, mux)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
