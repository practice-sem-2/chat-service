BEGIN;

ALTER TABLE chats
    DROP COLUMN is_direct;

END;