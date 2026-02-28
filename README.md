# zaehler2mqtt

SML smart meter to MQTT bridge with Home Assistant auto-discovery. Reads SML data from serial IR readers and publishes meter values to an MQTT broker.

## Features

- Reads SML V1.04 from multiple serial IR readers concurrently
- Publishes to MQTT with Home Assistant auto-discovery (retained config, device grouping, `state_class`)
- HTTP JSON API for current meter values
- YAML configuration
- Runs as systemd service

## Quick Start

```bash
git clone https://github.com/petesahatt/zaehler2mqtt.git
cd zaehler2mqtt
sudo make install
# edit /etc/zaehler2mqtt/config.yaml
sudo make enable
```

## Build

```bash
make build
```

## Configuration

```bash
# edit the installed config
sudo nano /etc/zaehler2mqtt/config.yaml
```

See `config.example.yaml` for all available options. Each meter entry defines:

- `device` — serial device path (e.g. `/dev/ttyUSB0`)
- `values` — list of OBIS codes to read, each with `name`, `device_class`, `state_class`, and `unit`
- `factor` — optional correction factor per value (default: 1.0)

Common OBIS codes for German smart meters:

| OBIS | Description |
|------|-------------|
| `1.0.1.8.0` | Bezug (energy consumed) |
| `1.0.2.8.0` | Einspeisung (energy fed back) |
| `1.0.16.7.0` | Leistung (current power) |

## Usage

```bash
# run with default config.yaml in current directory
./zaehler2mqtt

# run with explicit config path
./zaehler2mqtt -config /etc/zaehler2mqtt/config.yaml
```

The HTTP API is available at the configured listen address (default `:8081`):

```bash
curl http://localhost:8081/
```

## Install / Uninstall

```bash
sudo make install      # build, create user, install binary + config + service + udev
sudo make enable       # enable and start the service
sudo make status       # check service status
sudo make disable      # stop and disable
sudo make uninstall    # remove binary + service (config preserved)
```

Logs:

```bash
journalctl -u zaehler2mqtt -f
```

## MQTT Topics

State values are published to:

```
zaehler2mqtt/{meter}/{value}/state
```

Home Assistant discovery configs are published (retained) to:

```
homeassistant/sensor/zaehler2mqtt_{meter}_{value}/config
```

## License

MIT
