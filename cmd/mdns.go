package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/grandcat/zeroconf"
	"github.com/spf13/cobra"
)

// Common mDNS service types to browse.
var defaultMDNSServices = []string{
	"_airplay._tcp",         // AirPlay (Apple TV, speakers)
	"_raop._tcp",            // AirTunes (AirPlay audio)
	"_googlecast._tcp",      // Chromecast, Google Home, Nest
	"_spotify-connect._tcp", // Spotify Connect
	"_hap._tcp",             // HomeKit
	"_homekit._tcp",         // HomeKit alt
	"_printer._tcp",         // Network printers
	"_ipp._tcp",             // IPP printers
	"_ipps._tcp",            // IPP secure
	"_pdl-datastream._tcp",  // HP JetDirect
	"_smb._tcp",             // SMB/CIFS shares
	"_afpovertcp._tcp",      // Apple Filing Protocol
	"_ssh._tcp",             // SSH servers advertising via mDNS
	"_http._tcp",            // HTTP services
	"_https._tcp",           // HTTPS services
	"_workstation._tcp",     // Workstations
	"_device-info._tcp",     // Apple device info
	"_sleep-proxy._udp",     // Apple sleep proxy
	"_companion-link._tcp",  // Apple Companion
	"_rdlink._tcp",          // Apple Remote Desktop
}

var mdnsCmd = &cobra.Command{
	Use:   "mdns",
	Short: "Discover Bonjour/mDNS services on the local network",
	Long: `Browse local mDNS/Bonjour services to find devices like AirPlay speakers,
Chromecasts, HomeKit accessories, network printers, SMB shares, and more.

Examples:
  netscan mdns                           # browse common service types
  netscan mdns --timeout 10              # longer discovery window
  netscan mdns --service _googlecast._tcp  # browse a specific service
  netscan mdns --all                     # also include uncommon service types`,
	RunE: runMDNS,
}

func init() {
	mdnsCmd.Flags().IntP("timeout", "t", 5, "Discovery timeout in seconds")
	mdnsCmd.Flags().StringSlice("service", nil, "Specific mDNS service type(s) to browse (e.g. _googlecast._tcp)")
	mdnsCmd.Flags().Bool("all", false, "Browse all known service types (slower but thorough)")
	mdnsCmd.Flags().StringP("output", "o", "table", "Output format: table, json")
	rootCmd.AddCommand(mdnsCmd)
}

type mdnsEntry struct {
	Service  string   `json:"service"`
	Instance string   `json:"instance"`
	Host     string   `json:"host"`
	IPv4     []string `json:"ipv4,omitempty"`
	IPv6     []string `json:"ipv6,omitempty"`
	Port     int      `json:"port"`
	Text     []string `json:"text,omitempty"`
}

func runMDNS(cmd *cobra.Command, args []string) error {
	timeout, _ := cmd.Flags().GetInt("timeout")
	services, _ := cmd.Flags().GetStringSlice("service")

	if len(services) == 0 {
		services = defaultMDNSServices
	}

	fmt.Fprintf(os.Stderr, "Browsing %d mDNS service types for %ds...\n", len(services), timeout)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	entriesChan := make(chan mdnsEntry, 128)
	done := make(chan struct{})

	go func() {
		for _, svc := range services {
			resolver, err := zeroconf.NewResolver(nil)
			if err != nil {
				continue
			}

			serviceEntries := make(chan *zeroconf.ServiceEntry, 16)
			svcCopy := svc

			go func() {
				for entry := range serviceEntries {
					e := mdnsEntry{
						Service:  svcCopy,
						Instance: entry.Instance,
						Host:     entry.HostName,
						Port:     entry.Port,
						Text:     entry.Text,
					}
					for _, ip := range entry.AddrIPv4 {
						e.IPv4 = append(e.IPv4, ip.String())
					}
					for _, ip := range entry.AddrIPv6 {
						e.IPv6 = append(e.IPv6, ip.String())
					}
					entriesChan <- e
				}
			}()

			browseCtx, browseCancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
			_ = resolver.Browse(browseCtx, svc, "local.", serviceEntries)
			<-browseCtx.Done()
			browseCancel()
		}
		close(done)
	}()

	// Collect entries until done
	var entries []mdnsEntry
collectLoop:
	for {
		select {
		case e := <-entriesChan:
			entries = append(entries, e)
		case <-done:
			break collectLoop
		case <-ctx.Done():
			break collectLoop
		}
	}

	// Drain remaining entries
	for {
		select {
		case e := <-entriesChan:
			entries = append(entries, e)
		default:
			goto print
		}
	}

print:
	if len(entries) == 0 {
		fmt.Fprintln(os.Stderr, "No mDNS services found.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SERVICE\tINSTANCE\tHOST\tIP\tPORT")
	fmt.Fprintln(w, "-------\t--------\t----\t--\t----")
	for _, e := range entries {
		ip := ""
		if len(e.IPv4) > 0 {
			ip = e.IPv4[0]
		} else if len(e.IPv6) > 0 {
			ip = e.IPv6[0]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\n", e.Service, e.Instance, e.Host, ip, e.Port)
	}
	w.Flush()
	return nil
}
