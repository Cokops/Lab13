package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
)

type Alert struct {
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Anomaly    bool      `json:"anomaly"`
	Score      float64   `json:"score"`
	Timestamp  time.Time `json:"timestamp"`
}

type Notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	RawAlerts []Alert   `json:"raw_alerts"`
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

	// Subscribe to alert detection stream
	_, err = js.Subscribe("ALERTS.DETECTED", func(msg *nats.Msg) {
		ctx := context.Background()
		tracer := otel.Tracer("notifier-agent")
		ctx, span := tracer.Start(ctx, "process-alerts")
		defer span.End()

		var alerts []Alert
		err := json.Unmarshal(msg.Data, &alerts)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error unmarshaling alerts: %v", err)
			msg.Nak()
			return
		}

		// Create notification
		severity := "warning"
		if len(alerts) > 1 {
			severity = "critical"
		}

		notification := Notification{
			ID:        generateID(),
			Type:      "infrastructure-alert",
			Severity:  severity,
			Message:   createAlertMessage(alerts),
			Timestamp: time.Now(),
			RawAlerts: alerts,
		}

		// Marshal and publish notification
		data, _ := json.Marshal(notification)
		_, err = js.Publish("NOTIFICATIONS.SENT", data)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error publishing notification: %v", err)
		}

		log.Printf("Sent %s alert: %s", severity, notification.Message)

		// Simulate external notification (email, webhook, etc.)
		// In a real system, this would send emails, Slack messages, etc.
		time.Sleep(100 * time.Millisecond)

		// Acknowledge message
		msg.Ack()
	}, nats.Bind("alerts_consumer", "notifier"))

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Notifier agent started, waiting for alerts...")
	select {}
}

func generateID() string {
	return "not_" + fmt.Sprintf("%d", time.Now().UnixNano())
}

func createAlertMessage(alerts []Alert) string {
	if len(alerts) == 1 {
		return fmt.Sprintf("High %s usage detected: %.2f%% at %s",
			alerts[0].MetricName, alerts[0].Value*100, alerts[0].Timestamp.Format(time.RFC3339))
	}
	return fmt.Sprintf("Multiple anomalies detected: %d metrics above threshold at %s",
		len(alerts), alerts[0].Timestamp.Format(time.RFC3339))
}
