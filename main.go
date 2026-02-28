package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Meters) == 0 {
		log.Fatal("No meters configured")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect to MQTT broker
	pub, err := NewPublisher(cfg.MQTT)
	if err != nil {
		log.Fatalf("Failed to connect to MQTT broker: %v", err)
	}
	defer pub.Close()

	// Start HTTP server
	srv := NewServer(cfg.HTTP.Listen)
	go srv.Start()

	// Start a reader goroutine per meter
	var wg sync.WaitGroup
	for _, meterCfg := range cfg.Meters {
		wg.Add(1)
		go func(mc MeterConfig) {
			defer wg.Done()
			RunMeter(ctx, mc, pub, srv)
		}(meterCfg)
	}

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received %v, shutting down...", sig)

	cancel()
	wg.Wait()
	srv.Stop(context.Background())
	log.Println("Shutdown complete")
}
