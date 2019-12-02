CREATE TABLE channels (
  channel_name text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone,
  hiccup_count bigint NOT NULL
);

ALTER TABLE channels ADD CONSTRAINT channel_pkey PRIMARY KEY (channel_name);

CREATE TABLE messages (
  id integer NOT NULL,
  channel_name text NOT NULL,
  sender text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone,
  message text NOT NULL
);

ALTER TABLE messages ADD CONSTRAINT message_pkey PRIMARY KEY (id);
ALTER TABLE messages ADD CONSTRAINT messages_fkey FOREIGN KEY (channel_name, sender) REFERENCES users(channel_name, sender);

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

-- Join table
CREATE TABLE user_messages (
  channel_name text NOT NULL,
  sender text NOT NULL,
  message_id integer NOT NULL
);

-- Composite primary key
ALTER TABLE user_messages ADD CONSTRAINT user_message_pkey PRIMARY KEY (channel_name, sender, message_id);
ALTER TABLE user_messages ADD CONSTRAINT user_message_messages_fkey FOREIGN KEY (message_id) REFERENCES messages(id);
ALTER TABLE user_messages ADD CONSTRAINT user_message_users_fkey FOREIGN KEY (channel_name, sender) REFERENCES users(channel_name, sender);
