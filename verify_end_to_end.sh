#!/bin/bash
set -e

API_URL="http://localhost:8000/api/v1"
EMAIL="test@test.com"
PASSWORD="password" # pragma: allowlist secret

echo "1. Logging in..."
LOGIN_RES=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\": \"$EMAIL\", \"password\": \"$PASSWORD\"}")

TOKEN=$(echo $LOGIN_RES | python3 -c "import sys, json; print(json.load(sys.stdin).get('access_token', ''))")

if [ -z "$TOKEN" ]; then
  echo "Login failed: $LOGIN_RES"
  exit 1
fi
echo "Login successful."

echo "2. Creating Workspace (if needed) - Assuming ID 1 exists from previous steps"
# We verified ID 1 exists.

echo "3. Creating HTTP Test Workflow..."
WORKFLOW_RES=$(curl -s -X POST "$API_URL/workspaces/1/workflows" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "E2E Test Workflow",
    "trigger_type": "manual",
    "nodes": [
      {
        "id": "trigger",
        "type": "trigger_manual",
        "position": {"x": 100, "y": 100},
        "data": {"label": "Start"}
      },
      {
        "id": "http_action",
        "type": "action_http_request",
        "position": {"x": 300, "y": 100},
        "data": {
          "label": "Get Data",
          "method": "GET",
          "url": "https://httpbin.org/get"
        }
      }
    ],
    "edges": [
      {"id": "e1", "source": "trigger", "target": "http_action"}
    ]
  }')

echo "Workflow Response: $WORKFLOW_RES"

WORKFLOW_ID=$(echo "$WORKFLOW_RES" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('workflow', {}).get('id', data.get('data', {}).get('id', data.get('id', ''))))")

if [ -z "$WORKFLOW_ID" ]; then
  echo "Failed to create workflow. Response: $WORKFLOW_RES"
  exit 1
fi
echo "Workflow created: $WORKFLOW_ID"

echo "4. Executing Workflow..."
EXEC_RES=$(curl -s -X POST "$API_URL/workspaces/1/workflows/$WORKFLOW_ID/execute" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{}')

echo "Execution Response: $EXEC_RES"

EXECUTION_ID=$(echo "$EXEC_RES" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('execution', {}).get('id', data.get('execution_id', data.get('id', ''))))")

if [ -z "$EXECUTION_ID" ]; then
  echo "Failed to start execution: $EXEC_RES"
  exit 1
fi
echo "Execution started: $EXECUTION_ID"

echo "5. Polling Execution Status..."
for i in {1..10}; do
  STATUS_RES=$(curl -s -X GET "$API_URL/workspaces/1/executions/$EXECUTION_ID" \
    -H "Authorization: Bearer $TOKEN")

  echo "Attempt $i: Status Response: $STATUS_RES"
  STATUS=$(echo "$STATUS_RES" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('execution', {}).get('status', data.get('data', {}).get('status', data.get('status', 'unknown'))))")

  echo "Attempt $i: Status = $STATUS"

  if [ "$STATUS" == "completed" ] || [ "$STATUS" == "succeeded" ] || [ "$STATUS" == "2" ]; then
    echo "Execution Succeeded!"
    exit 0
  fi

  if [ "$STATUS" == "failed" ] || [ "$STATUS" == "3" ]; then
    echo "Execution Failed!"
    exit 1
  fi

  sleep 2
done

echo "Execution timed out."
exit 1
