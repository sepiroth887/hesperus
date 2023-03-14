package main

import (
	"strings"
	"time"
)

type StateUpdate struct {
	State      string                 `json:"state"`
	Attributes map[string]interface{} `json:"attributes"`
}

type Config struct {
	HassUpdateInterval time.Duration `yaml:"hassUpdateInterval"`
	HASSAPIToken       string        `yaml:"hassAPIToken"`
	HASSEntityName     string        `yaml:"hassEntity"`
	HASSURL            string        `yaml:"hassURL"`
	IBeacons           []struct {
		Name    string `yaml:"name"`
		UUID    string `yaml:"UUID"`
		Major   int    `yaml:"major"`
		Minor   int    `yaml:"minor"`
		MinRSSI int    `yaml:"minRSSI"`
	} `yaml:"iBeacons"`
}

type IBeacon struct {
	UUID    string
	Minor   uint16
	Major   uint16
	MinRSSI int
}

func (i IBeacon) match(uuid string, major, minor uint16, rssi int) bool {
	return uuid == strings.ToUpper(i.UUID) && major == i.Major && minor == i.Minor && rssi > i.MinRSSI
}
