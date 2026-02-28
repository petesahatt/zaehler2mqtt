package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Publisher struct {
	client mqtt.Client
}

func NewPublisher(cfg MQTTConfig) (*Publisher, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second)

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	log.Printf("Connected to MQTT broker %s", cfg.Broker)
	return &Publisher{client: client}, nil
}

func (p *Publisher) Close() {
	p.client.Disconnect(1000)
}

func (p *Publisher) PublishDiscovery(meterName string, sensorID string, val ValueConfig) {
	topic := fmt.Sprintf("homeassistant/sensor/%s/config", sensorID)

	payload := map[string]interface{}{
		"name":                val.Name,
		"unique_id":           sensorID,
		"state_topic":         fmt.Sprintf("zaehler2mqtt/%s/%s/state", meterName, val.Name),
		"value_template":      "{{ value }}",
		"device_class":        val.DeviceClass,
		"unit_of_measurement": val.Unit,
		"device": map[string]interface{}{
			"identifiers":  []string{fmt.Sprintf("zaehler2mqtt_%s", meterName)},
			"name":         meterName,
			"manufacturer": "zaehler2mqtt",
			"model":        "SML Meter Reader",
		},
	}
	if val.StateClass != "" {
		payload["state_class"] = val.StateClass
	}

	data, _ := json.Marshal(payload)
	token := p.client.Publish(topic, 1, true, data)
	token.WaitTimeout(5 * time.Second)
	if token.Error() != nil {
		log.Printf("Failed to publish discovery for %s: %v", sensorID, token.Error())
	} else {
		log.Printf("Published HA discovery: %s", sensorID)
	}
}

func (p *Publisher) PublishState(meterName string, valueName string, value float64) {
	topic := fmt.Sprintf("zaehler2mqtt/%s/%s/state", meterName, valueName)
	payload := fmt.Sprintf("%.4f", value)
	token := p.client.Publish(topic, 0, false, payload)
	token.WaitTimeout(50 * time.Millisecond)
}
