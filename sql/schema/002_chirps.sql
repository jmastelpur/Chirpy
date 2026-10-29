-- +goose up
CREATE TABLE chirps (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    body TEXT NOT NULL,
    user_id UUID NOT NULL CONSTRAINT chirps_user_id_fkey REFERENCES users(id) ON DELETE CASCADE
);

-- +goose down
DROP TABLE chirps;