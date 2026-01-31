#!/bin/bash
set -e

echo "🔒 Running Security Scans..."

# PHP Security Scan
echo "📦 Checking PHP dependencies..."
cd apps/api
composer audit --no-dev

# Go Security Scan
echo "🔍 Checking Go dependencies..."
cd ../engine
go list -json -m all | docker run --rm -i sonatypecommunity/nancy:latest sleuth

# Container Security Scan
echo "🐳 Scanning Docker images..."
cd ../..
if command -v docker &> /dev/null; then
    docker scout cves --only-severity critical,high . || echo "Docker Scout not available"
fi

# SAST Scan with Semgrep
echo "🔍 Running SAST scan..."
if command -v semgrep &> /dev/null; then
    semgrep --config=auto --error --quiet .
else
    echo "Semgrep not installed, skipping SAST scan"
fi

echo "✅ Security scans completed!"
