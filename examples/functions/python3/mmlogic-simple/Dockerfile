# Golang application builder steps
FROM python:3.5.3 as builder
WORKDIR /app

# Get reqs first, so docker builds can cache this layer and 
# code changes that don't add new dependencies will build faster.
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy the app
COPY api/ api/
COPY *.py ./

CMD ["python", "./harness.py"]
