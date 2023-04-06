BEGIN;

-- All created chats are counted as not direct
ALTER TABLE chats
    ADD COLUMN is_direct BOOLEAN NOT NULL DEFAULT FALSE;


-- Makes is_direct column required
ALTER TABLE chats
    ALTER COLUMN is_direct DROP DEFAULT;

COMMIT;