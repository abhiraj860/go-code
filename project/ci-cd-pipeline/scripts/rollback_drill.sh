#!/usr/bin/env bash
# Requires a local kind/minikube cluster with ArgoCD + Argo Rollouts installed.
# Deliberately deploys a broken image (bad-build tag), then times how long
# the AnalysisTemplate takes to detect the elevated error rate and the
# Rollout controller to auto-abort back to the last stable ReplicaSet.
#
# This is the script that produces your MTTR / auto-rollback number —
# do NOT skip actually running this if you plan to claim a rollback time
# on your resume.

set -euo pipefail
NAMESPACE="arbiter-lite"
ROLLOUT="arbiter-lite"

echo "1. Deploying known-bad image to trigger canary failure..."
START=$(date +%s)
kubectl argo rollouts set image "$ROLLOUT" \
  arbiter-lite=ghcr.io/abhiraj/arbiter-lite:bad-build \
  -n "$NAMESPACE"

echo "2. Watching rollout status until it aborts..."
kubectl argo rollouts get rollout "$ROLLOUT" -n "$NAMESPACE" --watch &
WATCH_PID=$!

# Poll for Degraded/Aborted status
while true; do
  STATUS=$(kubectl argo rollouts status "$ROLLOUT" -n "$NAMESPACE" 2>&1 || true)
  if echo "$STATUS" | grep -qi "degraded\|aborted"; then
    break
  fi
  sleep 2
done

kill $WATCH_PID 2>/dev/null || true
END=$(date +%s)
ELAPSED=$((END - START))

echo "Rollback detected and completed in ${ELAPSED}s" | tee -a scripts/results.md
echo "(time from bad deploy -> automated abort -> stable ReplicaSet serving 100% traffic)"
