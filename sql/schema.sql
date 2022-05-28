CREATE TABLE channels (
  channel_id text NOT NULL,
  autoreply_enabled boolean NOT NULL DEFAULT false,
  autoreply_frequency float NOT NULL DEFAULT 2,
  reply_safety int NOT NULL DEFAULT 2,
  openai_token text,
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