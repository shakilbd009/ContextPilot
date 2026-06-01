#!/bin/bash
# doctor.sh — Check local development environment prerequisites
set -euo pipefail

ERRORS=0
WARNINGS=0

echo "=== ContextPilot Doctor ==="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

add_error() {
  ERRORS=$((ERRORS + 1))
  echo -e "  ${RED}ERROR: $1${NC}"
}

add_warning() {
  WARNINGS=$((WARNINGS + 1))
  echo -e "  ${YELLOW}WARN: $1${NC}"
}

add_ok() {
  echo -e "  ${GREEN}OK: $1${NC}"
}

# ──────────────────────────────────────────
# Docker
# ──────────────────────────────────────────
echo "Checking Docker..."
if command -v docker &>/dev/null; then
  DOCKER_VERSION=$(docker --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
  add_ok "docker $DOCKER_VERSION"
else
  add_error "docker not found — required for local dev"
fi

# Docker daemon
if docker info &>/dev/null; then
  add_ok "Docker daemon is running"
else
  add_error "Docker daemon is not running — start Docker Desktop"
fi

# Docker Compose
if command -v docker &>/dev/null; then
  if docker compose version &>/dev/null 2>&1; then
    DC_VERSION=$(docker compose version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    add_ok "docker compose $DC_VERSION"
  elif docker-compose --version &>/dev/null 2>&1; then
    DC_VERSION=$(docker-compose --version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
    add_warning "docker-compose (legacy) $DC_VERSION — consider installing 'docker compose' plugin"
  else
    add_error "docker compose not found"
  fi
fi

# ──────────────────────────────────────────
# Go
# ──────────────────────────────────────────
echo ""
echo "Checking Go..."
GO_VERSION=$(go version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' || echo "")
if [ -n "$GO_VERSION" ]; then
  add_ok "go $GO_VERSION"
else
  add_error "go not found — required for backend development"
fi

# ──────────────────────────────────────────
# Node / pnpm
# ──────────────────────────────────────────
echo ""
echo "Checking Node..."
NODE_VERSION=$(node --version 2>/dev/null | sed 's/v//' || echo "")
if [ -n "$NODE_VERSION" ]; then
  add_ok "node $NODE_VERSION"
else
  add_error "node not found — required for frontend development"
fi

echo ""
echo "Checking pnpm..."
pnpm_ver=$(pnpm --version 2>/dev/null || echo "")
if [ -n "$pnpm_ver" ]; then
  add_ok "pnpm $pnpm_ver"
else
  add_warning "pnpm not found — install with: npm install -g pnpm"
fi

# ──────────────────────────────────────────
# Port conflicts
# ──────────────────────────────────────────
echo ""
echo "Checking port availability..."

check_port() {
  local port=$1
  local name=$2
  if lsof -i :$port >/dev/null 2>&1; then
    add_warning "port $port ($name) is in use — may cause conflicts"
  else
    add_ok "port $port ($name) is free"
  fi
}

check_port 5432 "PostgreSQL"
check_port 6379 "Redis"
check_port 5173 "SvelteKit (frontend)"
check_port 8025 "Mailpit Web"
check_port 1025 "Mailpit SMTP"

# ──────────────────────────────────────────
# Environment files
# ──────────────────────────────────────────
echo ""
echo "Checking environment files..."
if [ -f ".env.example" ]; then
  add_ok ".env.example exists"
else
  add_error ".env.example missing"
fi

if [ -f ".env" ]; then
  add_ok ".env exists (local overrides)"
else
  add_warning ".env missing — create from .env.example for full local dev"
fi

# ──────────────────────────────────────────
# Scripts
# ──────────────────────────────────────────
echo ""
echo "Checking scripts..."
for script in scripts/check-status-sync.sh scripts/doctor.sh scripts/init-eval.sh scripts/seed-dev.sh; do
  if [ -f "$script" ]; then
    if [ -x "$script" ]; then
      add_ok "$script is executable"
    else
      add_warning "$script is not executable — run: chmod +x $script"
    fi
  else
    add_error "$script missing"
  fi
done

# ──────────────────────────────────────────
# Architecture evals
# ──────────────────────────────────────────
echo ""
echo "Checking architecture fitness functions..."
for f in evals/architecture/check-*.sh; do
  if [ -f "$f" ]; then
    if [ -x "$f" ]; then
      add_ok "$(basename $f) is executable"
    else
      add_warning "$(basename $f) is not executable — run: chmod +x $f"
    fi
  fi
done

# ──────────────────────────────────────────
# Summary
# ──────────────────────────────────────────
echo ""
echo "─────────────────────────────────────"
if [ "$ERRORS" -gt 0 ]; then
  echo -e "${RED}FAIL: $ERRORS error(s), $WARNINGS warning(s)${NC}"
  exit 1
elif [ "$WARNINGS" -gt 0 ]; then
  echo -e "${YELLOW}WARN: $WARNINGS warning(s)${NC}"
  echo "Environment is usable but not fully configured."
  exit 0
else
  echo -e "${GREEN}OK: environment is healthy${NC}"
  exit 0
fi