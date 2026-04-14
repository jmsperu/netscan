package cmd

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

// metricsServer holds the in-memory gauges we expose.
type metricsServer struct {
	mu        sync.RWMutex
	hostsUp   int
	ports     map[string]int    // "ip:port" -> 1 if open
	scanStart time.Time
	scanEnd   time.Time
	scanCount int
}

var globalMetrics = &metricsServer{ports: map[string]int{}}

func (m *metricsServer) SetHostsUp(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hostsUp = n
}

func (m *metricsServer) AddOpenPort(ip string, port int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ports[fmt.Sprintf("%s:%d", ip, port)] = 1
}

func (m *metricsServer) ScanStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanStart = time.Now()
	m.ports = map[string]int{}
}

func (m *metricsServer) ScanFinished() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.scanEnd = time.Now()
	m.scanCount++
}

func (m *metricsServer) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		m.mu.RLock()
		defer m.mu.RUnlock()
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")

		fmt.Fprintln(w, "# HELP netscan_hosts_up Number of hosts that responded to the last scan")
		fmt.Fprintln(w, "# TYPE netscan_hosts_up gauge")
		fmt.Fprintf(w, "netscan_hosts_up %d\n", m.hostsUp)

		fmt.Fprintln(w, "# HELP netscan_scan_count Total number of scans completed")
		fmt.Fprintln(w, "# TYPE netscan_scan_count counter")
		fmt.Fprintf(w, "netscan_scan_count %d\n", m.scanCount)

		if !m.scanEnd.IsZero() {
			durMs := m.scanEnd.Sub(m.scanStart).Milliseconds()
			fmt.Fprintln(w, "# HELP netscan_scan_duration_ms Duration of the last scan in milliseconds")
			fmt.Fprintln(w, "# TYPE netscan_scan_duration_ms gauge")
			fmt.Fprintf(w, "netscan_scan_duration_ms %d\n", durMs)

			fmt.Fprintln(w, "# HELP netscan_last_scan_unixtime Timestamp of the last scan completion")
			fmt.Fprintln(w, "# TYPE netscan_last_scan_unixtime gauge")
			fmt.Fprintf(w, "netscan_last_scan_unixtime %d\n", m.scanEnd.Unix())
		}

		fmt.Fprintln(w, "# HELP netscan_port_open Whether a port was found open on a host (1=open)")
		fmt.Fprintln(w, "# TYPE netscan_port_open gauge")
		for hostPort := range m.ports {
			var ip string
			var port int
			fmt.Sscanf(hostPort, "%[^:]:%d", &ip, &port)
			fmt.Fprintf(w, "netscan_port_open{ip=%q,port=\"%d\"} 1\n", ip, port)
		}
	})
}

// startMetricsServer starts the Prometheus metrics endpoint on the given address.
// Returns immediately; the server runs in a goroutine.
func startMetricsServer(addr string) error {
	if addr == "" {
		return nil
	}
	mux := http.NewServeMux()
	mux.Handle("/metrics", globalMetrics.Handler())
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
<head><title>netscan metrics</title></head>
<body>
<h1>netscan Prometheus Exporter</h1>
<p><a href="/metrics">/metrics</a> — scrape this endpoint from Prometheus</p>
</body>
</html>`))
	})
	srv := &http.Server{Addr: addr, Handler: mux}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("metrics server error: %v\n", err)
		}
	}()
	return nil
}

// metricsCmd exposes the /metrics endpoint standalone (useful in watch mode).
var metricsCmd = &cobra.Command{
	Use:   "metrics",
	Short: "Run a standalone Prometheus metrics exporter (for use with watch mode)",
	Long: `Exposes netscan scan results as Prometheus metrics at /metrics.

Typically used in combination with watch mode. Run one netscan watch with
--prometheus-port, and Prometheus scrapes http://this-host:9100/metrics.

Example:
  netscan watch 10.0.0.0/24 --prometheus-port 9100`,
	RunE: func(cmd *cobra.Command, args []string) error {
		addr, _ := cmd.Flags().GetString("addr")
		fmt.Printf("Listening on %s/metrics\n", addr)
		http.Handle("/metrics", globalMetrics.Handler())
		return http.ListenAndServe(addr, nil)
	},
}

func init() {
	metricsCmd.Flags().String("addr", ":9100", "Listen address for metrics server")
	rootCmd.AddCommand(metricsCmd)
}
