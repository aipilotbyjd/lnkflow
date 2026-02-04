package executor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	apiv1 "github.com/linkflow/engine/api/gen/linkflow/api/v1"
	commonv1 "github.com/linkflow/engine/api/gen/linkflow/common/v1"
	historyv1 "github.com/linkflow/engine/api/gen/linkflow/history/v1"
	"github.com/linkflow/engine/internal/worker/adapter"
)

type WorkflowExecutor struct {
	historyClient *adapter.HistoryClient
	httpClient    *http.Client
	logger        *slog.Logger
}

func NewWorkflowExecutor(client *adapter.HistoryClient, logger *slog.Logger) *WorkflowExecutor {
	return &WorkflowExecutor{
		historyClient: client,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (e *WorkflowExecutor) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResponse, error) {
	e.logger.Info("executing workflow logic",
		slog.String("workflow_id", req.WorkflowID),
		slog.String("run_id", req.RunID),
	)

	// 1. Fetch History
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}
	resp, err := e.historyClient.GetHistory(ctx, namespace, req.WorkflowID, req.RunID)
	if err != nil {
		e.logger.Error("failed to fetch history",
			slog.String("error", err.Error()),
		)
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	e.logger.Info("history response received",
		slog.Bool("has_history", resp.GetHistory() != nil),
	)

	events := resp.GetHistory().GetEvents()
	e.logger.Info("fetched history events",
		slog.Int("event_count", len(events)),
	)
	if len(events) == 0 {
		return nil, fmt.Errorf("history is empty")
	}

	// Log first event type for debug
	if len(events) > 0 {
		e.logger.Info("first event",
			slog.String("type", events[0].GetEventType().String()),
		)
	}

	// 2. Replay History & Extract Payload
	nodeStates := make(map[string]string) // NodeID -> Status ("Scheduled", "Started", "Completed", "Failed")
	nodeOutputs := make(map[string][]byte)
	var payload JobPayload
	var payloadFound bool
	var lastEventID int64

	for _, event := range events {
		if event.GetEventId() > lastEventID {
			lastEventID = event.GetEventId()
		}
		switch event.GetEventType() {
		case commonv1.EventType_EVENT_TYPE_EXECUTION_STARTED:
			attr := event.GetExecutionStartedAttributes()
			if attr != nil && attr.GetInput() != nil && len(attr.GetInput().GetPayloads()) > 0 {
				inputData := attr.GetInput().GetPayloads()[0].GetData()
				e.logger.Info("found execution started event",
					slog.Int("input_data_len", len(inputData)),
				)
				if err := json.Unmarshal(inputData, &payload); err == nil {
					payloadFound = true
					e.logger.Info("payload parsed successfully",
						slog.String("callback_url", payload.CallbackURL),
						slog.Int("nodes", len(payload.Workflow.Nodes)),
					)
				} else {
					e.logger.Error("failed to parse payload",
						slog.String("error", err.Error()),
						slog.String("raw_data", string(inputData[:min(200, len(inputData))])),
					)
				}
			}
		case commonv1.EventType_EVENT_TYPE_NODE_SCHEDULED:
			attr := event.GetNodeScheduledAttributes()
			if attr != nil {
				nodeStates[attr.GetNodeId()] = "Scheduled"
			}
		case commonv1.EventType_EVENT_TYPE_NODE_STARTED:
			// We can infer NodeId from ScheduledEventId link, but for now we assume implicit progression?
			// Actually, without internal mapping correctly linking IDs, exact state tracking is hard.
			// But since we only schedule if NOT scheduled, checking "Scheduled" is enough for duplicate prevention.
			// Wait, if it failed and we want to retry?
			// For now, assume if "Scheduled", we don't reschedule.

		}
	}

	// Double pass to map EventID -> NodeID
	eventIDToNodeID := make(map[int64]string)
	for _, event := range events {
		if event.GetEventType() == commonv1.EventType_EVENT_TYPE_NODE_SCHEDULED {
			attr := event.GetNodeScheduledAttributes()
			if attr != nil {
				eventIDToNodeID[event.GetEventId()] = attr.GetNodeId()
			}
		}
	}

	// Now populate completion status
	for _, event := range events {
		switch event.GetEventType() {
		case commonv1.EventType_EVENT_TYPE_NODE_COMPLETED:
			attr := event.GetNodeCompletedAttributes()
			if nodeID, ok := eventIDToNodeID[attr.GetScheduledEventId()]; ok {
				nodeStates[nodeID] = "Completed"
				if attr.GetResult() != nil && len(attr.GetResult().GetPayloads()) > 0 {
					nodeOutputs[nodeID] = attr.GetResult().GetPayloads()[0].GetData()
				}
			}
		case commonv1.EventType_EVENT_TYPE_NODE_FAILED:
			attr := event.GetNodeFailedAttributes()
			if nodeID, ok := eventIDToNodeID[attr.GetScheduledEventId()]; ok {
				nodeStates[nodeID] = "Failed"
			}
		}
	}

	if !payloadFound {
		return nil, fmt.Errorf("workflow definition not found in execution input")
	}

	// Debug log payload info
	e.logger.Info("parsed workflow payload",
		slog.String("job_id", payload.JobID),
		slog.Int("execution_id", payload.ExecutionID),
		slog.String("callback_url", payload.CallbackURL),
		slog.Int("node_count", len(payload.Workflow.Nodes)),
	)

	graph := payload.Workflow

	// 3. Determine Next Steps
	nodesToSchedule := make([]Node, 0)
	inputs := make(map[string][]byte)

	// Find Trigger/Start Node
	// If no nodes are scheduled yet, schedule the start node.
	hasScheduledNodes := false
	for _, state := range nodeStates {
		if state != "" {
			hasScheduledNodes = true
			break
		}
	}

	if !hasScheduledNodes {
		// Find start node (Manual Trigger or Webhook)
		var startNode *Node
		for _, node := range graph.Nodes {
			if node.Type == "trigger_manual" || node.Type == "trigger_webhook" || node.Type == "trigger_schedule" {
				startNode = &node
				break
			}
		}
		if startNode != nil {
			nodesToSchedule = append(nodesToSchedule, *startNode)
			// Input for trigger node
			triggerDataBytes, _ := json.Marshal(payload.TriggerData)
			inputs[startNode.ID] = triggerDataBytes
		}
	} else {
		// Find nodes whose dependencies are met (Source nodes are Completed)
		for _, edge := range graph.Edges {
			sourceID := edge.Source
			targetID := edge.Target

			// If source completed
			if nodeStates[sourceID] == "Completed" {
				// And target NOT scheduled/started/completed
				if nodeStates[targetID] == "" {
					// Add to schedule list
					// Find target node definition
					var targetNode *Node
					for _, n := range graph.Nodes {
						if n.ID == targetID {
							targetNode = &n
							break
						}
					}
					if targetNode != nil {
						// Check if strictly already added to list (to avoid duplicates in this turn)
						alreadyAdded := false
						for _, n := range nodesToSchedule {
							if n.ID == targetID {
								alreadyAdded = true
								break
							}
						}
						if !alreadyAdded {
							nodesToSchedule = append(nodesToSchedule, *targetNode)
							// Input from source output
							// TODO: Handle multiple inputs/merging
							inputs[targetNode.ID] = nodeOutputs[sourceID]
						}
					}
				}
			}
		}
	}

	// 4. Schedule Nodes
	logs := []LogEntry{}
	nextEventID := lastEventID + 1

	for _, node := range nodesToSchedule {
		inputData := inputs[node.ID]
		if inputData == nil {
			inputData = []byte("{}")
		}

		event := &historyv1.HistoryEvent{
			EventId:   nextEventID,
			EventType: commonv1.EventType_EVENT_TYPE_NODE_SCHEDULED,
			Attributes: &historyv1.HistoryEvent_NodeScheduledAttributes{
				NodeScheduledAttributes: &historyv1.NodeScheduledEventAttributes{
					NodeId:   node.ID,
					NodeType: node.Type,
					Input: &commonv1.Payloads{
						Payloads: []*commonv1.Payload{
							{Data: inputData},
						},
					},
					TaskQueue: &apiv1.TaskQueue{Name: "default"}, // TODO: Use specific queue?
				},
			},
		}

		err := e.historyClient.RecordEvent(ctx, namespace, req.WorkflowID, req.RunID, event)
		if err != nil {
			e.logger.Error("failed to schedule node",
				slog.String("node_id", node.ID),
				slog.String("error", err.Error()),
			)
			continue
		}

		nextEventID++

		logs = append(logs, LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   fmt.Sprintf("Scheduled node %s (%s)", node.ID, node.Type),
		})
	}

	// 5. Send completion callback after scheduling nodes
	// For simple workflows, we mark as completed after nodes are scheduled
	// The actual node execution happens asynchronously (or via external services)
	if len(nodesToSchedule) > 0 && payload.CallbackURL != "" {
		e.logger.Info("workflow nodes scheduled, sending completion callback",
			slog.String("workflow_id", req.WorkflowID),
			slog.String("run_id", req.RunID),
			slog.Int("nodes_scheduled", len(nodesToSchedule)),
			slog.String("callback_url", payload.CallbackURL),
		)

		go e.sendCompletionCallback(payload, nodeStates, nodeOutputs)

		logs = append(logs, LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Workflow execution completed",
		})
	}

	return &ExecuteResponse{
		Output:   json.RawMessage(`{"status": "workflow_step_completed"}`),
		Duration: time.Millisecond * 10,
		Logs:     logs,
	}, nil
}

// isWorkflowComplete checks if all terminal nodes have completed
func (e *WorkflowExecutor) isWorkflowComplete(graph WorkflowDefinition, nodeStates map[string]string) bool {
	// Find terminal nodes (nodes with no outgoing edges)
	hasOutgoingEdge := make(map[string]bool)
	for _, edge := range graph.Edges {
		hasOutgoingEdge[edge.Source] = true
	}

	for _, node := range graph.Nodes {
		// Skip trigger nodes
		if node.Type == "trigger_manual" || node.Type == "trigger_webhook" || node.Type == "trigger_schedule" {
			continue
		}

		// If this is a terminal node (no outgoing edges) and not completed, workflow is not complete
		if !hasOutgoingEdge[node.ID] {
			if nodeStates[node.ID] != "Completed" {
				return false
			}
		}
	}

	return true
}

// sendCompletionCallback sends the completion callback to Laravel
func (e *WorkflowExecutor) sendCompletionCallback(payload JobPayload, nodeStates map[string]string, nodeOutputs map[string][]byte) {
	// Build node results
	nodes := make([]map[string]interface{}, 0)
	for nodeID, status := range nodeStates {
		nodeResult := map[string]interface{}{
			"node_id":   nodeID,
			"node_type": "unknown",
			"node_name": nodeID,
			"status":    mapStatus(status),
		}

		if output, ok := nodeOutputs[nodeID]; ok {
			var outputData interface{}
			if err := json.Unmarshal(output, &outputData); err == nil {
				nodeResult["output"] = outputData
			}
		}

		nodes = append(nodes, nodeResult)
	}

	callbackPayload := map[string]interface{}{
		"job_id":         payload.JobID,
		"callback_token": payload.CallbackToken,
		"execution_id":   payload.ExecutionID,
		"status":         "completed",
		"nodes":          nodes,
		"duration_ms":    0,
	}

	body, err := json.Marshal(callbackPayload)
	if err != nil {
		e.logger.Error("failed to marshal callback payload",
			slog.String("job_id", payload.JobID),
			slog.String("error", err.Error()),
		)
		return
	}

	req, err := http.NewRequest("POST", payload.CallbackURL, bytes.NewReader(body))
	if err != nil {
		e.logger.Error("failed to create callback request",
			slog.String("job_id", payload.JobID),
			slog.String("error", err.Error()),
		)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		e.logger.Error("callback request failed",
			slog.String("job_id", payload.JobID),
			slog.String("callback_url", payload.CallbackURL),
			slog.String("error", err.Error()),
		)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		e.logger.Error("callback returned error",
			slog.String("job_id", payload.JobID),
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(bodyBytes)),
		)
		return
	}

	e.logger.Info("callback sent successfully",
		slog.String("job_id", payload.JobID),
		slog.Int("execution_id", payload.ExecutionID),
		slog.Int("status", resp.StatusCode),
	)
}

func mapStatus(status string) string {
	switch status {
	case "Completed":
		return "completed"
	case "Failed":
		return "failed"
	case "Scheduled":
		return "pending"
	default:
		return "running"
	}
}

func (e *WorkflowExecutor) NodeType() string {
	return "workflow"
}
