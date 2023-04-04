BEGIN;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE chats
(
    chat_id uuid NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY
);


CREATE TABLE chat_members
(
    chat_id uuid        NOT NULL REFERENCES chats,
    user_id varchar(64) NOT NULL,
    PRIMARY KEY (chat_id, user_id)
);


CREATE TABLE messages
(
    message_id   uuid          NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    chat_id      uuid          NOT NULL REFERENCES chats,
    from_user    varchar(64)   NOT NULL,
    reply_to     uuid          NULL     DEFAULT NULL REFERENCES messages,
    sending_time TIMESTAMP     NOT NULL DEFAULT (now() at time zone 'utc'),
    text         VARCHAR(2048) NULL     DEFAULT NULL
);

CREATE TABLE attachments
(
    attachment_id uuid NOT NULL DEFAULT uuid_generate_v4() PRIMARY KEY,
    message_id    uuid NOT NULL REFERENCES messages,
    file_id       uuid NOT NULL
);

COMMIT;