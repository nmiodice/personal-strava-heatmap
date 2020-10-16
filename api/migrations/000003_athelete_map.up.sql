BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE AthleteMap (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
	athlete_id INT UNIQUE NOT NULL
);

END;
