BEGIN;

CREATE TABLE StravaToken (
	id                      SERIAL PRIMARY KEY,
	athlete_id              INT,
	access_token            VARCHAR(500) NOT NULL,
	access_token_expires_at TIMESTAMP,
	refresh_token           VARCHAR(500) NOT NULL,
	created_at              TIMESTAMP NOT NULL DEFAULT NOW(),
	UNIQUE(access_token)
);

END;
