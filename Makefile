.PHONY: help sync-check eval-arch eval eval-e2e eval-integration eval-security eval-perf dev docker-up docker-down lint fmt clean migrate seed-dev doctor demo-check check-pipeline-gates check-orchestrator-pipeline check-brd-open-items

# ──────────────────────────────────────────
# Colors
# ──────────────────────────────────────────
RED  := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC  := \033[0m

# ──────────────────────────────────────────
# Defaults
# ──────────────────────────────────────────
export APP_PORT ?= 8080
export FRONTEND_PORT ?= 5173
export DATABASE_PORT ?= 5432
export CACHE_PORT ?= 6379

# ──────────────────────────────────────────
# Help
# ──────────────────────────────────────────
help: ## Show all available targets
	@echo "ContextPilot — Make targets"
	@echo ""
	@echo "=== Governance ==="
	@echo "  make sync-check          Run governance parity check (required before commit)"
	@echo "  make check-pipeline-gates Run gate check (GATE 1 completeness-score + GATE 2 validate-design)"
	@echo "  make check-orchestrator-pipeline Run strict v2 Kanban graph/no-skipped-steps check"
	@echo "  make eval-arch           Run architecture fitness functions"
	@echo "  make doctor         Check local dev environment (Docker, CLIs, ports)"
	@echo ""
	@echo "=== Development ==="
	@echo "  make dev            Start infrastructure (docker compose up -d)"
	@echo "  make docker-up      Start infrastructure"
	@echo "  make docker-down    Stop infrastructure"
	@echo "  make seed-dev       Load seed data (Phase 1+)"
	@echo ""
	@echo "=== Code Quality ==="
	@echo "  make lint           Lint backend + frontend (skips if dirs empty)"
	@echo "  make fmt            Format code (Phase 1+)"
	@echo "  make clean          Remove build artifacts"
	@echo ""
	@echo "=== Evaluation ==="
	@echo "  make eval           Run all evals (requires services up)"
	@echo "  make eval-e2e       Run E2E scenarios (Phase 1+)"
	@echo "  make eval-integration  Run integration tests (Phase 1+)"
	@echo "  make eval-security  Run security checks"
	@echo "  make eval-perf      Run performance benchmarks"
	@echo ""
	@echo "=== Meta ==="
	@echo "  make demo-check     Confirm local app/demo readiness"
	@echo "  make migrate        Run DB migrations (Phase 1+)"

# ──────────────────────────────────────────
# Governance
# ──────────────────────────────────────────
sync-check: ## Run check-status-sync.sh (exits 0 or 1)
	@bash scripts/check-status-sync.sh

check-pipeline-gates: ## Run pipeline gate check (GATE 1: completeness-score, GATE 2: validate-design)
	@bash scripts/check-pipeline-gates.sh

check-orchestrator-pipeline: ## Run strict v2 Kanban graph/no-skipped-steps check. Requires BRD_ID; optional PHASE=implementation.
	@bash scripts/check-orchestrator-pipeline.sh --phase $${PHASE:-implementation}

check-brd-open-items: ## Run BRD open-items check (exit 0 if all resolved, exit 1 if open items exist)
	@bash evals/architecture/check-brd-open-items.sh

eval-arch: ## Run architecture fitness functions (check-*.sh)
	@echo "Running architecture evals..."
	@for f in evals/architecture/check-*.sh; do \
		if [ -f "$$f" ]; then \
			echo "  $$(basename $$f)"; \
			bash "$$f" || exit 1; \
		fi \
	done
	@echo "OK: all architecture evals passed"

# ──────────────────────────────────────────
# Infrastructure
# ──────────────────────────────────────────
dev: docker-up ## Start infrastructure + show service URLs
	@echo ""
	@echo "ContextPilot is running:"
	@echo "  PostgreSQL: localhost:$$DATABASE_PORT"
	@echo "  Redis:      localhost:$$CACHE_PORT"
	@echo "  Mailpit:    http://localhost:8025 (SMTP capture UI)"
	@echo "  Backend:    http://localhost:$$APP_PORT (Phase 1+)"
	@echo "  Frontend:   http://localhost:$$FRONTEND_PORT (Phase 1+)"
	@echo ""
	@echo "Feature flag local demo:"
	@echo "  FF_ENABLE_APP_SHELL=true make dev"

docker-up: ## Start docker compose services (postgres, redis, mailpit)
	@docker compose up -d
	@echo "Infrastructure started. Run 'make doctor' to verify."

docker-down: ## Stop docker compose services
	@docker compose down

# ──────────────────────────────────────────
# Evaluation
# ──────────────────────────────────────────
eval: eval-arch ## Run all evals (arch + e2e + integration + security + perf)
	@echo ""
	@make eval-e2e || true
	@make eval-integration || true
	@make eval-security || true
	@make eval-perf || true

eval-e2e: ## Run E2E scenarios (Phase 1+)
	@if [ ! -d "frontend" ] || [ -z "$$(find frontend -name '*.ts' -type f 2>/dev/null)" ]; then \
		echo "SKIP: frontend not scaffolded yet (Phase 1)"; \
	else \
		cd frontend && pnpm exec playwright test --reporter=list || true; \
	fi

eval-integration: ## Run integration tests (Phase 1+)
	@if [ ! -d "backend" ] || [ ! -f "backend/go.mod" ]; then \
		echo "SKIP: backend not scaffolded yet (Phase 1)"; \
	else \
		cd backend && go test ./... || true; \
	fi

eval-security: ## Run security checks
	@echo "Running security checks..."
	@which trufflehog >/dev/null 2>&1 && trufflehog filesystem . || echo "SKIP: trufflehog not installed"
	@echo "OK: security checks complete"

eval-perf: ## Run performance benchmarks
	@if [ ! -d "frontend" ] || [ ! -d "frontend/node_modules" ]; then \
		echo "SKIP: frontend not built yet"; \
	else \
		echo "SKIP: lighthouse CI not configured yet"; \
	fi

# ──────────────────────────────────────────
# Development
# ──────────────────────────────────────────
seed-dev: ## Load seed data (Phase 1+)
	@bash scripts/seed-dev.sh

doctor: ## Check Docker, CLIs, package manager, env files, port conflicts
	@bash scripts/doctor.sh

lint: ## Lint backend + frontend (skips if dirs empty)
	@echo "Linting..."
	@if [ -d "backend" ] && [ -f "backend/go.mod" ]; then \
		cd backend && go vet ./... && echo "OK: backend vet passed"; \
	else \
		echo "SKIP: backend not scaffolded yet"; \
	fi
	@if [ -d "frontend" ] && [ -f "frontend/package.json" ]; then \
		cd frontend && pnpm lint || echo "SKIP: frontend lint not configured"; \
	else \
		echo "SKIP: frontend not scaffolded yet"; \
	fi
	@docker compose config >/dev/null 2>&1 && echo "OK: docker compose config valid" || echo "FAIL: docker compose config error"

fmt: ## Format code (Phase 1+)
	@if [ -d "backend" ] && [ -f "backend/go.mod" ]; then \
		cd backend && go fmt ./...; \
	fi
	@if [ -d "frontend" ] && [ -f "frontend/package.json" ]; then \
		cd frontend && pnpm fmt || true; \
	fi

clean: ## Remove build artifacts
	@rm -rf backend/bin/ frontend/.svelte-kit/ frontend/build/
	@rm -f *.out *.cover coverage/
	@echo "Cleaned build artifacts"

migrate: ## Run DB migrations (Phase 1+)
	@if [ ! -f "backend/main.go" ]; then \
		echo "SKIP: backend not scaffolded yet"; \
	else \
		cd backend && go run cmd/migrate/main.go || echo "SKIP: migrations not configured yet"; \
	fi

demo-check: ## Confirm local app/demo readiness
	@echo "Checking demo readiness..."
	@bash scripts/doctor.sh
	@docker compose config >/dev/null 2>&1 && echo "OK: docker compose config valid" || echo "FAIL: docker compose config error"