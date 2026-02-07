<?php

namespace App\Http\Middleware;

use App\Models\Workspace;
use Closure;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class EnforceUsageLimits
{
    /**
     * Limit types that can be checked.
     */
    protected array $limitChecks = [
        'workflows' => 'checkWorkflowLimit',
        'executions' => 'checkExecutionLimit',
        'credentials' => 'checkCredentialLimit',
        'members' => 'checkMemberLimit',
    ];

    /**
     * Handle an incoming request.
     *
     * @param  \Closure(\Illuminate\Http\Request): (\Symfony\Component\HttpFoundation\Response)  $next
     */
    public function handle(Request $request, Closure $next, ?string $limitType = null): Response
    {
        $workspace = $request->route('workspace');

        if (! $workspace instanceof Workspace) {
            return $next($request);
        }

        // Only check limits on create operations
        if (! in_array($request->method(), ['POST'])) {
            return $next($request);
        }

        // Get the workspace's plan limits
        $limits = $this->getWorkspaceLimits($workspace);

        if (empty($limits)) {
            // No limits (free tier or unlimited plan)
            return $next($request);
        }

        // Check specific limit type if provided
        if ($limitType && isset($this->limitChecks[$limitType])) {
            $checkMethod = $this->limitChecks[$limitType];
            $result = $this->$checkMethod($workspace, $limits);

            if ($result !== true) {
                return response()->json([
                    'message' => $result,
                    'error' => 'usage_limit_exceeded',
                    'limit_type' => $limitType,
                    'upgrade_url' => config('app.url').'/billing',
                ], Response::HTTP_PAYMENT_REQUIRED);
            }
        }

        return $next($request);
    }

    /**
     * Get the workspace's plan limits.
     */
    protected function getWorkspaceLimits(Workspace $workspace): array
    {
        $subscription = $workspace->subscription()->with('plan')->first();

        if (! $subscription || ! $subscription->plan) {
            // Return default free tier limits
            return [
                'max_workflows' => 5,
                'max_executions_per_month' => 100,
                'max_credentials' => 3,
                'max_members' => 2,
            ];
        }

        return $subscription->plan->limits ?? [];
    }

    /**
     * Check workflow creation limit.
     */
    protected function checkWorkflowLimit(Workspace $workspace, array $limits): bool|string
    {
        $maxWorkflows = $limits['max_workflows'] ?? null;

        if ($maxWorkflows === null || $maxWorkflows === -1) {
            return true; // Unlimited
        }

        $currentCount = $workspace->workflows()->count();

        if ($currentCount >= $maxWorkflows) {
            return "You have reached the maximum number of workflows ({$maxWorkflows}). Please upgrade your plan.";
        }

        return true;
    }

    /**
     * Check execution limit (monthly).
     */
    protected function checkExecutionLimit(Workspace $workspace, array $limits): bool|string
    {
        $maxExecutions = $limits['max_executions_per_month'] ?? null;

        if ($maxExecutions === null || $maxExecutions === -1) {
            return true; // Unlimited
        }

        $currentMonthCount = $workspace->executions()
            ->where('created_at', '>=', now()->startOfMonth())
            ->count();

        if ($currentMonthCount >= $maxExecutions) {
            return "You have reached the maximum number of executions this month ({$maxExecutions}). Please upgrade your plan.";
        }

        return true;
    }

    /**
     * Check credential creation limit.
     */
    protected function checkCredentialLimit(Workspace $workspace, array $limits): bool|string
    {
        $maxCredentials = $limits['max_credentials'] ?? null;

        if ($maxCredentials === null || $maxCredentials === -1) {
            return true; // Unlimited
        }

        $currentCount = $workspace->credentials()->count();

        if ($currentCount >= $maxCredentials) {
            return "You have reached the maximum number of credentials ({$maxCredentials}). Please upgrade your plan.";
        }

        return true;
    }

    /**
     * Check member limit.
     */
    protected function checkMemberLimit(Workspace $workspace, array $limits): bool|string
    {
        $maxMembers = $limits['max_members'] ?? null;

        if ($maxMembers === null || $maxMembers === -1) {
            return true; // Unlimited
        }

        $currentCount = $workspace->members()->count();

        if ($currentCount >= $maxMembers) {
            return "You have reached the maximum number of team members ({$maxMembers}). Please upgrade your plan.";
        }

        return true;
    }
}
