package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	appVersion string
)

func SetVersion(v string) {
	appVersion = v
}

var rootCmd = &cobra.Command{
	Use:   "netscan",
	Short: "Fast network scanner — discover hosts, ports, and services",
	Long: `netscan - Fast cross-platform network scanner

Discover devices on your LAN, scan ports, identify services,
and resolve MAC vendor information. Single binary, no dependencies.

Examples:
  netscan                          # scan current subnet
  netscan 192.168.1.0/24           # scan specific subnet
  netscan -p 22,80,443             # scan specific ports
  netscan -p 1-1024                # scan port range
  netscan --fast                   # ping sweep only (no ports)
  netscan --all                    # scan all 65535 ports
  netscan -o json                  # JSON output
  netscan -o csv                   # CSV output
  netscan watch                    # continuous monitoring
  netscan ports <host>             # deep port scan a single host`,
	Version: appVersion,
	RunE:    runScan,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringP("ports", "p", "22,80,443,8080,3389,5900,8443,3306,5432,6379,27017", "Ports to scan (comma-separated or range)")
	rootCmd.Flags().BoolP("fast", "f", false, "Fast mode — ping sweep only, no port scan")
	rootCmd.Flags().Bool("all", false, "Scan all 65535 ports")
	rootCmd.Flags().StringP("output", "o", "table", "Output format: table, json, csv, wide")
	rootCmd.Flags().IntP("timeout", "t", 500, "Connection timeout in milliseconds")
	rootCmd.Flags().IntP("concurrency", "c", 256, "Max concurrent connections")
	rootCmd.Flags().BoolP("verbose", "v", false, "Show detailed output")
	rootCmd.Flags().Bool("no-resolve", false, "Skip hostname resolution")
	rootCmd.Flags().Bool("no-vendor", false, "Skip MAC vendor lookup")
	rootCmd.Flags().StringP("interface", "i", "", "Network interface to use")

	rootCmd.AddCommand(portsCmd)
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(wakeCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	// If a subnet is provided, use it; otherwise auto-detect
	var subnet string
	if len(args) > 0 {
		subnet = args[0]
	}

	fast, _ := cmd.Flags().GetBool("fast")
	allPorts, _ := cmd.Flags().GetBool("all")
	portStr, _ := cmd.Flags().GetString("ports")
	outputFmt, _ := cmd.Flags().GetString("output")
	timeout, _ := cmd.Flags().GetInt("timeout")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	noResolve, _ := cmd.Flags().GetBool("no-resolve")
	noVendor, _ := cmd.Flags().GetBool("no-vendor")
	iface, _ := cmd.Flags().GetString("interface")

	if allPorts {
		portStr = "1-65535"
	}

	_ = fast
	_ = portStr
	_ = outputFmt
	_ = timeout
	_ = concurrency
	_ = noResolve
	_ = noVendor
	_ = iface
	_ = subnet

	// Import and run scanner
	return runFullScan(subnet, portStr, fast, outputFmt, timeout, concurrency, noResolve, noVendor, iface)
}

func runFullScan(subnet, portStr string, fast bool, outputFmt string, timeout, concurrency int, noResolve, noVendor bool, iface string) error {
	fmt.Println("Scanning...")

	// This is implemented in scan.go
	return doScan(subnet, portStr, fast, outputFmt, timeout, concurrency, noResolve, noVendor, iface)
}
