CREATE TABLE users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR     NOT NULL UNIQUE,
    password_hash VARCHAR     NOT NULL,
    timezone      VARCHAR     NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
