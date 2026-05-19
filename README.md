# IT-Infrastructure Monitoring System

This project implements a distributed monitoring system with Go microservices, NATS messaging, OpenTelemetry tracing, Redis state management, and an LLM-powered agent.

## Architecture

- **Agents** (Go microservices):
  - `collector`: Collects metrics from monitored systems
  - `detector`: Detects anomalies in collected metrics
  - `notifier`: Sends alerts based on detected anomalies
  - `autohealer`: Attempts automatic recovery from issues

- **LLM Agent** (Python): Processes natural language queries and generates intelligent responses

- **Orchestrator**: Manages task pipelines and agent coordination

- **Web Interface**: Monitoring dashboard built with FastAPI and Jinja2

- **Infrastructure**:
  - NATS for message brokering
  - Redis for agent state persistence
  - Jaeger for distributed tracing
  - Docker Compose for orchestration

## Features Implemented

1. ✅ Multi-agent system in Go with NATS communication
2. ✅ Task pipelines with sequential processing
3. ✅ Distributed tracing with OpenTelemetry and Jaeger
4. ✅ Stateful agent with Redis persistence
5. ✅ Dynamic scaling based on queue length
6. ✅ Auction-based task distribution
7. ✅ LLM agent integration for intelligent processing
8. ✅ Web interface for monitoring and control

## Getting Started

1. Start infrastructure:
```bash
docker-compose up -d
```

2. Build and run agents:
```bash
go build -o bin/collector agents/collector/main.go
GO_BUILD_FLAGS="-o bin/detector" go build agents/detector/main.go
# ... build other agents
```

3. Run the web interface:
```bash
cd web && python -m pip install -r requirements.txt && python app.py
```

4. Access the dashboard at `http://localhost:8000`

## Project Structure

```
ZAD13/
├── agents/               # Go microservices
│   ├── collector/        # Metrics collection agent
│   ├── detector/         # Anomaly detection agent
│   ├── notifier/         # Alert notification agent
│   └── autohealer/       # Automatic recovery agent
├── llm-agent/            # Python LLM agent
├── orchestrator/         # Task orchestration service
├── tracing/              # OpenTelemetry configuration
├── web/                  # Web dashboard
├── docker-compose.yml    # Container orchestration
└── go.mod                # Go module definition
```
