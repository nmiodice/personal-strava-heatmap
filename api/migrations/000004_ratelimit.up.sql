BEGIN;

-- this is a single row table
CREATE TABLE StravaRateLimit (
    id bool       PRIMARY KEY DEFAULT TRUE,
    limited_until TIMESTAMP NOT NULL,
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT    id_unique CHECK (id)
);

END;
