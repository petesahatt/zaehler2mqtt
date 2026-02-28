BINARY    := zaehler2mqtt
PREFIX    := /usr/local
BINDIR    := $(PREFIX)/bin
CONFDIR   := /etc/$(BINARY)
UNITDIR   := /etc/systemd/system
UDEVDIR   := /etc/udev/rules.d
USER      := $(BINARY)
GROUP     := $(BINARY)

.PHONY: build clean install uninstall enable disable status check-config help

help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "  build          Build the binary"
	@echo "  clean          Remove the binary"
	@echo "  check-config   Validate config.yaml"
	@echo "  install        Build, check config, install binary + service + udev (requires root)"
	@echo "  uninstall      Stop service, remove binary + service + udev (config preserved)"
	@echo "  enable         Enable and start the service"
	@echo "  disable        Stop and disable the service"
	@echo "  status         Show service status"
	@echo ""
	@echo "Typical workflow:"
	@echo "  make build"
	@echo "  cp config.example.yaml config.yaml && nano config.yaml"
	@echo "  sudo make install"
	@echo "  sudo make enable"

build:
	go build -o $(BINARY) .
	@echo ""
	@echo "Build complete. Next steps:"
	@echo ""
	@echo "  1. Create config:  cp config.example.yaml config.yaml"
	@echo "     Set mqtt.broker, mqtt.username, mqtt.password"
	@echo "     Set meter device paths (ls /dev/ttyUSB*)"
	@echo ""
	@echo "  2. Adjust udev rules if needed:  nano 99-$(BINARY).rules"
	@echo "     Find your adapter:  lsusb"
	@echo "     Check attrs:        udevadm info -a /dev/ttyUSB0 | grep -E 'idVendor|idProduct'"
	@echo "     Install manually:   sudo cp 99-$(BINARY).rules $(UDEVDIR)/ && sudo udevadm control --reload-rules"
	@echo ""
	@echo "  3. Install service:  sudo make install"

clean:
	rm -f $(BINARY)

check-config:
	@if [ ! -f config.yaml ]; then \
		echo "ERROR: config.yaml not found."; \
		echo "  Run: cp config.example.yaml config.yaml"; \
		echo "  Then edit config.yaml with your MQTT credentials and meter devices."; \
		exit 1; \
	fi
	@if grep -q 'CHANGE_ME' config.yaml; then \
		echo "ERROR: config.yaml still contains CHANGE_ME placeholders."; \
		echo "  Edit config.yaml and set mqtt.username and mqtt.password."; \
		exit 1; \
	fi
	@if grep -q 'localhost' config.yaml; then \
		echo "WARNING: mqtt.broker is set to localhost — adjust if your broker is remote."; \
	fi
	@echo "Config OK."

install: build check-config
	@echo "==> Creating service user..."
	id -u $(USER) >/dev/null 2>&1 || useradd -r -s /usr/sbin/nologin $(USER)
	usermod -aG dialout $(USER)
	@echo "==> Installing binary..."
	install -m 755 $(BINARY) $(BINDIR)/$(BINARY)
	@echo "==> Installing config..."
	install -d -m 755 $(CONFDIR)
	test -f $(CONFDIR)/config.yaml || install -m 640 -g $(GROUP) config.yaml $(CONFDIR)/config.yaml
	@echo "==> Installing systemd service..."
	install -m 644 $(BINARY).service $(UNITDIR)/$(BINARY).service
	@echo "==> Installing udev rule..."
	install -m 644 99-$(BINARY).rules $(UDEVDIR)/99-$(BINARY).rules
	udevadm control --reload-rules 2>/dev/null || true
	@echo "==> Reloading systemd..."
	systemctl daemon-reload
	@echo ""
	@echo "Installation complete."
	@echo "  Run: sudo make enable"

uninstall:
	@echo "==> Stopping service..."
	-systemctl stop $(BINARY) 2>/dev/null
	-systemctl disable $(BINARY) 2>/dev/null
	@echo "==> Removing files..."
	rm -f $(BINDIR)/$(BINARY)
	rm -f $(UNITDIR)/$(BINARY).service
	rm -f $(UDEVDIR)/99-$(BINARY).rules
	systemctl daemon-reload
	udevadm control --reload-rules 2>/dev/null || true
	@echo "==> Done. Config in $(CONFDIR) preserved."

enable:
	systemctl enable --now $(BINARY)

disable:
	systemctl disable --now $(BINARY)

status:
	systemctl status $(BINARY)
