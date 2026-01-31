#!/bin/bash
set -e

echo "🔒 Running Security Scans..."

# PHP Security Scan
echo "📦 Checking PHP dependencies..."
cd apps/api
composer audit

# Go Security Scan
echo "🔍 Checking Go dependencies..."
cd ../engine
go list -json -deps ./... | nancy sleuth

# Docker Security Scan
echo "🐳 Scanning Docker images..."
cd ../..
docker scout cves --only-severity critical,high

echo "✅ Security scans completed!"