# LinkFlow API - Architecture

## Folder Structure

```
app/
├── Http/
│   ├── Controllers/
│   │   └── Api/
│   │       └── V1/
│   │           ├── AuthController.php
│   │           ├── UserController.php
│   │           ├── WorkspaceController.php
│   │           ├── WorkspaceMemberController.php
│   │           ├── InvitationController.php
│   │           └── SubscriptionController.php
│   ├── Requests/
│   ├── Resources/
│   └── Middleware/
│
├── Models/
│   ├── User.php
│   ├── Workspace.php
│   ├── WorkspaceMember.php
│   ├── Invitation.php
│   ├── Subscription.php
│   └── Plan.php
│
├── Enums/
│   ├── WorkspaceRole.php
│   ├── InvitationStatus.php
│   └── SubscriptionStatus.php
│
├── Services/
│   ├── WorkspaceService.php
│   ├── InvitationService.php
│   └── SubscriptionService.php
│
├── Jobs/
├── Events/
├── Listeners/
├── Notifications/
├── Policies/
├── Observers/
├── Exceptions/
└── Providers/
```

## Database Relationships

### User
- Has many `WorkspaceMember` (user can be member of multiple workspaces)
- Has many `Workspace` through `WorkspaceMember`

### Workspace
- Has many `WorkspaceMember`
- Has many `User` through `WorkspaceMember`
- Has many `Invitation`
- Has one `Subscription`
- Belongs to `Plan` (through Subscription)

### WorkspaceMember (Pivot)
- Belongs to `User`
- Belongs to `Workspace`
- Has `role` attribute (using `WorkspaceRole` enum)

### Invitation
- Belongs to `Workspace`
- Belongs to `User` (inviter)
- Has `status` attribute (using `InvitationStatus` enum)

### Plan
- Has many `Subscription`
- Contains plan limits (members, features, etc.)

### Subscription
- Belongs to `Workspace`
- Belongs to `Plan`
- Has `status` attribute (using `SubscriptionStatus` enum)

## Enums

### WorkspaceRole
- `Owner` - Full access, can delete workspace
- `Admin` - Manage members, settings
- `Member` - Standard access

### InvitationStatus
- `Pending` - Awaiting response
- `Accepted` - User joined
- `Declined` - User declined
- `Expired` - Invitation expired

### SubscriptionStatus
- `Active` - Currently active
- `Trialing` - In trial period
- `PastDue` - Payment failed
- `Canceled` - Subscription canceled
- `Expired` - Subscription expired

## API Endpoints (V1)

### Authentication
- `POST /api/v1/auth/register`
- `POST /api/v1/auth/login`
- `POST /api/v1/auth/logout`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/forgot-password`
- `POST /api/v1/auth/reset-password`

### User
- `GET /api/v1/user` - Current user profile
- `PUT /api/v1/user` - Update profile
- `PUT /api/v1/user/password` - Change password
- `DELETE /api/v1/user` - Delete account

### Workspaces
- `GET /api/v1/workspaces` - List user's workspaces
- `POST /api/v1/workspaces` - Create workspace
- `GET /api/v1/workspaces/{workspace}` - Get workspace
- `PUT /api/v1/workspaces/{workspace}` - Update workspace
- `DELETE /api/v1/workspaces/{workspace}` - Delete workspace

### Workspace Members
- `GET /api/v1/workspaces/{workspace}/members` - List members
- `POST /api/v1/workspaces/{workspace}/members` - Add member
- `PUT /api/v1/workspaces/{workspace}/members/{member}` - Update role
- `DELETE /api/v1/workspaces/{workspace}/members/{member}` - Remove member

### Invitations
- `GET /api/v1/workspaces/{workspace}/invitations` - List invitations
- `POST /api/v1/workspaces/{workspace}/invitations` - Send invitation
- `DELETE /api/v1/workspaces/{workspace}/invitations/{invitation}` - Cancel invitation
- `POST /api/v1/invitations/{token}/accept` - Accept invitation
- `POST /api/v1/invitations/{token}/decline` - Decline invitation

### Subscriptions
- `GET /api/v1/workspaces/{workspace}/subscription` - Get subscription
- `POST /api/v1/workspaces/{workspace}/subscription` - Create subscription
- `PUT /api/v1/workspaces/{workspace}/subscription` - Update subscription
- `DELETE /api/v1/workspaces/{workspace}/subscription` - Cancel subscription
- `GET /api/v1/plans` - List available plans

## Services

### WorkspaceService
- Create workspace with owner
- Switch current workspace
- Transfer ownership

### InvitationService
- Send invitation email
- Accept/decline invitation
- Expire old invitations

### SubscriptionService
- Handle plan changes
- Check feature access
- Manage billing (integrate with Stripe/Paddle)
