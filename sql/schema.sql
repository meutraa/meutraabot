-- DROP TABLE IF EXISTS channels;
-- DROP TABLE IF EXISTS messages;
-- DROP TABLE IF EXISTS users;
-- DROP TABLE IF EXISTS user_messages;

CREATE TABLE channels (
  channel_name text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone,
  hiccup_count bigint NOT NULL
);

ALTER TABLE channels ADD CONSTRAINT channel_pkey PRIMARY KEY (channel_name);

CREATE TABLE users (
  channel_name text NOT NULL,
  sender text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone,
  word_count bigint NOT NULL,
  message_count bigint NOT NULL,
  watch_time bigint NOT NULL,
  emoji text,
  text_color text
);

ALTER TABLE users ADD CONSTRAINT user_pkey PRIMARY KEY (channel_name, sender);

CREATE TABLE messages (
  id SERIAL NOT NULL,
  channel_name text NOT NULL,
  sender text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone,
  message text NOT NULL
);

ALTER TABLE messages ADD CONSTRAINT message_pkey PRIMARY KEY (id);
-- ALTER TABLE messages ADD CONSTRAINT messages_fkey FOREIGN KEY (channel_name, sender) REFERENCES users(channel_name, sender);
