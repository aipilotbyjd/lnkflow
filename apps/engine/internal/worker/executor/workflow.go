package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	apiv1 "github.com/linkflow/engine/api/gen/linkflow/api/v1"
	commonv1 "github.com/linkflow/engine/api/gen/linkflow/common/v1"
	historyv1 "github.com/linkflow/engine/api/gen/linkflow/history/v1"
	"github.com/linkflow/engine/internal/worker/adapter"
)

type WorkflowExecutor struct {
	historyClient *adapter.HistoryClient
	logger        *slog.Logger
}

func NewWorkflowExecutor(client *adapter.HistoryClient, logger *slog.Logger) *WorkflowExecutor {
	return &WorkflowExecutor{
		historyClient: client,
		logger:        logger,
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
		return nil, fmt.Errorf("failed to fetch history: %w", err)
	}

	events := resp.GetHistory().GetEvents()
	if len(events) == 0 {
		return nil, fmt.Errorf("history is empty")
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
				if err := json.Unmarshal(inputData, &payload); err == nil {
					payloadFound = true
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

	return &ExecuteResponse{
		Output:   json.RawMessage(`{"status": "workflow_step_completed"}`),
		Duration: time.Millisecond * 10,
		Logs:     logs,
	}, nil
}

func (e *WorkflowExecutor) NodeType() string {
	return "workflow"
}
