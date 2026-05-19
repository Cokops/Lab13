FROM python:3.11-slim

WORKDIR /app

# Copy requirements
COPY llm-agent/requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY llm-agent/app.py .

# Expose port if running a web service
# EXPOSE 8000

# Command to run the application
CMD ["python", "app.py"]