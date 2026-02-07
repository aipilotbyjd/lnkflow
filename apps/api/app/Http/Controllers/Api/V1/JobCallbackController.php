<?php

namespace App\Http\Controllers\Api\V1;

use App\Enums\LogLevel;
use App\Http\Controllers\Controller;
use App\Models\Execution;
use App\Models\ExecutionLog;
use App\Models\ExecutionNode;
use App\Models\JobStatus;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Support\Facades\DB;

class JobCallbackController extends Controller
{
    /**
     * Handle callback from Go engine.
     */
    public function handle(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'job_id' => 'required|uuid',
            'callback_token' => 'required|string|size:64', // Required token
            'execution_id' => 'required|integer',
            'status' => 'required|in:completed,failed',
            'nodes' => 'nullable|array',
            'nodes.*.node_id' => 'required|string',
            'nodes.*.node_type' => 'required|string',
            'nodes.*.node_name' => 'nullable|string',
            'nodes.*.status' => 'required|in:pending,running,completed,failed,skipped',
            'nodes.*.output' => 'nullable|array',
            'nodes.*.error' => 'nullable|array',
            'nodes.*.started_at' => 'nullable|date',
            'nodes.*.completed_at' => 'nullable|date',
            'nodes.*.sequence' => 'nullable|integer',
            'error' => 'nullable|array',
            'duration_ms' => 'nullable|integer',
        ]);

        // Find job status
        $jobStatus = JobStatus::where('job_id', $validated['job_id'])->first();

        if (! $jobStatus) {
            return response()->json(['error' => 'Job not found'], 404);
        }

        // Validate callback token (timing-safe comparison)
        if (! hash_equals($jobStatus->callback_token, $validated['callback_token'])) {
            return response()->json(['error' => 'Invalid callback token'], 401);
        }

        // Idempotent handling for repeated callbacks from retries.
        if (in_array($jobStatus->status, ['completed', 'failed'], true)) {
            return response()->json([
                'success' => true,
                'execution_id' => $jobStatus->execution_id,
                'status' => $jobStatus->status,
                'idempotent' => true,
            ]);
        }

        // Find execution
        $execution = Execution::find($validated['execution_id']);

        if (! $execution) {
            return response()->json(['error' => 'Execution not found'], 404);
        }

        if ((int) $jobStatus->execution_id !== (int) $execution->id) {
            return response()->json(['error' => 'Execution does not match job'], 403);
        }

        DB::transaction(function () use ($execution, $jobStatus, $validated): void {
            // Update execution nodes and create execution logs
            if (! empty($validated['nodes'])) {
                foreach ($validated['nodes'] as $nodeData) {
                    $executionNode = ExecutionNode::updateOrCreate(
                        [
                            'execution_id' => $execution->id,
                            'node_id' => $nodeData['node_id'],
                        ],
                        [
                            'node_type' => $nodeData['node_type'],
                            'node_name' => $nodeData['node_name'] ?? null,
                            'status' => $nodeData['status'],
                            'output_data' => $nodeData['output'] ?? null,
                            'error' => $nodeData['error'] ?? null,
                            'started_at' => $nodeData['started_at'] ?? null,
                            'finished_at' => $nodeData['completed_at'] ?? null,
                            'sequence' => $nodeData['sequence'] ?? 0,
                        ]
                    );

                    $nodeName = $nodeData['node_name'] ?? $nodeData['node_id'];
                    $nodeStatus = $nodeData['status'];

                    if ($nodeStatus === 'failed') {
                        $errorMessage = $nodeData['error']['message'] ?? 'Unknown error';
                        ExecutionLog::create([
                            'execution_id' => $execution->id,
                            'execution_node_id' => $executionNode->id,
                            'level' => LogLevel::Error,
                            'message' => "Node '{$nodeName}' failed: {$errorMessage}",
                            'context' => $nodeData['error'] ?? null,
                            'logged_at' => $nodeData['completed_at'] ?? now(),
                        ]);
                    } else {
                        ExecutionLog::create([
                            'execution_id' => $execution->id,
                            'execution_node_id' => $executionNode->id,
                            'level' => LogLevel::Info,
                            'message' => "Node '{$nodeName}' ({$nodeData['node_type']}) {$nodeStatus}",
                            'context' => null,
                            'logged_at' => $nodeData['completed_at'] ?? $nodeData['started_at'] ?? now(),
                        ]);
                    }
                }
            }

            $startedAt = $execution->started_at ?? (! empty($validated['nodes']) ? $validated['nodes'][0]['started_at'] ?? now() : now());

            $execution->update([
                'status' => $validated['status'],
                'started_at' => $startedAt,
                'finished_at' => now(),
                'duration_ms' => $validated['duration_ms'] ?? null,
                'error' => $validated['error'] ?? null,
            ]);

            if ($validated['status'] === 'completed') {
                $jobStatus->markCompleted([
                    'duration_ms' => $validated['duration_ms'] ?? null,
                    'nodes_count' => count($validated['nodes'] ?? []),
                ]);
            } else {
                $jobStatus->markFailed($validated['error'] ?? ['message' => 'Unknown error']);
            }
        });

        return response()->json([
            'success' => true,
            'execution_id' => $execution->id,
            'status' => $validated['status'],
        ]);
    }

    /**
     * Handle progress update from Go engine.
     */
    public function progress(Request $request): JsonResponse
    {
        $validated = $request->validate([
            'job_id' => 'required|uuid',
            'callback_token' => 'required|string|size:64',
            'progress' => 'required|integer|min:0|max:100',
            'current_node' => 'nullable|string',
        ]);

        $jobStatus = JobStatus::where('job_id', $validated['job_id'])->first();

        if (! $jobStatus) {
            return response()->json(['error' => 'Job not found'], 404);
        }

        // Validate callback token
        if (! hash_equals($jobStatus->callback_token, $validated['callback_token'])) {
            return response()->json(['error' => 'Invalid callback token'], 401);
        }

        if (in_array($jobStatus->status, ['completed', 'failed'], true)) {
            return response()->json(['success' => true, 'idempotent' => true]);
        }

        $jobStatus->updateProgress($validated['progress']);

        return response()->json(['success' => true]);
    }
}
