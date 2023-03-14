## About

Fairly simple bluetooth scanner to track iBeacon presence and report them as occupancy values (active/inactive) to homeassistant using the homeassistant API

## Usage
Build using go > v1.15 `go build .`
Create valid config using sample_config.yml as reference
and place as `config.yml` in same folder as hesperus binary

launch using sudo (needed to open the linux device at least on my raspberry and didn't check what capabilities are needed to run without)