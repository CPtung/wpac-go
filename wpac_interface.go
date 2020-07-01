package wpac

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/godbus/dbus/v5"
)

const (
	DefaultIfaceName = "wlan0"

	// ErrInterfaceExists A WPA Supplicant Error fi.w1.wpa_supplicant1.InterfaceExists
	ErrInterfaceExists = "wpa_supplicant already controls this interface."
)

type WPAInterface struct {
	Bus       *WPADBus
	Ctx       context.Context
	IfacePath dbus.ObjectPath
	Networks  map[string]WPANetwork
}

//func NewWPAInterface(bus *WPADBus, objectPath dbus.ObjectPath) *WPAInterface {
func NewWPAInterface(ctx context.Context, bus *WPADBus) *WPAInterface {
	wi := &WPAInterface{
		Bus:      bus,
		Ctx:      ctx,
		Networks: make(map[string]WPANetwork),
	}
	return wi
}

// CreateInterface ...
func (w *WPAInterface) CreateInterface(ifname string) error {
	if ifname == "" {
		ifname = DefaultIfaceName
	}
	if ifpath, err := w.GetInterface(ifname); err == nil {
		w.IfacePath = ifpath
		return nil
	}

	args := make(map[string]dbus.Variant)
	args["Ifname"] = dbus.MakeVariant(ifname)
	args["Driver"] = dbus.MakeVariant("nl80211")
	iface, err := w.Bus.CallWithVariant("fi.w1.wpa_supplicant1.CreateInterface", args)
	if err != nil && err.Error() != ErrInterfaceExists {
		w.IfacePath = ""
		return err
	}
	w.IfacePath = iface
	return nil
}

func (w *WPAInterface) GetInterface(ifname string) (dbus.ObjectPath, error) {
	iface, err := w.Bus.CallWithString("fi.w1.wpa_supplicant1.GetInterface", ifname)
	if err != nil {
		return "", err
	}
	return iface, nil
}

func (w *WPAInterface) CloseInterface() error {
	if w.IfacePath == "" {
		return errors.New("interface doesn't exist or doesn't represent an interface")
	}
	ifacePath := dbus.ObjectPath(w.IfacePath)
	if _, err := w.Bus.CallWithPath("fi.w1.wpa_supplicant1.RemoveInterface", ifacePath); err != nil {
		return err
	}
	return nil
}

func (self *WPAInterface) State() string {
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	state, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.State")
	if err != nil {
		return "unknown"
	}
	return state.Value().(string)
}

// GetScanInterval Time (in seconds) between scans for a suitable AP. Must be >= 0.
func (self *WPAInterface) GetScanInterval() (int32, error) {
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	interval, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.ScanInterval")
	if err != nil {
		return -1, err
	}
	return interval.Value().(int32), nil
}

func (self *WPAInterface) SetScanInterval(interval int32) error {
	value := dbus.MakeVariant(interval)
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	err := obj.SetProperty("fi.w1.wpa_supplicant1.Interface.ScanInterval", value)
	if err != nil {
		return err
	}
	return nil
}

func (self *WPAInterface) Scan() error {
	args := make(map[string]dbus.Variant)
	args["Type"] = dbus.MakeVariant("passive")
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	if call := obj.Call("fi.w1.wpa_supplicant1.Interface.Scan", 0, args); call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) GetBSSList() []WPABSS {
	newBSSs := []WPABSS{}
	tmpBSSs := make(map[string]string)
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	bsss, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.BSSs")
	if err == nil {
		re := regexp.MustCompile(`^[\w\s_.-]*$`)
		for _, bssObjectPath := range bsss.Value().([]dbus.ObjectPath) {
			bss := NewBSS(self.Bus, bssObjectPath)
			if re.MatchString(bss.SSID) && bss.SSID != "" {
				if _, found := tmpBSSs[bss.BSSID]; !found {
					tmpBSSs[bss.BSSID] = bss.BSSID
					newBSSs = append(newBSSs, bss)
				}
			}
		}
	}
	return newBSSs
}

func (self *WPAInterface) AutoScan() ([]WPABSS, error) {
	interval, err := self.GetScanInterval()
	if err != nil {
		return nil, err
	}

	done := make(chan struct{})
	signal := self.Bus.GetSignal()
	timeout := time.After(time.Duration(interval) * time.Second)
	if err := self.Scan(); err != nil {
		return nil, err
	}
	go func() {
		for {
			// wait for scan done or exit by timeout
			select {
			case <-done:
				return
			case <-timeout:
				close(done)
			case event := <-signal:
				if event.Name == SignalScanDone {
					close(done)
				}
			}
		}
	}()
	<-done

	return self.GetBSSList(), nil
}

func (self *WPAInterface) AddNetwork(args map[string]dbus.Variant) error {
	bssid := args["bssid"].Value().(string)
	if _, found := self.Networks[bssid]; found {
		return nil
	}

	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	call := obj.Call("fi.w1.wpa_supplicant1.Interface.AddNetwork", 0, args)
	if call.Err != nil || len(call.Body) == 0 {
		return call.Err
	}

	networkObj := dbus.ObjectPath(call.Body[0].(dbus.ObjectPath))
	network := NewWPANetwork(self.Bus, networkObj)
	self.Networks[network.BSSID] = network
	return nil
}

func (self *WPAInterface) SelectNetwork(bssid string) error {
	if network, found := self.Networks[bssid]; found {
		obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
		call := obj.Call("fi.w1.wpa_supplicant1.Interface.SelectNetwork", 0, network.Object)
		if call.Err != nil {
			return call.Err
		}
	}
	return nil
}

func (self *WPAInterface) RemoveNetwork(bssid string) error {
	if networkObj, found := self.Networks[bssid]; found {
		obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
		if call := obj.Call("fi.w1.wpa_supplicant1.Interface.RemoveNetwork", 0, networkObj); call.Err != nil {
			return call.Err
		}
	}
	return fmt.Errorf("network (%s) not found", bssid)
}

func (self *WPAInterface) RemoveAllNetwork() error {
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	if call := obj.Call("fi.w1.wpa_supplicant1.Interface.RemoveAllNetworks", 0); call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) GetNetworks() error {
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	ifaces, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.Networks")
	if err != nil {
		return err
	}
	networks := ifaces.Value().([]dbus.ObjectPath)
	for _, network := range networks {
		wn := NewWPANetwork(self.Bus, network)
		self.Networks[wn.BSSID] = wn
	}
	return nil
}

func (self *WPAInterface) Reassociate() error {
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	call := obj.Call("fi.w1.wpa_supplicant1.Interface.Reassociate", 0)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) Reattach() error {
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	call := obj.Call("fi.w1.wpa_supplicant1.Interface.Reattach", 0)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) Reconnect() error {
	obj := self.Bus.Connection.Object("fi.w1.wpa_supplicant1", self.IfacePath)
	call := obj.Call("fi.w1.wpa_supplicant1.Interface.Reconnect", 0)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func (w *WPAInterface) eventUpdate(name string, body []interface{}) {
	switch name {
	case "fi.w1.wpa_supplicant1.InterfaceRemoved":
	case "fi.w1.wpa_supplicant1.Interface.PropertiesChanged":
	case "fi.w1.wpa_supplicant1.Interface.ScanDone":
	}
}

func (w *WPAInterface) eventListener() {
	signal := w.Bus.GetSignal()
	for {
		select {
		case event := <-signal:
			w.eventUpdate(event.Name, event.Body)
		case <-w.Ctx.Done():
			return
		}
	}
}

func (w *WPAInterface) AddEventListener() error {
	if w.IfacePath == "" {
		return errors.New("interface not ready")
	}
	obj := w.Bus.Connection.Object("fi.w1.wpa_supplicant1", w.IfacePath)
	if err := w.Bus.AddSignalObserver("fi.w1.wpa_supplicant1.Interface", obj.Path()); err != nil {
		return err
	}
	return nil
}
