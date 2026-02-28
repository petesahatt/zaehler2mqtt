package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/petesahatt/gosml"
)

func configureSerial(device string) error {
	cmd := exec.Command("stty", "-F", device, "9600", "cs8", "-cstopb", "-parenb", "raw")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func RunMeter(ctx context.Context, cfg MeterConfig, pub *Publisher, srv *Server) {
	log.Printf("[%s] Starting meter reader on %s", cfg.Name, cfg.Device)
	srv.RegisterMeter(cfg.Name, cfg.Device)

	for {
		if ctx.Err() != nil {
			return
		}

		if err := configureSerial(cfg.Device); err != nil {
			log.Printf("[%s] Failed to configure serial: %v", cfg.Name, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		f, err := os.OpenFile(cfg.Device, os.O_RDONLY|syscall.O_NOCTTY, 0666)
		if err != nil {
			log.Printf("[%s] Failed to open device: %v", cfg.Name, err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				continue
			}
		}

		// Build OBIS callbacks
		readOpts := []gosml.ReadOption{}
		for _, v := range cfg.Values {
			obis, err := v.OBISBytes()
			if err != nil {
				log.Printf("[%s] Invalid OBIS code %s: %v", cfg.Name, v.OBIS, err)
				continue
			}
			val := v // capture for closure
			meterName := cfg.Name
			readOpts = append(readOpts, gosml.WithObisCallback(gosml.OctetString(obis), func(entry *gosml.ListEntry) {
				floatVal := entry.Float() * val.Factor
				pub.PublishState(meterName, val.Name, floatVal)
				srv.UpdateValue(meterName, val.Name, floatVal, val.Unit, entry.ObjectName())
			}))
		}

		// Publish HA discovery for all values of this meter
		for _, v := range cfg.Values {
			sensorID := fmt.Sprintf("zaehler2mqtt_%s_%s", cfg.Name, v.Name)
			pub.PublishDiscovery(cfg.Name, sensorID, v)
		}

		log.Printf("[%s] Reading SML data from %s", cfg.Name, cfg.Device)

		// Close file on context cancellation to unblock Read
		go func() {
			<-ctx.Done()
			f.Close()
		}()

		r := bufio.NewReader(f)
		err = gosml.Read(r, readOpts...)
		f.Close()

		if ctx.Err() != nil {
			log.Printf("[%s] Shutting down", cfg.Name)
			return
		}

		log.Printf("[%s] Read error: %v, restarting in 5s", cfg.Name, err)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}
