CREATE TABLE IF NOT EXISTS payments (
  payment_id               VARCHAR(36) PRIMARY KEY CHECK (payment_id ~* '^[0-9a-f-]{36,36}$'),
  asset_id                 VARCHAR(36) NOT NULL DEFAULT '',
  amount                   VARCHAR(128) NOT NULL DEFAULT '0',
  threshold                BIGINT	NOT NULL,
  receivers                VARCHAR[] NOT NULL DEFAULT '{}',
  memo                     VARCHAR(256) NOT NULL DEFAULT '',
  state                    VARCHAR(128) NOT NULL DEFAULT '',
  code_id                  VARCHAR(36) NOT NULL DEFAULT '',
  transaction_hash         VARCHAR(512) UNIQUE,
  raw_transaction          TEXT,
  created_at               TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS payments_memo_statex ON payments (memo, state);
CREATE INDEX IF NOT EXISTS payments_statex ON payments (state);
CREATE INDEX IF NOT EXISTS payments_codex ON payments (code_id);
