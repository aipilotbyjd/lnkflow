<?php

namespace App\Jobs;

use App\Models\Execution;
use App\Models\JobStatus;
use App\Models\Workflow;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldBeUnique;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Facades\Redis;
use Illuminate\Support\Str;

class ExecuteWorkflowJob implements ShouldQueue, ShouldBeUnique
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public string $jobId;
    public int $partition;

    /**
     * Number of times the job may be attempted.
     */
    public int $tries = 3;

    /**
     * Backoff between retries (seconds).
     *
     * @var array<int>
     */
    public array $backoff = [10, 60, 300];

    /**
     * Time (seconds) before job should timeout.
     */
    public int $timeout = 300;

    public function __construct(
        public Workflow $workflow,
        public Execution $execution,
        public string $priority = 'default',
        public array $triggerData = [],
    ) {
        $this->jobId = (string) Str::uuid();
        $this->partition = $workflow->workspace_id % 16;
        $this->onQueue("workflows-{$priority}");
    }

    /**
     * Unique ID for preventing duplicate jobs.
     */
    public function uniqueId(): string
    {
        return "workflow:{$this->workflow->id}:execution:{$this->execution->id}";
    }

    public function handle(): void
    {
        // Generate unique callback token (64-char hex)
        $callbackToken = bin2hex(random_bytes(32));

        // Create job status record with token
        $jobStatus = JobStatus::create([
            'job_id' => $this->jobId,
            'execution_id' => $this->execution->id,
            'partition' => $this->partition,
            'callback_token' => $callbackToken,
            'status' => 'pending',
        ]);

        // Prepare message for Go engine (includes token for callbacks)
        $message = $this->buildMessage($callbackToken);

        // Publish to Redis Stream (partitioned)
        $streamKey = "linkflow:jobs:partition:{$this->partition}";

        Redis::xadd($streamKey, '*', [
            'payload' => json_encode($message),
        ]);

        // Update job status
        $jobStatus->markProcessing();

        // Update execution status
        $this->execution->update(['status' => 'running']);
    }

    /**
     * Build the message payload for Go engine.
     *
     * @return array<string, mixed>
     */
    protected function buildMessage(string $callbackToken): array
    {
        return [
            'job_id' => $this->jobId,
            'callback_token' => $callbackToken, // Go must include this in callbacks
            'execution_id' => $this->execution->id,
            'workflow_id' => $this->workflow->id,
            'workspace_id' => $this->workflow->workspace_id,
            'partition' => $this->partition,
            'priority' => $this->priority,
            'workflow' => [
                'nodes' => $this->workflow->nodes,
                'edges' => $this->workflow->edges,
                'settings' => $this->workflow->settings,
            ],
            'trigger_data' => $this->triggerData,
            'credentials' => $this->getDecryptedCredentials(),
            'variables' => $this->getVariables(),
            'callback_url' => config('app.url') . '/api/v1/jobs/callback',
            'progress_url' => config('app.url') . '/api/v1/jobs/progress',
            'created_at' => now()->toIso8601String(),
        ];
    }

    /**
     * Get decrypted credentials used by this workflow.
     * Only decrypts credentials that are actually referenced by nodes.
     *
     * @return array<string, mixed>
     */
    protected function getDecryptedCredentials(): array
    {
        // Extract credential IDs from nodes to only decrypt what's needed
        $usedCredentialIds = collect($this->workflow->nodes)
            ->pluck('data.credentialId')
            ->merge(collect($this->workflow->nodes)->pluck('data.credential_id'))
            ->filter()
            ->unique()
            ->values()
            ->all();

        if (empty($usedCredentialIds)) {
            return [];
        }

        return $this->workflow->credentials()
            ->whereIn('credentials.id', $usedCredentialIds)
            ->get()
            ->mapWithKeys(fn ($credential) => [
                $credential->id => [
                    'type' => $credential->type,
                    'data' => $credential->getDecryptedData(),
                ],
            ])
            ->all();
    }

    /**
     * Get workspace variables.
     *
     * @return array<string, mixed>
     */
    protected function getVariables(): array
    {
        return $this->workflow->workspace->variables()
            ->get()
            ->mapWithKeys(fn ($variable) => [
                $variable->key => $variable->is_secret
                    ? $variable->getDecryptedValue()
                    : $variable->value,
            ])
            ->all();
    }

    /**
     * Handle job failure.
     */
    public function failed(\Throwable $exception): void
    {
        $this->execution->update([
            'status' => 'failed',
            'error' => [
                'message' => $exception->getMessage(),
                'trace' => $exception->getTraceAsString(),
            ],
            'completed_at' => now(),
        ]);

        JobStatus::where('job_id', $this->jobId)->first()?->markFailed([
            'message' => $exception->getMessage(),
            'code' => $exception->getCode(),
        ]);
    }
}
