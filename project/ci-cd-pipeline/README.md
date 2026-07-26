# arbiter-lite

A small Go service (Kafka-consumer-style incident ticket processor, modeled
on the priority-scoring logic in Arbiter) built specifically to carry a full,
real CI/CD pipeline — GitHub Actions **and** Jenkins, GitOps deploy via
ArgoCD, canary rollout + automated rollback via Argo Rollouts.

Verified in this build: `go build ./...`, `go vet ./...`, and `go test ./...`
all pass with zero external dependencies (54.5% coverage on the processor
package). Local run:

```
real vet time:  0.337s
real test time: 0.706s
```

Everything past this point — Docker build times, image sizes, load-test
throughput, canary rollback timing — needs a Docker daemon and (for the
rollback drill) a local Kubernetes cluster, neither of which exist in this
sandbox. Run the three scripts below **on your machine** to get real numbers.

## Why this matters for your resume

I'm not going to hand you fabricated benchmark numbers — a hiring manager or
interviewer who asks "walk me through how you measured that 43% number" will
find out immediately if you can't explain it. Every number below is designed
to be something *you* generate by running the scripts and reading the
output. That's what makes it defensible.

## Run it

```bash
# 1. Build and test (already verified above)
go build ./... && go test ./... -v -cover

# 2. Pipeline stage timings — lint, test, docker build (naive vs cached)
chmod +x scripts/*.sh
./scripts/benchmark_pipeline.sh

# 3. Load test — throughput / latency under load (needs `hey`)
./scripts/load_test.sh

# 4. Canary rollback timing — needs kind/minikube + ArgoCD + Argo Rollouts
#    installed (see deploy/ for manifests)
./scripts/rollback_drill.sh
```

All output is appended to `scripts/results.md` as you run each script —
that file becomes your source of truth for resume numbers.

## What each pipeline stage demonstrates

| Stage | GitHub Actions | Jenkins | What it proves |
|---|---|---|---|
| Lint/static analysis | `golangci-lint`, `gosec` | same, via `go install` | code quality gate |
| Unit tests + coverage | `go test -coverprofile` | same | test discipline |
| Build | multi-stage Dockerfile → distroless | same Dockerfile | image size/security discipline |
| Scan | Trivy, fails build on CRITICAL/HIGH | same | supply-chain security |
| Push | GHCR | private registry | — |
| Deploy | commits new image tag → ArgoCD auto-syncs | same GitOps pattern | GitOps, not push-deploy |
| Rollout | Argo Rollouts canary (20%→50%→100%) | same manifests | progressive delivery |
| Rollback | AnalysisTemplate on error rate, auto-abort | same | automated, no human in the loop |

Having both GitHub Actions and Jenkins implementations of the *same*
pipeline is itself worth a line on your resume — it shows you understand
CI/CD as a set of portable concepts (stages, gates, artifacts) rather than
one tool's syntax.

## Files

```
cmd/consumer/main.go              — HTTP service (stands in for Kafka consumer)
internal/processor/               — priority scoring logic + tests
internal/metrics/                 — dependency-free Prometheus-format exporter
Dockerfile                        — multi-stage, distroless runtime
.github/workflows/ci-cd.yml       — GitHub Actions pipeline
Jenkinsfile                       — Jenkins declarative pipeline
deploy/k8s/deployment.yaml        — Namespace, Service, Argo Rollout (canary)
deploy/argo-rollouts/             — AnalysisTemplate for automated rollback
deploy/argocd-application.yaml    — ArgoCD Application (GitOps sync)
scripts/benchmark_pipeline.sh     — stage timing, cached vs uncached builds
scripts/load_test.sh              — throughput/latency via `hey`
scripts/rollback_drill.sh         — measures real auto-rollback time
scripts/results.md                — appended results from your runs (create on first run)
```

## Resume bullet template — fill in only after you've run the scripts

Do not fill in a bracket until `scripts/results.md` has a real number
backing it. Suggested phrasing once you have real numbers:

- Designed and implemented CI/CD pipeline (GitHub Actions + Jenkins) with
  automated lint, test, Trivy vulnerability scanning, and GitOps deployment
  via ArgoCD, cutting deployment time from [naive docker build time] to
  [cached docker build time]
- Implemented progressive delivery using Argo Rollouts (canary) with an
  automated Prometheus-based rollback gate, reducing bad-deploy recovery
  time to [rollback_drill.sh result] with zero manual intervention
- Containerized service using multi-stage distroless builds, reducing image
  size from [naive image size] to [optimized image size]
- Built [X req/s at Yms p99] load-tested ingestion service with full
  observability (Prometheus-format metrics) as part of a self-built CI/CD
  demonstration pipeline

If you'd rather not wait to run everything, it's also completely fine to
frame this as a personal project on your resume ("Built and benchmarked a
CI/CD pipeline demonstrating GitOps and progressive delivery") without
hard numbers — reviewers respect a well-architected system bullet even
without a metric, and it avoids any number you'd have to defend under
questioning.
