# LinkFlow API - Complete System Overview

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              LINKFLOW API                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐  │
│  │   Frontend  │───▶│   Laravel   │───▶│   MySQL     │    │    Redis    │  │
│  │   (Client)  │◀───│     API     │◀───│  Database   │    │   (Queue)   │  │
│  └─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘  │
│         │                  │                                                 │
│         │           ┌──────┴──────┐                                         │
│         │           │             │                                         │
│         │    ┌──────▼─────┐ ┌─────▼──────┐                                  │
│         │    │  Passport  │ │   Spatie   │                                  │
│         │    │   (Auth)   │ │(Permissions)│                                  │
│         │    └────────────┘ └────────────┘                                  │
│         │                                                                    │
│  ┌──────▼──────┐                                                            │
│  │ Admin Panel │ (Session Auth - Separate Guard)                            │
│  └─────────────┘                                                            │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
app/
├── Enums/
│   └── SubscriptionStatus.php         # Subscription statuses
├── Http/
│   ├── Controllers/
│   │   ├── Admin/
│   │   │   └── AuthController.php     # Admin login/logout (session)
│   │   └── Api/V1/
│   │       ├── AuthController.php     # User auth (Passport)
│   │       ├── UserController.php     # Profile management
│   │       ├── WorkspaceController.php
│   │       ├── WorkspaceMemberController.php
│   │       ├── InvitationController.php
│   │       ├── PlanController.php
│   │       └── SubscriptionController.php
│   ├── Middleware/
│   │   └── CheckWorkspacePermission.php
│   ├── Requests/
│   │   ├── Admin/Auth/
│   │   └── Api/V1/
│   │       ├── Auth/
│   │       ├── User/
│   │       ├── Workspace/
│   │       └── Subscription/
│   └── Resources/
│       ├── Admin/
│       └── Api/V1/
├── Models/
│   ├── Admin.php                      # Super admin (separate table)
│   ├── User.php                       # Regular users
│   ├── Workspace.php
│   ├── Invitation.php
│   ├── Plan.php
│   └── Subscription.php
├── Notifications/
│   ├── ResetPasswordNotification.php
│   ├── VerifyEmailNotification.php
│   └── WorkspaceInvitationNotification.php
├── Providers/
└── Services/
    └── WorkspacePermissionService.php  # Workspace-scoped permissions

routes/
├── api.php                            # API routes (Passport auth)
├── admin.php                          # Admin routes (Session auth)
└── web.php
```

---

## Authentication System

### Two Separate Auth Systems

```
┌────────────────────────────────────────────────────────────────┐
│                     AUTHENTICATION                              │
├────────────────────────────────────────────────────────────────┤
│                                                                 │
│  API Users (Passport - Token Based)                            │
│  ─────────────────────────────────                             │
│  Table: users                                                   │
│  Guard: api                                                     │
│  Driver: passport                                               │
│  Routes: /api/v1/*                                              │
│  Auth Header: Authorization: Bearer {token}                     │
│                                                                 │
│  ─────────────────────────────────────────────────────────────│
│                                                                 │
│  Super Admin (Session Based)                                    │
│  ───────────────────────────                                   │
│  Table: admins                                                  │
│  Guard: admin                                                   │
│  Driver: session                                                │
│  Routes: /admin/*                                               │
│  Auth: Cookie/Session                                           │
│                                                                 │
└────────────────────────────────────────────────────────────────┘
```

### Auth Flow

```
┌─────────┐     POST /api/v1/auth/register     ┌─────────┐
│ Client  │ ──────────────────────────────────▶│  API    │
│         │     { email, password, ... }       │         │
└─────────┘                                    └────┬────┘
                                                    │
     ┌──────────────────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────────────────────────────┐
│ 1. Create User                                               │
│ 2. Create Default Workspace (owner)                          │
│ 3. Create Free Subscription                                  │
│ 4. Send Verification Email                                   │
│ 5. Generate Passport Token                                   │
│ 6. Return { user, access_token }                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Multi-Tenancy (Workspaces)

### Data Model

```
┌─────────────┐       ┌──────────────────┐       ┌─────────────┐
│    User     │       │ workspace_members│       │  Workspace  │
├─────────────┤       ├──────────────────┤       ├─────────────┤
│ id          │──┐    │ id               │    ┌──│ id          │
│ first_name  │  │    │ workspace_id ────│────┘  │ name        │
│ last_name   │  └────│ user_id          │       │ slug        │
│ email       │       │ role             │       │ owner_id ───│──┐
│ avatar      │       │ joined_at        │       │ settings    │  │
│ password    │       └──────────────────┘       └─────────────┘  │
└─────────────┘                                         │         │
       ▲                                                │         │
       └────────────────────────────────────────────────┴─────────┘
```

### Workspace Hierarchy

```
User
├── Can own multiple workspaces (owner_id)
├── Can be member of multiple workspaces (workspace_members)
└── Each membership has a role: owner | admin | member | viewer

Workspace
├── Has one owner (owner_id → User)
├── Has many members (workspace_members pivot)
├── Has many invitations
└── Has one subscription → Plan
```

---

## Permission System

### How It Works

```
┌─────────────────────────────────────────────────────────────────┐
│                    PERMISSION CHECK FLOW                         │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Request: PUT /api/v1/workspaces/{workspace}/members/{user}     │
│                           │                                      │
│                           ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ WorkspaceMemberController::update()                      │    │
│  │                                                          │    │
│  │ $this->permissionService->authorize(                     │    │
│  │     $request->user(),    // Current user                 │    │
│  │     $workspace,          // Target workspace             │    │
│  │     'member.update'      // Required permission          │    │
│  │ );                                                       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                           │                                      │
│                           ▼                                      │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │ WorkspacePermissionService::authorize()                  │    │
│  │                                                          │    │
│  │ 1. Get user's role in workspace (from pivot table)       │    │
│  │ 2. Look up permissions for that role                     │    │
│  │ 3. Check if 'member.update' is in permissions            │    │
│  │ 4. If not → abort(403)                                   │    │
│  │ 5. If yes → continue                                     │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Role → Permission Matrix

| Permission | Owner | Admin | Member | Viewer |
|------------|:-----:|:-----:|:------:|:------:|
| workspace.view | ✅ | ✅ | ✅ | ✅ |
| workspace.update | ✅ | ✅ | ❌ | ❌ |
| workspace.delete | ✅ | ❌ | ❌ | ❌ |
| workspace.manage-billing | ✅ | ❌ | ❌ | ❌ |
| member.view | ✅ | ✅ | ✅ | ✅ |
| member.invite | ✅ | ✅ | ❌ | ❌ |
| member.update | ✅ | ✅ | ❌ | ❌ |
| member.remove | ✅ | ✅ | ❌ | ❌ |
| workflow.view | ✅ | ✅ | ✅ | ✅ |
| workflow.create | ✅ | ✅ | ✅ | ❌ |
| workflow.update | ✅ | ✅ | ✅ | ❌ |
| workflow.delete | ✅ | ✅ | ✅ | ❌ |
| workflow.execute | ✅ | ✅ | ✅ | ❌ |
| credential.view | ✅ | ✅ | ✅ | ✅ |
| credential.create | ✅ | ✅ | ✅ | ❌ |
| credential.update | ✅ | ✅ | ✅ | ❌ |
| credential.delete | ✅ | ✅ | ✅ | ❌ |
| execution.view | ✅ | ✅ | ✅ | ✅ |
| execution.delete | ✅ | ✅ | ❌ | ❌ |

---

## Invitation Flow

```
┌─────────┐                                              ┌─────────┐
│  Admin  │                                              │  Invitee│
└────┬────┘                                              └────┬────┘
     │                                                        │
     │  POST /workspaces/{id}/invitations                     │
     │  { email: "new@user.com", role: "member" }             │
     │──────────────────────────────────────┐                 │
     │                                      │                 │
     │                              ┌───────▼───────┐         │
     │                              │ Create        │         │
     │                              │ Invitation    │         │
     │                              │ (token, 7d)   │         │
     │                              └───────┬───────┘         │
     │                                      │                 │
     │                              ┌───────▼───────┐         │
     │                              │ Send Email    │─────────▶ Email with link
     │                              │ Notification  │         │
     │                              └───────────────┘         │
     │                                                        │
     │                                                        │
     │                                        Click link      │
     │                                        ┌───────────────┤
     │                                        │               │
     │                              ┌─────────▼─────────┐     │
     │                              │ POST /invitations │     │
     │                              │ /{token}/accept   │     │
     │                              └─────────┬─────────┘     │
     │                                        │               │
     │                              ┌─────────▼─────────┐     │
     │                              │ Add to workspace  │     │
     │                              │ with assigned role│     │
     │                              └───────────────────┘     │
     │                                                        │
```

---

## Billing System

### Data Model

```
┌─────────────┐       ┌──────────────────┐       ┌─────────────┐
│    Plan     │       │   Subscription   │       │  Workspace  │
├─────────────┤       ├──────────────────┤       ├─────────────┤
│ id          │◀──────│ plan_id          │       │ id          │
│ name        │       │ workspace_id ────│──────▶│ name        │
│ slug        │       │ status           │       │ ...         │
│ price_monthly│      │ trial_ends_at    │       └─────────────┘
│ price_yearly│       │ current_period_* │
│ limits      │       │ canceled_at      │
│ features    │       └──────────────────┘
└─────────────┘
```

### Default Plans

| Plan | Monthly | Yearly | Workflows | Executions | Members | Webhooks |
|------|---------|--------|-----------|------------|---------|----------|
| Free | $0 | $0 | 5 | 500 | 1 | ❌ |
| Pro | $19 | $190 | 50 | 10,000 | 5 | ✅ |
| Business | $49 | $490 | Unlimited | 100,000 | 20 | ✅ |

### Subscription Flow

```
User Registers
     │
     ▼
┌─────────────────────────┐
│ Create User             │
│ Create Workspace        │
│ Create Free Subscription│◀── Auto-assigned
└─────────────────────────┘
     │
     ▼
User Upgrades (POST /workspaces/{id}/subscription)
     │
     ▼
┌─────────────────────────┐
│ Update Subscription     │
│ - plan_id = new plan    │
│ - status = active       │
│ - period_start = now    │
│ - period_end = +1 month │
└─────────────────────────┘
```

---

## API Routes Summary

### Public Routes (No Auth)

```
GET  /api/v1/plans                              # List available plans
POST /api/v1/auth/register                      # Register new user
POST /api/v1/auth/login                         # Login
POST /api/v1/auth/forgot-password               # Request password reset
POST /api/v1/auth/reset-password                # Reset password
GET  /api/v1/verify-email/{id}/{hash}           # Verify email (signed)
POST /api/v1/invitations/{token}/accept         # Accept invitation
POST /api/v1/invitations/{token}/decline        # Decline invitation
```

### Protected Routes (Require Auth)

```
# Auth
POST /api/v1/auth/logout                        # Logout
POST /api/v1/auth/resend-verification-email     # Resend verification

# User Profile
GET    /api/v1/user                             # Get profile
PUT    /api/v1/user                             # Update profile
PUT    /api/v1/user/password                    # Change password
POST   /api/v1/user/avatar                      # Upload avatar
DELETE /api/v1/user/avatar                      # Delete avatar
DELETE /api/v1/user                             # Delete account

# Workspaces
GET    /api/v1/workspaces                       # List workspaces
POST   /api/v1/workspaces                       # Create workspace
GET    /api/v1/workspaces/{workspace}           # Get workspace
PUT    /api/v1/workspaces/{workspace}           # Update workspace
DELETE /api/v1/workspaces/{workspace}           # Delete workspace

# Workspace Members
GET    /api/v1/workspaces/{workspace}/members           # List members
PUT    /api/v1/workspaces/{workspace}/members/{user}    # Update role
DELETE /api/v1/workspaces/{workspace}/members/{user}    # Remove member
POST   /api/v1/workspaces/{workspace}/leave             # Leave workspace

# Invitations
GET    /api/v1/workspaces/{workspace}/invitations               # List
POST   /api/v1/workspaces/{workspace}/invitations               # Send
DELETE /api/v1/workspaces/{workspace}/invitations/{invitation}  # Cancel

# Subscriptions
GET    /api/v1/workspaces/{workspace}/subscription      # Get subscription
POST   /api/v1/workspaces/{workspace}/subscription      # Create/Update
DELETE /api/v1/workspaces/{workspace}/subscription      # Cancel
```

### Admin Routes (Session Auth)

```
POST /admin/login                               # Admin login
POST /admin/logout                              # Admin logout
GET  /admin/me                                  # Get current admin
```

---

## Request/Response Flow

```
┌─────────┐      ┌───────────┐      ┌────────────┐      ┌────────────┐
│ Request │─────▶│ Middleware│─────▶│ Controller │─────▶│  Response  │
└─────────┘      └───────────┘      └────────────┘      └────────────┘
                       │                   │
                       │                   │
              ┌────────▼────────┐   ┌──────▼──────┐
              │ auth:api        │   │ FormRequest │
              │ (Passport)      │   │ (Validation)│
              └─────────────────┘   └─────────────┘
                                          │
                                   ┌──────▼──────┐
                                   │ Permission  │
                                   │ Service     │
                                   └─────────────┘
                                          │
                                   ┌──────▼──────┐
                                   │ Model /     │
                                   │ Database    │
                                   └─────────────┘
                                          │
                                   ┌──────▼──────┐
                                   │ Resource    │
                                   │ (Transform) │
                                   └─────────────┘
```

---

## Database Tables

```sql
-- Core Tables
users                    # Regular users
admins                   # Super admins (platform level)
workspaces               # Workspaces (tenants)
workspace_members        # Pivot: user ↔ workspace with role

-- Billing
plans                    # Subscription plans
subscriptions            # Workspace subscriptions

-- Invitations
invitations              # Workspace invitations

-- Auth
password_reset_tokens    # Password reset
oauth_*                  # Passport tables (tokens, clients, etc.)

-- Permissions (Spatie - Global roles, not used in workspace context)
permissions
roles
model_has_permissions
model_has_roles
role_has_permissions
```

---

## Environment Variables

```env
# App
APP_NAME=LinkFlow
APP_URL=https://linkflow-api.test
FRONTEND_URL=https://app.linkflow.com

# Database
DB_CONNECTION=mysql
DB_HOST=127.0.0.1
DB_DATABASE=linkflow
DB_USERNAME=root
DB_PASSWORD=

# Mail (for notifications)
MAIL_MAILER=smtp
MAIL_HOST=mailhog
MAIL_PORT=1025

# Queue
QUEUE_CONNECTION=redis

# Passport
PASSPORT_PRIVATE_KEY=
PASSPORT_PUBLIC_KEY=
```

---

## Key Files Reference

| File | Purpose |
|------|---------|
| `config/auth.php` | Guards: api (passport), admin (session) |
| `bootstrap/app.php` | Middleware aliases, route registration |
| `routes/api.php` | All API routes (v1) |
| `routes/admin.php` | Admin panel routes |
| `app/Services/WorkspacePermissionService.php` | Workspace-scoped permission checks |
| `app/Http/Middleware/CheckWorkspacePermission.php` | Permission middleware |
| `database/seeders/RolesAndPermissionsSeeder.php` | Spatie permissions (global) |
| `database/seeders/PlanSeeder.php` | Default plans |
| `database/seeders/AdminSeeder.php` | Default super admin |

---

## Default Credentials

**Super Admin:**
- Email: `admin@linkflow.com`
- Password: `password`

---

## Next Phase (Workflows)

Phase 2 will add:
- Workflows (visual workflow builder data)
- Nodes (trigger, action definitions)
- Credentials (encrypted API keys storage)
- Executions (workflow run history)
- Webhooks (trigger endpoints)

The Go execution engine will communicate with this API to:
1. Fetch workflow definitions
2. Fetch credentials
3. Store execution results
4. Trigger webhook-based workflows
