CREATE TABLE appointments (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title               VARCHAR     NOT NULL,
    description         VARCHAR,
    start_time          TIMESTAMPTZ NOT NULL,
    end_time            TIMESTAMPTZ NOT NULL,
    status              VARCHAR     NOT NULL DEFAULT 'scheduled' CHECK (status IN ('scheduled', 'cancelled')),
    recurrence_group_id UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_appointments_user_id             ON appointments(user_id);
CREATE INDEX idx_appointments_start_time          ON appointments(start_time);
CREATE INDEX idx_appointments_end_time            ON appointments(end_time);
CREATE INDEX idx_appointments_recurrence_group_id ON appointments(recurrence_group_id);

CREATE TABLE idempotency_keys (
    key             VARCHAR   NOT NULL,
    user_id         UUID      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    appointment_ids UUID[]    NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (key, user_id)
);
