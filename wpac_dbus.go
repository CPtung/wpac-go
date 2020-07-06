package wpac

import (
	"context"
	"errors"
	"fmt"

	"github.com/godbus/dbus/v5"
)

type DBusProp struct {
	Interface string
	Name      string
	Value     string
}

type WPADBus struct {
	Connection *dbus.Conn
	Object     dbus.BusObject
	Signal     *WPASignal
}

func NewWpaDBus(ctx context.Context) (*WPADBus, error) {
	var wdbus *WPADBus
	if conn, err := dbus.SystemBus(); err == nil {
		if obj := conn.Object("fi.w1.wpa_supplicant1", "/fi/w1/wpa_supplicant1"); obj != nil {
			wdbus = &WPADBus{
				Connection: conn,
				Object:     obj,
				Signal:     NewWPASignal(conn),
			}
			if err := wdbus.AddSignalObserver("fi.w1.wpa_supplicant1", "/fi/w1/wpa_supplicant1"); err != nil {
				wdbus.Close()
				return nil, fmt.Errorf("create dbus signal hook failed (%s)", err.Error())
			}
		} else {
			conn.Close()
			return nil, errors.New("Can't create WPA object")
		}
	}
	return wdbus, nil
}

func (self *WPADBus) SetProperty(prop DBusProp) error {
	value := dbus.MakeVariant(prop.Value)
	call := self.Object.Call("org.freedesktop.DBus.Properties.Set", 0, prop.Interface, prop.Name, value)
	if call.Err != nil {
		return call.Err
	}
	return nil
}

func (self *WPADBus) GetProperty(name string) (value interface{}, e error) {
	variant, err := self.Object.GetProperty(name)
	if err != nil {
		return nil, err
	}
	return variant.Value(), nil
}

func (self *WPADBus) GetSignal() chan *dbus.Signal {
	return self.Signal.Get()
}

func (self *WPADBus) AddSignalObserver(dbusInterface string, object dbus.ObjectPath) error {
	return self.Signal.AddObserver(dbusInterface, object)
}

func (self *WPADBus) Call(path string) (dbus.ObjectPath, error) {
	var objectPath dbus.ObjectPath
	call := self.Object.Call(path, 0)
	if call.Err != nil {
		return "", call.Err
	}
	if len(call.Body) > 0 {
		objectPath = dbus.ObjectPath(call.Body[0].(dbus.ObjectPath))
	}
	return objectPath, nil
}

func (self *WPADBus) CallWithPath(path string, args dbus.ObjectPath) (dbus.ObjectPath, error) {
	var objectPath dbus.ObjectPath
	call := self.Object.Call(path, 0, args)
	if call.Err != nil {
		return "", call.Err
	}
	if len(call.Body) > 0 {
		objectPath = dbus.ObjectPath(call.Body[0].(dbus.ObjectPath))
	}
	return objectPath, nil
}

func (self *WPADBus) CallWithString(path string, args string) (dbus.ObjectPath, error) {
	var objectPath dbus.ObjectPath
	call := self.Object.Call(path, 0, args)
	if call.Err != nil {
		return "", call.Err
	}
	if len(call.Body) > 0 {
		objectPath = dbus.ObjectPath(call.Body[0].(dbus.ObjectPath))
	}
	return objectPath, nil
}

func (self *WPADBus) CallWithVariant(path string, args map[string]dbus.Variant) (dbus.ObjectPath, error) {
	var objectPath dbus.ObjectPath
	call := self.Object.Call(path, 0, args)
	if call.Err != nil {
		return "", call.Err
	}
	if len(call.Body) > 0 {
		objectPath = dbus.ObjectPath(call.Body[0].(dbus.ObjectPath))
	}
	return objectPath, nil
}

func (self *WPADBus) MakeVariant(variant map[string]interface{}) map[string]dbus.Variant {
	args := make(map[string]dbus.Variant)
	for k, v := range variant {
		args[k] = dbus.MakeVariant(v)
	}
	return args
}

func (self *WPADBus) Close() {
	self.Signal.Close()
	self.Connection.Close()
}
