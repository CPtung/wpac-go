package wpac

import (
	"encoding/hex"
	"strings"

	"github.com/godbus/dbus/v5"
)

type BSSWPA struct {
	KeyMgmt  []string `json:"key_mgmt"`
	PairWise []string `json:"pairwise"`
	Group    string   `json:"group"`
}

type BSSWPA2 struct {
	KeyMgmt  []string `json:"key_mgmt"`
	PairWise []string `json:"pairwise"`
	Group    string   `json:"group"`
}

// WPABSS ...
type WPABSS struct {
	busObject dbus.BusObject `json:"busObj"`
	BSSID     string         `json:"bssid"`
	SSID      string         `json:"ssid"`
	PSK       string         `json:"psk"`
	WPA       *BSSWPA        `json:"wpa"`
	WPA2      *BSSWPA2       `json:"wpa2"`
	WPS       string         `json:"wps"`
	Frequency uint16         `json:"frequency"`
	Signal    int16          `json:"signal"`
	Age       uint32         `json:"age"`
	Mode      string         `json:"mode"`
	Privacy   bool           `json:"privacy"`
	Priority  int            `json:"priority"`
}

// NewBSS ...
func NewBSS(bus *WPADBus, objPath dbus.ObjectPath) WPABSS {
	obj := bus.Connection.Object("fi.w1.wpa_supplicant1", objPath)
	bss := WPABSS{busObject: obj, WPA: &BSSWPA{}, WPA2: &BSSWPA2{}}
	bss.readWPA()
	bss.readRSN()
	bss.readBSSID()
	bss.readSSID()
	bss.readAge()
	bss.readSignal()
	bss.readMode()
	bss.readPrivacy()
	bss.readFrequency()
	return bss
}

func (wb *WPABSS) readWPA() error {
	if prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.WPA"); err == nil {
		if value, ok := prop.Value().(map[string]dbus.Variant); ok {
			for key, variant := range value {
				switch key {
				case "KeyMgmt":
					wb.WPA.KeyMgmt = variant.Value().([]string)
				case "Pairwise":
					wb.WPA.PairWise = variant.Value().([]string)
				case "Group":
					wb.WPA.Group = variant.Value().(string)
				}
			}
			if len(wb.WPA.KeyMgmt) == 0 {
				wb.WPA = nil
			}
		}
	}
	return nil
}

func (wb *WPABSS) readRSN() error {
	prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.RSN")
	if err != nil {
		return err
	}
	if value, ok := prop.Value().(map[string]dbus.Variant); ok {
		for key, variant := range value {
			switch key {
			case "KeyMgmt":
				wb.WPA2.KeyMgmt = variant.Value().([]string)
			case "Pairwise":
				wb.WPA2.PairWise = variant.Value().([]string)
			case "Group":
				wb.WPA2.Group = variant.Value().(string)
			}
		}
		if len(wb.WPA2.KeyMgmt) == 0 {
			wb.WPA2 = nil
		}
	}
	return nil
}

func (wb *WPABSS) readBSSID() error {
	prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.BSSID")
	if err == nil {
		var hexstring strings.Builder
		bssid := prop.Value().([]byte)

		i := 0
		for {
			hexstring.WriteString(hex.EncodeToString(bssid[i : i+1]))
			i++
			if i >= len(bssid) {
				break
			}
			hexstring.WriteRune(':')
		}
		wb.BSSID = hexstring.String()
	}
	return err
}

func (wb *WPABSS) readSSID() error {
	prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.SSID")
	if err == nil {
		wb.SSID = string(prop.Value().([]byte))
	}
	return err
}

func (wb *WPABSS) readFrequency() error {
	if prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.Frequency"); err == nil {
		wb.Frequency = prop.Value().(uint16)
	}
	return nil
}

func (wb *WPABSS) readSignal() error {
	if prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.Signal"); err == nil {
		wb.Signal = prop.Value().(int16)
	}
	return nil
}

func (wb *WPABSS) readAge() error {
	if prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.Age"); err == nil {
		wb.Age = prop.Value().(uint32)
	}
	return nil
}

func (wb *WPABSS) readMode() error {
	if prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.Mode"); err == nil {
		wb.Mode = prop.Value().(string)
	}
	return nil
}

func (wb *WPABSS) readPrivacy() error {
	if prop, err := wb.busObject.GetProperty("fi.w1.wpa_supplicant1.BSS.Privacy"); err == nil {
		wb.Privacy = prop.Value().(bool)
	}
	return nil
}
