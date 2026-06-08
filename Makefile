.PHONY: build test run-ingress run-worker run-ui run-review-agent act-run clean

# Default target
all: build

## build: compile Go binaries
build:
	go build -o bin/agent ./agent-cli
	go build -o bin/ingress ./ingress
	go build -o bin/worker ./temporal/worker
	go build -o bin/client ./temporal/client

## test: run Go tests
test:
	go test ./... -v -timeout 60s

## run-ingress: start the HTTP ingress server on :8080
run-ingress:
	go run ./ingress

## run-worker: start the Temporal worker
run-worker:
	go run ./temporal/worker

## run-ui: start the Nuxt dev server on :3000
run-ui:
	cd ui && npm run dev

## run-review-agent: start the review agent NATS consumer + FastAPI
run-review-agent:
	cd review-agent && uvicorn server:app --host 0.0.0.0 --port 8081 &
	cd review-agent && python main.py

## act-run: run the agent pipeline locally via act
act-run:
	act workflow_dispatch -j agent-run --input site=$(SITE)

## clean: remove build artifacts
clean:
	rm -rf bin/ out/ ui/.nuxt ui/.output ui/node_modules/.cache
