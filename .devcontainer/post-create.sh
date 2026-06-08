#!/usr/bin/env bash
set -euo pipefail

echo "==> Installing Playwright browsers"
npx playwright install --with-deps chromium

echo "==> Building agent CLI"
go build -o /usr/local/bin/agent ./agent-cli

echo "==> Installing Python deps"
pip install -r review-agent/requirements.txt

echo "==> Installing Temporal CLI"
curl -sSf https://temporal.download/cli.sh | sh
mv temporal /usr/local/bin/temporal || true

echo "==> Installing act (GitHub Actions local runner)"
curl -sSf https://raw.githubusercontent.com/nektos/act/master/install.sh | bash -s -- -b /usr/local/bin

echo "==> Installing UI deps"
cd ui && npm install

echo "==> Done!"
