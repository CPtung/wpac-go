package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"text/tabwriter"

	wpa "github.com/CPtung/wpac-go"
	"github.com/godbus/dbus/v5"
	"github.com/spf13/cobra"
)

var (
	ifname   string
	cfile    string
	security string
	interval int32
	ctx      context.Context
	wpacli   *wpa.WPA
)

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "wpac state",
	Run:   stateMode,
}

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "wpac scan",
	Run:   scanMode,
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "wpac connect",
	Run:   connectMode,
}

var reattachCmd = &cobra.Command{
	Use:   "reattach",
	Short: "wpac reattach",
	Run:   reattachMode,
}

var reassociateCmd = &cobra.Command{
	Use:   "reassociate",
	Short: "wpac reassociate",
	Run:   reassociateMode,
}

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "wpac remove",
	Run:   removeMode,
}

var shutdownCmd = &cobra.Command{
	Use:   "shutdown",
	Short: "wpac shutdown",
	Run:   shutdownMode,
}

var eventCmd = &cobra.Command{
	Use:   "event",
	Short: "wpac event",
	Run:   eventMode,
}

var rootCmd = &cobra.Command{
	Use:   "wpa",
	Short: "WPA Client Util for MOXA ThingsPro",
}

func loadConfig(path string, bss *wpa.WPABSS) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	re := regexp.MustCompile(`\s*(.*)=\"*([\w_.-@!~:]*)\"*`)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) >= 3 {
			switch matches[1] {
			case "ssid":
				bss.SSID = matches[2]
			case "bssid":
				bss.BSSID = matches[2]
			case "psk":
				bss.PSK = matches[2]
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func printUsage(cmd *cobra.Command, err error) {
	fmt.Println(err.Error())
	cmd.Usage()
	os.Exit(1)
}

func stateMode(cmd *cobra.Command, args []string) {
	fmt.Printf("network state: %s\n", wpacli.GetInterface(ifname).State())
}

func scanMode(cmd *cobra.Command, args []string) {
	if interval == 0 {
		printUsage(cmd, errors.New("scan interval cannot be smaller than 1sec"))
	} else if interval > 0 {
		err := wpacli.GetInterface(ifname).SetScanInterval(interval)
		if err != nil {
			printUsage(cmd, err)
		}
	}

	list, err := wpacli.GetInterface(ifname).AutoScan()
	if err != nil {
		printUsage(cmd, err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "bssid\tssid\tfrequency\tsignal")
	for _, bss := range list {
		ap := strings.Builder{}
		ap.WriteString(fmt.Sprintf("%s", bss.BSSID))
		ap.WriteString(fmt.Sprintf("\t%s", bss.SSID))
		ap.WriteString(fmt.Sprintf("\t%d", bss.Frequency))
		ap.WriteString(fmt.Sprintf("\t%d", bss.Signal))
		if bss.WPA != nil {
			ap.WriteString(fmt.Sprintf("\t[WPA-%s-%s]",
				strings.ToUpper(bss.WPA.KeyMgmt[0]),
				strings.ToUpper(bss.WPA.Group)))
		}
		if bss.WPA2 != nil {
			ap.WriteString(fmt.Sprintf("\t[WPA2-%s-%s]",
				strings.ToUpper(bss.WPA2.KeyMgmt[0]),
				strings.ToUpper(bss.WPA2.Group)))
		}
		fmt.Fprintln(w, ap.String())
	}
	w.Flush()
}

func connectMode(cmd *cobra.Command, args []string) {
	var (
		bss    wpa.WPABSS
		config map[string]dbus.Variant
	)

	bss = wpa.WPABSS{}
	if err := loadConfig(cfile, &bss); err != nil {
		printUsage(cmd, err)
	}

	switch security {
	case "none":
		config = wpa.WPAConfig().GetWPANone(bss)
	case "wpa":
		config = wpa.WPAConfig().GetWPA2(bss)
	case "wpa2":
		config = wpa.WPAConfig().GetWPA2(bss)
	}
	if err := wpacli.GetInterface(ifname).AddNetwork(config); err != nil {
		printUsage(cmd, fmt.Errorf("add network error (%s)", err.Error()))
	}

	// select network
	if err := wpacli.GetInterface(ifname).SelectNetwork(bss.BSSID); err != nil {
		printUsage(cmd, fmt.Errorf("select network error (%s)", err.Error()))
	}
}

func removeMode(cmd *cobra.Command, args []string) {
	err := wpacli.GetInterface(ifname).RemoveAllNetwork()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func reassociateMode(cmd *cobra.Command, args []string) {
	err := wpacli.GetInterface(ifname).Reassociate()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func scanIntervalMode(cmd *cobra.Command, args []string) {
	err := wpacli.GetInterface(ifname).SetScanInterval(interval)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func reattachMode(cmd *cobra.Command, args []string) {
	err := wpacli.GetInterface(ifname).Reattach()
	if err != nil {
		fmt.Println(err.Error())
	}
}

func reInitInterface(prop map[string]dbus.Variant) {
	if name, found := prop["Ifname"]; found {
		fmt.Println("interface (%s) up" + name.Value().(string))
		if err := wpacli.AddInterface(name.Value().(string)); err != nil {
			fmt.Printf("add interface error (%s)\n", err.Error())
		}
	}
}

func printState(prop map[string]dbus.Variant) {
	if state, found := prop["State"]; found {
		fmt.Printf("network State: %s\n", state.Value().(string))
	}
}

func eventMode(cmd *cobra.Command, args []string) {
	done := make(chan struct{})
	go func() {
		sig := wpacli.GetEventSignal()
		interrupt := make(chan os.Signal)
		signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
		for {
			select {
			case <-interrupt:
				close(done)
				return
			case event := <-sig:
				switch event.Name {
				case wpa.SignalPropertiesChanged:
					for _, data := range event.Body {
						printState(data.(map[string]dbus.Variant))
					}
				case wpa.SignalInterfaceAdded:
					for _, data := range event.Body {
						if prop, ok := data.(map[string]dbus.Variant); ok {
							reInitInterface(prop)
							break
						}
					}
				case wpa.SignalInterfaceRemoved:
					fmt.Printf("interface (%s) Down\n", event.Body[0])
				}
			}
		}
	}()
	<-done
}

func shutdownMode(cmd *cobra.Command, args []string) {
	err := wpacli.GetInterface(ifname).CloseInterface()
	if err != nil {
		fmt.Println(err.Error())
	}
	wpacli.RemoveInterface(ifname)
}

func init() {
	rootCmd.Flags().StringVarP(&ifname, "iface", "i", "wlan0", "target interface")
	connectCmd.Flags().StringVarP(&cfile, "config", "c", "", "target network config")
	connectCmd.Flags().StringVarP(&security, "security", "s", "wpa2", "target network security (\"none\", \"wpa\", \"wpa2\")")
	scanCmd.Flags().Int32VarP(&interval, "interval", "i", -1, "target scan interval (interval > 0)")
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(reattachCmd)
	rootCmd.AddCommand(reassociateCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(shutdownCmd)
	rootCmd.AddCommand(eventCmd)
}

func main() {
	var err error

	if wpacli, err = wpa.NewWPA(context.TODO()); err != nil {
		log.Fatalf(err.Error())
	}
	defer wpacli.Close()

	if err := wpacli.AddInterface(ifname); err != nil {
		log.Fatalf(err.Error())
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}