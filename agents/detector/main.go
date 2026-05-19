package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
)

type Metrics struct {
	CPU       float64   `json:"cpu"`
	Memory    float64   `json:"memory"`
	Network   float64   `json:"network"`
	Timestamp time.Time `json:"timestamp"`
}

type AnomalyDetectionResult struct {
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Anomaly    bool      `json:"anomaly"`
	Score      float64   `json:"score"`
	Timestamp  time.Time `json:"timestamp"`
}

func main() {
	// Connect to NATS
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	// Create consumer for metrics stream
	_, err = js.Subscribe("METRICS.RAW", func(msg *nats.Msg) {
		ctx := context.Background()
		tracer := otel.Tracer("detector-agent")
		ctx, span := tracer.Start(ctx, "process-metrics")
		defer span.End()

		var metrics Metrics
		err := json.Unmarshal(msg.Data, &metrics)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error unmarshaling metrics: %v", err)
			msg.Nak()
			return
		}

		// Simple anomaly detection (values > 0.8 are considered anomalous)
		results := []AnomalyDetectionResult{}
		if metrics.CPU > 0.8 {
			results = append(results, AnomalyDetectionResult{
				MetricName: "CPU",
				Value:      metrics.CPU,
				Anomaly:    true,
				Score:      metrics.CPU,
				Timestamp:  time.Now(),
			})
		}
		if metrics.Memory > 0.8 {
			results = append(results, AnomalyDetectionResult{
				MetricName: "Memory",
				Value:      metrics.Memory,
				Anomaly:    true,
				Score:      metrics.Memory,
				Timestamp:  time.Now(),
			})
		}
		if metrics.Network > 0.8 {
			results = append(results, AnomalyDetectionResult{
				MetricName: "Network",
				Value:      metrics.Network,
				Anomaly:    true,
				Score:      metrics.Network,
				Timestamp:  time.Now(),
			})
		}

		// If anomalies detected, publish to alert stream
		if len(results) > 0 {
			data, _ := json.Marshal(results)
			_, err = js.Publish("ALERTS.DETECTED", data)
			if err != nil {
				span.RecordError(err)
				log.Printf("Error publishing alerts: %v", err)
			}
		}

		// Acknowledge message
		msg.Ack()
	}, nats.Bind("metrics_consumer", "detector"))

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Detector agent started, waiting for metrics...")
	select {}
}
