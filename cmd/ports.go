package cmd

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"
	"time"

	"github.com/jmsperu/netscan/internal/portscan"
	"github.com/jmsperu/netscan/internal/scanner"
	"github.com/spf13/cobra"
)

var portsCmd = &cobra.Command{
	Use:   "ports <host>",
	Short: "Deep port scan a single host",
	Long: `Scan all ports (or specified range) on a single host.

Examples:
  netscan ports 192.168.1.1              # common ports
  netscan ports 192.168.1.1 -p 1-65535   # all ports
  netscan ports 192.168.1.1 -p 1-1024    # well-known ports
  netscan ports 192.168.1.1 --all        # all 65535 ports`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		host := args[0]
		portStr, _ := cmd.Flags().GetString("ports")
		allPorts, _ := cmd.Flags().GetBool("all")
		timeout, _ := cmd.Flags().GetInt("timeout")
		concurrency, _ := cmd.Flags().GetInt("concurrency")

		if allPorts {
			portStr = "1-65535"
		}

		ports, err := portscan.ParsePorts(portStr)
		if err != nil {
			return err
		}

		timeoutDur := time.Duration(timeout) * time.Millisecond

		fmt.Printf("Scanning %d ports on %s...\n", len(ports), host)
		start := time.Now()

		h := &scanner.Host{IP: host, Alive: true}
		portscan.ScanPorts(h, ports, timeoutDur, concurrency)

		// Sort ports
		sort.Slice(h.Ports, func(i, j int) bool {
			return h.Ports[i].Number < h.Ports[j].Number
		})

		fmt.Println()

		if len(h.Ports) == 0 {
			fmt.Println("No open ports found.")
		} else {
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintf(w, "PORT\tSTATE\tSERVICE\tBANNER\n")
			fmt.Fprintf(w, "----\t-----\t-------\t------\n")
			for _, p := range h.Ports {
				banner := p.Banner
				if len(banner) > 60 {
					banner = banner[:60] + "..."
				}
				fmt.Fprintf(w, "%d/%s\t%s\t%s\t%s\n",
					p.Number, p.Proto, p.State, p.Service, banner)
			}
			w.Flush()
			fmt.Printf("\n%d open ports\n", len(h.Ports))
		}

		elapsed := time.Since(start)
		fmt.Printf("Scan completed in %s\n", elapsed.Round(time.Millisecond))

		return nil
	},
}

func init() {
	portsCmd.Flags().StringP("ports", "p", "1-1024", "Ports to scan")
	portsCmd.Flags().Bool("all", false, "Scan all 65535 ports")
	portsCmd.Flags().IntP("timeout", "t", 500, "Connection timeout in ms")
	portsCmd.Flags().IntP("concurrency", "c", 1000, "Max concurrent connections")
}
