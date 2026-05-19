import ollama
import json
from typing import Dict, Any, List
import asyncio
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class LLMAgent:
    def __init__(self, model: str = "llama2"):
        self.model = model
        self.conversation_history = []
        logger.info(f"LLM Agent initialized with model {model}")
    
    async def analyze_alert(self, alert_data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Analyze an alert using LLM and provide intelligent insights
        """
        prompt = f"""
Analyze the following infrastructure alert and provide assessment:

Alert Type: {alert_data.get('type', 'Unknown')}
Severity: {alert_data.get('severity', 'Unknown')}
Metrics: {json.dumps(alert_data.get('metrics', {}), indent=2)}
Timestamp: {alert_data.get('timestamp', 'Unknown')}

Provide:
1. Root cause analysis
2. Business impact assessment (Low/Medium/High/Critical)
3. Recommended actions
4. Related historical incidents

Respond in JSON format with keys: 'root_cause', 'impact', 'recommendations', 'related_incidents'.
"""
        
        try:
            response = ollama.generate(
                model=self.model,
                prompt=prompt,
                format='json'
            )
            
            # Parse and validate response
            content = response['response'].strip()
            analysis = json.loads(content)
            
            # Add metadata
            analysis['generated_at'] = datetime.utcnow().isoformat()
            analysis['model'] = self.model
            analysis['input_alert_id'] = alert_data.get('id')
            
            logger.info(f"Successfully analyzed alert {alert_data.get('id')}")
            return analysis
            
        except Exception as e:
            logger.error(f"Error analyzing alert: {str(e)}")
            return {
                'error': str(e),
                'fallback_analysis': 'Unable to perform LLM analysis, using basic rules',
                'root_cause': 'Unknown',
                'impact': 'Medium',
                'recommendations': ['Check system logs', 'Monitor for recurrence', 'Verify backup systems'],
                'related_incidents': [],
                'generated_at': datetime.utcnow().isoformat()
            }
    
    async def classify_incident(self, description: str) -> Dict[str, Any]:
        """
        Classify an incident based on its description
        """
        prompt = f"""
Classify the following incident description:
\"{description}\"

Provide classification in JSON format with keys:
- 'category' (one of: hardware, software, network, security, performance, availability)
- 'subcategory'
- 'urgency' (1-5, 5 highest)
- 'confidence' (0-1)
- 'keywords'
"""
        
        try:
            response = ollama.generate(
                model=self.model,
                prompt=prompt,
                format='json'
            )
            
            content = response['response'].strip()
            classification = json.loads(content)
            classification['description'] = description
            classification['generated_at'] = datetime.utcnow().isoformat()
            
            logger.info(f"Classified incident: {description[:100]}...")
            return classification
            
        except Exception as e:
            logger.error(f"Error classifying incident: {str(e)}")
            return {
                'error': str(e),
                'category': 'unknown',
                'subcategory': 'unknown',
                'urgency': 3,
                'confidence': 0.5,
                'keywords': [],
                'description': description,
                'generated_at': datetime.utcnow().isoformat()
            }
    
    async def generate_report(self, data: Dict[str, Any]) -> Dict[str, Any]:
        """
        Generate a comprehensive report from monitoring data
        """
        prompt = f"""
Generate a comprehensive infrastructure health report based on the following data:

System Overview:
{json.dumps(data, indent=2)}

Include:
1. Executive summary
2. Key metrics analysis
3. Trend observations
4. Risk assessment
5. Recommendations

Format as structured JSON with appropriate sections.
"""
        
        try:
            response = ollama.generate(
                model=self.model,
                prompt=prompt,
                format='json'
            )
            
            content = response['response'].strip()
            report = json.loads(content)
            report['generated_at'] = datetime.utcnow().isoformat()
            report['period'] = data.get('period', 'unknown')
            
            logger.info("Generated infrastructure report")
            return report
            
        except Exception as e:
            logger.error(f"Error generating report: {str(e)}")
            return {
                'error': str(e),
                'executive_summary': 'Unable to generate AI report',
                'key_metrics': {},
                'trends': [],
                'risks': ['Monitoring system operational'],
                'recommendations': ['Enable LLM integration'],
                'generated_at': datetime.utcnow().isoformat()
            }

# Example usage
if __name__ == "__main__":
    agent = LLMAgent(model="llama2")
    
    # Example alert
    test_alert = {
        'id': 'alert_123',
        'type': 'high_cpu_usage',
        'severity': 'critical',
        'metrics': {
            'cpu_usage': 0.95,
            'process_count': 150,
            'load_average': 12.5
        },
        'timestamp': datetime.utcnow().isoformat()
    }
    
    # Run analysis
    result = asyncio.run(agent.analyze_alert(test_alert))
    print(json.dumps(result, indent=2))