package wpac

import (
	"context"

	"github.com/godbus/dbus/v5"
)

// WPA ...
type WPA struct {
	bus    *WPADBus
	ctx    context.Context
	ifaces map[string]*WPAInterface
}

// NewWPA ...
func NewWPA(ctx context.Context) (wpa *WPA, e error) {
	var (
		err error
		bus *WPADBus
	)
	bus, err = NewWpaDBus(ctx)
	if err != nil {
		return nil, err
	}
	// init wpa instance
	wpa = &WPA{
		bus:    bus,
		ctx:    ctx,
		ifaces: make(map[string]*WPAInterface),
	}
	return wpa, e
}

func (w *WPA) InitInterface(ifname string) error {
	iface := NewWPAInterface(w.ctx, w.bus)
	if err := iface.CreateInterface(ifname); err != nil {
		return err
	}

	// scan wpa network profiles on machine
	if _, err := iface.GetNetworks(); err != nil {
		return err
	}

	if err := iface.AddEventListener(); err != nil {
		return err
	}

	w.ifaces[ifname] = iface
	return nil
}

func (w *WPA) GetInterface(ifname string) *WPAInterface {
	if iface, found := w.ifaces[ifname]; found {
		return iface
	}
	return nil
}

func (w *WPA) RemoveInterface(ifname string) {
	if _, found := w.ifaces[ifname]; found {
		delete(w.ifaces, ifname)
	}
}

func (w *WPA) GetEventSignal() <-chan *dbus.Signal {
	return w.bus.GetSignal()
}

func (w *WPA) Close() {
	w.bus.Close()
}
