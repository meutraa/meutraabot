CREATE TABLE channels (
  channel_id text NOT NULL PRIMARY KEY,
  autoreply_enabled boolean NOT NULL DEFAULT false,
  autoreply_frequency float NOT NULL DEFAULT 2,
  reply_safety int NOT NULL DEFAULT 2,
  openai_token text
);

CREATE TABLE approvals (
  channel_id text NOT NULL,
  user_id text NOT NULL,
  manual boolean NOT NULL,
  UNIQUE (channel_id, user_id)
);

CREATE TABLE commands (
  channel_id text NOT NULL,
  name text NOT NULL,
  template text NOT NULL,
  UNIQUE (channel_id, name)
);

CREATE TABLE numbers (
  channel_id text NOT NULL,
  name text NOT NULL,
  value int NOT NULL DEFAULT 0,
  UNIQUE (channel_id, name)
);