-- LinkFlow Database Initialization
-- This runs automatically when PostgreSQL container starts (first time only)

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Grant permissions (in case needed)
GRANT ALL PRIVILEGES ON DATABASE linkflow TO linkflow;

-- Log
DO $$
BEGIN
    RAISE NOTICE 'LinkFlow database initialized successfully!';
END $$;
