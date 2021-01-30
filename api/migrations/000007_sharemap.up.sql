BEGIN;

ALTER TABLE 
    AthleteMap
ADD COLUMN 
    sharable bool DEFAULT false;

END;
