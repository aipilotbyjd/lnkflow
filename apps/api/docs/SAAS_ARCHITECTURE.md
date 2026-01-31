# LinkFlow SaaS Architecture

## Core Modules Overview

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           LINKFLOW SAAS                                  │
├─────────────────────────────────────────────────────────────────────────┤
│  AUTH & USERS  │  WORKSPACES  │  BILLING  │  WORKFLOWS  │  EXECUTION   │
│                │              │           │             │   (Go Engine) │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## Phase 1: Core SaaS Foundation (Build First)

### 1. Authentication & Users
| Feature | Description |
|---------|-------------|
| Register/Login | Email + password with Passport |
| Email Verification | Verify email before access |
| Password Reset | Forgot password flow |
| Two-Factor Auth (2FA) | Optional TOTP-based 2FA |
| Social Login | Google, GitHub OAuth (optional) |
| Profile Management | Update name, email, avatar |

### 2. Workspaces (Multi-Tenancy)
| Feature | Description |
|---------|-------------|
| Create Workspace | User creates workspace on signup |
| Multiple Workspaces | User can own/join multiple workspaces |
| Workspace Settings | Name, slug, logo, timezone |
| Switch Workspace | Toggle between workspaces |
| Delete Workspace | Owner can delete (with confirmation) |

### 3. Workspace Members & Roles
| Role | Permissions |
|------|-------------|
| Owner | Full access, billing, delete workspace, transfer ownership |
| Admin | Manage members, settings, all workflows |
| Member | Create/edit own workflows, view shared workflows |
| Viewer | View-only access to workflows |

| Feature | Description |
|---------|-------------|
| Invite Members | Send email invitation with role |
| Accept/Decline Invite | Join workspace via invite link |
| Remove Members | Admin+ can remove members |
| Change Roles | Owner/Admin can change member roles |
| Leave Workspace | Member can leave (except owner) |

### 4. Billing & Subscriptions
| Feature | Description |
|---------|-------------|
| Plans | Free, Pro, Business, Enterprise |
| Subscription | Stripe/Paddle integration |
| Usage Limits | Workflows, executions, team members per plan |
| Billing Portal | Manage payment methods, invoices |
| Plan Upgrades/Downgrades | Change plans anytime |
| Trial Period | 14-day trial for paid plans |

---

## Phase 2: Workflow Builder (After SaaS Foundation)

### 5. Workflows
| Feature | Description |
|---------|-------------|
| Create Workflow | Name, description, folder |
| Workflow Editor | Visual node-based editor (frontend) |
| Workflow Versions | Save versions, rollback |
| Duplicate Workflow | Clone existing workflow |
| Import/Export | JSON export/import |
| Workflow Tags | Organize with tags |
| Workflow Folders | Organize in folders |

### 6. Nodes
| Category | Examples |
|----------|----------|
| Triggers | Webhook, Schedule (Cron), Manual, App Events |
| Actions | HTTP Request, Send Email, Database Query |
| Logic | IF/Else, Switch, Merge, Split |
| Data | Set, Transform, Filter, Aggregate |
| Integrations | Slack, Discord, Google Sheets, Notion, etc. |

### 7. Credentials
| Feature | Description |
|---------|-------------|
| Store Credentials | Encrypted storage for API keys, OAuth tokens |
| Credential Types | API Key, OAuth2, Basic Auth, Custom |
| Shared Credentials | Share across workflows in workspace |
| Test Credentials | Verify credentials work |

### 8. Executions
| Feature | Description |
|---------|-------------|
| Run Workflow | Manual or triggered execution |
| Execution History | List all past executions |
| Execution Details | Input/output of each node |
| Execution Status | Success, Failed, Running, Cancelled |
| Retry Failed | Re-run failed executions |
| Execution Logs | Detailed logs per execution |

---

## Phase 3: Advanced Features

### 9. Webhooks
| Feature | Description |
|---------|-------------|
| Webhook URLs | Unique URL per workflow trigger |
| Webhook Auth | Optional authentication |
| Webhook Logs | Log incoming webhook requests |

### 10. Templates
| Feature | Description |
|---------|-------------|
| Workflow Templates | Pre-built workflow templates |
| Template Categories | By use case (Marketing, Dev, etc.) |
| Publish Template | Users can share templates |

### 11. Activity & Audit Logs
| Feature | Description |
|---------|-------------|
| Activity Log | Track all actions in workspace |
| Audit Trail | Who did what, when |

### 12. API Access
| Feature | Description |
|---------|-------------|
| API Keys | Workspace-level API keys |
| API Rate Limits | Based on plan |
| API Documentation | Swagger/OpenAPI docs |

---

## Database Schema

### Core Tables

```
users
├── id
├── first_name
├── last_name
├── email
├── password
├── email_verified_at
├── two_factor_enabled
├── avatar
└── timestamps

workspaces
├── id
├── name
├── slug (unique)
├── logo
├── settings (json)
├── owner_id (user)
├── plan_id
└── timestamps

workspace_members
├── id
├── workspace_id
├── user_id
├── role (enum: owner, admin, member, viewer)
├── joined_at
└── timestamps

invitations
├── id
├── workspace_id
├── email
├── role
├── token (unique)
├── invited_by (user_id)
├── accepted_at
├── expires_at
└── timestamps

plans
├── id
├── name
├── slug
├── price_monthly
├── price_yearly
├── limits (json: workflows, executions, members)
├── features (json)
└── timestamps

subscriptions
├── id
├── workspace_id
├── plan_id
├── stripe_subscription_id
├── status (enum)
├── trial_ends_at
├── current_period_start
├── current_period_end
└── timestamps
```

### Workflow Tables (Phase 2)

```
workflows
├── id
├── workspace_id
├── name
├── description
├── nodes (json)
├── edges (json)
├── settings (json)
├── is_active
├── created_by (user_id)
└── timestamps

credentials
├── id
├── workspace_id
├── name
├── type
├── data (encrypted json)
├── created_by (user_id)
└── timestamps

executions
├── id
├── workflow_id
├── status (enum)
├── started_at
├── finished_at
├── error_message
├── data (json)
└── timestamps

webhooks
├── id
├── workflow_id
├── path (unique)
├── method
├── is_active
└── timestamps
```

---

## Plan Limits Example

| Feature | Free | Pro ($19/mo) | Business ($49/mo) | Enterprise |
|---------|------|--------------|-------------------|------------|
| Workflows | 5 | 50 | Unlimited | Unlimited |
| Executions/mo | 500 | 10,000 | 100,000 | Unlimited |
| Team Members | 1 | 5 | 20 | Unlimited |
| Execution History | 7 days | 30 days | 90 days | 1 year |
| Credentials | 10 | 50 | Unlimited | Unlimited |
| Webhook Triggers | ❌ | ✅ | ✅ | ✅ |
| Priority Support | ❌ | ❌ | ✅ | ✅ |
| SSO/SAML | ❌ | ❌ | ❌ | ✅ |

---

## API Endpoints Structure

### Auth
- `POST /v1/register`
- `POST /v1/login`
- `POST /v1/logout`
- `POST /v1/forgot-password`
- `POST /v1/reset-password`
- `POST /v1/verify-email`

### User
- `GET /v1/user`
- `PUT /v1/user`
- `PUT /v1/user/password`
- `DELETE /v1/user`

### Workspaces
- `GET /v1/workspaces`
- `POST /v1/workspaces`
- `GET /v1/workspaces/{workspace}`
- `PUT /v1/workspaces/{workspace}`
- `DELETE /v1/workspaces/{workspace}`

### Workspace Members
- `GET /v1/workspaces/{workspace}/members`
- `PUT /v1/workspaces/{workspace}/members/{member}`
- `DELETE /v1/workspaces/{workspace}/members/{member}`

### Invitations
- `GET /v1/workspaces/{workspace}/invitations`
- `POST /v1/workspaces/{workspace}/invitations`
- `DELETE /v1/workspaces/{workspace}/invitations/{invitation}`
- `POST /v1/invitations/{token}/accept`
- `POST /v1/invitations/{token}/decline`

### Subscriptions
- `GET /v1/plans`
- `GET /v1/workspaces/{workspace}/subscription`
- `POST /v1/workspaces/{workspace}/subscription`
- `PUT /v1/workspaces/{workspace}/subscription`
- `DELETE /v1/workspaces/{workspace}/subscription`
- `GET /v1/workspaces/{workspace}/billing-portal`

### Workflows (Phase 2)
- `GET /v1/workspaces/{workspace}/workflows`
- `POST /v1/workspaces/{workspace}/workflows`
- `GET /v1/workspaces/{workspace}/workflows/{workflow}`
- `PUT /v1/workspaces/{workspace}/workflows/{workflow}`
- `DELETE /v1/workspaces/{workspace}/workflows/{workflow}`
- `POST /v1/workspaces/{workspace}/workflows/{workflow}/execute`
- `POST /v1/workspaces/{workspace}/workflows/{workflow}/activate`
- `POST /v1/workspaces/{workspace}/workflows/{workflow}/deactivate`

### Executions (Phase 2)
- `GET /v1/workspaces/{workspace}/executions`
- `GET /v1/workspaces/{workspace}/executions/{execution}`
- `POST /v1/workspaces/{workspace}/executions/{execution}/retry`
- `DELETE /v1/workspaces/{workspace}/executions/{execution}`

### Credentials (Phase 2)
- `GET /v1/workspaces/{workspace}/credentials`
- `POST /v1/workspaces/{workspace}/credentials`
- `PUT /v1/workspaces/{workspace}/credentials/{credential}`
- `DELETE /v1/workspaces/{workspace}/credentials/{credential}`
- `POST /v1/workspaces/{workspace}/credentials/{credential}/test`

### Webhooks (Phase 2)
- `POST /v1/webhooks/{path}` (Public endpoint for triggers)

---

## Recommended Build Order

### Week 1-2: Auth & Users
- [x] Register, Login, Logout
- [ ] Email Verification
- [ ] Password Reset
- [ ] Profile with Avatar

### Week 3-4: Workspaces & Members
- [ ] Create/List/Update/Delete Workspaces
- [ ] Workspace Members CRUD
- [ ] Roles & Permissions
- [ ] Invitations Flow

### Week 5-6: Billing
- [ ] Plans CRUD
- [ ] Stripe Integration
- [ ] Subscription Management
- [ ] Usage Tracking

### Week 7+: Workflows
- [ ] Start workflow builder (coordinate with Go engine)
- [ ] Credentials management
- [ ] Execution history

---

## Tech Stack Summary

| Layer | Technology |
|-------|------------|
| API | Laravel 12 (this project) |
| Execution Engine | Go (separate service) |
| Database | MySQL/PostgreSQL |
| Cache | Redis |
| Queue | Redis + Laravel Horizon |
| Storage | S3/MinIO |
| Payments | Stripe or Paddle |
| Email | Postmark, Resend, or SES |
