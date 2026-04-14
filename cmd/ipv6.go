package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var ipv6Cmd = &cobra.Command{
	Use:   "ipv6",
	Short: "Discover IPv6 neighbors on the local link",
	Long: `Show IPv6 neighbors from the system neighbor discovery cache.

Reads /proc/net/ipv6_neigh on Linux and runs 'ndp -an' on macOS/BSD.
For fresh discovery, first ping the all-nodes multicast address:

  ping6 -c 3 ff02::1%en0    (macOS/BSD)
  ping6 -c 3 ff02::1         (Linux, via the correct interface)

Examples:
  netscan ipv6                    # list all discovered neighbors
  netscan ipv6 --reachable        # show only reachable neighbors
  netscan ipv6 --refresh          # send multicast ping first to populate cache`,
	RunE: runIPv6,
}

func init() {
	ipv6Cmd.Flags().Bool("reachable", false, "Only show reachable neighbors")
	ipv6Cmd.Flags().Bool("refresh", false, "Send IPv6 multicast ping first to refresh neighbor table")
	ipv6Cmd.Flags().StringP("interface", "i", "", "Network interface (e.g. en0, eth0)")
	rootCmd.AddCommand(ipv6Cmd)
}

type ipv6Neighbor struct {
	Address   string
	MAC       string
	Interface string
	State     string
}

func runIPv6(cmd *cobra.Command, args []string) error {
	refresh, _ := cmd.Flags().GetBool("refresh")
	iface, _ := cmd.Flags().GetString("interface")
	reachableOnly, _ := cmd.Flags().GetBool("reachable")

	if refresh {
		refreshIPv6Neighbors(iface)
	}

	var neighbors []ipv6Neighbor
	var err error

	switch runtime.GOOS {
	case "linux":
		neighbors, err = readLinuxIPv6Neighbors(iface)
	case "darwin", "freebsd", "openbsd", "netbsd":
		neighbors, err = readBSDIPv6Neighbors(iface)
	default:
		return fmt.Errorf("IPv6 neighbor discovery not supported on %s", runtime.GOOS)
	}

	if err != nil {
		return err
	}

	if len(neighbors) == 0 {
		fmt.Fprintln(os.Stderr, "No IPv6 neighbors found.")
		fmt.Fprintln(os.Stderr, "Try: netscan ipv6 --refresh --interface en0")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ADDRESS\tMAC\tINTERFACE\tSTATE")
	fmt.Fprintln(w, "-------\t---\t---------\t-----")
	for _, n := range neighbors {
		if reachableOnly && !strings.EqualFold(n.State, "reachable") && !strings.EqualFold(n.State, "R") {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", n.Address, n.MAC, n.Interface, n.State)
	}
	w.Flush()
	return nil
}

func refreshIPv6Neighbors(iface string) {
	var cmd *exec.Cmd
	target := "ff02::1"
	if iface != "" {
		target = fmt.Sprintf("ff02::1%%%s", iface)
	}

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("ping6", "-c", "3", "-i", "0.3", target)
	case "linux":
		if iface != "" {
			cmd = exec.Command("ping", "-6", "-c", "3", "-I", iface, "ff02::1")
		} else {
			cmd = exec.Command("ping", "-6", "-c", "3", "ff02::1")
		}
	default:
		return
	}
	_ = cmd.Run()
}

func readLinuxIPv6Neighbors(iface string) ([]ipv6Neighbor, error) {
	// Use `ip -6 neigh show` for better formatting
	args := []string{"-6", "neigh", "show"}
	if iface != "" {
		args = append(args, "dev", iface)
	}
	out, err := exec.Command("ip", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("run ip -6 neigh: %w", err)
	}

	var results []ipv6Neighbor
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		n := ipv6Neighbor{}
		for i := 0; i < len(fields); i++ {
			switch fields[i] {
			case "dev":
				if i+1 < len(fields) {
					n.Interface = fields[i+1]
					i++
				}
			case "lladdr":
				if i+1 < len(fields) {
					n.MAC = fields[i+1]
					i++
				}
			default:
				if n.Address == "" && strings.Contains(fields[i], ":") {
					n.Address = fields[i]
				} else if i == len(fields)-1 {
					n.State = fields[i]
				}
			}
		}
		if n.Address != "" {
			results = append(results, n)
		}
	}
	return results, nil
}

func readBSDIPv6Neighbors(iface string) ([]ipv6Neighbor, error) {
	args := []string{"-an"}
	if iface != "" {
		args = append(args, "-i", iface)
	}
	out, err := exec.Command("ndp", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("run ndp -an: %w", err)
	}

	var results []ipv6Neighbor
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	// Skip header
	_ = scanner.Scan()
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		// ndp output: Neighbor Linklayer Address Netif Expire St Flgs Prbs
		if len(fields) < 5 {
			continue
		}
		n := ipv6Neighbor{
			Address:   fields[0],
			MAC:       fields[1],
			Interface: fields[2],
		}
		if len(fields) >= 5 {
			n.State = fields[4]
		}
		// Translate ndp state codes
		switch n.State {
		case "R":
			n.State = "reachable"
		case "S":
			n.State = "stale"
		case "D":
			n.State = "delay"
		case "P":
			n.State = "probe"
		case "N":
			n.State = "nostate"
		case "W":
			n.State = "waiting"
		case "I":
			n.State = "incomplete"
		}
		results = append(results, n)
	}
	return results, nil
}
