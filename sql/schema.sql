CREATE TABLE channels (
  channel_id text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone
);

ALTER TABLE
  channels
ADD
  CONSTRAINT channel_pkey PRIMARY KEY (channel_id);

CREATE TABLE approvals (
  channel_id text NOT NULL,
  user_id text NOT NULL
);

ALTER TABLE
  approvals
ADD
  CONSTRAINT approvals_pkey PRIMARY KEY (channel_id, user_id);

CREATE TABLE commands (
  channel_id text NOT NULL,
  name text NOT NULL,
  template text NOT NULL
);

ALTER TABLE
  commands
ADD
  CONSTRAINT command_pkey PRIMARY KEY (channel_id, name);

CREATE TABLE counter (
  channel_id text NOT NULL,
  name text NOT NULL,
  value bigint NOT NULL DEFAULT 0
);

ALTER TABLE
  counter
ADD
  CONSTRAINT counter_pkey PRIMARY KEY (channel_id, name);

CREATE TABLE users (
  channel_id text NOT NULL,
  sender_id text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  updated_at timestamp with time zone,
  word_count bigint NOT NULL,
  message_count bigint NOT NULL,
  watch_time bigint NOT NULL
);

ALTER TABLE
  users
ADD
  CONSTRAINT user_pkey PRIMARY KEY (channel_id, sender_id);

CREATE TABLE messages (
  id SERIAL NOT NULL,
  channel_id text NOT NULL,
  sender_id text NOT NULL,
  created_at timestamp with time zone NOT NULL,
  message text NOT NULL
);

ALTER TABLE
  messages
ADD
  CONSTRAINT message_pkey PRIMARY KEY (id);