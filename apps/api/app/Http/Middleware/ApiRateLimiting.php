<?php

namespace App\Http\Middleware;

use Closure;
use Illuminate\Cache\RateLimiter;
use Illuminate\Http\Request;
use Symfony\Component\HttpFoundation\Response;

class ApiRateLimiting
{
    public function __construct(
        protected RateLimiter $limiter
    ) {}

    /**
     * Handle an incoming request.
     *
     * @param  \Closure(\Illuminate\Http\Request): (\Symfony\Component\HttpFoundation\Response)  $next
     */
    public function handle(Request $request, Closure $next, string $tier = 'default'): Response
    {
        $key = $this->resolveRequestSignature($request);
        $limits = $this->getLimitsForTier($tier, $request);

        if ($this->limiter->tooManyAttempts($key, $limits['max_attempts'])) {
            return $this->buildResponse($key, $limits['max_attempts']);
        }

        $this->limiter->hit($key, $limits['decay_seconds']);

        $response = $next($request);

        return $this->addHeaders(
            $response,
            $limits['max_attempts'],
            $this->calculateRemainingAttempts($key, $limits['max_attempts'])
        );
    }

    /**
     * Resolve request signature for rate limiting.
     */
    protected function resolveRequestSignature(Request $request): string
    {
        if ($user = $request->user()) {
            return 'user:'.$user->id;
        }

        return 'ip:'.$request->ip();
    }

    /**
     * Get rate limits for the specified tier.
     */
    protected function getLimitsForTier(string $tier, Request $request): array
    {
        // Check if user has a subscription with custom limits
        if ($user = $request->user()) {
            $workspace = $request->route('workspace');
            if ($workspace) {
                $subscription = $workspace->subscription()->with('plan')->first();
                if ($subscription && $subscription->plan) {
                    $apiLimit = $subscription->plan->limits['api_requests_per_minute'] ?? null;
                    if ($apiLimit) {
                        return [
                            'max_attempts' => $apiLimit,
                            'decay_seconds' => 60,
                        ];
                    }
                }
            }
        }

        return match ($tier) {
            'high' => [
                'max_attempts' => 1000,
                'decay_seconds' => 60,
            ],
            'medium' => [
                'max_attempts' => 300,
                'decay_seconds' => 60,
            ],
            'low' => [
                'max_attempts' => 60,
                'decay_seconds' => 60,
            ],
            'auth' => [
                'max_attempts' => 5,
                'decay_seconds' => 60,
            ],
            default => [
                'max_attempts' => 100,
                'decay_seconds' => 60,
            ],
        };
    }

    /**
     * Calculate remaining attempts.
     */
    protected function calculateRemainingAttempts(string $key, int $maxAttempts): int
    {
        return $this->limiter->remaining($key, $maxAttempts);
    }

    /**
     * Build rate limit exceeded response.
     */
    protected function buildResponse(string $key, int $maxAttempts): Response
    {
        $retryAfter = $this->limiter->availableIn($key);

        return response()->json([
            'message' => 'Too many requests. Please slow down.',
            'error' => 'rate_limit_exceeded',
            'retry_after' => $retryAfter,
        ], Response::HTTP_TOO_MANY_REQUESTS)
            ->header('Retry-After', $retryAfter)
            ->header('X-RateLimit-Limit', $maxAttempts)
            ->header('X-RateLimit-Remaining', 0);
    }

    /**
     * Add rate limit headers to response.
     */
    protected function addHeaders(Response $response, int $maxAttempts, int $remainingAttempts): Response
    {
        return $response
            ->header('X-RateLimit-Limit', $maxAttempts)
            ->header('X-RateLimit-Remaining', max(0, $remainingAttempts));
    }
}
