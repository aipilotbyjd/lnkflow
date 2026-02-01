#!/bin/bash

echo "🔐 Environment Security Check"

# Check for required environment variables
required_vars=("LINKFLOW_SECRET" "JWT_SECRET" "DB_PASSWORD")

for var in "${required_vars[@]}"; do
    if [ -z "${!var}" ]; then
        echo "❌ Missing required environment variable: $var"
        exit 1
    fi
done

# Check secret strength
if [ ${#LINKFLOW_SECRET} -lt 32 ]; then
    echo "⚠️  LINKFLOW_SECRET should be at least 32 characters"
fi

if [ ${#JWT_SECRET} -lt 32 ]; then
    echo "⚠️  JWT_SECRET should be at least 32 characters"
fi

echo "✅ Environment security check passed"
