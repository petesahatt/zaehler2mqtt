package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"
)

type MeterValue struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	OBIS  string  `json:"obis"`
}

type MeterState struct {
	Device     string                `json:"device"`
	LastUpdate time.Time             `json:"last_update"`
	Values     map[string]MeterValue `json:"values"`
}

type Server struct {
	listen string
	server *http.Server
	mu     sync.RWMutex
	meters map[string]*MeterState
}

func NewServer(listen string) *Server {
	s := &Server{
		listen: listen,
		meters: make(map[string]*MeterState),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	s.server = &http.Server{
		Addr:    listen,
		Handler: mux,
	}
	return s
}

func (s *Server) Start() {
	log.Printf("HTTP server listening on %s", s.listen)
	if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
		log.Printf("HTTP server error: %v", err)
	}
}

func (s *Server) Stop(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	s.server.Shutdown(shutdownCtx)
}

func (s *Server) RegisterMeter(meterName string, device string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.meters[meterName]; !ok {
		s.meters[meterName] = &MeterState{
			Device: device,
			Values: make(map[string]MeterValue),
		}
	}
}

func (s *Server) UpdateValue(meterName, valueName string, value float64, unit string, obis string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state, ok := s.meters[meterName]
	if !ok {
		return
	}
	state.LastUpdate = time.Now()
	state.Values[valueName] = MeterValue{
		Value: value,
		Unit:  unit,
		OBIS:  obis,
	}
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"meters": s.meters,
	})
}
