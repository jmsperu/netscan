package cmd

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jmsperu/netscan/internal/oui"
	"github.com/jmsperu/netscan/internal/output"
	"github.com/jmsperu/netscan/internal/scanner"
	"github.com/spf13/cobra"
)

// suppress unused import
var _ = output.Print

var watchCmd = &cobra.Command{
	Use:   "watch [subnet]",
	Short: "Continuously monitor the network for changes",
	Long: `Watch the network and report when devices appear or disappear.

Examples:
  netscan watch                    # watch current subnet
  netscan watch 192.168.1.0/24     # watch specific subnet
  netscan watch -n 30              # scan every 30 seconds`,
	RunE: func(cmd *cobra.Command, args []string) error {
		interval, _ := cmd.Flags().GetInt("interval")
		iface, _ := cmd.Flags().GetString("interface")
		promPort, _ := cmd.Flags().GetString("prometheus-port")

		var subnet string
		if len(args) > 0 {
			subnet = args[0]
		} else {
			var err error
			subnet, err = scanner.GetLocalSubnet(iface)
			if err != nil {
				return fmt.Errorf("auto-detecting subnet: %w", err)
			}
		}

		if promPort != "" {
			if err := startMetricsServer(promPort); err != nil {
				return fmt.Errorf("metrics server: %w", err)
			}
			fmt.Printf("Prometheus metrics on http://localhost%s/metrics\n", promPort)
		}

		fmt.Printf("Watching %s (every %ds, Ctrl+C to stop)\n\n", subnet, interval)

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		ips, err := scanner.ExpandSubnet(subnet)
		if err != nil {
			return err
		}

		known := make(map[string]bool)
		timeout := 500 * time.Millisecond

		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		scan := func() {
			globalMetrics.ScanStarted()
			hosts := scanner.PingSweep(ips, timeout, 256)

			arpTable := scanner.GetARPTable()
			for i := range hosts {
				if mac, ok := arpTable[hosts[i].IP]; ok {
					hosts[i].MAC = mac
					hosts[i].Vendor = oui.Lookup(mac)
				}
			}

			scanner.ResolveHostnames(hosts)

			globalMetrics.SetHostsUp(len(hosts))
			globalMetrics.ScanFinished()

			current := make(map[string]bool)
			for _, h := range hosts {
				current[h.IP] = true
				if !known[h.IP] {
					vendor := h.Vendor
					if vendor == "" {
						vendor = "unknown"
					}
					hostname := h.Hostname
					if hostname != "" {
						hostname = " (" + hostname + ")"
					}
					fmt.Printf("[+] NEW  %s  %s  %s%s\n",
						h.IP, h.MAC, vendor, hostname)
				}
			}

			for ip := range known {
				if !current[ip] {
					fmt.Printf("[-] GONE %s\n", ip)
				}
			}

			known = current
			fmt.Printf("--- %s: %d hosts ---\n", time.Now().Format("15:04:05"), len(hosts))
		}

		scan()

		for {
			select {
			case <-ticker.C:
				scan()
			case <-sigCh:
				fmt.Println("\nStopped.")
				return nil
			}
		}
	},
}

var wakeCmd = &cobra.Command{
	Use:   "wake <mac-address>",
	Short: "Send Wake-on-LAN magic packet",
	Long: `Send a WOL magic packet to wake a device.

Examples:
  netscan wake AA:BB:CC:DD:EE:FF
  netscan wake aa:bb:cc:dd:ee:ff -b 192.168.1.255`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		macStr := args[0]
		broadcast, _ := cmd.Flags().GetString("broadcast")

		// Parse MAC address
		var mac [6]byte
		n, err := fmt.Sscanf(macStr, "%02x:%02x:%02x:%02x:%02x:%02x",
			&mac[0], &mac[1], &mac[2], &mac[3], &mac[4], &mac[5])
		if err != nil || n != 6 {
			n, err = fmt.Sscanf(macStr, "%02x-%02x-%02x-%02x-%02x-%02x",
				&mac[0], &mac[1], &mac[2], &mac[3], &mac[4], &mac[5])
			if err != nil || n != 6 {
				return fmt.Errorf("invalid MAC address: %s", macStr)
			}
		}

		// Build magic packet: 6 bytes of 0xFF + 16 repetitions of MAC
		var packet [102]byte
		for i := 0; i < 6; i++ {
			packet[i] = 0xFF
		}
		for i := 0; i < 16; i++ {
			copy(packet[6+i*6:], mac[:])
		}

		if broadcast == "" {
			broadcast = "255.255.255.255"
		}

		target := fmt.Sprintf("%s:%d", broadcast, 9)
		conn, err := net.Dial("udp", target)
		if err != nil {
			return fmt.Errorf("dial: %w", err)
		}
		defer conn.Close()

		_, err = conn.Write(packet[:])
		if err != nil {
			return fmt.Errorf("send: %w", err)
		}

		fmt.Printf("Magic packet sent to %s via %s:9\n", macStr, broadcast)
		return nil
	},
}

func init() {
	watchCmd.Flags().IntP("interval", "n", 60, "Scan interval in seconds")
	watchCmd.Flags().StringP("interface", "i", "", "Network interface")
	watchCmd.Flags().String("prometheus-port", "", "Expose Prometheus metrics on this address (e.g. :9100)")

	wakeCmd.Flags().StringP("broadcast", "b", "255.255.255.255", "Broadcast address")
}
