# LinkFlow API

The **Control Plane** for LinkFlow - a Laravel-based REST API that handles user authentication, workspace management, workflow configuration, and job orchestration.

## Tech Stack

| Component | Technology | Version |
|-----------|------------|---------|
| Framework | Laravel | 12.x |
| Language | PHP | 8.4+ |
| Authentication | Laravel Passport | 13.x |
| Permissions | spatie/laravel-permission | 6.x |
| Testing | Pest PHP | 4.x |
| Static Analysis | PHPStan/Larastan | 3.x |
| Formatter | Laravel Pint | 1.x |

## Getting Started

### Prerequisites

- PHP 8.4+
- Composer 2.x
- PostgreSQL 16+ (or use Docker)
- Redis 7+

### Installation

```bash
# Install dependencies
composer install

# Copy environment file
cp .env.example .env

# Generate application key
php artisan key:generate

# Run migrations
php artisan migrate

# Seed the database (optional)
php artisan db:seed
```

### Development Server

```bash
# Using Laravel Herd (recommended)
# The app is automatically available at https://lnkflow.test

# Using built-in server
php artisan serve

# Using Laravel Sail (Docker)
./vendor/bin/sail up
```

## API Structure

The API follows RESTful conventions with versioned endpoints:

```
/api/v1/
├── health                    # Health check endpoint
├── auth/                     # Authentication
│   ├── register              # POST - User registration
│   ├── login                 # POST - User login
│   ├── logout                # POST - User logout
│   ├── forgot-password       # POST - Password reset request
│   └── reset-password        # POST - Password reset
├── user/                     # User profile management
├── workspaces/               # Workspace CRUD
│   └── {workspace}/
│       ├── members/          # Member management
│       ├── invitations/      # Team invitations
│       ├── subscription/     # Billing & subscription
│       ├── workflows/        # Workflow CRUD + execution
│       ├── credentials/      # Credential storage
│       ├── executions/       # Execution history
│       ├── webhooks/         # Webhook management
│       ├── variables/        # Environment variables
│       ├── tags/             # Resource tagging
│       └── activity/         # Activity logs
├── nodes/                    # Available node types
├── credential-types/         # Supported credentials
└── plans/                    # Subscription plans
```

## Commands

```bash
# Run tests
php artisan test

# Run tests with coverage
php artisan test --coverage

# Format code
vendor/bin/pint

# Static analysis
vendor/bin/phpstan analyse

# List all routes
php artisan route:list --path=api

# Clear caches
php artisan optimize:clear
```

## Project Structure

```
apps/api/
├── app/
│   ├── Enums/           # Enum definitions
│   ├── Http/
│   │   ├── Controllers/ # API controllers (versioned)
│   │   ├── Middleware/  # Custom middleware
│   │   ├── Requests/    # Form request validation
│   │   └── Resources/   # API resource transformers
│   ├── Jobs/            # Queue jobs
│   ├── Models/          # Eloquent models
│   ├── Notifications/   # Email notifications
│   ├── Providers/       # Service providers
│   └── Services/        # Business logic
├── config/              # Configuration files
├── database/
│   ├── factories/       # Model factories
│   ├── migrations/      # Database migrations
│   └── seeders/         # Database seeders
├── routes/
│   ├── api.php          # API routes
│   └── admin.php        # Admin routes
└── tests/
    ├── Feature/         # Feature tests
    └── Unit/            # Unit tests
```

## Communication with Engine

The API communicates with the Go Engine via:

1. **HTTP/REST**: For dispatching workflow executions
2. **Redis Streams**: For real-time job status updates
3. **Callbacks**: Engine sends execution results back via signed HTTP callbacks

The `LINKFLOW_SECRET` environment variable must match between the API and Engine for secure callback authentication.

## License

Proprietary - All rights reserved.
