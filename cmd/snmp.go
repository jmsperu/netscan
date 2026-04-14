package cmd

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	g "github.com/gosnmp/gosnmp"
	"github.com/spf13/cobra"
)

// Common OIDs for quick device overview.
var commonOIDs = map[string]string{
	"sysDescr":      "1.3.6.1.2.1.1.1.0",
	"sysObjectID":   "1.3.6.1.2.1.1.2.0",
	"sysUpTime":     "1.3.6.1.2.1.1.3.0",
	"sysContact":    "1.3.6.1.2.1.1.4.0",
	"sysName":       "1.3.6.1.2.1.1.5.0",
	"sysLocation":   "1.3.6.1.2.1.1.6.0",
	"sysServices":   "1.3.6.1.2.1.1.7.0",
	"ifNumber":      "1.3.6.1.2.1.2.1.0",
}

// ifTable columns for interface listing.
const (
	oidIfDescr  = "1.3.6.1.2.1.2.2.1.2"
	oidIfType   = "1.3.6.1.2.1.2.2.1.3"
	oidIfSpeed  = "1.3.6.1.2.1.2.2.1.5"
	oidIfOperStatus = "1.3.6.1.2.1.2.2.1.8"
	oidIfInOctets   = "1.3.6.1.2.1.2.2.1.10"
	oidIfOutOctets  = "1.3.6.1.2.1.2.2.1.16"
)

var snmpCmd = &cobra.Command{
	Use:   "snmp [host]",
	Short: "Query SNMP-enabled network devices (switches, routers, printers)",
	Long: `Query SNMP v1/v2c/v3 devices for system info and interface stats.

Examples:
  netscan snmp 192.168.1.1                         # quick summary (v2c, "public")
  netscan snmp 10.0.0.1 -c private                 # custom community
  netscan snmp 192.168.1.1 --interfaces            # list interfaces
  netscan snmp 192.168.1.1 --walk 1.3.6.1.2.1.1    # walk a specific OID
  netscan snmp 192.168.1.1 --oid 1.3.6.1.2.1.1.5.0 # get a single OID
  netscan snmp 192.168.1.1 -v 3 -u admin --authkey PASS --authproto SHA --privkey PRIV --privproto AES`,
	Args: cobra.ExactArgs(1),
	RunE: runSNMP,
}

func init() {
	snmpCmd.Flags().StringP("community", "c", "public", "SNMP community string (v1/v2c)")
	snmpCmd.Flags().IntP("version", "v", 2, "SNMP version: 1, 2 (v2c), or 3")
	snmpCmd.Flags().IntP("port", "p", 161, "SNMP port")
	snmpCmd.Flags().Int("timeout", 2, "Timeout in seconds")
	snmpCmd.Flags().StringP("oid", "O", "", "Get a single OID")
	snmpCmd.Flags().String("walk", "", "Walk a subtree starting at this OID")
	snmpCmd.Flags().Bool("interfaces", false, "List network interfaces (ifTable)")
	// v3
	snmpCmd.Flags().StringP("user", "u", "", "SNMPv3 username")
	snmpCmd.Flags().String("authkey", "", "SNMPv3 auth passphrase")
	snmpCmd.Flags().String("authproto", "SHA", "SNMPv3 auth protocol: MD5, SHA, SHA256, SHA512")
	snmpCmd.Flags().String("privkey", "", "SNMPv3 privacy passphrase")
	snmpCmd.Flags().String("privproto", "AES", "SNMPv3 privacy protocol: DES, AES, AES192, AES256")

	rootCmd.AddCommand(snmpCmd)
}

func buildSNMPClient(cmd *cobra.Command, host string) (*g.GoSNMP, error) {
	community, _ := cmd.Flags().GetString("community")
	version, _ := cmd.Flags().GetInt("version")
	port, _ := cmd.Flags().GetInt("port")
	timeout, _ := cmd.Flags().GetInt("timeout")

	client := &g.GoSNMP{
		Target:    host,
		Port:      uint16(port),
		Community: community,
		Timeout:   time.Duration(timeout) * time.Second,
		Retries:   1,
	}

	switch version {
	case 1:
		client.Version = g.Version1
	case 2:
		client.Version = g.Version2c
	case 3:
		client.Version = g.Version3
		user, _ := cmd.Flags().GetString("user")
		if user == "" {
			return nil, fmt.Errorf("SNMPv3 requires --user")
		}
		authKey, _ := cmd.Flags().GetString("authkey")
		authProto, _ := cmd.Flags().GetString("authproto")
		privKey, _ := cmd.Flags().GetString("privkey")
		privProto, _ := cmd.Flags().GetString("privproto")

		msgFlags := g.NoAuthNoPriv
		if authKey != "" && privKey != "" {
			msgFlags = g.AuthPriv
		} else if authKey != "" {
			msgFlags = g.AuthNoPriv
		}

		secParams := &g.UsmSecurityParameters{
			UserName:                 user,
			AuthenticationPassphrase: authKey,
			PrivacyPassphrase:        privKey,
			AuthenticationProtocol:   snmpAuthProto(authProto),
			PrivacyProtocol:          snmpPrivProto(privProto),
		}
		client.MsgFlags = msgFlags
		client.SecurityModel = g.UserSecurityModel
		client.SecurityParameters = secParams
	default:
		return nil, fmt.Errorf("unsupported SNMP version: %d", version)
	}

	return client, nil
}

func snmpAuthProto(s string) g.SnmpV3AuthProtocol {
	switch strings.ToUpper(s) {
	case "MD5":
		return g.MD5
	case "SHA":
		return g.SHA
	case "SHA256":
		return g.SHA256
	case "SHA512":
		return g.SHA512
	default:
		return g.NoAuth
	}
}

func snmpPrivProto(s string) g.SnmpV3PrivProtocol {
	switch strings.ToUpper(s) {
	case "DES":
		return g.DES
	case "AES":
		return g.AES
	case "AES192":
		return g.AES192
	case "AES256":
		return g.AES256
	default:
		return g.NoPriv
	}
}

func runSNMP(cmd *cobra.Command, args []string) error {
	host := args[0]

	client, err := buildSNMPClient(cmd, host)
	if err != nil {
		return err
	}

	if err := client.Connect(); err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer client.Conn.Close()

	singleOID, _ := cmd.Flags().GetString("oid")
	walkOID, _ := cmd.Flags().GetString("walk")
	interfaces, _ := cmd.Flags().GetBool("interfaces")

	// Single OID
	if singleOID != "" {
		return snmpGet(client, singleOID)
	}
	// Walk
	if walkOID != "" {
		return snmpWalk(client, walkOID)
	}
	// Interfaces
	if interfaces {
		return snmpInterfaces(client)
	}
	// Default: summary
	return snmpSummary(client, host)
}

func snmpGet(client *g.GoSNMP, oid string) error {
	result, err := client.Get([]string{oid})
	if err != nil {
		return err
	}
	for _, v := range result.Variables {
		fmt.Printf("%s = %s\n", v.Name, snmpValueToString(v))
	}
	return nil
}

func snmpWalk(client *g.GoSNMP, oid string) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "OID\tVALUE")
	fmt.Fprintln(w, "---\t-----")
	err := client.BulkWalk(oid, func(pdu g.SnmpPDU) error {
		fmt.Fprintf(w, "%s\t%s\n", pdu.Name, snmpValueToString(pdu))
		return nil
	})
	w.Flush()
	return err
}

func snmpSummary(client *g.GoSNMP, host string) error {
	oids := make([]string, 0, len(commonOIDs))
	names := make([]string, 0, len(commonOIDs))
	for name, oid := range commonOIDs {
		names = append(names, name)
		oids = append(oids, oid)
	}

	result, err := client.Get(oids)
	if err != nil {
		return err
	}

	fmt.Printf("SNMP Summary for %s\n", host)
	fmt.Println(strings.Repeat("=", 60))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for i, v := range result.Variables {
		if v.Type == g.NoSuchObject || v.Type == g.NoSuchInstance {
			continue
		}
		fmt.Fprintf(w, "%s\t%s\n", names[i], snmpValueToString(v))
	}
	w.Flush()
	return nil
}

func snmpInterfaces(client *g.GoSNMP) error {
	descrs := map[string]string{}
	statuses := map[string]string{}
	speeds := map[string]string{}

	err := client.BulkWalk(oidIfDescr, func(pdu g.SnmpPDU) error {
		idx := strings.TrimPrefix(pdu.Name, "."+oidIfDescr+".")
		descrs[idx] = snmpValueToString(pdu)
		return nil
	})
	if err != nil {
		return err
	}

	_ = client.BulkWalk(oidIfOperStatus, func(pdu g.SnmpPDU) error {
		idx := strings.TrimPrefix(pdu.Name, "."+oidIfOperStatus+".")
		v := snmpValueToString(pdu)
		switch v {
		case "1":
			statuses[idx] = "up"
		case "2":
			statuses[idx] = "down"
		case "3":
			statuses[idx] = "testing"
		default:
			statuses[idx] = v
		}
		return nil
	})

	_ = client.BulkWalk(oidIfSpeed, func(pdu g.SnmpPDU) error {
		idx := strings.TrimPrefix(pdu.Name, "."+oidIfSpeed+".")
		speeds[idx] = snmpValueToString(pdu)
		return nil
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "IDX\tINTERFACE\tSTATUS\tSPEED(bps)")
	fmt.Fprintln(w, "---\t---------\t------\t----------")
	for idx, name := range descrs {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", idx, name, statuses[idx], speeds[idx])
	}
	w.Flush()
	return nil
}

func snmpValueToString(v g.SnmpPDU) string {
	switch v.Type {
	case g.OctetString:
		if b, ok := v.Value.([]byte); ok {
			return string(b)
		}
	case g.Integer, g.Counter32, g.Counter64, g.Gauge32, g.Uinteger32, g.TimeTicks:
		return fmt.Sprintf("%v", v.Value)
	case g.IPAddress:
		return fmt.Sprintf("%v", v.Value)
	case g.ObjectIdentifier:
		return fmt.Sprintf("%v", v.Value)
	}
	return fmt.Sprintf("%v", v.Value)
}
