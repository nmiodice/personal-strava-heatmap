BEGIN;

DROP TYPE IF EXISTS PSTATE;
CREATE TYPE PSTATE AS ENUM ('QUEUED', 'COMPLETE', 'FAILED');

CREATE TABLE QueueProcessingState (
	id         SERIAL PRIMARY KEY,
	map_id     uuid NOT NULL,
	message_id VARCHAR(100) NOT NULL,
    pstate     PSTATE NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX message_id_idx ON QueueProcessingState (message_id);

END;
