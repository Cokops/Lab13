package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

type MonitoringTask struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // metrics, alert, healing, etc.
	Payload   map[string]interface{} `json:"payload"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Priority  int                    `json:"priority"` // 1-10, 10 highest
}

type AgentInfo struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	LastSeen     time.Time `json:"last_seen"`
	Load         int       `json:"load"`         // current tasks being processed
	MaxLoad      int       `json:"max_load"`     // maximum concurrent tasks
	Capabilities []string  `json:"capabilities"` // what types of tasks it can handle
	Cost         float64   `json:"cost"`         // cost per task unit
	Skill        float64   `json:"skill"`        // proficiency in its domain (0-1)
	Available    bool      `json:"available"`    // true if can accept new tasks
}

type AuctionBid struct {
	TaskID     string  `json:"task_id"`
	AgentID    string  `json:"agent_id"`
	Bid        float64 `json:"bid"`        // calculated cost/suitability score
	Confidence float64 `json:"confidence"` // 0-1, how confident agent is
}

type AuctionResult struct {
	TaskID     string    `json:"task_id"`
	AgentID    string    `json:"agent_id"`
	WinningBid float64   `json:"winning_bid"`
	Timestamp  time.Time `json:"timestamp"`
}

var (
	agents      = make(map[string]*AgentInfo)
	agentsMutex sync.RWMutex

	taskQueue = make(chan *MonitoringTask, 1000)
	random    = rand.New(rand.NewSource(time.Now().UnixNano()))
)

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

	// Create streams if they don't exist
	createStreams(js)

	// Start agent registry
	startAgentRegistry(nc, js)

	// Start auction system
	startAuctionSystem(nc, js)

	// Start dynamic scaling monitor
	startScalingMonitor(js)

	// Start web API (in real system)
	// go startWebAPI()

	log.Println("Orchestrator started, managing agent ecosystem...")
	select {}
}

func createStreams(js jetstream.JetStream) {
	// Metrics stream
	_, err := js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "METRICS",
		Subjects: []string{"METRICS.RAW", "METRICS.PROCESSED"},
	})
	if err != nil && !isStreamExists(err) {
		log.Printf("Error creating METRICS stream: %v", err)
	}

	// Alerts stream
	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "ALERTS",
		Subjects: []string{"ALERTS.DETECTED", "ALERTS.SUPPRESSED"},
	})
	if err != nil && !isStreamExists(err) {
		log.Printf("Error creating ALERTS stream: %v", err)
	}

	// Notifications stream
	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "NOTIFICATIONS",
		Subjects: []string{"NOTIFICATIONS.SENT", "NOTIFICATIONS.DELIVERED"},
	})
	if err != nil && !isStreamExists(err) {
		log.Printf("Error creating NOTIFICATIONS stream: %v", err)
	}

	// Healing stream
	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "HEALING",
		Subjects: []string{"HEALING.REQUESTS", "HEALING.RESULTS"},
	})
	if err != nil && !isStreamExists(err) {
		log.Printf("Error creating HEALING stream: %v", err)
	}

	// Auction stream
	_, err = js.CreateStream(context.Background(), jetstream.StreamConfig{
		Name:     "AUCTION",
		Subjects: []string{"AUCTION.BIDS", "AUCTION.RESULTS", "AUCTION.TASKS"},
	})
	if err != nil && !isStreamExists(err) {
		log.Printf("Error creating AUCTION stream: %v", err)
	}
}

func isStreamExists(err error) bool {
	return err != nil && (err.Error() == "stream name already in use" ||
		strings.Contains(err.Error(), "already exists"))
}

func startAgentRegistry(nc *nats.Conn, js jetstream.JetStream) {
	// Subscribe to agent heartbeats
	_, err := js.Subscribe("AGENT.HEARTBEAT", func(msg *nats.Msg) {
		var agent AgentInfo
		err := json.Unmarshal(msg.Data, &agent)
		if err != nil {
			log.Printf("Error unmarshaling agent heartbeat: %v", err)
			msg.Nak()
			return
		}

		agent.LastSeen = time.Now()
		agent.Available = agent.Load < agent.MaxLoad

		// Register or update agent
		agentsMutex.Lock()
		agents[agent.ID] = &agent
		agentsMutex.Unlock()

		// Acknowledge
		msg.Ack()

		log.Printf("Registered/updated agent %s (%s)", agent.ID, agent.Type)
	})
	if err != nil {
		log.Fatal(err)
	}

	// Clean up stale agents
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for {
			<-ticker.C
			cleanupStaleAgents()
		}
	}()
}

func cleanupStaleAgents() {
	agentsMutex.Lock()
	defer agentsMutex.Unlock()

	now := time.Now()
	for id, agent := range agents {
		if now.Sub(agent.LastSeen) > 60*time.Second {
			log.Printf("Removing stale agent %s", id)
			delete(agents, id)
		}
	}
}

func startAuctionSystem(nc *nats.Conn, js jetstream.JetStream) {
	// Subscribe to new tasks that need auction
	_, err := js.Subscribe("AUCTION.TASKS", func(msg *nats.Msg) {
		ctx := context.Background()
		tracer := otel.Tracer("orchestrator")
		ctx, span := tracer.Start(ctx, "run-auction")
		span.SetAttributes(attribute.String("messaging.message.id", string(msg.Header.Get("Nats-Msg-Id"))))
		defer span.End()

		var task MonitoringTask
		err := json.Unmarshal(msg.Data, &task)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error unmarshaling auction task: %v", err)
			msg.Nak()
			return
		}

		// Collect bids from agents
		bids := collectBidsForTask(&task, js, ctx)

		if len(bids) == 0 {
			// No bids received, log and discard
			log.Printf("No bids received for task %s", task.ID)
			msg.Ack()
			return
		}

		// Select winner (lowest bid wins)
		winner := bids[0]
		for _, bid := range bids[1:] {
			if bid.Bid < winner.Bid {
				winner = bid
			}
		}

		// Publish auction result
		result := AuctionResult{
			TaskID:     task.ID,
			AgentID:    winner.AgentID,
			WinningBid: winner.Bid,
			Timestamp:  time.Now(),
		}

		data, _ := json.Marshal(result)
		_, err = js.Publish("AUCTION.RESULTS", data)
		if err != nil {
			span.RecordError(err)
			log.Printf("Error publishing auction result: %v", err)
		}

		log.Printf("Auction completed: Task %s won by %s with bid %.3f",
			task.ID, winner.AgentID, winner.Bid)

		// Acknowledge original task message
		msg.Ack()
	}, nats.Bind("auction_tasks", "orchestrator"))
	if err != nil {
		log.Fatal(err)
	}
}

func collectBidsForTask(task *MonitoringTask, js jetstream.JetStream, ctx context.Context) []AuctionBid {
	tracer := otel.Tracer("orchestrator")
	ctx, span := tracer.Start(ctx, "collect-bids")
	defer span.End()

	var bids []AuctionBid

	// Use request-reply pattern to collect bids
	reqData, _ := json.Marshal(task)
	msg, err := js.PublishRequest("AUCTION.REQUEST_BIDS", "", reqData)
	if err != nil {
		span.RecordError(err)
		log.Printf("Error requesting bids: %v", err)
		return bids
	}

	// Set up subscription to collect bids
	sub, err := js.SubscribeSync("AUCTION.BIDS")
	if err != nil {
		span.RecordError(err)
		log.Printf("Error subscribing to bids: %v", err)
		return bids
	}
	defer sub.Unsubscribe()

	// Wait for bids with timeout
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return bids
		default:
			msg, err := sub.NextMsg(100 * time.Millisecond)
			if err != nil {
				if err != nats.Timeout {
					log.Printf("Error reading bid: %v", err)
				}
				continue
			}

			var bid AuctionBid
			err := json.Unmarshal(msg.Data, &bid)
			if err != nil {
				log.Printf("Error unmarshaling bid: %v", err)
				msg.Nak()
				continue
			}

			// Validate bid is for this task
			if bid.TaskID != task.ID {
				msg.Nak()
				continue
			}

			// Add to bids
			bids = append(bids, bid)
			msg.Ack()
		}
	}
}

func startScalingMonitor(js jetstream.JetStream) {

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for {
			<-ticker.C

			// Check metrics queue length
			info, err := js.StreamInfo(context.Background(), "METRICS")
			if err == nil && info.State.Msgs > 100 {
				log.Printf("HIGH LOAD: Metrics queue has %d messages, would scale up detector agents", info.State.Msgs)
				// In real system: trigger scaling via Docker/Kubernetes API
			}

			// Check alerts queue
			info, err = js.StreamInfo(context.Background(), "ALERTS")
			if err == nil && info.State.Msgs > 50 {
				log.Printf("HIGH LOAD: Alerts queue has %d messages, would scale up notifier agents", info.State.Msgs)
			}

			// Check healing queue
			info, err = js.StreamInfo(context.Background(), "HEALING")
			if err == nil && info.State.Msgs > 20 {
				log.Printf("HIGH LOAD: Healing queue has %d messages, would scale up autohealer agents", info.State.Msgs)
			}
		}
	}()
}
