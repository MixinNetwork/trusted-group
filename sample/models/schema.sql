CREATE TABLE IF NOT EXISTS users (
  user_id                  VARCHAR(36) PRIMARY KEY CHECK (user_id ~* '^[0-9a-f-]{36,36}$'),
  identity_number          VARCHAR(128) NOT NULL DEFAULT '',
  full_name                VARCHAR(512) NOT NULL DEFAULT '',
  avatar_url               VARCHAR(1024) NOT NULL DEFAULT '',
  access_token             VARCHAR(1024) NOT NULL DEFAULT '',
  authentication_token     VARCHAR(512) UNIQUE NOT NULL,
  created_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);


CREATE TABLE IF NOT EXISTS payments (
  payment_id               VARCHAR(36) PRIMARY KEY CHECK (payment_id ~* '^[0-9a-f-]{36,36}$'),
  asset_id                 VARCHAR(36) NOT NULL DEFAULT '',
  amount                   VARCHAR(128) NOT NULL DEFAULT '0',
  threshold                BIGINT	NOT NULL,
  receivers                VARCHAR[] NOT NULL DEFAULT '{}',
  memo                     VARCHAR(256) NOT NULL DEFAULT '',
  state                   VARCHAR(128) NOT NULL DEFAULT '',
  code_id                  VARCHAR(36) NOT NULL DEFAULT '',
  transaction_hash         VARCHAR(512) NOT NULL DEFAULT '',
  raw_transaction          TEXT NOT NULL DEFAULT '',
  user_id                  VARCHAR(36) NOT NULL,
  created_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS payments_statex ON payments (state);
CREATE INDEX IF NOT EXISTS payments_codex ON payments (code_id);


CREATE TABLE IF NOT EXISTS transfers (
  transfer_id              VARCHAR(36) PRIMARY KEY CHECK (transfer_id ~* '^[0-9a-f-]{36,36}$'),
  user_id                  VARCHAR(36) NOT NULL DEFAULT '',
  asset_id                 VARCHAR(36) NOT NULL DEFAULT '',
  amount                   VARCHAR(128) NOT NULL,
  memo                     VARCHAR(256) NOT NULL DEFAULT '',
  trace_id                 VARCHAR(36) NOT NULL DEFAULT '',
  state                   VARCHAR(128) NOT NULL DEFAULT '',
  created_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS transfers_statex ON transfers (state);