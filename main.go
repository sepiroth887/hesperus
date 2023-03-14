package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/go-ble/ble"
	"github.com/go-ble/ble/examples/lib/dev"
)

type Hesperus struct {
	updateCh    chan bool
	config      Config
	activityMap map[string]time.Time
	beacons     map[string]IBeacon
}

func main() {
	d, err := dev.NewDevice("0")
	if err != nil {
		fmt.Println("failed init: ", err)
		os.Exit(1)
	}

	data, err := os.ReadFile("./config.yml")
	if err != nil {
		fmt.Println("failed to read config: ", err)
		os.Exit(2)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println("failed to parse config: ", err)
		os.Exit(3)
	}

	beacons := make(map[string]IBeacon)
	for _, beacon := range config.IBeacons {
		beacons[beacon.Name] = IBeacon{
			UUID:    beacon.UUID,
			Major:   uint16(beacon.Major),
			Minor:   uint16(beacon.Minor),
			MinRSSI: beacon.MinRSSI,
		}
		fmt.Printf("added beacon watch for: %s(%s)\n", beacon.Name, beacon.UUID)
	}

	hesp := Hesperus{
		updateCh:    make(chan bool),
		activityMap: make(map[string]time.Time),
		beacons:     beacons,
		config:      config,
	}

	ble.SetDefaultDevice(d)
	go ble.Scan(context.Background(), true, hesp.scan, nil)
	go func() {
		for range time.Tick(hesp.config.HassUpdateInterval) {
			active := false
			remove := []string{}
			for k, v := range hesp.activityMap {
				if time.Since(v) < time.Minute*5 {
					active = true
					break
				} else {
					remove = append(remove, k)
				}
			}
			hesp.updateCh <- active

			for _, k := range remove {
				delete(hesp.activityMap, k)
			}
		}
	}()

	sigCh := make(chan os.Signal, 1)

	for {
		select {
		case <-sigCh:
			fmt.Println("quit detected. Stopping ops")
			ble.Stop()
			os.Exit(0)
		case active := <-hesp.updateCh:
			if active {
				hesp.updateState("active")
			} else {
				hesp.updateState("inactive")
			}
		}
	}
}

func (h Hesperus) scan(a ble.Advertisement) {
	if len(a.ManufacturerData()) > 0 {
		data := a.ManufacturerData()
		if len(a.ManufacturerData()) < 25 || binary.BigEndian.Uint32(a.ManufacturerData()) != 0x4c000215 {
			return
		} else {
			uuid := strings.ToUpper(hex.EncodeToString(data[4:8]) + "-" + hex.EncodeToString(data[8:10]) + "-" + hex.EncodeToString(data[10:12]) + "-" + hex.EncodeToString(data[12:14]) + "-" + hex.EncodeToString(data[14:20]))
			major := binary.BigEndian.Uint16(data[20:22])
			minor := binary.BigEndian.Uint16(data[22:24])

			for k, v := range h.beacons {
				if v.match(uuid, major, minor, a.RSSI()) {
					if _, ok := h.activityMap[uuid]; !ok {
						fmt.Printf("discovered new activity for %s: RSSI %d\n", k, a.RSSI())
						h.updateCh <- true
					}
					h.activityMap[uuid] = time.Now()
				}
			}

		}
	}
}

func (h Hesperus) updateState(state string) (err error) {
	stateUpdate := StateUpdate{
		State: state,
		Attributes: map[string]interface{}{
			"last_seen": time.Now().UTC().Format(time.RFC3339),
			"state":     state,
		},
	}

	data, err := json.Marshal(&stateUpdate)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/api/states/%s.occupancy", h.config.HASSURL, h.config.HASSEntityName), bytes.NewBuffer(data))
	if err != nil {
		return
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", h.config.HASSAPIToken))
	req.Header.Add("Content-type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	defer res.Body.Close()
	data, err = io.ReadAll(res.Body)
	if err != nil || res.StatusCode < 200 || res.StatusCode > 299 {
		return fmt.Errorf("failed to update state entity: %s\n%v", string(data), err)
	}

	return
}
