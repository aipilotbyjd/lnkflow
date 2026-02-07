<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Http\Requests\Api\V1\Workflow\StoreWorkflowRequest;
use App\Http\Requests\Api\V1\Workflow\UpdateWorkflowRequest;
use App\Http\Resources\Api\V1\WorkflowResource;
use App\Models\Workflow;
use App\Models\Workspace;
use App\Services\ContractCompilerService;
use App\Services\WorkspacePermissionService;
use App\Services\WorkspacePolicyService;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Illuminate\Http\Resources\Json\AnonymousResourceCollection;

class WorkflowController extends Controller
{
    public function __construct(
        private WorkspacePermissionService $permissionService,
        private ContractCompilerService $contractCompilerService,
        private WorkspacePolicyService $workspacePolicyService
    ) {}

    public function index(Request $request, Workspace $workspace): AnonymousResourceCollection
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.view');

        $workflows = $workspace->workflows()
            ->with('creator')
            ->latest()
            ->paginate($request->input('per_page', 15));

        return WorkflowResource::collection($workflows);
    }

    public function store(StoreWorkflowRequest $request, Workspace $workspace): JsonResponse
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.create');

        $validated = $request->validated();
        $policyViolations = $this->workspacePolicyService->violations($workspace, $validated['nodes'] ?? []);
        if ($policyViolations !== []) {
            return response()->json([
                'message' => 'Workflow violates workspace policy.',
                'violations' => $policyViolations,
            ], 422);
        }

        $workflow = $workspace->workflows()->create([
            ...$validated,
            'created_by' => $request->user()->id,
        ]);

        $contractValidation = $this->contractCompilerService->validateAndSnapshot(
            workflow: $workflow,
            nodes: $validated['nodes'] ?? null,
            edges: $validated['edges'] ?? null
        );

        if ($contractValidation['status'] === 'invalid') {
            $workflow->delete();

            return response()->json([
                'message' => 'Workflow has invalid data contracts.',
                'issues' => $contractValidation['issues'],
            ], 422);
        }

        $settings = $workflow->settings ?? [];
        $settings['contract_snapshot_id'] = $contractValidation['snapshot']->id;
        $workflow->update(['settings' => $settings]);

        $workflow->load('creator');

        return response()->json([
            'message' => 'Workflow created successfully.',
            'workflow' => new WorkflowResource($workflow),
        ], 201);
    }

    public function show(Request $request, Workspace $workspace, Workflow $workflow): JsonResponse
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.view');
        $this->ensureWorkflowBelongsToWorkspace($workflow, $workspace);

        $workflow->load('creator');

        return response()->json([
            'workflow' => new WorkflowResource($workflow),
        ]);
    }

    public function update(UpdateWorkflowRequest $request, Workspace $workspace, Workflow $workflow): JsonResponse
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.update');
        $this->ensureWorkflowBelongsToWorkspace($workflow, $workspace);

        if ($workflow->is_locked) {
            return response()->json([
                'message' => 'This workflow is currently locked and cannot be edited.',
            ], 423);
        }

        $validated = $request->validated();
        $candidateNodes = $validated['nodes'] ?? $workflow->nodes ?? [];
        $candidateEdges = $validated['edges'] ?? $workflow->edges ?? [];

        $policyViolations = $this->workspacePolicyService->violations($workspace, $candidateNodes);
        if ($policyViolations !== []) {
            return response()->json([
                'message' => 'Workflow violates workspace policy.',
                'violations' => $policyViolations,
            ], 422);
        }

        $contractValidation = $this->contractCompilerService->validateAndSnapshot(
            workflow: $workflow,
            nodes: $candidateNodes,
            edges: $candidateEdges
        );

        if ($contractValidation['status'] === 'invalid') {
            return response()->json([
                'message' => 'Workflow has invalid data contracts.',
                'issues' => $contractValidation['issues'],
            ], 422);
        }

        $settings = $validated['settings'] ?? $workflow->settings ?? [];
        $settings['contract_snapshot_id'] = $contractValidation['snapshot']->id;
        $validated['settings'] = $settings;

        $workflow->update($validated);
        $workflow->load('creator');

        return response()->json([
            'message' => 'Workflow updated successfully.',
            'workflow' => new WorkflowResource($workflow),
        ]);
    }

    public function destroy(Request $request, Workspace $workspace, Workflow $workflow): JsonResponse
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.delete');
        $this->ensureWorkflowBelongsToWorkspace($workflow, $workspace);

        if ($workflow->is_locked) {
            return response()->json([
                'message' => 'This workflow is currently locked and cannot be deleted.',
            ], 423);
        }

        $workflow->delete();

        return response()->json([
            'message' => 'Workflow deleted successfully.',
        ]);
    }

    public function activate(Request $request, Workspace $workspace, Workflow $workflow): JsonResponse
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.update');
        $this->ensureWorkflowBelongsToWorkspace($workflow, $workspace);

        $workflow->activate();
        $workflow->load('creator');

        return response()->json([
            'message' => 'Workflow activated successfully.',
            'workflow' => new WorkflowResource($workflow),
        ]);
    }

    public function deactivate(Request $request, Workspace $workspace, Workflow $workflow): JsonResponse
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.update');
        $this->ensureWorkflowBelongsToWorkspace($workflow, $workspace);

        $workflow->deactivate();
        $workflow->load('creator');

        return response()->json([
            'message' => 'Workflow deactivated successfully.',
            'workflow' => new WorkflowResource($workflow),
        ]);
    }

    public function duplicate(Request $request, Workspace $workspace, Workflow $workflow): JsonResponse
    {
        $this->permissionService->authorize($request->user(), $workspace, 'workflow.create');
        $this->ensureWorkflowBelongsToWorkspace($workflow, $workspace);

        $newWorkflow = $workflow->replicate(['execution_count', 'last_executed_at', 'success_rate']);
        $newWorkflow->name = $workflow->name.' (Copy)';
        $newWorkflow->is_active = false;
        $newWorkflow->created_by = $request->user()->id;
        $newWorkflow->save();

        $newWorkflow->load('creator');

        return response()->json([
            'message' => 'Workflow duplicated successfully.',
            'workflow' => new WorkflowResource($newWorkflow),
        ], 201);
    }

    private function ensureWorkflowBelongsToWorkspace(Workflow $workflow, Workspace $workspace): void
    {
        if ($workflow->workspace_id !== $workspace->id) {
            abort(404, 'Workflow not found.');
        }
    }
}
