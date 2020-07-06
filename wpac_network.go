package wpac

import (
	"regexp"
	"strconv"

	"github.com/godbus/dbus/v5"
)

const (
	WPANetworkSSID     = "ssid"
	WPANetworkBSSID    = "bssid"
	WPANetworkPSK      = "psk"
	WPANetworkKeyMgmt  = "key_mgmt"
	WPANetworkProto    = "proto"
	WPANetworkMode     = "mode"
	WPANetworkPairWise = "pairwise"
	WPANetworkGroup    = "group"
	WPANetworkPriority = "priority"
)

// WPABSS ...
type WPANetwork struct {
	busObject dbus.BusObject
	Object    dbus.ObjectPath
	Enable    bool
	ID        int
	BSSID     string
	SSID      string
	PSK       string
	KeyMgmt   string
	Proto     string
	Mode      string
	PairWise  string
	Group     string
	Frequency uint16
	Priority  int64
}

// NewNetwork ...
func NewWPANetwork(bus *WPADBus, objPath dbus.ObjectPath, ID int) WPANetwork {
	obj := bus.Connection.Object("fi.w1.wpa_supplicant1", objPath)
	network := WPANetwork{busObject: obj, Object: objPath, ID: ID}
	network.readEnable()
	network.readProp()
	return network
}

func (wn *WPANetwork) writeEnable(enabled bool) error {
	v := dbus.MakeVariant(enabled)
	return wn.busObject.SetProperty("fi.w1.wpa_supplicant1.Network.Enabled", v)
}

func (wn *WPANetwork) writeProp(props map[string]dbus.Variant) error {
	v := dbus.MakeVariant(props)
	return wn.busObject.SetProperty("fi.w1.wpa_supplicant1.Network.Properties", v)
}

func (wn *WPANetwork) readEnable() error {
	prop, err := wn.busObject.GetProperty("fi.w1.wpa_supplicant1.Network.Enabled")
	if err == nil {
		if enable, ok := prop.Value().(bool); ok {
			wn.Enable = enable
		}
	}
	return err
}

func (wn *WPANetwork) readProp() error {
	var value string
	prop, err := wn.busObject.GetProperty("fi.w1.wpa_supplicant1.Network.Properties")
	if err == nil {
		re := regexp.MustCompile(`^"(.*)"$`)
		if dict, ok := prop.Value().(map[string]dbus.Variant); ok {
			for k, v := range dict {
				value = re.ReplaceAllString(v.Value().(string), `$1`)
				switch k {
				case WPANetworkSSID:
					wn.SSID = value
				case WPANetworkBSSID:
					wn.BSSID = value
				case WPANetworkPSK:
					wn.PSK = value
				case WPANetworkKeyMgmt:
					wn.KeyMgmt = value
				case WPANetworkProto:
					wn.Proto = value
				case WPANetworkMode:
					wn.Mode = value
				case WPANetworkPairWise:
					wn.PairWise = value
				case WPANetworkGroup:
					wn.Group = value
				case WPANetworkPriority:
					wn.Priority, _ = strconv.ParseInt(value, 0, 64)
				}
			}
		}
	}
	return err
}
