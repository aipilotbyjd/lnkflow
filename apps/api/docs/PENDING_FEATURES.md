# LinkFlow - Complete Pending Features Specification

> **Status**: Phase 2 - Core Workflow Features  
> **Last Updated**: 2026-01-30

---

## Table of Contents

1. [Workflows](#1-workflows)
2. [Nodes (Triggers & Actions)](#2-nodes-triggers--actions)
3. [Credentials](#3-credentials)
4. [Executions](#4-executions)
5. [Webhooks](#5-webhooks)
6. [Variables & Secrets](#6-variables--secrets)
7. [Tags](#7-tags)
8. [Activity Logs](#8-activity-logs)
9. [Queue Infrastructure](#9-queue-infrastructure)
10. [File Structure](#10-file-structure)
11. [Database Schema](#11-database-schema)
12. [API Endpoints Summary](#12-api-endpoints-summary)

---

## 1. Workflows

### Overview

Workflows are the core feature - visual automation flows with triggers and actions.

### Database Schema

```sql
CREATE TABLE workflows (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workspace_id        BIGINT UNSIGNED NOT NULL,
    created_by          BIGINT UNSIGNED NOT NULL,
    
    -- Basic Info
    name                VARCHAR(255) NOT NULL,
    description         TEXT NULL,
    icon                VARCHAR(50) DEFAULT 'workflow',
    color               VARCHAR(20) DEFAULT '#6366f1',
    
    -- Status
    is_active           BOOLEAN DEFAULT FALSE,
    is_locked           BOOLEAN DEFAULT FALSE,          -- Prevent edits during execution
    
    -- Trigger Configuration
    trigger_type        ENUM('manual', 'webhook', 'schedule', 'event') NOT NULL,
    trigger_config      JSON NULL,                       -- Cron expression, webhook path, etc.
    
    -- Workflow Definition (Node-based)
    nodes               JSON NOT NULL,                   -- Array of node definitions
    edges               JSON NOT NULL,                   -- Connections between nodes
    viewport            JSON NULL,                       -- Canvas position/zoom for UI
    
    -- Settings
    settings            JSON NULL,                       -- Retry policy, timeout, notifications
    
    -- Statistics (cached)
    execution_count     INT UNSIGNED DEFAULT 0,
    last_executed_at    TIMESTAMP NULL,
    success_rate        DECIMAL(5,2) DEFAULT 0.00,
    
    -- Timestamps
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at          TIMESTAMP NULL,                  -- Soft delete
    
    -- Indexes
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_workspace_active (workspace_id, is_active),
    INDEX idx_trigger_type (trigger_type)
);
```

### JSON Structure: `nodes`

```json
[
    {
        "id": "node_1",
        "type": "trigger_webhook",
        "position": { "x": 100, "y": 100 },
        "data": {
            "label": "Webhook Trigger",
            "config": {
                "method": "POST",
                "path": "my-webhook"
            }
        }
    },
    {
        "id": "node_2",
        "type": "action_http_request",
        "position": { "x": 100, "y": 250 },
        "data": {
            "label": "Send to API",
            "config": {
                "url": "https://api.example.com/data",
                "method": "POST",
                "headers": {
                    "Authorization": "Bearer {{credentials.api_key}}"
                },
                "body": "{{trigger.body}}"
            },
            "credential_id": 5
        }
    },
    {
        "id": "node_3",
        "type": "action_condition",
        "position": { "x": 100, "y": 400 },
        "data": {
            "label": "Check Response",
            "config": {
                "conditions": [
                    {
                        "field": "{{node_2.response.status}}",
                        "operator": "equals",
                        "value": 200
                    }
                ]
            }
        }
    }
]
```

### JSON Structure: `edges`

```json
[
    {
        "id": "edge_1",
        "source": "node_1",
        "target": "node_2",
        "type": "default"
    },
    {
        "id": "edge_2",
        "source": "node_2",
        "target": "node_3",
        "type": "default"
    },
    {
        "id": "edge_3",
        "source": "node_3",
        "target": "node_4",
        "sourceHandle": "true",
        "type": "success"
    },
    {
        "id": "edge_4",
        "source": "node_3",
        "target": "node_5",
        "sourceHandle": "false",
        "type": "failure"
    }
]
```

### JSON Structure: `settings`

```json
{
    "retry": {
        "enabled": true,
        "max_attempts": 3,
        "delay_seconds": 60,
        "backoff": "exponential"
    },
    "timeout": {
        "workflow": 3600,
        "node": 300
    },
    "notifications": {
        "on_failure": true,
        "on_success": false,
        "channels": ["email", "slack"]
    },
    "execution": {
        "mode": "sequential",
        "save_data": true,
        "log_level": "info"
    }
}
```

### JSON Structure: `trigger_config`

```json
// For schedule trigger
{
    "cron": "0 9 * * 1-5",
    "timezone": "America/New_York"
}

// For webhook trigger
{
    "path": "my-custom-webhook",
    "method": ["POST", "PUT"],
    "authentication": {
        "type": "header",
        "key": "X-API-Key",
        "value_credential_id": 10
    }
}

// For event trigger
{
    "event": "user.created",
    "source": "internal"
}
```

### Model: `Workflow.php`

```php
<?php

namespace App\Models;

use Illuminate\Database\Eloquent\Model;
use Illuminate\Database\Eloquent\SoftDeletes;
use Illuminate\Database\Eloquent\Relations\BelongsTo;
use Illuminate\Database\Eloquent\Relations\HasMany;
use Illuminate\Database\Eloquent\Relations\BelongsToMany;
use Illuminate\Database\Eloquent\Factories\HasFactory;

class Workflow extends Model
{
    use HasFactory, SoftDeletes;

    protected $fillable = [
        'workspace_id',
        'created_by',
        'name',
        'description',
        'icon',
        'color',
        'is_active',
        'is_locked',
        'trigger_type',
        'trigger_config',
        'nodes',
        'edges',
        'viewport',
        'settings',
    ];

    protected function casts(): array
    {
        return [
            'is_active' => 'boolean',
            'is_locked' => 'boolean',
            'trigger_config' => 'array',
            'nodes' => 'array',
            'edges' => 'array',
            'viewport' => 'array',
            'settings' => 'array',
            'last_executed_at' => 'datetime',
            'success_rate' => 'decimal:2',
        ];
    }

    // ─────────────────────────────────────────────────────────────
    // Relationships
    // ─────────────────────────────────────────────────────────────

    public function workspace(): BelongsTo
    {
        return $this->belongsTo(Workspace::class);
    }

    public function creator(): BelongsTo
    {
        return $this->belongsTo(User::class, 'created_by');
    }

    public function executions(): HasMany
    {
        return $this->hasMany(Execution::class);
    }

    public function webhook(): HasOne
    {
        return $this->hasOne(Webhook::class);
    }

    public function tags(): BelongsToMany
    {
        return $this->belongsToMany(Tag::class, 'workflow_tags');
    }

    public function credentials(): BelongsToMany
    {
        return $this->belongsToMany(Credential::class, 'workflow_credentials');
    }

    // ─────────────────────────────────────────────────────────────
    // Scopes
    // ─────────────────────────────────────────────────────────────

    public function scopeActive($query)
    {
        return $query->where('is_active', true);
    }

    public function scopeByTriggerType($query, string $type)
    {
        return $query->where('trigger_type', $type);
    }

    // ─────────────────────────────────────────────────────────────
    // Methods
    // ─────────────────────────────────────────────────────────────

    public function activate(): void
    {
        $this->update(['is_active' => true]);
    }

    public function deactivate(): void
    {
        $this->update(['is_active' => false]);
    }

    public function isScheduled(): bool
    {
        return $this->trigger_type === 'schedule';
    }

    public function isWebhookTriggered(): bool
    {
        return $this->trigger_type === 'webhook';
    }

    public function getNodeById(string $nodeId): ?array
    {
        return collect($this->nodes)->firstWhere('id', $nodeId);
    }

    public function incrementExecutionCount(bool $success): void
    {
        $this->increment('execution_count');
        
        // Recalculate success rate
        $totalSuccess = $this->executions()->where('status', 'completed')->count();
        $this->update([
            'last_executed_at' => now(),
            'success_rate' => ($totalSuccess / $this->execution_count) * 100,
        ]);
    }
}
```

### API Endpoints

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/workspaces/{workspace}/workflows` | List workflows | workflow.view |
| POST | `/workspaces/{workspace}/workflows` | Create workflow | workflow.create |
| GET | `/workspaces/{workspace}/workflows/{workflow}` | Get workflow | workflow.view |
| PUT | `/workspaces/{workspace}/workflows/{workflow}` | Update workflow | workflow.update |
| DELETE | `/workspaces/{workspace}/workflows/{workflow}` | Delete workflow | workflow.delete |
| POST | `/workspaces/{workspace}/workflows/{workflow}/execute` | Manual execute | workflow.execute |
| POST | `/workspaces/{workspace}/workflows/{workflow}/activate` | Activate | workflow.update |
| POST | `/workspaces/{workspace}/workflows/{workflow}/deactivate` | Deactivate | workflow.update |
| POST | `/workspaces/{workspace}/workflows/{workflow}/duplicate` | Clone workflow | workflow.create |
| GET | `/workspaces/{workspace}/workflows/{workflow}/versions` | Version history | workflow.view |

### Request/Response Examples

**Create Workflow Request:**
```json
POST /api/v1/workspaces/1/workflows
{
    "name": "New Customer Notification",
    "description": "Send Slack message when new customer signs up",
    "trigger_type": "webhook",
    "trigger_config": {
        "path": "new-customer",
        "method": ["POST"]
    },
    "nodes": [
        {
            "id": "trigger_1",
            "type": "trigger_webhook",
            "position": { "x": 100, "y": 100 },
            "data": { "label": "Webhook Received" }
        },
        {
            "id": "action_1",
            "type": "action_slack_message",
            "position": { "x": 100, "y": 250 },
            "data": {
                "label": "Send Slack Message",
                "config": {
                    "channel": "#customers",
                    "message": "New customer: {{trigger.body.name}}"
                },
                "credential_id": 5
            }
        }
    ],
    "edges": [
        { "id": "e1", "source": "trigger_1", "target": "action_1" }
    ],
    "settings": {
        "retry": { "enabled": true, "max_attempts": 3 }
    }
}
```

**Response:**
```json
{
    "data": {
        "id": 15,
        "name": "New Customer Notification",
        "description": "Send Slack message when new customer signs up",
        "icon": "workflow",
        "color": "#6366f1",
        "is_active": false,
        "trigger_type": "webhook",
        "trigger_config": { ... },
        "nodes": [ ... ],
        "edges": [ ... ],
        "settings": { ... },
        "webhook_url": "https://api.linkflow.com/webhooks/abc123/new-customer",
        "execution_count": 0,
        "success_rate": 0,
        "last_executed_at": null,
        "created_by": {
            "id": 1,
            "name": "John Doe"
        },
        "created_at": "2026-01-30T10:00:00Z",
        "updated_at": "2026-01-30T10:00:00Z"
    }
}
```

### Form Requests

**StoreWorkflowRequest.php:**
```php
<?php

namespace App\Http\Requests\Api\V1\Workflow;

use App\Enums\TriggerType;
use Illuminate\Foundation\Http\FormRequest;
use Illuminate\Validation\Rule;

class StoreWorkflowRequest extends FormRequest
{
    public function authorize(): bool
    {
        return true; // Handled by controller
    }

    public function rules(): array
    {
        return [
            'name' => ['required', 'string', 'max:255'],
            'description' => ['nullable', 'string', 'max:1000'],
            'icon' => ['nullable', 'string', 'max:50'],
            'color' => ['nullable', 'string', 'regex:/^#[0-9A-Fa-f]{6}$/'],
            
            'trigger_type' => ['required', Rule::enum(TriggerType::class)],
            'trigger_config' => ['nullable', 'array'],
            'trigger_config.cron' => ['required_if:trigger_type,schedule', 'string'],
            'trigger_config.path' => ['required_if:trigger_type,webhook', 'string', 'max:100'],
            
            'nodes' => ['required', 'array', 'min:1'],
            'nodes.*.id' => ['required', 'string'],
            'nodes.*.type' => ['required', 'string'],
            'nodes.*.position' => ['required', 'array'],
            'nodes.*.position.x' => ['required', 'numeric'],
            'nodes.*.position.y' => ['required', 'numeric'],
            'nodes.*.data' => ['required', 'array'],
            
            'edges' => ['present', 'array'],
            'edges.*.id' => ['required', 'string'],
            'edges.*.source' => ['required', 'string'],
            'edges.*.target' => ['required', 'string'],
            
            'viewport' => ['nullable', 'array'],
            'settings' => ['nullable', 'array'],
            'settings.retry' => ['nullable', 'array'],
            'settings.retry.enabled' => ['boolean'],
            'settings.retry.max_attempts' => ['integer', 'min:1', 'max:10'],
            'settings.timeout' => ['nullable', 'array'],
            'settings.timeout.workflow' => ['integer', 'min:60', 'max:86400'],
            
            'tag_ids' => ['nullable', 'array'],
            'tag_ids.*' => ['exists:tags,id'],
        ];
    }
}
```

### Files to Create

```
app/
├── Enums/
│   └── TriggerType.php
├── Http/
│   ├── Controllers/Api/V1/
│   │   └── WorkflowController.php
│   ├── Requests/Api/V1/Workflow/
│   │   ├── StoreWorkflowRequest.php
│   │   └── UpdateWorkflowRequest.php
│   └── Resources/Api/V1/
│       ├── WorkflowResource.php
│       └── WorkflowCollection.php
├── Models/
│   └── Workflow.php
└── Policies/
    └── WorkflowPolicy.php

database/
├── factories/
│   └── WorkflowFactory.php
├── migrations/
│   └── 2026_01_30_000001_create_workflows_table.php
└── seeders/
    └── WorkflowSeeder.php
```

---

## 2. Nodes (Triggers & Actions)

### Overview

Nodes define what actions a workflow can perform. These are predefined templates.

### Database Schema

```sql
-- Node Categories
CREATE TABLE node_categories (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    name            VARCHAR(100) NOT NULL,
    slug            VARCHAR(100) NOT NULL UNIQUE,
    description     TEXT NULL,
    icon            VARCHAR(50) NOT NULL,
    color           VARCHAR(20) NOT NULL,
    sort_order      INT DEFAULT 0,
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- Node Definitions
CREATE TABLE nodes (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    category_id     BIGINT UNSIGNED NOT NULL,
    
    -- Identification
    type            VARCHAR(100) NOT NULL UNIQUE,    -- e.g., 'action_http_request'
    name            VARCHAR(255) NOT NULL,           -- e.g., 'HTTP Request'
    description     TEXT NULL,
    
    -- Display
    icon            VARCHAR(50) NOT NULL,
    color           VARCHAR(20) NOT NULL,
    
    -- Classification
    node_kind       ENUM('trigger', 'action', 'logic', 'transform') NOT NULL,
    
    -- Configuration Schema (JSON Schema format)
    config_schema   JSON NOT NULL,                   -- Input fields definition
    output_schema   JSON NULL,                       -- Expected output structure
    
    -- Credential requirement
    credential_type VARCHAR(100) NULL,               -- Required credential type
    
    -- Availability
    is_active       BOOLEAN DEFAULT TRUE,
    is_premium      BOOLEAN DEFAULT FALSE,           -- Pro/Business only
    
    -- Documentation
    docs_url        VARCHAR(500) NULL,
    
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (category_id) REFERENCES node_categories(id),
    INDEX idx_node_kind (node_kind),
    INDEX idx_category (category_id)
);
```

### Node Categories (Seed Data)

```php
$categories = [
    [
        'name' => 'Triggers',
        'slug' => 'triggers',
        'icon' => 'bolt',
        'color' => '#f59e0b',
        'description' => 'Start your workflow'
    ],
    [
        'name' => 'HTTP & APIs',
        'slug' => 'http',
        'icon' => 'globe',
        'color' => '#3b82f6',
        'description' => 'Make HTTP requests'
    ],
    [
        'name' => 'Communication',
        'slug' => 'communication',
        'icon' => 'chat',
        'color' => '#8b5cf6',
        'description' => 'Email, Slack, SMS'
    ],
    [
        'name' => 'Data',
        'slug' => 'data',
        'icon' => 'database',
        'color' => '#10b981',
        'description' => 'Transform and store data'
    ],
    [
        'name' => 'Logic',
        'slug' => 'logic',
        'icon' => 'code',
        'color' => '#6366f1',
        'description' => 'Conditions, loops, delays'
    ],
    [
        'name' => 'Integrations',
        'slug' => 'integrations',
        'icon' => 'puzzle',
        'color' => '#ec4899',
        'description' => 'Third-party services'
    ],
];
```

### Node Definitions (Seed Data)

```php
$nodes = [
    // ─────────────────────────────────────────────────────────────
    // TRIGGERS
    // ─────────────────────────────────────────────────────────────
    [
        'type' => 'trigger_manual',
        'name' => 'Manual Trigger',
        'category' => 'triggers',
        'node_kind' => 'trigger',
        'icon' => 'play',
        'color' => '#f59e0b',
        'description' => 'Start workflow manually',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'input_schema' => [
                    'type' => 'object',
                    'description' => 'Define expected input data'
                ]
            ]
        ],
        'output_schema' => [
            'type' => 'object',
            'properties' => [
                'input' => ['type' => 'object'],
                'triggered_at' => ['type' => 'string', 'format' => 'date-time'],
                'triggered_by' => ['type' => 'object']
            ]
        ]
    ],
    [
        'type' => 'trigger_webhook',
        'name' => 'Webhook',
        'category' => 'triggers',
        'node_kind' => 'trigger',
        'icon' => 'webhook',
        'color' => '#f59e0b',
        'description' => 'Receive HTTP webhook calls',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'method' => [
                    'type' => 'array',
                    'items' => ['type' => 'string', 'enum' => ['GET', 'POST', 'PUT', 'DELETE']],
                    'default' => ['POST']
                ],
                'path' => [
                    'type' => 'string',
                    'description' => 'Custom webhook path'
                ],
                'authentication' => [
                    'type' => 'object',
                    'properties' => [
                        'type' => ['type' => 'string', 'enum' => ['none', 'header', 'basic', 'bearer']],
                        'header_name' => ['type' => 'string'],
                        'credential_id' => ['type' => 'integer']
                    ]
                ],
                'response' => [
                    'type' => 'object',
                    'properties' => [
                        'status_code' => ['type' => 'integer', 'default' => 200],
                        'body' => ['type' => 'string']
                    ]
                ]
            ],
            'required' => ['method']
        ],
        'output_schema' => [
            'type' => 'object',
            'properties' => [
                'headers' => ['type' => 'object'],
                'query' => ['type' => 'object'],
                'body' => ['type' => 'object'],
                'method' => ['type' => 'string'],
                'ip' => ['type' => 'string']
            ]
        ]
    ],
    [
        'type' => 'trigger_schedule',
        'name' => 'Schedule',
        'category' => 'triggers',
        'node_kind' => 'trigger',
        'icon' => 'clock',
        'color' => '#f59e0b',
        'description' => 'Run on a schedule (cron)',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'cron' => [
                    'type' => 'string',
                    'description' => 'Cron expression (e.g., "0 9 * * 1-5")'
                ],
                'timezone' => [
                    'type' => 'string',
                    'default' => 'UTC'
                ]
            ],
            'required' => ['cron']
        ],
        'output_schema' => [
            'type' => 'object',
            'properties' => [
                'scheduled_time' => ['type' => 'string', 'format' => 'date-time'],
                'execution_time' => ['type' => 'string', 'format' => 'date-time']
            ]
        ]
    ],
    
    // ─────────────────────────────────────────────────────────────
    // HTTP & APIS
    // ─────────────────────────────────────────────────────────────
    [
        'type' => 'action_http_request',
        'name' => 'HTTP Request',
        'category' => 'http',
        'node_kind' => 'action',
        'icon' => 'globe',
        'color' => '#3b82f6',
        'description' => 'Make HTTP requests to any URL',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'url' => [
                    'type' => 'string',
                    'format' => 'uri',
                    'description' => 'Request URL'
                ],
                'method' => [
                    'type' => 'string',
                    'enum' => ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'],
                    'default' => 'GET'
                ],
                'headers' => [
                    'type' => 'object',
                    'additionalProperties' => ['type' => 'string']
                ],
                'query_params' => [
                    'type' => 'object',
                    'additionalProperties' => ['type' => 'string']
                ],
                'body_type' => [
                    'type' => 'string',
                    'enum' => ['none', 'json', 'form', 'raw', 'binary'],
                    'default' => 'none'
                ],
                'body' => [
                    'type' => ['object', 'string', 'null']
                ],
                'timeout' => [
                    'type' => 'integer',
                    'default' => 30,
                    'minimum' => 1,
                    'maximum' => 300
                ],
                'follow_redirects' => [
                    'type' => 'boolean',
                    'default' => true
                ],
                'ignore_ssl' => [
                    'type' => 'boolean',
                    'default' => false
                ]
            ],
            'required' => ['url', 'method']
        ],
        'output_schema' => [
            'type' => 'object',
            'properties' => [
                'status' => ['type' => 'integer'],
                'status_text' => ['type' => 'string'],
                'headers' => ['type' => 'object'],
                'body' => ['type' => ['object', 'string', 'array']],
                'duration_ms' => ['type' => 'integer']
            ]
        ]
    ],
    [
        'type' => 'action_graphql',
        'name' => 'GraphQL',
        'category' => 'http',
        'node_kind' => 'action',
        'icon' => 'graphql',
        'color' => '#e535ab',
        'description' => 'Execute GraphQL queries',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'endpoint' => ['type' => 'string', 'format' => 'uri'],
                'query' => ['type' => 'string'],
                'variables' => ['type' => 'object'],
                'headers' => ['type' => 'object']
            ],
            'required' => ['endpoint', 'query']
        ]
    ],
    
    // ─────────────────────────────────────────────────────────────
    // COMMUNICATION
    // ─────────────────────────────────────────────────────────────
    [
        'type' => 'action_send_email',
        'name' => 'Send Email',
        'category' => 'communication',
        'node_kind' => 'action',
        'icon' => 'envelope',
        'color' => '#8b5cf6',
        'description' => 'Send an email',
        'credential_type' => 'smtp',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'to' => [
                    'type' => 'array',
                    'items' => ['type' => 'string', 'format' => 'email']
                ],
                'cc' => [
                    'type' => 'array',
                    'items' => ['type' => 'string', 'format' => 'email']
                ],
                'bcc' => [
                    'type' => 'array',
                    'items' => ['type' => 'string', 'format' => 'email']
                ],
                'subject' => ['type' => 'string'],
                'body' => ['type' => 'string'],
                'body_type' => ['type' => 'string', 'enum' => ['text', 'html']],
                'attachments' => [
                    'type' => 'array',
                    'items' => [
                        'type' => 'object',
                        'properties' => [
                            'filename' => ['type' => 'string'],
                            'content' => ['type' => 'string'],
                            'content_type' => ['type' => 'string']
                        ]
                    ]
                ]
            ],
            'required' => ['to', 'subject', 'body']
        ]
    ],
    [
        'type' => 'action_slack_message',
        'name' => 'Slack Message',
        'category' => 'communication',
        'node_kind' => 'action',
        'icon' => 'slack',
        'color' => '#4a154b',
        'credential_type' => 'slack',
        'description' => 'Send a Slack message',
        'is_premium' => true,
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'channel' => ['type' => 'string'],
                'message' => ['type' => 'string'],
                'blocks' => ['type' => 'array'],
                'thread_ts' => ['type' => 'string'],
                'reply_broadcast' => ['type' => 'boolean']
            ],
            'required' => ['channel', 'message']
        ]
    ],
    [
        'type' => 'action_discord_message',
        'name' => 'Discord Message',
        'category' => 'communication',
        'node_kind' => 'action',
        'icon' => 'discord',
        'color' => '#5865f2',
        'credential_type' => 'discord_webhook',
        'description' => 'Send a Discord message',
        'is_premium' => true,
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'content' => ['type' => 'string'],
                'username' => ['type' => 'string'],
                'avatar_url' => ['type' => 'string', 'format' => 'uri'],
                'embeds' => ['type' => 'array']
            ],
            'required' => ['content']
        ]
    ],
    [
        'type' => 'action_sms',
        'name' => 'Send SMS',
        'category' => 'communication',
        'node_kind' => 'action',
        'icon' => 'phone',
        'color' => '#22c55e',
        'credential_type' => 'twilio',
        'description' => 'Send SMS via Twilio',
        'is_premium' => true,
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'to' => ['type' => 'string'],
                'message' => ['type' => 'string', 'maxLength' => 1600]
            ],
            'required' => ['to', 'message']
        ]
    ],
    
    // ─────────────────────────────────────────────────────────────
    // LOGIC
    // ─────────────────────────────────────────────────────────────
    [
        'type' => 'logic_condition',
        'name' => 'IF Condition',
        'category' => 'logic',
        'node_kind' => 'logic',
        'icon' => 'git-branch',
        'color' => '#6366f1',
        'description' => 'Branch based on conditions',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'conditions' => [
                    'type' => 'array',
                    'items' => [
                        'type' => 'object',
                        'properties' => [
                            'field' => ['type' => 'string'],
                            'operator' => [
                                'type' => 'string',
                                'enum' => [
                                    'equals', 'not_equals',
                                    'contains', 'not_contains',
                                    'starts_with', 'ends_with',
                                    'greater_than', 'less_than',
                                    'greater_or_equal', 'less_or_equal',
                                    'is_empty', 'is_not_empty',
                                    'is_true', 'is_false',
                                    'matches_regex'
                                ]
                            ],
                            'value' => ['type' => ['string', 'number', 'boolean', 'null']]
                        ]
                    ]
                ],
                'combine' => [
                    'type' => 'string',
                    'enum' => ['and', 'or'],
                    'default' => 'and'
                ]
            ],
            'required' => ['conditions']
        ],
        'output_schema' => [
            'type' => 'object',
            'properties' => [
                'result' => ['type' => 'boolean'],
                'matched_conditions' => ['type' => 'array']
            ]
        ]
    ],
    [
        'type' => 'logic_switch',
        'name' => 'Switch',
        'category' => 'logic',
        'node_kind' => 'logic',
        'icon' => 'switch',
        'color' => '#6366f1',
        'description' => 'Route to different paths based on value',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'value' => ['type' => 'string'],
                'cases' => [
                    'type' => 'array',
                    'items' => [
                        'type' => 'object',
                        'properties' => [
                            'value' => ['type' => 'string'],
                            'output' => ['type' => 'string']
                        ]
                    ]
                ],
                'default_output' => ['type' => 'string']
            ],
            'required' => ['value', 'cases']
        ]
    ],
    [
        'type' => 'logic_delay',
        'name' => 'Delay',
        'category' => 'logic',
        'node_kind' => 'logic',
        'icon' => 'clock',
        'color' => '#6366f1',
        'description' => 'Wait before continuing',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'delay_type' => [
                    'type' => 'string',
                    'enum' => ['fixed', 'until']
                ],
                'duration' => ['type' => 'integer'],
                'unit' => [
                    'type' => 'string',
                    'enum' => ['seconds', 'minutes', 'hours', 'days']
                ],
                'until' => ['type' => 'string', 'format' => 'date-time']
            ],
            'required' => ['delay_type']
        ]
    ],
    [
        'type' => 'logic_loop',
        'name' => 'Loop',
        'category' => 'logic',
        'node_kind' => 'logic',
        'icon' => 'refresh',
        'color' => '#6366f1',
        'description' => 'Iterate over array items',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'items' => ['type' => 'string', 'description' => 'Expression returning array'],
                'batch_size' => ['type' => 'integer', 'default' => 1],
                'parallel' => ['type' => 'boolean', 'default' => false]
            ],
            'required' => ['items']
        ]
    ],
    [
        'type' => 'logic_stop',
        'name' => 'Stop Workflow',
        'category' => 'logic',
        'node_kind' => 'logic',
        'icon' => 'stop',
        'color' => '#ef4444',
        'description' => 'Stop workflow execution',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'status' => [
                    'type' => 'string',
                    'enum' => ['success', 'error', 'cancelled']
                ],
                'message' => ['type' => 'string']
            ]
        ]
    ],
    
    // ─────────────────────────────────────────────────────────────
    // DATA TRANSFORM
    // ─────────────────────────────────────────────────────────────
    [
        'type' => 'transform_set',
        'name' => 'Set Values',
        'category' => 'data',
        'node_kind' => 'transform',
        'icon' => 'edit',
        'color' => '#10b981',
        'description' => 'Set or modify data values',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'values' => [
                    'type' => 'array',
                    'items' => [
                        'type' => 'object',
                        'properties' => [
                            'key' => ['type' => 'string'],
                            'value' => ['type' => ['string', 'number', 'boolean', 'object', 'array']]
                        ]
                    ]
                ],
                'mode' => [
                    'type' => 'string',
                    'enum' => ['set', 'append', 'merge'],
                    'default' => 'set'
                ]
            ],
            'required' => ['values']
        ]
    ],
    [
        'type' => 'transform_code',
        'name' => 'Code (JavaScript)',
        'category' => 'data',
        'node_kind' => 'transform',
        'icon' => 'code',
        'color' => '#f59e0b',
        'description' => 'Run custom JavaScript code',
        'is_premium' => true,
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'code' => ['type' => 'string'],
                'timeout' => ['type' => 'integer', 'default' => 10]
            ],
            'required' => ['code']
        ]
    ],
    [
        'type' => 'transform_json_parse',
        'name' => 'Parse JSON',
        'category' => 'data',
        'node_kind' => 'transform',
        'icon' => 'brackets',
        'color' => '#10b981',
        'description' => 'Parse JSON string to object',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'input' => ['type' => 'string']
            ],
            'required' => ['input']
        ]
    ],
    [
        'type' => 'transform_filter',
        'name' => 'Filter Array',
        'category' => 'data',
        'node_kind' => 'transform',
        'icon' => 'filter',
        'color' => '#10b981',
        'description' => 'Filter array items by condition',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'input' => ['type' => 'string'],
                'condition' => [
                    'type' => 'object',
                    'properties' => [
                        'field' => ['type' => 'string'],
                        'operator' => ['type' => 'string'],
                        'value' => ['type' => ['string', 'number', 'boolean']]
                    ]
                ]
            ],
            'required' => ['input', 'condition']
        ]
    ],
    [
        'type' => 'transform_aggregate',
        'name' => 'Aggregate',
        'category' => 'data',
        'node_kind' => 'transform',
        'icon' => 'calculator',
        'color' => '#10b981',
        'description' => 'Aggregate array data (sum, avg, count)',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'input' => ['type' => 'string'],
                'operations' => [
                    'type' => 'array',
                    'items' => [
                        'type' => 'object',
                        'properties' => [
                            'field' => ['type' => 'string'],
                            'operation' => [
                                'type' => 'string',
                                'enum' => ['sum', 'avg', 'min', 'max', 'count', 'first', 'last']
                            ],
                            'output_key' => ['type' => 'string']
                        ]
                    ]
                ]
            ],
            'required' => ['input', 'operations']
        ]
    ],
    
    // ─────────────────────────────────────────────────────────────
    // INTEGRATIONS
    // ─────────────────────────────────────────────────────────────
    [
        'type' => 'integration_google_sheets',
        'name' => 'Google Sheets',
        'category' => 'integrations',
        'node_kind' => 'action',
        'icon' => 'sheets',
        'color' => '#34a853',
        'credential_type' => 'google',
        'is_premium' => true,
        'description' => 'Read/write Google Sheets',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'operation' => [
                    'type' => 'string',
                    'enum' => ['read', 'append', 'update', 'clear']
                ],
                'spreadsheet_id' => ['type' => 'string'],
                'sheet_name' => ['type' => 'string'],
                'range' => ['type' => 'string'],
                'data' => ['type' => 'array']
            ],
            'required' => ['operation', 'spreadsheet_id']
        ]
    ],
    [
        'type' => 'integration_airtable',
        'name' => 'Airtable',
        'category' => 'integrations',
        'node_kind' => 'action',
        'icon' => 'airtable',
        'color' => '#fcb400',
        'credential_type' => 'airtable',
        'is_premium' => true,
        'description' => 'Airtable operations',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'operation' => [
                    'type' => 'string',
                    'enum' => ['list', 'get', 'create', 'update', 'delete']
                ],
                'base_id' => ['type' => 'string'],
                'table_name' => ['type' => 'string'],
                'record_id' => ['type' => 'string'],
                'fields' => ['type' => 'object'],
                'filter' => ['type' => 'string']
            ],
            'required' => ['operation', 'base_id', 'table_name']
        ]
    ],
    [
        'type' => 'integration_stripe',
        'name' => 'Stripe',
        'category' => 'integrations',
        'node_kind' => 'action',
        'icon' => 'stripe',
        'color' => '#635bff',
        'credential_type' => 'stripe',
        'is_premium' => true,
        'description' => 'Stripe payment operations',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'resource' => [
                    'type' => 'string',
                    'enum' => ['customer', 'charge', 'invoice', 'subscription', 'payment_intent']
                ],
                'operation' => [
                    'type' => 'string',
                    'enum' => ['list', 'get', 'create', 'update', 'delete']
                ],
                'data' => ['type' => 'object']
            ],
            'required' => ['resource', 'operation']
        ]
    ],
    [
        'type' => 'integration_github',
        'name' => 'GitHub',
        'category' => 'integrations',
        'node_kind' => 'action',
        'icon' => 'github',
        'color' => '#181717',
        'credential_type' => 'github',
        'is_premium' => true,
        'description' => 'GitHub API operations',
        'config_schema' => [
            'type' => 'object',
            'properties' => [
                'resource' => [
                    'type' => 'string',
                    'enum' => ['issue', 'pull_request', 'repository', 'release', 'workflow']
                ],
                'operation' => [
                    'type' => 'string',
                    'enum' => ['list', 'get', 'create', 'update', 'delete', 'trigger']
                ],
                'owner' => ['type' => 'string'],
                'repo' => ['type' => 'string'],
                'data' => ['type' => 'object']
            ],
            'required' => ['resource', 'operation', 'owner', 'repo']
        ]
    ],
];
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/nodes` | List all available nodes |
| GET | `/nodes/categories` | List node categories |
| GET | `/nodes/{type}` | Get node details by type |
| GET | `/nodes/search?q=` | Search nodes |

### Files to Create

```
app/
├── Http/
│   ├── Controllers/Api/V1/
│   │   └── NodeController.php
│   └── Resources/Api/V1/
│       ├── NodeResource.php
│       └── NodeCategoryResource.php
├── Models/
│   ├── Node.php
│   └── NodeCategory.php

database/
├── migrations/
│   ├── 2026_01_30_000002_create_node_categories_table.php
│   └── 2026_01_30_000003_create_nodes_table.php
└── seeders/
    ├── NodeCategorySeeder.php
    └── NodeSeeder.php
```

---

## 3. Credentials

### Overview

Securely store API keys, tokens, and other secrets for use in workflows.

### Database Schema

```sql
-- Credential Types (templates)
CREATE TABLE credential_types (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    type            VARCHAR(100) NOT NULL UNIQUE,    -- e.g., 'slack', 'stripe'
    name            VARCHAR(255) NOT NULL,
    description     TEXT NULL,
    icon            VARCHAR(50) NOT NULL,
    color           VARCHAR(20) NOT NULL,
    
    -- Field Schema
    fields_schema   JSON NOT NULL,                   -- What fields are needed
    
    -- Test endpoint
    test_config     JSON NULL,                       -- How to test the credential
    
    -- Documentation
    docs_url        VARCHAR(500) NULL,
    
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- User Credentials
CREATE TABLE credentials (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workspace_id    BIGINT UNSIGNED NOT NULL,
    created_by      BIGINT UNSIGNED NOT NULL,
    
    -- Info
    name            VARCHAR(255) NOT NULL,
    type            VARCHAR(100) NOT NULL,           -- References credential_types.type
    
    -- Encrypted Data
    data            TEXT NOT NULL,                   -- AES-256 encrypted JSON
    
    -- Metadata
    last_used_at    TIMESTAMP NULL,
    expires_at      TIMESTAMP NULL,
    
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at      TIMESTAMP NULL,
    
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id),
    INDEX idx_workspace_type (workspace_id, type),
    UNIQUE KEY unique_workspace_name (workspace_id, name, deleted_at)
);

-- Track which workflows use which credentials
CREATE TABLE workflow_credentials (
    id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workflow_id     BIGINT UNSIGNED NOT NULL,
    credential_id   BIGINT UNSIGNED NOT NULL,
    node_id         VARCHAR(100) NOT NULL,           -- Which node uses it
    
    created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    FOREIGN KEY (credential_id) REFERENCES credentials(id) ON DELETE CASCADE,
    UNIQUE KEY unique_workflow_node (workflow_id, node_id)
);
```

### Credential Types (Seed Data)

```php
$types = [
    [
        'type' => 'api_key',
        'name' => 'API Key',
        'icon' => 'key',
        'color' => '#6366f1',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'api_key' => [
                    'type' => 'string',
                    'title' => 'API Key',
                    'secret' => true
                ],
                'header_name' => [
                    'type' => 'string',
                    'title' => 'Header Name',
                    'default' => 'X-API-Key'
                ]
            ],
            'required' => ['api_key']
        ]
    ],
    [
        'type' => 'bearer_token',
        'name' => 'Bearer Token',
        'icon' => 'shield',
        'color' => '#8b5cf6',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'token' => [
                    'type' => 'string',
                    'title' => 'Token',
                    'secret' => true
                ]
            ],
            'required' => ['token']
        ]
    ],
    [
        'type' => 'basic_auth',
        'name' => 'Basic Auth',
        'icon' => 'user',
        'color' => '#10b981',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'username' => ['type' => 'string', 'title' => 'Username'],
                'password' => ['type' => 'string', 'title' => 'Password', 'secret' => true]
            ],
            'required' => ['username', 'password']
        ]
    ],
    [
        'type' => 'oauth2',
        'name' => 'OAuth2',
        'icon' => 'lock',
        'color' => '#f59e0b',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'client_id' => ['type' => 'string', 'title' => 'Client ID'],
                'client_secret' => ['type' => 'string', 'title' => 'Client Secret', 'secret' => true],
                'access_token' => ['type' => 'string', 'title' => 'Access Token', 'secret' => true],
                'refresh_token' => ['type' => 'string', 'title' => 'Refresh Token', 'secret' => true],
                'token_url' => ['type' => 'string', 'title' => 'Token URL']
            ],
            'required' => ['access_token']
        ]
    ],
    [
        'type' => 'smtp',
        'name' => 'SMTP',
        'icon' => 'envelope',
        'color' => '#ec4899',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'host' => ['type' => 'string', 'title' => 'SMTP Host'],
                'port' => ['type' => 'integer', 'title' => 'Port', 'default' => 587],
                'username' => ['type' => 'string', 'title' => 'Username'],
                'password' => ['type' => 'string', 'title' => 'Password', 'secret' => true],
                'encryption' => ['type' => 'string', 'enum' => ['tls', 'ssl', 'none'], 'default' => 'tls'],
                'from_email' => ['type' => 'string', 'title' => 'From Email'],
                'from_name' => ['type' => 'string', 'title' => 'From Name']
            ],
            'required' => ['host', 'port', 'username', 'password']
        ],
        'test_config' => [
            'method' => 'smtp_connect',
            'success_message' => 'SMTP connection successful'
        ]
    ],
    [
        'type' => 'slack',
        'name' => 'Slack',
        'icon' => 'slack',
        'color' => '#4a154b',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'bot_token' => [
                    'type' => 'string',
                    'title' => 'Bot Token',
                    'description' => 'Starts with xoxb-',
                    'secret' => true
                ]
            ],
            'required' => ['bot_token']
        ],
        'test_config' => [
            'method' => 'http',
            'url' => 'https://slack.com/api/auth.test',
            'headers' => ['Authorization' => 'Bearer {{bot_token}}']
        ],
        'docs_url' => 'https://api.slack.com/authentication/token-types'
    ],
    [
        'type' => 'stripe',
        'name' => 'Stripe',
        'icon' => 'stripe',
        'color' => '#635bff',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'secret_key' => [
                    'type' => 'string',
                    'title' => 'Secret Key',
                    'description' => 'Starts with sk_',
                    'secret' => true
                ],
                'publishable_key' => [
                    'type' => 'string',
                    'title' => 'Publishable Key',
                    'description' => 'Starts with pk_'
                ]
            ],
            'required' => ['secret_key']
        ],
        'test_config' => [
            'method' => 'http',
            'url' => 'https://api.stripe.com/v1/balance',
            'headers' => ['Authorization' => 'Bearer {{secret_key}}']
        ]
    ],
    [
        'type' => 'github',
        'name' => 'GitHub',
        'icon' => 'github',
        'color' => '#181717',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'token' => [
                    'type' => 'string',
                    'title' => 'Personal Access Token',
                    'secret' => true
                ]
            ],
            'required' => ['token']
        ],
        'test_config' => [
            'method' => 'http',
            'url' => 'https://api.github.com/user',
            'headers' => ['Authorization' => 'token {{token}}']
        ]
    ],
    [
        'type' => 'twilio',
        'name' => 'Twilio',
        'icon' => 'phone',
        'color' => '#f22f46',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'account_sid' => ['type' => 'string', 'title' => 'Account SID'],
                'auth_token' => ['type' => 'string', 'title' => 'Auth Token', 'secret' => true],
                'from_number' => ['type' => 'string', 'title' => 'From Phone Number']
            ],
            'required' => ['account_sid', 'auth_token', 'from_number']
        ]
    ],
    [
        'type' => 'google',
        'name' => 'Google',
        'icon' => 'google',
        'color' => '#4285f4',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'client_id' => ['type' => 'string', 'title' => 'Client ID'],
                'client_secret' => ['type' => 'string', 'title' => 'Client Secret', 'secret' => true],
                'refresh_token' => ['type' => 'string', 'title' => 'Refresh Token', 'secret' => true],
                'scopes' => ['type' => 'array', 'items' => ['type' => 'string']]
            ],
            'required' => ['client_id', 'client_secret', 'refresh_token']
        ]
    ],
    [
        'type' => 'database_mysql',
        'name' => 'MySQL',
        'icon' => 'database',
        'color' => '#00758f',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'host' => ['type' => 'string', 'title' => 'Host', 'default' => 'localhost'],
                'port' => ['type' => 'integer', 'title' => 'Port', 'default' => 3306],
                'database' => ['type' => 'string', 'title' => 'Database Name'],
                'username' => ['type' => 'string', 'title' => 'Username'],
                'password' => ['type' => 'string', 'title' => 'Password', 'secret' => true],
                'ssl' => ['type' => 'boolean', 'title' => 'Use SSL', 'default' => false]
            ],
            'required' => ['host', 'database', 'username', 'password']
        ]
    ],
    [
        'type' => 'database_postgres',
        'name' => 'PostgreSQL',
        'icon' => 'database',
        'color' => '#336791',
        'fields_schema' => [
            'type' => 'object',
            'properties' => [
                'host' => ['type' => 'string', 'title' => 'Host', 'default' => 'localhost'],
                'port' => ['type' => 'integer', 'title' => 'Port', 'default' => 5432],
                'database' => ['type' => 'string', 'title' => 'Database Name'],
                'username' => ['type' => 'string', 'title' => 'Username'],
                'password' => ['type' => 'string', 'title' => 'Password', 'secret' => true],
                'ssl_mode' => ['type' => 'string', 'enum' => ['disable', 'require', 'verify-full']]
            ],
            'required' => ['host', 'database', 'username', 'password']
        ]
    ],
];
```

### Encryption Service

```php
<?php

namespace App\Services;

use Illuminate\Support\Facades\Crypt;
use Illuminate\Contracts\Encryption\DecryptException;

class CredentialEncryptionService
{
    public function encrypt(array $data): string
    {
        return Crypt::encryptString(json_encode($data));
    }

    public function decrypt(string $encrypted): array
    {
        try {
            $decrypted = Crypt::decryptString($encrypted);
            return json_decode($decrypted, true);
        } catch (DecryptException $e) {
            throw new \RuntimeException('Failed to decrypt credential data');
        }
    }

    public function mask(array $data, array $schema): array
    {
        $masked = [];
        
        foreach ($data as $key => $value) {
            $fieldSchema = $schema['properties'][$key] ?? [];
            
            if (isset($fieldSchema['secret']) && $fieldSchema['secret']) {
                // Mask secret fields
                $masked[$key] = str_repeat('•', 8) . substr($value, -4);
            } else {
                $masked[$key] = $value;
            }
        }
        
        return $masked;
    }
}
```

### API Endpoints

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/credential-types` | List credential types | - |
| GET | `/workspaces/{workspace}/credentials` | List credentials | credential.view |
| POST | `/workspaces/{workspace}/credentials` | Create credential | credential.create |
| GET | `/workspaces/{workspace}/credentials/{credential}` | Get credential (masked) | credential.view |
| PUT | `/workspaces/{workspace}/credentials/{credential}` | Update credential | credential.update |
| DELETE | `/workspaces/{workspace}/credentials/{credential}` | Delete credential | credential.delete |
| POST | `/workspaces/{workspace}/credentials/{credential}/test` | Test credential | credential.view |

### Files to Create

```
app/
├── Http/
│   ├── Controllers/Api/V1/
│   │   ├── CredentialController.php
│   │   └── CredentialTypeController.php
│   ├── Requests/Api/V1/Credential/
│   │   ├── StoreCredentialRequest.php
│   │   └── UpdateCredentialRequest.php
│   └── Resources/Api/V1/
│       ├── CredentialResource.php
│       └── CredentialTypeResource.php
├── Models/
│   ├── Credential.php
│   └── CredentialType.php
└── Services/
    └── CredentialEncryptionService.php

database/
├── migrations/
│   ├── 2026_01_30_000004_create_credential_types_table.php
│   ├── 2026_01_30_000005_create_credentials_table.php
│   └── 2026_01_30_000006_create_workflow_credentials_table.php
└── seeders/
    └── CredentialTypeSeeder.php
```

---

## 4. Executions

### Overview

Track every workflow run with detailed node-by-node execution data.

### Database Schema

```sql
-- Workflow Executions
CREATE TABLE executions (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workflow_id         BIGINT UNSIGNED NOT NULL,
    workspace_id        BIGINT UNSIGNED NOT NULL,      -- Denormalized for queries
    
    -- Execution Info
    status              ENUM('pending', 'running', 'completed', 'failed', 'cancelled', 'waiting') NOT NULL DEFAULT 'pending',
    mode                ENUM('manual', 'webhook', 'schedule', 'retry') NOT NULL,
    triggered_by        BIGINT UNSIGNED NULL,          -- User who triggered (if manual)
    
    -- Timing
    started_at          TIMESTAMP NULL,
    finished_at         TIMESTAMP NULL,
    duration_ms         INT UNSIGNED NULL,
    
    -- Data
    trigger_data        JSON NULL,                     -- Input data that triggered
    result_data         JSON NULL,                     -- Final output
    error               JSON NULL,                     -- Error details if failed
    
    -- Retry Info
    attempt             INT UNSIGNED DEFAULT 1,
    max_attempts        INT UNSIGNED DEFAULT 1,
    parent_execution_id BIGINT UNSIGNED NULL,          -- If this is a retry
    
    -- Metadata
    ip_address          VARCHAR(45) NULL,
    user_agent          VARCHAR(500) NULL,
    
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (triggered_by) REFERENCES users(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_execution_id) REFERENCES executions(id) ON DELETE SET NULL,
    INDEX idx_workflow_status (workflow_id, status),
    INDEX idx_workspace_created (workspace_id, created_at),
    INDEX idx_status_created (status, created_at)
);

-- Individual Node Executions
CREATE TABLE execution_nodes (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    execution_id        BIGINT UNSIGNED NOT NULL,
    
    -- Node Info
    node_id             VARCHAR(100) NOT NULL,         -- Matches workflow node id
    node_type           VARCHAR(100) NOT NULL,
    node_name           VARCHAR(255) NULL,
    
    -- Status
    status              ENUM('pending', 'running', 'completed', 'failed', 'skipped') NOT NULL DEFAULT 'pending',
    
    -- Timing
    started_at          TIMESTAMP(3) NULL,             -- Millisecond precision
    finished_at         TIMESTAMP(3) NULL,
    duration_ms         INT UNSIGNED NULL,
    
    -- Data
    input_data          JSON NULL,                     -- What went into the node
    output_data         JSON NULL,                     -- What came out
    error               JSON NULL,                     -- Error if failed
    
    -- Execution order
    sequence            INT UNSIGNED NOT NULL,
    
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (execution_id) REFERENCES executions(id) ON DELETE CASCADE,
    INDEX idx_execution_sequence (execution_id, sequence)
);

-- Execution Logs (detailed logs for debugging)
CREATE TABLE execution_logs (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    execution_id        BIGINT UNSIGNED NOT NULL,
    execution_node_id   BIGINT UNSIGNED NULL,
    
    level               ENUM('debug', 'info', 'warning', 'error') NOT NULL DEFAULT 'info',
    message             TEXT NOT NULL,
    context             JSON NULL,
    
    logged_at           TIMESTAMP(3) DEFAULT CURRENT_TIMESTAMP(3),
    
    FOREIGN KEY (execution_id) REFERENCES executions(id) ON DELETE CASCADE,
    FOREIGN KEY (execution_node_id) REFERENCES execution_nodes(id) ON DELETE CASCADE,
    INDEX idx_execution_level (execution_id, level)
);
```

### API Endpoints

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/workspaces/{workspace}/executions` | List all executions | execution.view |
| GET | `/workspaces/{workspace}/executions/{execution}` | Get execution detail | execution.view |
| GET | `/workspaces/{workspace}/executions/{execution}/nodes` | Get node executions | execution.view |
| GET | `/workspaces/{workspace}/executions/{execution}/logs` | Get execution logs | execution.view |
| DELETE | `/workspaces/{workspace}/executions/{execution}` | Delete execution | execution.delete |
| POST | `/workspaces/{workspace}/executions/{execution}/retry` | Retry failed execution | workflow.execute |
| POST | `/workspaces/{workspace}/executions/{execution}/cancel` | Cancel running execution | workflow.execute |
| GET | `/workspaces/{workspace}/workflows/{workflow}/executions` | Workflow executions | execution.view |
| GET | `/workspaces/{workspace}/executions/stats` | Execution statistics | execution.view |

### Response Example

```json
{
    "data": {
        "id": 12345,
        "workflow": {
            "id": 15,
            "name": "New Customer Notification"
        },
        "status": "completed",
        "mode": "webhook",
        "started_at": "2026-01-30T10:00:00.000Z",
        "finished_at": "2026-01-30T10:00:01.245Z",
        "duration_ms": 1245,
        "trigger_data": {
            "body": { "name": "John Doe", "email": "john@example.com" },
            "headers": { "content-type": "application/json" }
        },
        "result_data": {
            "slack_message_ts": "1234567890.123456"
        },
        "nodes": [
            {
                "id": "trigger_1",
                "type": "trigger_webhook",
                "name": "Webhook Received",
                "status": "completed",
                "duration_ms": 5,
                "output_data": { "body": { ... } }
            },
            {
                "id": "action_1",
                "type": "action_slack_message",
                "name": "Send Slack Message",
                "status": "completed",
                "duration_ms": 1240,
                "input_data": { "channel": "#customers", "message": "..." },
                "output_data": { "ok": true, "ts": "1234567890.123456" }
            }
        ],
        "attempt": 1,
        "triggered_by": null,
        "ip_address": "192.168.1.1"
    }
}
```

### Files to Create

```
app/
├── Http/
│   ├── Controllers/Api/V1/
│   │   └── ExecutionController.php
│   └── Resources/Api/V1/
│       ├── ExecutionResource.php
│       ├── ExecutionNodeResource.php
│       └── ExecutionLogResource.php
├── Models/
│   ├── Execution.php
│   ├── ExecutionNode.php
│   └── ExecutionLog.php

database/
├── migrations/
│   ├── 2026_01_30_000007_create_executions_table.php
│   ├── 2026_01_30_000008_create_execution_nodes_table.php
│   └── 2026_01_30_000009_create_execution_logs_table.php
```

---

## 5. Webhooks

### Overview

HTTP endpoints that trigger workflows when called.

### Database Schema

```sql
CREATE TABLE webhooks (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workflow_id         BIGINT UNSIGNED NOT NULL UNIQUE,
    workspace_id        BIGINT UNSIGNED NOT NULL,
    
    -- Webhook Path
    uuid                CHAR(36) NOT NULL UNIQUE,      -- Public identifier
    path                VARCHAR(100) NULL,             -- Custom path (optional)
    
    -- Configuration
    methods             JSON NOT NULL DEFAULT '["POST"]',
    is_active           BOOLEAN DEFAULT TRUE,
    
    -- Authentication
    auth_type           ENUM('none', 'header', 'basic', 'bearer') DEFAULT 'none',
    auth_config         JSON NULL,                     -- Encrypted auth details
    
    -- Rate Limiting
    rate_limit          INT UNSIGNED NULL,             -- Requests per minute
    
    -- Response Config
    response_mode       ENUM('immediate', 'wait') DEFAULT 'immediate',
    response_status     INT DEFAULT 200,
    response_body       JSON NULL,
    
    -- Statistics
    call_count          BIGINT UNSIGNED DEFAULT 0,
    last_called_at      TIMESTAMP NULL,
    
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    INDEX idx_uuid (uuid),
    INDEX idx_path (workspace_id, path)
);
```

### Webhook URL Format

```
https://api.linkflow.com/webhooks/{uuid}
https://api.linkflow.com/webhooks/{uuid}/{custom-path}
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| ANY | `/webhooks/{uuid}` | Receive webhook call |
| ANY | `/webhooks/{uuid}/{path}` | Receive webhook with custom path |

### Webhook Controller (Public)

```php
<?php

namespace App\Http\Controllers\Api;

use App\Http\Controllers\Controller;
use App\Models\Webhook;
use App\Jobs\ExecuteWorkflowJob;
use Illuminate\Http\Request;

class WebhookReceiverController extends Controller
{
    public function handle(Request $request, string $uuid, ?string $path = null)
    {
        $webhook = Webhook::where('uuid', $uuid)
            ->where('is_active', true)
            ->with('workflow')
            ->first();

        if (!$webhook) {
            return response()->json(['error' => 'Webhook not found'], 404);
        }

        // Validate custom path if set
        if ($webhook->path && $webhook->path !== $path) {
            return response()->json(['error' => 'Invalid webhook path'], 404);
        }

        // Validate HTTP method
        if (!in_array($request->method(), $webhook->methods)) {
            return response()->json(['error' => 'Method not allowed'], 405);
        }

        // Validate authentication
        if (!$this->validateAuth($request, $webhook)) {
            return response()->json(['error' => 'Unauthorized'], 401);
        }

        // Check rate limit
        if (!$this->checkRateLimit($webhook)) {
            return response()->json(['error' => 'Rate limit exceeded'], 429);
        }

        // Prepare trigger data
        $triggerData = [
            'method' => $request->method(),
            'headers' => $request->headers->all(),
            'query' => $request->query(),
            'body' => $request->all(),
            'ip' => $request->ip(),
            'path' => $path,
        ];

        // Dispatch workflow execution
        ExecuteWorkflowJob::dispatch(
            $webhook->workflow,
            'webhook',
            $triggerData
        );

        // Update statistics
        $webhook->increment('call_count');
        $webhook->update(['last_called_at' => now()]);

        // Return response
        if ($webhook->response_mode === 'wait') {
            // Wait for execution (sync mode)
            // ... implementation
        }

        return response()->json(
            $webhook->response_body ?? ['success' => true],
            $webhook->response_status
        );
    }
}
```

### Files to Create

```
app/
├── Http/
│   ├── Controllers/
│   │   └── Api/
│   │       └── WebhookReceiverController.php      # Public webhook handler
│   └── Controllers/Api/V1/
│       └── WebhookController.php                   # Manage webhooks
├── Models/
│   └── Webhook.php

database/
├── migrations/
│   └── 2026_01_30_000010_create_webhooks_table.php

routes/
└── webhooks.php                                    # Separate route file
```

---

## 6. Variables & Secrets

### Overview

Workspace-level reusable variables and secrets.

### Database Schema

```sql
CREATE TABLE variables (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workspace_id        BIGINT UNSIGNED NOT NULL,
    created_by          BIGINT UNSIGNED NOT NULL,
    
    -- Variable Info
    key                 VARCHAR(100) NOT NULL,
    value               TEXT NOT NULL,               -- Encrypted if is_secret
    description         TEXT NULL,
    
    -- Type
    is_secret           BOOLEAN DEFAULT FALSE,
    
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (created_by) REFERENCES users(id),
    UNIQUE KEY unique_workspace_key (workspace_id, key)
);
```

### API Endpoints

| Method | Endpoint | Description | Permission |
|--------|----------|-------------|------------|
| GET | `/workspaces/{workspace}/variables` | List variables | credential.view |
| POST | `/workspaces/{workspace}/variables` | Create variable | credential.create |
| PUT | `/workspaces/{workspace}/variables/{variable}` | Update variable | credential.update |
| DELETE | `/workspaces/{workspace}/variables/{variable}` | Delete variable | credential.delete |

### Files to Create

```
app/
├── Http/
│   ├── Controllers/Api/V1/
│   │   └── VariableController.php
│   ├── Requests/Api/V1/Variable/
│   │   ├── StoreVariableRequest.php
│   │   └── UpdateVariableRequest.php
│   └── Resources/Api/V1/
│       └── VariableResource.php
├── Models/
│   └── Variable.php

database/
├── migrations/
│   └── 2026_01_30_000011_create_variables_table.php
```

---

## 7. Tags

### Overview

Organize workflows with tags.

### Database Schema

```sql
CREATE TABLE tags (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workspace_id        BIGINT UNSIGNED NOT NULL,
    
    name                VARCHAR(50) NOT NULL,
    color               VARCHAR(20) DEFAULT '#6366f1',
    
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    UNIQUE KEY unique_workspace_name (workspace_id, name)
);

CREATE TABLE workflow_tags (
    workflow_id         BIGINT UNSIGNED NOT NULL,
    tag_id              BIGINT UNSIGNED NOT NULL,
    
    PRIMARY KEY (workflow_id, tag_id),
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workspaces/{workspace}/tags` | List tags |
| POST | `/workspaces/{workspace}/tags` | Create tag |
| PUT | `/workspaces/{workspace}/tags/{tag}` | Update tag |
| DELETE | `/workspaces/{workspace}/tags/{tag}` | Delete tag |

### Files to Create

```
app/
├── Http/
│   ├── Controllers/Api/V1/
│   │   └── TagController.php
│   └── Resources/Api/V1/
│       └── TagResource.php
├── Models/
│   └── Tag.php

database/
├── migrations/
│   ├── 2026_01_30_000012_create_tags_table.php
│   └── 2026_01_30_000013_create_workflow_tags_table.php
```

---

## 8. Activity Logs

### Overview

Track all actions in a workspace for audit purposes.

### Database Schema

```sql
CREATE TABLE activity_logs (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    workspace_id        BIGINT UNSIGNED NOT NULL,
    user_id             BIGINT UNSIGNED NULL,
    
    -- Action Info
    action              VARCHAR(100) NOT NULL,        -- e.g., 'workflow.created'
    description         VARCHAR(500) NULL,
    
    -- Subject
    subject_type        VARCHAR(100) NULL,            -- e.g., 'App\Models\Workflow'
    subject_id          BIGINT UNSIGNED NULL,
    
    -- Changes
    old_values          JSON NULL,
    new_values          JSON NULL,
    
    -- Metadata
    ip_address          VARCHAR(45) NULL,
    user_agent          VARCHAR(500) NULL,
    
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_workspace_created (workspace_id, created_at),
    INDEX idx_subject (subject_type, subject_id)
);
```

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/workspaces/{workspace}/activity` | List activity logs |

### Files to Create

```
app/
├── Http/
│   ├── Controllers/Api/V1/
│   │   └── ActivityLogController.php
│   └── Resources/Api/V1/
│       └── ActivityLogResource.php
├── Models/
│   └── ActivityLog.php
├── Services/
│   └── ActivityLogService.php

database/
├── migrations/
│   └── 2026_01_30_000014_create_activity_logs_table.php
```

---

## 9. Queue Infrastructure

### Overview

Jobs that dispatch workflow execution to the Go engine.

### Database Schema

```sql
CREATE TABLE job_statuses (
    id                  BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    job_id              VARCHAR(100) NOT NULL UNIQUE,
    execution_id        BIGINT UNSIGNED NULL,
    
    status              ENUM('pending', 'processing', 'completed', 'failed') NOT NULL DEFAULT 'pending',
    progress            TINYINT UNSIGNED DEFAULT 0,   -- 0-100
    
    result              JSON NULL,
    error               JSON NULL,
    
    started_at          TIMESTAMP NULL,
    completed_at        TIMESTAMP NULL,
    
    created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (execution_id) REFERENCES executions(id) ON DELETE SET NULL,
    INDEX idx_status (status)
);
```

### Job Class

```php
<?php

namespace App\Jobs;

use App\Models\Workflow;
use App\Models\Execution;
use Illuminate\Bus\Queueable;
use Illuminate\Contracts\Queue\ShouldQueue;
use Illuminate\Foundation\Bus\Dispatchable;
use Illuminate\Queue\InteractsWithQueue;
use Illuminate\Queue\SerializesModels;
use Illuminate\Support\Str;

class ExecuteWorkflowJob implements ShouldQueue
{
    use Dispatchable, InteractsWithQueue, Queueable, SerializesModels;

    public string $jobId;
    
    public function __construct(
        public Workflow $workflow,
        public string $mode,
        public array $triggerData = [],
        public ?int $triggeredBy = null
    ) {
        $this->jobId = (string) Str::uuid();
        $this->onQueue('workflows');
    }

    public function handle(): void
    {
        // Create execution record
        $execution = Execution::create([
            'workflow_id' => $this->workflow->id,
            'workspace_id' => $this->workflow->workspace_id,
            'status' => 'pending',
            'mode' => $this->mode,
            'triggered_by' => $this->triggeredBy,
            'trigger_data' => $this->triggerData,
            'attempt' => 1,
            'max_attempts' => $this->workflow->settings['retry']['max_attempts'] ?? 1,
        ]);

        // Push to Redis for Go engine to consume
        // Format that Go engine expects
        $message = [
            'job_id' => $this->jobId,
            'execution_id' => $execution->id,
            'workflow_id' => $this->workflow->id,
            'workspace_id' => $this->workflow->workspace_id,
            'workflow' => [
                'nodes' => $this->workflow->nodes,
                'edges' => $this->workflow->edges,
                'settings' => $this->workflow->settings,
            ],
            'trigger_data' => $this->triggerData,
            'credentials' => $this->getDecryptedCredentials(),
            'variables' => $this->getVariables(),
            'callback_url' => route('api.v1.webhooks.job-callback'),
            'created_at' => now()->toIso8601String(),
        ];

        // Push to Redis list that Go engine consumes
        \Redis::rpush('linkflow:jobs:pending', json_encode($message));
    }

    protected function getDecryptedCredentials(): array
    {
        // Get all credentials used by this workflow
        // Decrypt and return them for Go engine
        return [];
    }

    protected function getVariables(): array
    {
        return $this->workflow->workspace->variables()
            ->get()
            ->mapWithKeys(fn ($v) => [$v->key => $v->value])
            ->all();
    }
}
```

### Callback Controller

```php
<?php

namespace App\Http\Controllers\Api\V1;

use App\Http\Controllers\Controller;
use App\Models\Execution;
use App\Models\ExecutionNode;
use Illuminate\Http\Request;

class JobCallbackController extends Controller
{
    public function handle(Request $request)
    {
        $validated = $request->validate([
            'job_id' => 'required|string',
            'execution_id' => 'required|integer',
            'status' => 'required|in:running,completed,failed',
            'node_results' => 'nullable|array',
            'result_data' => 'nullable|array',
            'error' => 'nullable|array',
            'duration_ms' => 'nullable|integer',
        ]);

        $execution = Execution::findOrFail($validated['execution_id']);

        // Update execution status
        $execution->update([
            'status' => $validated['status'],
            'result_data' => $validated['result_data'],
            'error' => $validated['error'],
            'duration_ms' => $validated['duration_ms'],
            'finished_at' => in_array($validated['status'], ['completed', 'failed']) 
                ? now() 
                : null,
        ]);

        // Save node results
        if (!empty($validated['node_results'])) {
            foreach ($validated['node_results'] as $index => $nodeResult) {
                ExecutionNode::create([
                    'execution_id' => $execution->id,
                    'node_id' => $nodeResult['node_id'],
                    'node_type' => $nodeResult['node_type'],
                    'node_name' => $nodeResult['node_name'],
                    'status' => $nodeResult['status'],
                    'started_at' => $nodeResult['started_at'],
                    'finished_at' => $nodeResult['finished_at'],
                    'duration_ms' => $nodeResult['duration_ms'],
                    'input_data' => $nodeResult['input_data'],
                    'output_data' => $nodeResult['output_data'],
                    'error' => $nodeResult['error'],
                    'sequence' => $index,
                ]);
            }
        }

        // Update workflow statistics
        $execution->workflow->incrementExecutionCount(
            $validated['status'] === 'completed'
        );

        // Broadcast real-time update (to Go WebSocket server)
        \Redis::publish('linkflow:events', json_encode([
            'event' => 'execution.' . $validated['status'],
            'channel' => 'workspace.' . $execution->workspace_id,
            'data' => [
                'execution_id' => $execution->id,
                'workflow_id' => $execution->workflow_id,
                'status' => $validated['status'],
            ],
        ]));

        return response()->json(['success' => true]);
    }
}
```

### Files to Create

```
app/
├── Http/
│   └── Controllers/Api/V1/
│       └── JobCallbackController.php
├── Jobs/
│   ├── ExecuteWorkflowJob.php
│   └── RetryWorkflowJob.php
├── Models/
│   └── JobStatus.php

database/
├── migrations/
│   └── 2026_01_30_000015_create_job_statuses_table.php

config/
└── engine.php                                      # Go engine configuration
```

---

## 10. File Structure

### Complete Files to Create

```
app/
├── Enums/
│   ├── TriggerType.php
│   ├── ExecutionStatus.php
│   ├── NodeKind.php
│   └── CredentialAuthType.php
│
├── Http/
│   ├── Controllers/
│   │   └── Api/
│   │       ├── WebhookReceiverController.php
│   │       └── V1/
│   │           ├── WorkflowController.php
│   │           ├── NodeController.php
│   │           ├── CredentialController.php
│   │           ├── CredentialTypeController.php
│   │           ├── ExecutionController.php
│   │           ├── VariableController.php
│   │           ├── TagController.php
│   │           ├── ActivityLogController.php
│   │           └── JobCallbackController.php
│   │
│   ├── Requests/Api/V1/
│   │   ├── Workflow/
│   │   │   ├── StoreWorkflowRequest.php
│   │   │   ├── UpdateWorkflowRequest.php
│   │   │   └── ExecuteWorkflowRequest.php
│   │   ├── Credential/
│   │   │   ├── StoreCredentialRequest.php
│   │   │   └── UpdateCredentialRequest.php
│   │   ├── Variable/
│   │   │   ├── StoreVariableRequest.php
│   │   │   └── UpdateVariableRequest.php
│   │   └── Tag/
│   │       ├── StoreTagRequest.php
│   │       └── UpdateTagRequest.php
│   │
│   └── Resources/Api/V1/
│       ├── WorkflowResource.php
│       ├── WorkflowCollection.php
│       ├── NodeResource.php
│       ├── NodeCategoryResource.php
│       ├── CredentialResource.php
│       ├── CredentialTypeResource.php
│       ├── ExecutionResource.php
│       ├── ExecutionNodeResource.php
│       ├── ExecutionLogResource.php
│       ├── VariableResource.php
│       ├── TagResource.php
│       ├── WebhookResource.php
│       └── ActivityLogResource.php
│
├── Models/
│   ├── Workflow.php
│   ├── Node.php
│   ├── NodeCategory.php
│   ├── Credential.php
│   ├── CredentialType.php
│   ├── Execution.php
│   ├── ExecutionNode.php
│   ├── ExecutionLog.php
│   ├── Webhook.php
│   ├── Variable.php
│   ├── Tag.php
│   ├── ActivityLog.php
│   └── JobStatus.php
│
├── Jobs/
│   ├── ExecuteWorkflowJob.php
│   └── RetryWorkflowJob.php
│
├── Services/
│   ├── CredentialEncryptionService.php
│   ├── WorkflowValidationService.php
│   ├── ActivityLogService.php
│   └── WebhookService.php
│
└── Policies/
    ├── WorkflowPolicy.php
    ├── CredentialPolicy.php
    └── ExecutionPolicy.php

database/
├── factories/
│   ├── WorkflowFactory.php
│   ├── CredentialFactory.php
│   ├── ExecutionFactory.php
│   └── TagFactory.php
│
├── migrations/
│   ├── 2026_01_30_000001_create_workflows_table.php
│   ├── 2026_01_30_000002_create_node_categories_table.php
│   ├── 2026_01_30_000003_create_nodes_table.php
│   ├── 2026_01_30_000004_create_credential_types_table.php
│   ├── 2026_01_30_000005_create_credentials_table.php
│   ├── 2026_01_30_000006_create_workflow_credentials_table.php
│   ├── 2026_01_30_000007_create_executions_table.php
│   ├── 2026_01_30_000008_create_execution_nodes_table.php
│   ├── 2026_01_30_000009_create_execution_logs_table.php
│   ├── 2026_01_30_000010_create_webhooks_table.php
│   ├── 2026_01_30_000011_create_variables_table.php
│   ├── 2026_01_30_000012_create_tags_table.php
│   ├── 2026_01_30_000013_create_workflow_tags_table.php
│   ├── 2026_01_30_000014_create_activity_logs_table.php
│   └── 2026_01_30_000015_create_job_statuses_table.php
│
└── seeders/
    ├── NodeCategorySeeder.php
    ├── NodeSeeder.php
    ├── CredentialTypeSeeder.php
    └── WorkflowSeeder.php

config/
└── engine.php

routes/
├── api.php                 # Updated with new routes
└── webhooks.php            # Public webhook routes
```

---

## 11. Database Schema

### Entity Relationship Diagram

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   workspaces    │────<│    workflows    │────<│   executions    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        │                       │                       │
        │                       │               ┌───────┴───────┐
        │                       │               │               │
        │               ┌───────┴───────┐       │               │
        │               │               │       ▼               ▼
        │               ▼               ▼  execution_nodes  execution_logs
        │          webhooks      workflow_tags
        │                               │
        │                               ▼
        │                             tags
        │
        ├────<│   credentials   │
        │
        ├────<│    variables    │
        │
        └────<│  activity_logs  │


┌─────────────────┐     ┌─────────────────┐
│ node_categories │────<│     nodes       │
└─────────────────┘     └─────────────────┘

┌─────────────────┐
│credential_types │
└─────────────────┘
```

---

## 12. API Endpoints Summary

### Total New Endpoints: 45+

| Resource | Endpoints | Priority |
|----------|-----------|----------|
| Workflows | 10 | 🔴 Critical |
| Nodes | 4 | 🔴 Critical |
| Credentials | 6 | 🔴 Critical |
| Credential Types | 2 | 🔴 Critical |
| Executions | 9 | 🔴 Critical |
| Webhooks (public) | 2 | 🔴 Critical |
| Variables | 4 | 🟡 Medium |
| Tags | 4 | 🟡 Medium |
| Activity Logs | 1 | 🟢 Low |
| Job Callback | 1 | 🔴 Critical |

### Routes File Update

```php
// routes/api.php - Add these routes

Route::middleware('auth:api')->group(function () {
    
    // ... existing routes ...

    Route::prefix('workspaces/{workspace}')->as('workspaces.')->group(function () {
        
        // Workflows
        Route::apiResource('workflows', WorkflowController::class);
        Route::post('workflows/{workflow}/execute', [WorkflowController::class, 'execute']);
        Route::post('workflows/{workflow}/activate', [WorkflowController::class, 'activate']);
        Route::post('workflows/{workflow}/deactivate', [WorkflowController::class, 'deactivate']);
        Route::post('workflows/{workflow}/duplicate', [WorkflowController::class, 'duplicate']);
        Route::get('workflows/{workflow}/versions', [WorkflowController::class, 'versions']);
        Route::get('workflows/{workflow}/executions', [WorkflowController::class, 'executions']);
        
        // Credentials
        Route::apiResource('credentials', CredentialController::class);
        Route::post('credentials/{credential}/test', [CredentialController::class, 'test']);
        
        // Executions
        Route::apiResource('executions', ExecutionController::class)->only(['index', 'show', 'destroy']);
        Route::get('executions/{execution}/nodes', [ExecutionController::class, 'nodes']);
        Route::get('executions/{execution}/logs', [ExecutionController::class, 'logs']);
        Route::post('executions/{execution}/retry', [ExecutionController::class, 'retry']);
        Route::post('executions/{execution}/cancel', [ExecutionController::class, 'cancel']);
        Route::get('executions/stats', [ExecutionController::class, 'stats']);
        
        // Variables
        Route::apiResource('variables', VariableController::class)->except(['show']);
        
        // Tags
        Route::apiResource('tags', TagController::class)->except(['show']);
        
        // Activity Logs
        Route::get('activity', [ActivityLogController::class, 'index']);
    });
    
    // Nodes (not workspace-scoped)
    Route::get('nodes', [NodeController::class, 'index']);
    Route::get('nodes/categories', [NodeController::class, 'categories']);
    Route::get('nodes/search', [NodeController::class, 'search']);
    Route::get('nodes/{type}', [NodeController::class, 'show']);
    
    // Credential Types (not workspace-scoped)
    Route::get('credential-types', [CredentialTypeController::class, 'index']);
    Route::get('credential-types/{type}', [CredentialTypeController::class, 'show']);
    
    // Job Callback (internal)
    Route::post('internal/job-callback', [JobCallbackController::class, 'handle'])
        ->name('webhooks.job-callback');
});

// routes/webhooks.php - Public webhook routes
Route::any('webhooks/{uuid}/{path?}', [WebhookReceiverController::class, 'handle'])
    ->where('path', '.*')
    ->name('webhooks.receive');
```

---

## Implementation Order

### Phase 2A (Week 1-2): Core Models
1. ✅ Workflows (Model, Migration, Controller, APIs)
2. ✅ Nodes & NodeCategories (Model, Migration, Seeder, Controller)
3. ✅ Credentials & CredentialTypes (Model, Migration, Seeder, Service, Controller)

### Phase 2B (Week 3-4): Execution System
4. ✅ Executions & ExecutionNodes (Model, Migration, Controller)
5. ✅ Webhooks (Model, Migration, Controller, Routes)
6. ✅ Queue Jobs (ExecuteWorkflowJob, JobCallback)

### Phase 2C (Week 5): Supporting Features
7. ✅ Variables (Model, Migration, Controller)
8. ✅ Tags (Model, Migration, Controller)
9. ✅ Activity Logs (Model, Migration, Service, Controller)

### Phase 2D (Week 6): Polish
10. ✅ Form Requests (all validation)
11. ✅ Resources (all API responses)
12. ✅ Tests (feature tests for all endpoints)
13. ✅ Documentation (OpenAPI/Swagger)

---

*This specification is the complete guide for implementing LinkFlow Phase 2.*
