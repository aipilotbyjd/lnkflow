# Authentication

The LinkFlow API uses **Bearer Token** authentication.

## Getting a Token

### 1. Login (User)
To get a token programmatically, use the login endpoint:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "your-password"
}
```

**Response:**
```json
{
  "token": "123|abcdef..."
}
```

### 2. API Keys (Service Accounts)
For server-to-server communication, create an API Token in the dashboard under **Settings > API Keys**.

## Authenticating Requests

Include the token in the `Authorization` header of every request:

```http
GET /api/v1/workspaces
Authorization: Bearer 123|abcdef...
Accept: application/json
```

## Token Expiration

-   **User Tokens**: Expire after 30 days of inactivity.
-   **API Keys**: Do not expire until revoked.

If you receive a `401 Unauthorized` response, your token is invalid or expired.
