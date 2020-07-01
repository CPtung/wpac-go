package wpac

import (
	"github.com/godbus/dbus/v5"
)

type WPASupplicantConfig struct{}

var instance *WPASupplicantConfig

func WPAConfig() *WPASupplicantConfig {
	if instance == nil {
		instance = &WPASupplicantConfig{}
	}
	return instance
}

func (config *WPASupplicantConfig) GetWPA(bss WPABSS) map[string]dbus.Variant {
	template := make(map[string]dbus.Variant)
	template["bssid"] = dbus.MakeVariant(bss.BSSID)
	template["ssid"] = dbus.MakeVariant(bss.SSID)
	template["psk"] = dbus.MakeVariant(bss.PSK)
	template["proto"] = dbus.MakeVariant("WPA")
	template["pairwise"] = dbus.MakeVariant("TKIP")
	template["group"] = dbus.MakeVariant("TKIP")
	template["key_mgmt"] = dbus.MakeVariant("WPA-PSK")
	return template
}

func (config *WPASupplicantConfig) GetWPA2(bss WPABSS) map[string]dbus.Variant {
	template := make(map[string]dbus.Variant)
	template["bssid"] = dbus.MakeVariant(bss.BSSID)
	template["ssid"] = dbus.MakeVariant(bss.SSID)
	template["psk"] = dbus.MakeVariant(bss.PSK)
	template["proto"] = dbus.MakeVariant("RSN")
	template["pairwise"] = dbus.MakeVariant("CCMP")
	template["group"] = dbus.MakeVariant("CCMP")
	template["key_mgmt"] = dbus.MakeVariant("WPA-PSK")
	return template
}

func (config *WPASupplicantConfig) GetWPANone(bss WPABSS) map[string]dbus.Variant {
	template := make(map[string]dbus.Variant)
	template["ssid"] = dbus.MakeVariant(bss.SSID)
	template["key_mgmt"] = dbus.MakeVariant("NONE")
	return template
}

func (config *WPASupplicantConfig) GetWPAWPA2(bss WPABSS) map[string]dbus.Variant {
	template := make(map[string]dbus.Variant)
	template["bssid"] = dbus.MakeVariant(bss.BSSID)
	template["ssid"] = dbus.MakeVariant(bss.SSID)
	template["psk"] = dbus.MakeVariant(bss.PSK)
	template["proto"] = dbus.MakeVariant("WPA RSN")
	template["pairwise"] = dbus.MakeVariant("CCMP")
	template["group"] = dbus.MakeVariant("CCMP")
	template["key_mgmt"] = dbus.MakeVariant("WPA-PSK")
	return template
}
