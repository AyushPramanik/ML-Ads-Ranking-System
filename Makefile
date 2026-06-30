# ===========================================================================
# ML Ads Ranking System — developer task runner
# ===========================================================================
.DEFAULT_GOAL := help

# Directories
ML_DIR        := ml
RANKING_DIR   := ranking
DATA_DIR      := data
ARTIFACTS_DIR := artifacts
VENV          := $(ML_DIR)/.venv
PY            := $(VENV)/bin/python
PIP           := $(VENV)/bin/pip

# ---------------------------------------------------------------------------
# Meta
# ---------------------------------------------------------------------------
.PHONY: help
help: ## Show this help
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'

# ---------------------------------------------------------------------------
# Python ML pipeline
# ---------------------------------------------------------------------------
.PHONY: ml-setup
ml-setup: ## Create the Python virtualenv and install the ML package
	python3 -m venv $(VENV)
	$(PIP) install --upgrade pip
	$(PIP) install -e "$(ML_DIR)[dev]"

.PHONY: ml-data
ml-data: ## Generate the synthetic ad-click dataset
	$(PY) -m ads_ml.cli generate --out $(DATA_DIR)

.PHONY: ml-train
ml-train: ## Train the CTR model and export artifacts for serving
	$(PY) -m ads_ml.cli train --data $(DATA_DIR) --out $(ARTIFACTS_DIR)

.PHONY: ml-pipeline
ml-pipeline: ml-data ml-train ## Run the full pipeline: generate data then train

.PHONY: ml-test
ml-test: ## Run the Python unit tests
	$(VENV)/bin/pytest $(ML_DIR)/tests -q

.PHONY: ml-lint
ml-lint: ## Lint and format-check the Python code
	$(VENV)/bin/ruff check $(ML_DIR)
	$(VENV)/bin/ruff format --check $(ML_DIR)

.PHONY: ml-fmt
ml-fmt: ## Auto-format the Python code
	$(VENV)/bin/ruff format $(ML_DIR)
	$(VENV)/bin/ruff check --fix $(ML_DIR)

# ---------------------------------------------------------------------------
# Go ranking service
# ---------------------------------------------------------------------------
.PHONY: go-build
go-build: ## Build the ranking server binary
	cd $(RANKING_DIR) && go build -o bin/ranking ./cmd/server

.PHONY: go-run
go-run: ## Run the ranking server locally
	cd $(RANKING_DIR) && go run ./cmd/server

.PHONY: go-test
go-test: ## Run the Go tests with the race detector
	cd $(RANKING_DIR) && go test -race ./...

.PHONY: go-bench
go-bench: ## Run the Go scoring/ranking benchmarks
	cd $(RANKING_DIR) && go test -run=^$$ -bench=. -benchmem ./internal/...

.PHONY: go-lint
go-lint: ## Vet and format-check the Go code
	cd $(RANKING_DIR) && go vet ./...
	@test -z "$$(cd $(RANKING_DIR) && gofmt -l .)" || (echo "gofmt needed:" && cd $(RANKING_DIR) && gofmt -l . && exit 1)

.PHONY: go-fmt
go-fmt: ## Format the Go code
	cd $(RANKING_DIR) && gofmt -w .

# ---------------------------------------------------------------------------
# Combined
# ---------------------------------------------------------------------------
.PHONY: test
test: ml-test go-test ## Run all tests

.PHONY: lint
lint: ml-lint go-lint ## Lint everything

# ---------------------------------------------------------------------------
# Docker / infrastructure
# ---------------------------------------------------------------------------
.PHONY: up
up: ## Build and start the full stack (postgres, redis, ranking) in the background
	docker compose up --build -d

.PHONY: down
down: ## Stop the stack and remove volumes
	docker compose down -v

.PHONY: logs
logs: ## Tail logs from all services
	docker compose logs -f

.PHONY: ps
ps: ## Show running services
	docker compose ps

.PHONY: clean
clean: ## Remove generated data, artifacts, and build output
	rm -rf $(DATA_DIR)/*.parquet $(DATA_DIR)/*.csv
	rm -rf $(ARTIFACTS_DIR)/*.json $(ARTIFACTS_DIR)/*.txt
	rm -rf $(RANKING_DIR)/bin
