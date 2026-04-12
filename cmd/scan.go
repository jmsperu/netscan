package cmd

import (
	"fmt"
	"time"

	"github.com/jmsperu/netscan/internal/oui"
	"github.com/jmsperu/netscan/internal/output"
	"github.com/jmsperu/netscan/internal/portscan"
	"github.com/jmsperu/netscan/internal/scanner"
)

func doScan(subnet, portStr string, fast bool, outputFmt string, timeout, concurrency int, noResolve, noVendor bool, iface string) error {
	// Auto-detect subnet if not provided
	if subnet == "" {
		var err error
		subnet, err = scanner.GetLocalSubnet(iface)
		if err != nil {
			return fmt.Errorf("auto-detecting subnet: %w\nSpecify a subnet manually: netscan 192.168.1.0/24", err)
		}
	}

	fmt.Printf("Scanning %s", subnet)
	if !fast {
		fmt.Printf(" (ports: %s)", portStr)
	}
	fmt.Println()
	fmt.Println()

	start := time.Now()

	// Expand subnet to IPs
	ips, err := scanner.ExpandSubnet(subnet)
	if err != nil {
		return err
	}

	timeoutDur := time.Duration(timeout) * time.Millisecond

	// Phase 1: Ping sweep
	fmt.Printf("Discovering hosts (%d IPs)... ", len(ips))
	hosts := scanner.PingSweep(ips, timeoutDur, concurrency)
	fmt.Printf("%d alive\n", len(hosts))

	if len(hosts) == 0 {
		fmt.Println("No hosts found.")
		return nil
	}

	// Phase 2: ARP table for MAC addresses
	arpTable := scanner.GetARPTable()
	for i := range hosts {
		if mac, ok := arpTable[hosts[i].IP]; ok {
			hosts[i].MAC = mac
		}
	}

	// Phase 3: MAC vendor lookup
	if !noVendor {
		for i := range hosts {
			if hosts[i].MAC != "" {
				hosts[i].Vendor = oui.Lookup(hosts[i].MAC)
			}
		}
	}

	// Phase 4: Hostname resolution
	if !noResolve {
		fmt.Print("Resolving hostnames... ")
		scanner.ResolveHostnames(hosts)
		fmt.Println("done")
	}

	// Phase 5: Port scan
	if !fast {
		ports, err := portscan.ParsePorts(portStr)
		if err != nil {
			return fmt.Errorf("parsing ports: %w", err)
		}

		fmt.Printf("Scanning %d ports on %d hosts... ", len(ports), len(hosts))
		perHostConcurrency := concurrency / len(hosts)
		if perHostConcurrency < 10 {
			perHostConcurrency = 10
		}

		for i := range hosts {
			portscan.ScanPorts(&hosts[i], ports, timeoutDur, perHostConcurrency)
		}
		fmt.Println("done")
	}

	fmt.Println()

	// Output
	output.Print(hosts, outputFmt)

	elapsed := time.Since(start)
	fmt.Printf("\nScan completed in %s\n", elapsed.Round(time.Millisecond))

	return nil
}
