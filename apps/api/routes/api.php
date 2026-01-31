<?php

use App\Http\Controllers\Api\V1\ActivityLogController;
use App\Http\Controllers\Api\V1\AuthController;
use App\Http\Controllers\Api\V1\CredentialController;
use App\Http\Controllers\Api\V1\CredentialTypeController;
use App\Http\Controllers\Api\V1\ExecutionController;
use App\Http\Controllers\Api\V1\InvitationController;
use App\Http\Controllers\Api\V1\JobCallbackController;
use App\Http\Controllers\Api\V1\NodeController;
use App\Http\Controllers\Api\V1\PlanController;
use App\Http\Controllers\Api\V1\SubscriptionController;
use App\Http\Controllers\Api\V1\TagController;
use App\Http\Controllers\Api\V1\UserController;
use App\Http\Controllers\Api\V1\VariableController;
use App\Http\Controllers\Api\V1\WebhookController;
use App\Http\Controllers\Api\V1\WorkflowController;
use App\Http\Controllers\Api\V1\WorkspaceController;
use App\Http\Controllers\Api\V1\WorkspaceMemberController;
use App\Http\Controllers\Api\WebhookReceiverController;
use Illuminate\Support\Facades\Route;

Route::prefix('v1')->as('v1.')->group(function () {

    /*
    |--------------------------------------------------------------------------
    | Health Check (for Docker/K8s)
    |--------------------------------------------------------------------------
    */

    Route::get('health', fn () => response()->json(['status' => 'ok', 'timestamp' => now()->toIso8601String()]))->name('health');

    /*
    |--------------------------------------------------------------------------
    | Public Routes
    |--------------------------------------------------------------------------
    */

    Route::get('plans', [PlanController::class, 'index'])->name('plans.index');

    /*
    |--------------------------------------------------------------------------
    | Auth Routes (Guest)
    |--------------------------------------------------------------------------
    */

    Route::prefix('auth')->as('auth.')->group(function () {
        Route::post('register', [AuthController::class, 'register'])->name('register');
        Route::post('login', [AuthController::class, 'login'])->name('login');
        Route::post('forgot-password', [AuthController::class, 'forgotPassword'])->name('forgot-password');
        Route::post('reset-password', [AuthController::class, 'resetPassword'])->name('reset-password');
    });

    Route::get('verify-email/{id}/{hash}', [AuthController::class, 'verifyEmail'])
        ->middleware('signed')
        ->name('verification.verify');

    /*
    |--------------------------------------------------------------------------
    | Invitation Routes (Public)
    |--------------------------------------------------------------------------
    */

    Route::prefix('invitations/{token}')->as('invitations.')->group(function () {
        Route::post('accept', [InvitationController::class, 'accept'])->name('accept');
        Route::post('decline', [InvitationController::class, 'decline'])->name('decline');
    });

    /*
    |--------------------------------------------------------------------------
    | Authenticated Routes
    |--------------------------------------------------------------------------
    */

    Route::middleware('auth:api')->group(function () {

        // Auth (Authenticated)
        Route::prefix('auth')->as('auth.')->group(function () {
            Route::post('logout', [AuthController::class, 'logout'])->name('logout');
            Route::post('resend-verification-email', [AuthController::class, 'resendVerificationEmail'])->name('resend-verification');
        });

        // User Profile
        Route::prefix('user')->as('user.')->group(function () {
            Route::get('/', [UserController::class, 'show'])->name('show');
            Route::put('/', [UserController::class, 'update'])->name('update');
            Route::put('password', [UserController::class, 'changePassword'])->name('password');
            Route::post('avatar', [UserController::class, 'uploadAvatar'])->name('avatar.upload');
            Route::delete('avatar', [UserController::class, 'deleteAvatar'])->name('avatar.delete');
            Route::delete('/', [UserController::class, 'destroy'])->name('destroy');
        });

        // Workspaces
        Route::apiResource('workspaces', WorkspaceController::class);

        // Workspace Nested Routes
        Route::prefix('workspaces/{workspace}')->as('workspaces.')->group(function () {

            // Members
            Route::prefix('members')->as('members.')->group(function () {
                Route::get('/', [WorkspaceMemberController::class, 'index'])->name('index');
                Route::put('{user}', [WorkspaceMemberController::class, 'update'])->name('update');
                Route::delete('{user}', [WorkspaceMemberController::class, 'destroy'])->name('destroy');
            });

            // Leave Workspace
            Route::post('leave', [WorkspaceMemberController::class, 'leave'])->name('leave');

            // Invitations
            Route::prefix('invitations')->as('invitations.')->group(function () {
                Route::get('/', [InvitationController::class, 'index'])->name('index');
                Route::post('/', [InvitationController::class, 'store'])->name('store');
                Route::delete('{invitation}', [InvitationController::class, 'destroy'])->name('destroy');
            });

            // Subscription
            Route::prefix('subscription')->as('subscription.')->group(function () {
                Route::get('/', [SubscriptionController::class, 'show'])->name('show');
                Route::post('/', [SubscriptionController::class, 'store'])->name('store');
                Route::delete('/', [SubscriptionController::class, 'destroy'])->name('destroy');
            });

            // Workflows
            Route::apiResource('workflows', WorkflowController::class);
            Route::post('workflows/{workflow}/activate', [WorkflowController::class, 'activate'])->name('workflows.activate');
            Route::post('workflows/{workflow}/deactivate', [WorkflowController::class, 'deactivate'])->name('workflows.deactivate');
            Route::post('workflows/{workflow}/duplicate', [WorkflowController::class, 'duplicate'])->name('workflows.duplicate');

            // Credentials
            Route::apiResource('credentials', CredentialController::class);
            Route::post('credentials/{credential}/test', [CredentialController::class, 'test'])->name('credentials.test');

            // Executions
            Route::get('executions/stats', [ExecutionController::class, 'stats'])->name('executions.stats');
            Route::apiResource('executions', ExecutionController::class)->only(['index', 'show', 'destroy']);
            Route::get('executions/{execution}/nodes', [ExecutionController::class, 'nodes'])->name('executions.nodes');
            Route::get('executions/{execution}/logs', [ExecutionController::class, 'logs'])->name('executions.logs');
            Route::post('executions/{execution}/retry', [ExecutionController::class, 'retry'])->name('executions.retry');
            Route::post('executions/{execution}/cancel', [ExecutionController::class, 'cancel'])->name('executions.cancel');
            Route::get('workflows/{workflow}/executions', [ExecutionController::class, 'workflowExecutions'])->name('workflows.executions');

            // Webhooks
            Route::apiResource('webhooks', WebhookController::class);
            Route::post('webhooks/{webhook}/regenerate-uuid', [WebhookController::class, 'regenerateUuid'])->name('webhooks.regenerate-uuid');
            Route::post('webhooks/{webhook}/activate', [WebhookController::class, 'activate'])->name('webhooks.activate');
            Route::post('webhooks/{webhook}/deactivate', [WebhookController::class, 'deactivate'])->name('webhooks.deactivate');
            Route::get('workflows/{workflow}/webhook', [WebhookController::class, 'forWorkflow'])->name('workflows.webhook');

            // Variables
            Route::apiResource('variables', VariableController::class);

            // Tags
            Route::apiResource('tags', TagController::class)->except(['show']);

            // Activity Logs
            Route::get('activity', [ActivityLogController::class, 'index'])->name('activity.index');
        });

        // Nodes (Global - not workspace-scoped)
        Route::prefix('nodes')->as('nodes.')->group(function () {
            Route::get('/', [NodeController::class, 'index'])->name('index');
            Route::get('categories', [NodeController::class, 'categories'])->name('categories');
            Route::get('search', [NodeController::class, 'search'])->name('search');
            Route::get('{type}', [NodeController::class, 'show'])->name('show');
        });

        // Credential Types (Global)
        Route::prefix('credential-types')->as('credential-types.')->group(function () {
            Route::get('/', [CredentialTypeController::class, 'index'])->name('index');
            Route::get('{type}', [CredentialTypeController::class, 'show'])->name('show');
        });
    });
});

/*
|--------------------------------------------------------------------------
| Public Webhook Receiver Routes
|--------------------------------------------------------------------------
*/

Route::prefix('webhooks')->as('webhooks.')->group(function () {
    Route::any('{uuid}', [WebhookReceiverController::class, 'handle'])->name('receive');
    Route::any('{uuid}/{path}', [WebhookReceiverController::class, 'handle'])->name('receive.path');
});

/*
|--------------------------------------------------------------------------
| Job Callback Routes (Go Engine)
|--------------------------------------------------------------------------
*/

Route::prefix('v1/jobs')->as('v1.jobs.')->group(function () {
    Route::post('callback', [JobCallbackController::class, 'handle'])->name('callback');
    Route::post('progress', [JobCallbackController::class, 'progress'])->name('progress');
});
