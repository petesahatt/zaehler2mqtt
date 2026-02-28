package main

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// OBISBytes
// ---------------------------------------------------------------------------

func TestOBISBytes_Valid(t *testing.T) {
	v := ValueConfig{OBIS: "1.0.1.8.0"}
	got, err := v.OBISBytes()
	if err != nil {
		t.Fatalf("OBISBytes error: %v", err)
	}
	want := []byte{1, 0, 1, 8, 0}
	if len(got) != len(want) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestOBISBytes_Power(t *testing.T) {
	v := ValueConfig{OBIS: "1.0.16.7.0"}
	got, err := v.OBISBytes()
	if err != nil {
		t.Fatalf("OBISBytes error: %v", err)
	}
	want := []byte{1, 0, 16, 7, 0}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("byte %d: got %d, want %d", i, got[i], want[i])
		}
	}
}

func TestOBISBytes_InvalidLetter(t *testing.T) {
	v := ValueConfig{OBIS: "1.0.X.8.0"}
	_, err := v.OBISBytes()
	if err == nil {
		t.Fatal("expected error for invalid OBIS code")
	}
}

func TestOBISBytes_OutOfRange(t *testing.T) {
	v := ValueConfig{OBIS: "1.0.256.8.0"}
	_, err := v.OBISBytes()
	if err == nil {
		t.Fatal("expected error for OBIS byte > 255")
	}
}

func TestOBISBytes_Negative(t *testing.T) {
	v := ValueConfig{OBIS: "1.0.-1.8.0"}
	_, err := v.OBISBytes()
	if err == nil {
		t.Fatal("expected error for negative OBIS byte")
	}
}

func TestOBISBytes_SingleByte(t *testing.T) {
	v := ValueConfig{OBIS: "42"}
	got, err := v.OBISBytes()
	if err != nil {
		t.Fatalf("OBISBytes error: %v", err)
	}
	if len(got) != 1 || got[0] != 42 {
		t.Fatalf("expected [42], got %v", got)
	}
}

// ---------------------------------------------------------------------------
// LoadConfig
// ---------------------------------------------------------------------------

func writeTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadConfig_Valid(t *testing.T) {
	yaml := `
mqtt:
  broker: "tcp://localhost:1883"
  username: "user"
  password: "pass"
http:
  listen: ":8081"
meters:
  - name: nutzstrom
    device: /dev/ttyUSB0
    values:
      - obis: "1.0.1.8.0"
        name: Bezug
        device_class: energy
        state_class: total_increasing
        unit: kWh
        factor: 0.001
      - obis: "1.0.16.7.0"
        name: Leistung
        device_class: power
        state_class: measurement
        unit: W
`
	cfg, err := LoadConfig(writeTestConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.MQTT.Broker != "tcp://localhost:1883" {
		t.Fatalf("broker = %q", cfg.MQTT.Broker)
	}
	if cfg.MQTT.ClientID != "zaehler2mqtt" {
		t.Fatalf("client_id should default to zaehler2mqtt, got %q", cfg.MQTT.ClientID)
	}
	if len(cfg.Meters) != 1 {
		t.Fatalf("expected 1 meter, got %d", len(cfg.Meters))
	}
	m := cfg.Meters[0]
	if m.Name != "nutzstrom" {
		t.Fatalf("meter name = %q", m.Name)
	}
	if len(m.Values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(m.Values))
	}
	if m.Values[0].Factor != 0.001 {
		t.Fatalf("Bezug factor = %f, want 0.001", m.Values[0].Factor)
	}
	// Leistung has no factor → should default to 1.0
	if m.Values[1].Factor != 1.0 {
		t.Fatalf("Leistung factor = %f, want 1.0", m.Values[1].Factor)
	}
}

func TestLoadConfig_DefaultHTTPListen(t *testing.T) {
	yaml := `
mqtt:
  broker: "tcp://localhost:1883"
  username: "user"
  password: "pass"
meters: []
`
	cfg, err := LoadConfig(writeTestConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.HTTP.Listen != ":8080" {
		t.Fatalf("default listen = %q, want :8080", cfg.HTTP.Listen)
	}
}

func TestLoadConfig_RejectsCHANGE_ME(t *testing.T) {
	yaml := `
mqtt:
  broker: "tcp://localhost:1883"
  username: "CHANGE_ME"
  password: "CHANGE_ME"
meters: []
`
	_, err := LoadConfig(writeTestConfig(t, yaml))
	if err == nil {
		t.Fatal("expected error for CHANGE_ME credentials")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	_, err := LoadConfig(writeTestConfig(t, "{{{{ not yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadConfig_CustomClientID(t *testing.T) {
	yaml := `
mqtt:
  broker: "tcp://localhost:1883"
  client_id: "my-custom-id"
  username: "user"
  password: "pass"
meters: []
`
	cfg, err := LoadConfig(writeTestConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if cfg.MQTT.ClientID != "my-custom-id" {
		t.Fatalf("client_id = %q, want my-custom-id", cfg.MQTT.ClientID)
	}
}

func TestLoadConfig_MultipleMeters(t *testing.T) {
	yaml := `
mqtt:
  broker: "tcp://localhost:1883"
  username: "user"
  password: "pass"
meters:
  - name: nutzstrom
    device: /dev/ttyUSB0
    values:
      - obis: "1.0.1.8.0"
        name: Bezug
        unit: kWh
  - name: waermestrom
    device: /dev/ttyUSB1
    values:
      - obis: "1.0.1.8.0"
        name: Bezug
        unit: kWh
`
	cfg, err := LoadConfig(writeTestConfig(t, yaml))
	if err != nil {
		t.Fatalf("LoadConfig error: %v", err)
	}
	if len(cfg.Meters) != 2 {
		t.Fatalf("expected 2 meters, got %d", len(cfg.Meters))
	}
	if cfg.Meters[1].Name != "waermestrom" {
		t.Fatalf("second meter name = %q", cfg.Meters[1].Name)
	}
}
