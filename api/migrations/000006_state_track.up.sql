BEGIN;

CREATE TABLE AthleteProcessingState (
	athlete_id int PRIMARY KEY,
	state      TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

END;
