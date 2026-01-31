# LinkFlow API Documentation

**Base URL:** `https://linkflow-api.test/api/v1`

**Authentication:** Bearer Token (Passport)

---

## Authentication

### Register

Create a new user account with default workspace.

**Endpoint:** `POST /register`

**Request Body:**
```json
{
    "first_name": "John",
    "last_name": "Doe",
    "email": "john@example.com",
    "password": "password123",
    "password_confirmation": "password123"
}
```

**Success Response (201):**
```json
{
    "message": "User registered successfully.",
    "user": {
        "id": 1,
        "first_name": "John",
        "last_name": "Doe",
        "email": "john@example.com",
        "email_verified_at": null,
        "created_at": "2026-01-30T10:00:00.000000Z",
        "updated_at": "2026-01-30T10:00:00.000000Z"
    },
    "access_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9...",
    "token_type": "Bearer"
}
```

---

### Login

**Endpoint:** `POST /login`

**Request Body:**
```json
{
    "email": "john@example.com",
    "password": "password123"
}
```

**Success Response (200):**
```json
{
    "message": "Login successful.",
    "user": { ... },
    "access_token": "eyJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9...",
    "token_type": "Bearer"
}
```

---

### Logout

**Endpoint:** `POST /logout`  
**Auth:** Required

**Success Response (200):**
```json
{
    "message": "Logged out successfully."
}
```

---

### Forgot Password

**Endpoint:** `POST /forgot-password`

**Request Body:**
```json
{
    "email": "john@example.com"
}
```

**Success Response (200):**
```json
{
    "message": "Password reset link sent to your email."
}
```

---

### Reset Password

**Endpoint:** `POST /reset-password`

**Request Body:**
```json
{
    "token": "reset-token-from-email",
    "email": "john@example.com",
    "password": "newpassword123",
    "password_confirmation": "newpassword123"
}
```

**Success Response (200):**
```json
{
    "message": "Password reset successfully."
}
```

---

### Verify Email

**Endpoint:** `GET /verify-email/{id}/{hash}` (Signed URL from email)

**Success Response (200):**
```json
{
    "message": "Email verified successfully."
}
```

---

### Resend Verification Email

**Endpoint:** `POST /resend-verification-email`  
**Auth:** Required

**Success Response (200):**
```json
{
    "message": "Verification email sent."
}
```

---

## User

### Get Profile

**Endpoint:** `GET /user`  
**Auth:** Required

**Success Response (200):**
```json
{
    "user": {
        "id": 1,
        "first_name": "John",
        "last_name": "Doe",
        "email": "john@example.com",
        "email_verified_at": "2026-01-30T10:00:00.000000Z",
        "created_at": "2026-01-30T10:00:00.000000Z",
        "updated_at": "2026-01-30T10:00:00.000000Z"
    }
}
```

---

### Update Profile

**Endpoint:** `PUT /user`  
**Auth:** Required

**Request Body:**
```json
{
    "first_name": "John",
    "last_name": "Updated",
    "email": "john.updated@example.com"
}
```

---

### Change Password

**Endpoint:** `PUT /user/password`  
**Auth:** Required

**Request Body:**
```json
{
    "current_password": "password123",
    "password": "newpassword123",
    "password_confirmation": "newpassword123"
}
```

---

### Delete Account

**Endpoint:** `DELETE /user`  
**Auth:** Required

---

## Workspaces

### List Workspaces

**Endpoint:** `GET /workspaces`  
**Auth:** Required

**Success Response (200):**
```json
{
    "data": [
        {
            "id": 1,
            "name": "My Workspace",
            "slug": "my-workspace",
            "logo": null,
            "settings": null,
            "owner": { ... },
            "created_at": "2026-01-30T10:00:00.000000Z",
            "updated_at": "2026-01-30T10:00:00.000000Z"
        }
    ]
}
```

---

### Create Workspace

**Endpoint:** `POST /workspaces`  
**Auth:** Required

**Request Body:**
```json
{
    "name": "New Workspace"
}
```

---

### Get Workspace

**Endpoint:** `GET /workspaces/{workspace}`  
**Auth:** Required

---

### Update Workspace

**Endpoint:** `PUT /workspaces/{workspace}`  
**Auth:** Required (Owner/Admin)

**Request Body:**
```json
{
    "name": "Updated Name",
    "slug": "updated-slug"
}
```

---

### Delete Workspace

**Endpoint:** `DELETE /workspaces/{workspace}`  
**Auth:** Required (Owner only)

---

## Workspace Members

### List Members

**Endpoint:** `GET /workspaces/{workspace}/members`  
**Auth:** Required

**Success Response (200):**
```json
{
    "data": [
        {
            "id": 1,
            "first_name": "John",
            "last_name": "Doe",
            "email": "john@example.com",
            "role": "owner",
            "joined_at": "2026-01-30T10:00:00.000000Z"
        }
    ]
}
```

---

### Update Member Role

**Endpoint:** `PUT /workspaces/{workspace}/members/{user}`  
**Auth:** Required (Owner/Admin)

**Request Body:**
```json
{
    "role": "admin"
}
```

**Available Roles:** `admin`, `member`, `viewer`

---

### Remove Member

**Endpoint:** `DELETE /workspaces/{workspace}/members/{user}`  
**Auth:** Required (Owner/Admin)

---

## Invitations

### List Pending Invitations

**Endpoint:** `GET /workspaces/{workspace}/invitations`  
**Auth:** Required

---

### Send Invitation

**Endpoint:** `POST /workspaces/{workspace}/invitations`  
**Auth:** Required (Owner/Admin)

**Request Body:**
```json
{
    "email": "newmember@example.com",
    "role": "member"
}
```

---

### Cancel Invitation

**Endpoint:** `DELETE /workspaces/{workspace}/invitations/{invitation}`  
**Auth:** Required (Owner/Admin)

---

### Accept Invitation (Public)

**Endpoint:** `POST /invitations/{token}/accept`

**Success Response (200):**
```json
{
    "message": "Invitation accepted successfully.",
    "workspace": { ... }
}
```

---

### Decline Invitation (Public)

**Endpoint:** `POST /invitations/{token}/decline`

---

## Plans

### List Plans (Public)

**Endpoint:** `GET /plans`

**Success Response (200):**
```json
{
    "data": [
        {
            "id": 1,
            "name": "Free",
            "slug": "free",
            "description": "Perfect for getting started",
            "price_monthly": 0,
            "price_yearly": 0,
            "limits": {
                "workflows": 5,
                "executions": 500,
                "members": 1
            },
            "features": {
                "webhooks": false,
                "priority_support": false
            }
        },
        {
            "id": 2,
            "name": "Pro",
            "slug": "pro",
            "price_monthly": 1900,
            "price_yearly": 19000,
            "limits": {
                "workflows": 50,
                "executions": 10000,
                "members": 5
            },
            "features": {
                "webhooks": true,
                "priority_support": false
            }
        }
    ]
}
```

---

## Subscriptions

### Get Subscription

**Endpoint:** `GET /workspaces/{workspace}/subscription`  
**Auth:** Required

**Success Response (200):**
```json
{
    "data": {
        "id": 1,
        "plan": { ... },
        "status": "active",
        "trial_ends_at": null,
        "current_period_start": "2026-01-30T10:00:00.000000Z",
        "current_period_end": "2026-02-28T10:00:00.000000Z"
    }
}
```

---

### Create/Update Subscription

**Endpoint:** `POST /workspaces/{workspace}/subscription`  
**Auth:** Required (Owner)

**Request Body:**
```json
{
    "plan_id": 2
}
```

---

### Cancel Subscription

**Endpoint:** `DELETE /workspaces/{workspace}/subscription`  
**Auth:** Required (Owner)

---

## Error Responses

### Unauthenticated (401)
```json
{
    "message": "Unauthenticated."
}
```

### Forbidden (403)
```json
{
    "message": "You do not have permission to perform this action."
}
```

### Validation Error (422)
```json
{
    "message": "The given data was invalid.",
    "errors": {
        "field_name": ["Error message."]
    }
}
```

---

## Environment Variables

Add these to your `.env` file:

```env
FRONTEND_URL=https://app.linkflow.com
```
