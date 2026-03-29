CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),
    role          VARCHAR(50)  NOT NULL DEFAULT 'dev',
    phone         VARCHAR(50),
    provider      VARCHAR(50)  NOT NULL DEFAULT 'local',
    provider_id   VARCHAR(255),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX users_email_idx ON users (email);
CREATE UNIQUE INDEX users_provider_provider_id_idx ON users (provider, provider_id) WHERE provider_id IS NOT NULL;
