package wpac

import (
	"context"
	"errors"
	"fmt"
	"log"
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
	bus       *WPADBus
	ctx       context.Context
	ifacePath dbus.ObjectPath
	networks  map[int]WPANetwork
}

//func NewWPAInterface(bus *WPADBus, objectPath dbus.ObjectPath) *WPAInterface {
func NewWPAInterface(ctx context.Context, bus *WPADBus) *WPAInterface {
	wi := &WPAInterface{
		bus:      bus,
		ctx:      ctx,
		networks: make(map[int]WPANetwork),
	}
	return wi
}

// CreateInterface ...
func (w *WPAInterface) CreateInterface(ifname string) error {
	if ifname == "" {
		ifname = DefaultIfaceName
	}
	if ifpath, err := w.GetInterface(ifname); err == nil {
		w.ifacePath = ifpath
		return nil
	}

	args := make(map[string]dbus.Variant)
	args["Ifname"] = dbus.MakeVariant(ifname)
	args["Driver"] = dbus.MakeVariant("nl80211")
	iface, err := w.bus.CallWithVariant("fi.w1.wpa_supplicant1.CreateInterface", args)
	if err != nil && err.Error() != ErrInterfaceExists {
		w.ifacePath = ""
		return err
	}
	w.ifacePath = iface
	return nil
}

func (w *WPAInterface) GetInterface(ifname string) (dbus.ObjectPath, error) {
	iface, err := w.bus.CallWithString("fi.w1.wpa_supplicant1.GetInterface", ifname)
	if err != nil {
		return "", err
	}
	return iface, nil
}

func (w *WPAInterface) GetInterfaces() ([]dbus.ObjectPath, error) {
	prop, err := w.bus.GetProperty("fi.w1.wpa_supplicant1.Interfaces")
	if err != nil {
		return nil, err
	}
	ifaces, ok := prop.([]dbus.ObjectPath)
	if !ok {
		return nil, errors.New("get interfaces error")
	}
	for _, iface := range ifaces {
		log.Printf("iface: %s\n", iface)
	}
	return nil, nil
}

func (w *WPAInterface) CloseInterface() error {
	if w.ifacePath == "" {
		return errors.New("interface doesn't exist or doesn't represent an interface")
	}
	ifacePath := dbus.ObjectPath(w.ifacePath)
	if _, err := w.bus.CallWithPath("fi.w1.wpa_supplicant1.RemoveInterface", ifacePath); err != nil {
		return err
	}
	return nil
}

func (self *WPAInterface) State() string {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	state, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.State")
	if err != nil {
		return "unknown"
	}
	return state.Value().(string)
}

// GetScanInterval Time (in seconds) between scans for a suitable AP. Must be >= 0.
func (self *WPAInterface) GetScanInterval() (int32, error) {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	interval, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.ScanInterval")
	if err != nil {
		return -1, err
	}
	return interval.Value().(int32), nil
}

func (self *WPAInterface) SetScanInterval(interval int32) error {
	value := dbus.MakeVariant(interval)
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	err := obj.SetProperty("fi.w1.wpa_supplicant1.Interface.ScanInterval", value)
	if err != nil {
		return err
	}
	return nil
}

func (self *WPAInterface) Scan() error {
	args := make(map[string]dbus.Variant)
	args["Type"] = dbus.MakeVariant("passive")
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	if call := obj.Call("fi.w1.wpa_supplicant1.Interface.Scan", 0, args); call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) GetBSSList() []WPABSS {
	newBSSs := []WPABSS{}
	tmpBSSs := make(map[string]string)
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	bsss, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.BSSs")
	if err == nil {
		re := regexp.MustCompile(`^[\w\s_.-]*$`)
		for _, bssObjectPath := range bsss.Value().([]dbus.ObjectPath) {
			bss := NewBSS(self.bus, bssObjectPath)
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
	signal := self.bus.GetSignal()
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

func (self *WPAInterface) AddNetwork(args map[string]dbus.Variant) (*WPANetwork, error) {
	if _, ok := args["bssid"]; ok {
		bssid := args["bssid"].Value().(string)
		for _, network := range self.networks {
			if network.BSSID == bssid {
				return &network, nil
			}
		}
	}

	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	call := obj.Call("fi.w1.wpa_supplicant1.Interface.AddNetwork", 0, args)
	if call.Err != nil || len(call.Body) == 0 {
		return nil, call.Err
	}

	networkObj := dbus.ObjectPath(call.Body[0].(dbus.ObjectPath))
	id := len(self.networks)
	network := NewWPANetwork(self.bus, networkObj, id)
	self.networks[id] = network
	return &network, nil
}

func (self *WPAInterface) SetNetwork(id int, args map[string]dbus.Variant) error {
	if network, found := self.networks[id]; found {
		return network.writeProp(args)
	}
	return fmt.Errorf("network %d not found", id)
}

func (self *WPAInterface) SetNetworkEnabled(id int, enabled bool) error {
	if network, found := self.networks[id]; found {
		return network.writeEnable(enabled)
	}
	return fmt.Errorf("network %d not found", id)
}

func (self *WPAInterface) SelectNetwork(id int) error {
	if network, found := self.networks[id]; found {
		obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
		call := obj.Call("fi.w1.wpa_supplicant1.Interface.SelectNetwork", 0, network.Object)
		if call.Err != nil {
			return call.Err
		}
	}
	return nil
}

func (self *WPAInterface) RemoveNetwork(id int) error {
	if networkObj, found := self.networks[id]; found {
		obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
		if call := obj.Call("fi.w1.wpa_supplicant1.Interface.RemoveNetwork", 0, networkObj.Object); call.Err != nil {
			return call.Err
		}
	}
	return fmt.Errorf("network (%d) not found", id)
}

func (self *WPAInterface) RemoveAllNetwork() error {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	if call := obj.Call("fi.w1.wpa_supplicant1.Interface.RemoveAllNetworks", 0); call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) Disconnect() error {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	if call := obj.Call("fi.w1.wpa_supplicant1.Interface.Disconnect", 0); call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) GetNetworks() (map[int]WPANetwork, error) {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	ifaces, err := obj.GetProperty("fi.w1.wpa_supplicant1.Interface.Networks")
	if err != nil {
		return nil, err
	}

	networks := ifaces.Value().([]dbus.ObjectPath)
	for id, network := range networks {
		wn := NewWPANetwork(self.bus, network, id)
		self.networks[id] = wn
	}
	return self.networks, nil
}

func (self *WPAInterface) Reassociate() error {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	call := obj.Call("fi.w1.wpa_supplicant1.Interface.Reassociate", 0)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) Reattach() error {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
	call := obj.Call("fi.w1.wpa_supplicant1.Interface.Reattach", 0)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPAInterface) Reconnect() error {
	obj := self.bus.Connection.Object("fi.w1.wpa_supplicant1", self.ifacePath)
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
	signal := w.bus.GetSignal()
	for {
		select {
		case event := <-signal:
			w.eventUpdate(event.Name, event.Body)
		case <-w.ctx.Done():
			return
		}
	}
}

func (w *WPAInterface) AddEventListener() error {
	if w.ifacePath == "" {
		return errors.New("interface not ready")
	}
	obj := w.bus.Connection.Object("fi.w1.wpa_supplicant1", w.ifacePath)
	if err := w.bus.AddSignalObserver("fi.w1.wpa_supplicant1.Interface", obj.Path()); err != nil {
		return err
	}
	return nil
}
