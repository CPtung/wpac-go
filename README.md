# wpac-go

wpac-go provides WiFi connectivity API for [ThingsPro](https://www.moxa.com/en/products/industrial-computing/system-software/thingspro-2). It was implemented by pure golang and following wpa_supplicant D-Bus API interfaces.


Environment
------------

### Install wpa supplicant
```bash
# install
apt-get update
apt-get install wpasupplicant -y

# start service
systemctl start wpa_supplicant
```

### Access host wpa supplicant from a container
Since wpac-go communicates with wpa_supplicant over DBus. If your process has to work in a container environment. Some pathes need to be mounted from the host beforehand.
```yaml
    volumes:
      - /var/lib/dbus:/var/lib/dbus
      - /var/run/dbus:/var/run/dbus
```


### Go Vendor

```bash
    go get github.com/CPtung/wpac-go
```

Examples
--------------
### Basic Client
```go
package main

import (
	"context"
	"errors"
	"log"
	"os"	
	wpa "github.com/CPtung/wpac-go"
	"github.com/spf13/cobra"
)

func main() {
	var (
	    err error
	    wpacli *wpa.WPA
	)
	if wpacli, err = wpa.NewWPA(context.TODO()); err != nil {
		log.Fatalf(err.Error())
	}
	defer wpacli.Close()
    
	if err := wpacli.InitInterface("wlan0"); err != nil {
		log.Fatalf(err.Error())
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
```

### Event Listener
```go
func eventLietener(wpacli *wpa.WPA) {
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
				case wpa.SignalScanDone:
					if event.Body[0] == true {
						fmt.Println("network State: scan completed")
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
```
Pre-defined signal category please refer to [definition](https://github.com/CPtung/wpac-go/blob/f0e9146aa3a26475ba6ab74929bb19140d737959/wpac_signal.go#L9-L15)
