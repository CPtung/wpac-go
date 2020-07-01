package wpac

import (
	"fmt"

	"github.com/godbus/dbus/v5"
)

const (
	SignalScanDone          = "fi.w1.wpa_supplicant1.Interface.ScanDone"
	SignalScanTimeout       = "fi.w1.wpa_supplicant1.Interface.ScanTimeout"
	SignalPropertiesChanged = "fi.w1.wpa_supplicant1.Interface.PropertiesChanged"
	SignalInterfaceAdded    = "fi.w1.wpa_supplicant1.InterfaceAdded"
	SignalInterfaceRemoved  = "fi.w1.wpa_supplicant1.InterfaceRemoved"
)

type WPASignal struct {
	conn   *dbus.Conn
	signal chan *dbus.Signal
	paths  map[dbus.ObjectPath]string
}

func NewWPASignal(conn *dbus.Conn) *WPASignal {
	ws := WPASignal{
		conn:   conn,
		signal: make(chan *dbus.Signal, 10),
		paths:  make(map[dbus.ObjectPath]string),
	}
	ws.conn.Signal(ws.signal)
	return &ws
}

func (ws *WPASignal) Get() chan *dbus.Signal {
	return ws.signal
}

func (ws *WPASignal) Close() {
	for path, iface := range ws.paths {
		ws.RemoveObserver(iface, path)
	}
	ws.conn.RemoveSignal(ws.signal)
}

func (ws *WPASignal) AddObserver(iface string, path dbus.ObjectPath) error {
	match := fmt.Sprintf("type='signal',interface='%s',path='%s'", iface, path)
	if call := ws.conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, match); call.Err != nil {
		return call.Err
	}
	ws.paths[path] = iface
	return nil
}

func (ws *WPASignal) RemoveObserver(iface string, path dbus.ObjectPath) error {
	match := fmt.Sprintf("type='signal',interface='%s',path='%s'", iface, path)
	if call := ws.conn.BusObject().Call("org.freedesktop.DBus.RemoveMatch", 0, match); call.Err != nil {
		return call.Err
	}
	return nil
}
