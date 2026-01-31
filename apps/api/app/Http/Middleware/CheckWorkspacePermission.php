<?php

namespace App\Http\Middleware;

use App\Models\Workspace;
use App\Services\WorkspacePermissionService;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class CheckWorkspacePermission
{
    public function __construct(
        private WorkspacePermissionService $permissionService
    ) {}

    /**
     * @param  Closure(Request): Response  $next
     */
    public function handle(Request $request, Closure $next, string $permission): Response
    {
        $workspace = $request->route('workspace');

        if (! $workspace instanceof Workspace) {
            $workspace = Workspace::findOrFail($workspace);
        }

        $user = $request->user();

        $this->permissionService->authorize($user, $workspace, $permission);

        return $next($request);
    }
}
