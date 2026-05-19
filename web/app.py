from fastapi import FastAPI, Request, Depends
from fastapi.responses import HTMLResponse
from fastapi.templating import Jinja2Templates
from fastapi.staticfiles import StaticFiles
import httpx
import asyncio
import logging
from typing import List, Dict, Any
import json
from datetime import datetime, timedelta

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

app = FastAPI(title="IT Monitoring Dashboard")

# Mount static files
app.mount("/static", StaticFiles(directory="web/static"), name="static")

# Setup templates
templates = Jinja2Templates(directory="web/templates")

# Configuration
ORCHESTRATOR_URL = "http://localhost:8080"
JAEGER_URL = "http://localhost:16686"

# Global HTTP client
client = httpx.AsyncClient(timeout=10.0)

class DashboardService:
    @staticmethod
    async def get_agent_status() -> List[Dict[str, Any]]:
        """Get status of all agents from orchestrator"""
        try:
            response = await client.get(f"{ORCHESTRATOR_URL}/api/v1/agents")
            if response.status_code == 200:
                return response.json().get("agents", [])
        except Exception as e:
            logger.error(f"Error fetching agent status: {e}")
            
        # Fallback data
        return [
            {"id": "collector-01", "type": "collector", "status": "online", "load": 2, "last_seen": "2025-01-15T10:30:00Z", "metrics_collected": 15420},
            {"id": "detector-01", "type": "detector", "status": "online", "load": 1, "last_seen": "2025-01-15T10:30:00Z", "anomalies_detected": 23},
            {"id": "notifier-01", "type": "notifier", "status": "online", "load": 0, "last_seen": "2025-01-15T10:30:00Z", "notifications_sent": 18},
            {"id": "autohealer-01", "type": "autohealer", "status": "online", "load": 0, "last_seen": "2025-01-15T10:30:00Z", "healing_attempts": 5, "success_rate": 0.8},
            {"id": "llm-agent-01", "type": "llm", "status": "online", "load": 1, "last_seen": "2025-01-15T10:30:00Z", "requests_processed": 42}
        ]
    
    @staticmethod
    async def get_recent_alerts() -> List[Dict[str, Any]]:
        """Get recent alerts"""
        try:
            response = await client.get(f"{ORCHESTRATOR_URL}/api/v1/alerts/recent?limit=10")
            if response.status_code == 200:
                return response.json().get("alerts", [])
        except Exception as e:
            logger.error(f"Error fetching recent alerts: {e}")
            
        # Fallback data
        now = datetime.utcnow()
        return [
            {
                "id": "alert_001",
                "severity": "critical",
                "metric": "CPU Usage",
                "value": 0.95,
                "service": "web-server-01",
                "detected_at": (now - timedelta(minutes=2)).isoformat(),
                "status": "active",
                "llm_analysis": {
                    "root_cause": "Memory leak in application",
                    "impact": "High",
                    "recommendations": ["Restart service", "Check logs", "Deploy patch"]
                }
            },
            {
                "id": "alert_002",
                "severity": "warning",
                "metric": "Disk Space",
                "value": 0.85,
                "service": "database-01",
                "detected_at": (now - timedelta(minutes=15)).isoformat(),
                "status": "resolved",
                "resolved_at": (now - timedelta(minutes=5)).isoformat()
            }
        ]
    
    @staticmethod
    async def get_system_metrics() -> Dict[str, Any]:
        """Get system metrics for dashboard charts"""
        now = datetime.utcnow()
        
        # Generate time series data for the last 24 hours
        hours = 24
        cpu_data = []
        memory_data = []
        
        for i in range(hours * 4):  # 15-minute intervals
            ts = now - timedelta(minutes=15 * i)
            cpu_data.append({
                "timestamp": ts.isoformat(),
                "value": max(0.1, min(0.9, 0.3 + (0.6 * (i % 24) / 24) + (0.2 * random.random())))
            })
            memory_data.append({
                "timestamp": ts.isoformat(),
                "value": max(0.2, min(0.95, 0.4 + (0.5 * (i % 24) / 24) + (0.1 * random.random())))
            })
        
        return {
            "cpu_usage": {
                "current": cpu_data[0]["value"],
                "history": sorted(cpu_data, key=lambda x: x["timestamp"])
            },
            "memory_usage": {
                "current": memory_data[0]["value"],
                "history": sorted(memory_data, key=lambda x: x["timestamp"])
            },
            "network_io": {
                "current_in": 45.2,
                "current_out": 67.8,
                "unit": "MB/s"
            },
            "request_rate": {
                "current": 234.5,
                "unit": "req/s"
            }
        }
    
    @staticmethod
    async def get_tracing_data() -> List[Dict[str, Any]]:
        """Get tracing data from Jaeger"""
        # In a real system, this would query Jaeger API
        # For demo, return sample data
        return [
            {
                "trace_id": "abc123def456",
                "service": "collector",
                "operation": "collect_metrics",
                "duration_ms": 45,
                "start_time": "2025-01-15T10:29:30Z",
                "status": "ok"
            },
            {
                "trace_id": "ghi789jkl012",
                "service": "detector",
                "operation": "detect_anomalies",
                "duration_ms": 23,
                "start_time": "2025-01-15T10:29:31Z",
                "status": "ok"
            },
            {
                "trace_id": "mno345pqr678",
                "service": "notifier",
                "operation": "send_notification",
                "duration_ms": 156,
                "start_time": "2025-01-15T10:29:32Z",
                "status": "ok"
            }
        ]

@app.get("/", response_class=HTMLResponse)
async def dashboard(request: Request):
    """Main dashboard page"""
    # Fetch data concurrently
    agent_status, recent_alerts, system_metrics, tracing_data = await asyncio.gather(
        DashboardService.get_agent_status(),
        DashboardService.get_recent_alerts(),
        DashboardService.get_system_metrics(),
        DashboardService.get_tracing_data()
    )
    
    # Calculate stats
    total_agents = len(agent_status)
    online_agents = sum(1 for a in agent_status if a["status"] == "online")
    critical_alerts = sum(1 for a in recent_alerts if a["severity"] == "critical")
    
    return templates.TemplateResponse(
        "dashboard.html",
        {
            "request": request,
            "agent_status": agent_status,
            "recent_alerts": recent_alerts,
            "system_metrics": system_metrics,
            "tracing_data": tracing_data,
            "stats": {
                "total_agents": total_agents,
                "online_agents": online_agents,
                "offline_agents": total_agents - online_agents,
                "critical_alerts": critical_alerts,
                "total_alerts": len(recent_alerts)
            },
            "jaeger_url": JAEGER_URL
        }
    )

@app.get("/agents", response_class=HTMLResponse)
async def agents_page(request: Request):
    """Agents management page"""
    agents = await DashboardService.get_agent_status()
    return templates.TemplateResponse(
        "agents.html",
        {
            "request": request,
            "agents": agents
        }
    )

@app.get("/alerts", response_class=HTMLResponse)
async def alerts_page(request: Request):
    """Alerts page"""
    alerts = await DashboardService.get_recent_alerts()
    return templates.TemplateResponse(
        "alerts.html",
        {
            "request": request,
            "alerts": alerts
        }
    )

@app.get("/tracing", response_class=HTMLResponse)
async def tracing_page(request: Request):
    """Distributed tracing page"""
    traces = await DashboardService.get_tracing_data()
    return templates.TemplateResponse(
        "tracing.html",
        {
            "request": request,
            "traces": traces,
            "jaeger_url": JAEGER_URL
        }
    )

@app.get"/metrics", response_class=HTMLResponse)
async def metrics_page(request: Request):
    """Metrics visualization page"""
    metrics = await DashboardService.get_system_metrics()
    return templates.TemplateResponse(
        "metrics.html",
        {
            "request": request,
            "metrics": metrics
        }
    )

@app.post"/api/v1/tasks/submit")
async def submit_task(request: Request):
    """Submit a new monitoring task"""
    try:
        data = await request.json()
        # Forward to orchestrator
        response = await client.post(f"{ORCHESTRATOR_URL}/api/v1/tasks", json=data)
        return response.json()
    except Exception as e:
        logger.error(f"Error submitting task: {e}")
        return {"error": str(e), "status": "failed"}

@app.on_event("startup")
async def startup_event():
    logger.info("Web dashboard starting up...")

@app.on_event("shutdown")
async def shutdown_event():
    await client.aclose()
    logger.info("Web dashboard shutting down...")