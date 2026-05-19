FROM python:3.11-slim

WORKDIR /app

# Copy requirements
COPY web/requirements.txt .

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy source code
COPY web/app.py web/templates/ web/static/ ./

# Expose web port
EXPOSE 8000

# Command to run the application
CMD ["python", "app.py"]