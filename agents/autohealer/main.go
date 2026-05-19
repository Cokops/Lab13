package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
)

type Notification struct {
	Severity  string     `json:"severity"`
	RawAlerts []RawAlert `json:"raw_alerts"`
}

type RawAlert struct {
	MetricName string  `json:"metric_name"`
	Value      float64 `json:"value"`
}

type Incident struct {
	ID       string    `json:"id"`
	Metric   string    `json:"metric"`
	Value    float64   `json:"value"`
	Detected time.Time `json:"detected"`
	Resolved time.Time `json:"resolved,omitempty"`
	Status   string    `json:"status"`
	Attempts int       `json:"attempts"`
}

type HealingAction struct {
	IncidentID string    `json:"incident_id"`
	Action     string    `json:"action"`
	Timestamp  time.Time `json:"timestamp"`
	Success    bool      `json:"success"`
	Output     string    `json:"output,omitempty"`
}

var rdb *redis.Client

func init() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	// Используем context.Background() напрямую
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		log.Printf("Redis connection error: %v", err)
	}
}

func getIncidentKey(id string) string {
	return "incident:" + id
}

func loadIncident(id string) (*Incident, error) {
	var incident Incident
	val, err := rdb.Get(context.Background(), getIncidentKey(id)).Result()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(val), &incident)
	return &incident, err
}

func saveIncident(incident *Incident) error {
	data, err := json.Marshal(incident)
	if err != nil {
		return err
	}
	return rdb.Set(context.Background(), getIncidentKey(incident.ID), data, 24*time.Hour).Err()
}

func attemptHealing(incident *Incident) HealingAction {
	tracer := otel.Tracer("autohealer-agent")
	_, span := tracer.Start(context.Background(), "attempt-healing")
	defer span.End()

	var action string
	var success bool
	var output string

	switch incident.Metric {
	case "CPU":
		action = "restart_high_cpu_process"
		success = rand.Float64() < 0.7
		if success {
			output = "High CPU process identified and restarted successfully"
		} else {
			output = "Attempted to restart high CPU process but it failed to terminate"
		}
	case "Memory":
		action = "clear_memory_cache"
		success = true
		output = "System memory cache cleared successfully"
	case "Network":
		action = "reset_network_interface"
		success = rand.Float64() < 0.8
		if success {
			output = "Network interface reset and connectivity restored"
		} else {
			output = "Network interface reset failed, hardware issue suspected"
		}
	default:
		action = "generic_recovery_procedure"
		success = rand.Float64() < 0.5
		output = "Executed generic recovery procedure"
	}

	time.Sleep(1 * time.Second)

	return HealingAction{
		IncidentID: incident.ID,
		Action:     action,
		Timestamp:  time.Now(),
		Success:    success,
		Output:     output,
	}
}

func main() {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}

	_, err = js.Subscribe("NOTIFICATIONS.SENT", func(msg *nats.Msg) {
		tracer := otel.Tracer("autohealer-agent")
		_, span := tracer.Start(context.Background(), "handle-notification") // Убрали переменную ctx
		defer span.End()

		var notification Notification
		err := json.Unmarshal(msg.Data, &notification)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error unmarshaling notification: %v", err)
			msg.Nak()
			return
		}

		if notification.Severity != "critical" {
			msg.Ack()
			return
		}

		if len(notification.RawAlerts) == 0 {
			log.Printf("No raw alerts in notification")
			msg.Ack()
			return
		}

		incident := &Incident{
			ID:       "inc_" + fmt.Sprintf("%d", time.Now().UnixNano()),
			Metric:   notification.RawAlerts[0].MetricName,
			Value:    notification.RawAlerts[0].Value,
			Detected: time.Now(),
			Status:   "active",
			Attempts: 0,
		}

		err = saveIncident(incident)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error saving incident to Redis: %v", err)
		}

		var lastAction *HealingAction
		for attempt := 0; attempt < 3; attempt++ {
			incident.Attempts++
			action := attemptHealing(incident)
			lastAction = &action

			incident.Status = "resolved"
			if !action.Success {
				incident.Status = "active"
			}

			err = saveIncident(incident)
			if err != nil {
				log.Printf("Error saving updated incident: %v", err)
			}

			if action.Success {
				break
			}

			time.Sleep(2 * time.Second)
		}

		result := map[string]interface{}{
			"incident_id": incident.ID,
			"status":      incident.Status,
			"attempts":    incident.Attempts,
			"last_action": lastAction,
			"timestamp":   time.Now(),
		}
		data, _ := json.Marshal(result)
		_, err = js.Publish("HEALING.RESULTS", data)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error publishing healing result: %v", err)
		}

		log.Printf("Healing attempt completed for incident %s: %s (%d attempts)",
			incident.ID, incident.Status, incident.Attempts)

		msg.Ack()
	}, nats.Bind("notifications_consumer", "autohealer"))

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Autohealer agent started, waiting for critical notifications...")
	select {}
}
