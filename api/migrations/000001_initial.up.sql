BEGIN;

CREATE TABLE StravaToken (
	athlete_id INT PRIMARY KEY,
	access_token VARCHAR(500) NOT NULL,
	access_token_expires_at TIMESTAMP,
	refresh_token VARCHAR(500) NOT NULL
);

END;
