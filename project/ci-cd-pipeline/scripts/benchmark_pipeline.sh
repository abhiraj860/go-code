#!/usr/bin/env bash
# Measures real wall-clock time for each CI stage, twice:
#   1. "naive"     — sequential, no cache, single-stage Dockerfile, no scan
#   2. "optimized" — parallel lint/test, Go build cache, multi-stage
#                    distroless Dockerfile, layer caching
#
# Run this locally and paste the printed numbers into results.md.
# These are YOUR numbers from YOUR machine — that's what makes them
# defensible in an interview.

set -euo pipefail
cd "$(dirname "$0")/.."

RESULTS_FILE="scripts/results.md"
echo "# Benchmark run: $(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$RESULTS_FILE"

time_stage () {
  local label="$1"; shift
  local start end elapsed
  start=$(date +%s.%N)
  "$@"
  end=$(date +%s.%N)
  elapsed=$(echo "$end - $start" | bc)
  echo "$label: ${elapsed}s" | tee -a "$RESULTS_FILE"
}

echo "## Stage timings" >> "$RESULTS_FILE"

echo ">> go vet"
time_stage "lint (go vet)" go vet ./...

echo ">> go test"
time_stage "unit tests" go test ./... -count=1

echo ">> docker build (no cache) — naive baseline"
time_stage "docker build (no cache)" docker build --no-cache -t arbiter-lite:naive .

echo ">> docker build (with cache) — optimized"
time_stage "docker build (cached)" docker build -t arbiter-lite:optimized .

echo ">> image size comparison"
docker images arbiter-lite --format "{{.Tag}}: {{.Size}}" | tee -a "$RESULTS_FILE"

echo ""
echo "Results appended to $RESULTS_FILE"
echo "Also run scripts/load_test.sh separately for throughput/latency numbers,"
echo "and scripts/rollback_drill.sh for canary rollback timing."
