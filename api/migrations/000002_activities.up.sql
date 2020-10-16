BEGIN;

CREATE TABLE StravaActivity (
    activity_id       BIGINT PRIMARY KEY,
	athlete_id        BIGINT,
    activity_data_ref TEXT,
    imported_at       TIMESTAMP NOT NULL DEFAULT NOW()
);

END;
