# LinkFlow Workflow Engine - Complete Testing Plan

## Executive Summary

This document provides a comprehensive testing plan to verify LinkFlow workflow engine achieves 100% production readiness with all core functionality working identically to established Wall system baseline.

## System Verification Checklist

### ✅ Current Status Check
```bash
# Verify all services running
make ps
# Expected: All 10 services showing "healthy" status

# Test API endpoints
curl -s http://localhost:8000/api/v1/health | jq .
curl -s http://localhost:8080/health | jq .
# Expected: {"status": "ok"} or {"status": "healthy"}
```

**Checkpoint 1**: ✓ All services operational and responding

## Core Functionality Tests

### Test 1: HTTP Request Execution
**Baseline Comparison**: Wall system HTTP node behavior

```bash
# Create HTTP test workflow
curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "HTTP Test Workflow",
    "trigger_type": "manual",
    "nodes": [
      {
        "id": "wh1",
        "type": "trigger_manual",
        "position": {"x": 100, "y": 100},
        "data": {"label": "Manual Trigger"}
      },
      {
        "id": "http1",
        "type": "action_http_request",
        "position": {"x": 100, "y": 200},
        "data": {
          "label": "GET Request",
          "method": "GET",
          "url": "https://httpbin.org/get"
        }
      }
    ],
    "edges": [
      {"id": "e1", "source": "wh1", "target": "http1"}
    ]
  }'

# Execute workflow
curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows/1/execute \
  -H "Content-Type: application/json" \
  -d '{"input":{"test":"data"}}'

# Monitor execution
docker-compose logs -f worker | grep -E "(http|execute|task)"
```

**Checkpoint 2**: ✓ HTTP request executes successfully with 200 response

### Test 2: Data Transformation Operations
**Baseline Comparison**: Wall system data manipulation capabilities

```bash
# Test data transformation workflow
curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Transform Test Workflow",
    "trigger_type": "manual",
    "nodes": [
      {
        "id": "wh1",
        "type": "trigger_manual",
        "position": {"x": 100, "y": 100}
      },
      {
        "id": "transform1",
        "type": "transform_filter",
        "position": {"x": 100, "y": 200},
        "data": {
          "operation": "set",
          "field": "processed_data",
          "value": "transformed_value"
        }
      }
    ],
    "edges": [{"source": "wh1", "target": "transform1"}]
  }'
```

**Checkpoint 3**: ✓ Data transformation operations working correctly

### Test 3: Loop/Iteration Processing
**Baseline Comparison**: Wall system array iteration behavior

```bash
# Test array iteration workflow
curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Loop Test Workflow",
    "trigger_type": "manual",
    "nodes": [
      {
        "id": "wh1",
        "type": "trigger_manual",
        "position": {"x": 100, "y": 100}
      },
      {
        "id": "loop1",
        "type": "logic_loop",
        "position": {"x": 100, "y": 200},
        "data": {
          "items_field": "items_array",
          "item_alias": "current_item"
        }
      }
    ],
    "edges": [{"source": "wh1", "target": "loop1"}]
  }'
```

**Checkpoint 4**: ✓ Array iteration and loop processing functional

## Integration Testing

### Test 4: Email Sending
**Requirements**: SMTP configuration in environment variables

```bash
# Test email functionality
curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Email Test Workflow",
    "trigger_type": "manual",
    "nodes": [
      {
        "id": "wh1",
        "type": "trigger_manual",
        "position": {"x": 100, "y": 100}
      },
      {
        "id": "email1",
        "type": "action_send_email",
        "position": {"x": 100, "y": 200},
        "data": {
          "to": "test@example.com",
          "subject": "Test Email",
          "body": "This is a test email from LinkFlow"
        }
      }
    ],
    "edges": [{"source": "wh1", "target": "email1"}]
  }'
```

**Checkpoint 5**: ✓ Email sending functionality working with SMTP

### Test 5: Slack Integration
**Requirements**: SLACK_BOT_TOKEN environment variable

```bash
# Test Slack integration
curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Slack Test Workflow",
    "trigger_type": "manual",
    "nodes": [
      {
        "id": "wh1",
        "type": "trigger_manual",
        "position": {"x": 100, "y": 100}
      },
      {
        "id": "slack1",
        "type": "action_slack_message",
        "position": {"x": 100, "y": 200},
        "data": {
          "channel": "#general",
          "text": "Hello from LinkFlow!"
        }
      }
    ],
    "edges": [{"source": "wh1", "target": "slack1"}]
  }'
```

**Checkpoint 6**: ✓ Slack message sending functional

### Test 6: AI Integration
**Requirements**: OPENAI_API_KEY or ANTHROPIC_API_KEY

```bash
# Test AI processing
curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows \
  -H "Content-Type: application/json" \
  -d '{
    "name": "AI Test Workflow",
    "trigger_type": "manual",
    "nodes": [
      {
        "id": "wh1",
        "type": "trigger_manual",
        "position": {"x": 100, "y": 100}
      },
      {
        "id": "ai1",
        "type": "ai_completion",
        "position": {"x": 100, "y": 200},
        "data": {
          "provider": "openai",
          "prompt": "Say hello in 3 languages"
        }
      }
    ],
    "edges": [{"source": "wh1", "target": "ai1"}]
  }'
```

**Checkpoint 7**: ✓ AI processing with external providers functional

## Performance & Load Testing

### Test 7: Concurrent Workflow Execution
```bash
# Test concurrent execution capability
for i in {1..10}; do
  curl -X POST http://localhost:8000/api/v1/workspaces/1/workflows/1/execute \
    -H "Content-Type: application/json" \
    -d "{\"input\":{\"batch_id\":\"$i\"}}" &
done
wait
```

**Checkpoint 8**: ✓ 10 concurrent workflows execute without errors

### Test 8: Database Performance Verification
```bash
# Check workflow execution records and performance
docker-compose exec postgres psql -U linkflow -d linkflow -c "
  SELECT
    w.name as workflow_name,
    COUNT(e.id) as execution_count,
    AVG(e.duration_ms) as avg_duration_ms
  FROM workflows w
  LEFT JOIN executions e ON w.id = e.workflow_id
  GROUP BY w.id, w.name
  ORDER BY execution_count DESC;
"
```

**Checkpoint 9**: ✓ Database queries performant (< 50ms average)

## Environment Configuration Verification

### Required Environment Variables Check
```bash
# Verify critical environment variables
echo "Critical Environment Variables:"
echo "LINKFLOW_SECRET: ${LINKFLOW_SECRET:-NOT_SET}"
echo "JWT_SECRET: ${JWT_SECRET:-NOT_SET}"
echo "DATABASE_URL: ${DATABASE_URL:-NOT_SET}"

# Check integration credentials
echo "Integration Credentials Status:"
echo "SLACK_BOT_TOKEN: ${SLACK_BOT_TOKEN:+SET}"
echo "OPENAI_API_KEY: ${OPENAI_API_KEY:+SET}"
echo "TWILIO_ACCOUNT_SID: ${TWILIO_ACCOUNT_SID:+SET}"
```

**Checkpoint 10**: ✓ All required environment variables configured

## Success Criteria Matrix

| Feature | Test Status | Baseline Comparison | Notes |
|---------|-------------|-------------------|-------|
| HTTP Requests | ⬜ Pending | Wall System | Basic GET/POST operations |
| Email Sending | ⬜ Pending | Wall System | SMTP integration |
| Data Transform | ⬜ Pending | Wall System | Set/Rename/Delete operations |
| Array Loops | ⬜ Pending | Wall System | Item iteration processing |
| Conditional Logic | ⬜ Pending | Wall System | IF/ELSE branching |
| Delay/Timers | ⬜ Pending | Wall System | Wait node functionality |
| Slack Integration | ⬜ Pending | Wall System | Message sending |
| Discord Integration | ⬜ Pending | Wall System | Webhook posts |
| Twilio SMS | ⬜ Pending | Wall System | Text messaging |
| AI Processing | ⬜ Pending | Wall System | OpenAI/Claude integration |
| Webhook Triggers | ⬜ Pending | Wall System | HTTP endpoint triggers |

## Performance Benchmarks

- **Response Time**: < 100ms for simple operations
- **Concurrent Executions**: 10+ workflows simultaneously
- **Database Queries**: < 50ms average
- **Memory Usage**: < 100MB per service
- **CPU Usage**: < 5% baseline

## Quick Start Command Sequence

```bash
# Run complete verification sequence:
cd /Users/jaydeep/Herd/lnkflow

# 1. Verify services
make ps

# 2. Test API endpoints
curl -s http://localhost:8000/api/v1/health | jq .
curl -s http://localhost:8080/health | jq .

# 3. Create and run HTTP test workflow
# (Use the JSON from Test 1 above)

# 4. Monitor execution
docker-compose logs -f worker

# 5. Check results in database
docker-compose exec postgres psql -U linkflow -d linkflow -c "SELECT * FROM executions ORDER BY id DESC LIMIT 5;"
```

## Troubleshooting Guide

### Common Issues and Solutions

1. **Services not starting**: Run `make restart`
2. **Database connection errors**: Check `DATABASE_URL` environment variable
3. **Integration failures**: Verify API keys in environment variables
4. **Performance issues**: Monitor with `docker stats` and check logs

### Monitoring Commands

```bash
# Real-time service monitoring
docker-compose logs -f --tail 100

# Resource usage monitoring
docker stats

# Specific service logs
docker-compose logs -f worker
docker-compose logs -f api
```

**Your LinkFlow engine is production-ready when all checklist items are verified and passing!** 🚀
