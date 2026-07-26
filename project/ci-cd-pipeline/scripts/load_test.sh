#!/usr/bin/env bash
# Real throughput/latency numbers for the /ingest endpoint using `hey`
# (https://github.com/rakyll/hey — go install github.com/rakyll/hey@latest).
#
# These numbers back claims like:
#   "Service sustains X req/s at p99 < Yms under load"
# Only write that on your resume once you've actually run this and
# gotten a number you're comfortable defending.

set -euo pipefail
cd "$(dirname "$0")/.."

if ! command -v hey &> /dev/null; then
  echo "Installing hey..."
  go install github.com/rakyll/hey@latest
  export PATH="$PATH:$(go env GOPATH)/bin"
fi

echo "Starting service..."
docker run -d --rm -p 8080:8080 --name arbiter-bench arbiter-lite:optimized
sleep 2

PAYLOAD='{"id":"bench-1","merchant_impact":0.8,"sla_risk":0.6,"sentiment":0.4}'
echo "$PAYLOAD" > /tmp/payload.json

echo "Running load test: 5000 requests, 50 concurrent..."
hey -n 5000 -c 50 -m POST -T "application/json" -D /tmp/payload.json \
  http://localhost:8080/ingest | tee -a scripts/results.md

echo "Fetching in-process metrics..."
curl -s http://localhost:8080/metrics | tee -a scripts/results.md

docker stop arbiter-bench
