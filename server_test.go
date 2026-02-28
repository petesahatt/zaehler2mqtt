package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

func TestServer_EmptyMeters(t *testing.T) {
	srv := NewServer(":0")
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleRoot(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type = %q", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	meters, ok := resp["meters"]
	if !ok {
		t.Fatal("missing 'meters' key")
	}
	m := meters.(map[string]interface{})
	if len(m) != 0 {
		t.Fatalf("expected empty meters, got %d", len(m))
	}
}

func TestServer_RegisterAndUpdate(t *testing.T) {
	srv := NewServer(":0")
	srv.RegisterMeter("nutzstrom", "/dev/ttyUSB0")
	srv.UpdateValue("nutzstrom", "Bezug", 8782.4, "kWh", "1-0:1.8.0*255")
	srv.UpdateValue("nutzstrom", "Leistung", 246.0, "W", "1-0:16.7.0*255")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleRoot(rr, req)

	var resp map[string]json.RawMessage
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	var meters map[string]MeterState
	if err := json.Unmarshal(resp["meters"], &meters); err != nil {
		t.Fatalf("invalid meters JSON: %v", err)
	}

	state, ok := meters["nutzstrom"]
	if !ok {
		t.Fatal("missing nutzstrom meter")
	}
	if state.Device != "/dev/ttyUSB0" {
		t.Fatalf("device = %q", state.Device)
	}
	if len(state.Values) != 2 {
		t.Fatalf("expected 2 values, got %d", len(state.Values))
	}

	bezug := state.Values["Bezug"]
	if bezug.Value != 8782.4 {
		t.Fatalf("Bezug = %f", bezug.Value)
	}
	if bezug.Unit != "kWh" {
		t.Fatalf("Bezug unit = %q", bezug.Unit)
	}
	if bezug.OBIS != "1-0:1.8.0*255" {
		t.Fatalf("Bezug OBIS = %q", bezug.OBIS)
	}

	leistung := state.Values["Leistung"]
	if leistung.Value != 246.0 {
		t.Fatalf("Leistung = %f, want 246.0 (not 2460)", leistung.Value)
	}
}

func TestServer_UpdateUnregistered(t *testing.T) {
	srv := NewServer(":0")
	// Should not panic
	srv.UpdateValue("nonexistent", "Bezug", 100.0, "kWh", "1-0:1.8.0*255")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleRoot(rr, req)

	var resp map[string]map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(resp["meters"]) != 0 {
		t.Fatalf("expected no meters, got %d", len(resp["meters"]))
	}
}

func TestServer_MultipleMeters(t *testing.T) {
	srv := NewServer(":0")
	srv.RegisterMeter("nutzstrom", "/dev/ttyUSB0")
	srv.RegisterMeter("waermestrom", "/dev/ttyUSB1")
	srv.UpdateValue("nutzstrom", "Bezug", 8782.4, "kWh", "1-0:1.8.0*255")
	srv.UpdateValue("waermestrom", "Bezug", 17271.4, "kWh", "1-0:1.8.0*255")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleRoot(rr, req)

	var resp map[string]json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &resp)
	var meters map[string]MeterState
	json.Unmarshal(resp["meters"], &meters)

	if len(meters) != 2 {
		t.Fatalf("expected 2 meters, got %d", len(meters))
	}
	if meters["nutzstrom"].Values["Bezug"].Value != 8782.4 {
		t.Fatalf("nutzstrom Bezug = %f", meters["nutzstrom"].Values["Bezug"].Value)
	}
	if meters["waermestrom"].Values["Bezug"].Value != 17271.4 {
		t.Fatalf("waermestrom Bezug = %f", meters["waermestrom"].Values["Bezug"].Value)
	}
}

func TestServer_OverwriteValue(t *testing.T) {
	srv := NewServer(":0")
	srv.RegisterMeter("nutzstrom", "/dev/ttyUSB0")
	srv.UpdateValue("nutzstrom", "Leistung", 100.0, "W", "1-0:16.7.0*255")
	srv.UpdateValue("nutzstrom", "Leistung", 250.0, "W", "1-0:16.7.0*255")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleRoot(rr, req)

	var resp map[string]json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &resp)
	var meters map[string]MeterState
	json.Unmarshal(resp["meters"], &meters)

	if meters["nutzstrom"].Values["Leistung"].Value != 250.0 {
		t.Fatalf("Leistung should be 250.0 (latest), got %f", meters["nutzstrom"].Values["Leistung"].Value)
	}
}

func TestServer_RegisterIdempotent(t *testing.T) {
	srv := NewServer(":0")
	srv.RegisterMeter("nutzstrom", "/dev/ttyUSB0")
	srv.UpdateValue("nutzstrom", "Bezug", 100.0, "kWh", "1-0:1.8.0*255")
	// Register again — must not reset values
	srv.RegisterMeter("nutzstrom", "/dev/ttyUSB0")

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	srv.handleRoot(rr, req)

	var resp map[string]json.RawMessage
	json.Unmarshal(rr.Body.Bytes(), &resp)
	var meters map[string]MeterState
	json.Unmarshal(resp["meters"], &meters)

	if meters["nutzstrom"].Values["Bezug"].Value != 100.0 {
		t.Fatal("re-registering meter should not reset values")
	}
}
