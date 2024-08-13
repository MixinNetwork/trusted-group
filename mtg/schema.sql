CREATE TABLE IF NOT EXISTS caches (
  key           VARCHAR NOT NULL,
  value         VARCHAR NOT NULL,
  created_at    TIMESTAMP NOT NULL,
  PRIMARY KEY ('key')
);

CREATE INDEX IF NOT EXISTS caches_by_created ON caches(created_at);



CREATE TABLE IF NOT EXISTS properties (
  key           VARCHAR NOT NULL,
  value         VARCHAR NOT NULL,
  created_at    TIMESTAMP NOT NULL,
  updated_at    TIMESTAMP NOT NULL,
  PRIMARY KEY ('key')
);



CREATE TABLE IF NOT EXISTS iterations (
  node_id          VARCHAR NOT NULL,
  action           INTEGER NOT NULL,
  threshold        INTEGER NOT NULL,
  created_at       INTEGER NOT NULL,
  PRIMARY KEY ('node_id')
);

CREATE INDEX IF NOT EXISTS iterations_by_node_created ON iterations(node_id, created_at);



CREATE TABLE IF NOT EXISTS outputs (
  output_id            VARCHAR NOT NULL,
  request_id           VARCHAR NOT NULL,
  transaction_hash     VARCHAR NOT NULL,
  output_index         INTEGER NOT NULL,
  asset_id             VARCHAR NOT NULL,
  kernel_asset_id      VARCHAR NOT NULL,
  amount               VARCHAR NOT NULL,
  senders_threshold    INTEGER NOT NULL,
  senders              VARCHAR NOT NULL,
  receivers_threshold  INTEGER NOT NULL,
  extra                VARCHAR NOT NULL,
  state                VARCHAR NOT NULL,
  sequence             INTEGER NOT NULL,
  created_at           TIMESTAMP NOT NULL,
  updated_at           TIMESTAMP NOT NULL,
  signers              VARCHAR NOT NULL,
  signed_by            VARCHAR NOT NULL,
  trace_id             VARCHAR NOT NULL,
  app_id               VARCHAR NOT NULL,
  PRIMARY KEY ('output_id')
);

CREATE UNIQUE INDEX IF NOT EXISTS outputs_by_sequence ON outputs(sequence);
CREATE INDEX IF NOT EXISTS outputs_by_trace_sequence ON outputs(trace_id, sequence);
CREATE INDEX IF NOT EXISTS outputs_by_app_asset_state_sequence ON outputs(app_id, asset_id, state, sequence);



CREATE TABLE IF NOT EXISTS actions (
  output_id            VARCHAR NOT NULL,
  transaction_hash     VARCHAR NOT NULL,
  action_state         INTEGER NOT NULL,
  sequence             INTEGER NOT NULL,
  restore_sequence     INTEGER NOT NULL,
  PRIMARY KEY ('output_id')
);

CREATE UNIQUE INDEX IF NOT EXISTS actions_by_sequence ON actions(sequence);
CREATE INDEX IF NOT EXISTS actions_by_state_hash ON actions(action_state, transaction_hash);



CREATE TABLE IF NOT EXISTS transactions (
  trace_id             VARCHAR NOT NULL,
  app_id               VARCHAR NOT NULL,
  opponent_app_id      VARCHAR NOT NULL,
  state                INTEGER NOT NULL,
  asset_id             VARCHAR NOT NULL,
  receivers            VARCHAR NOT NULL,
  threshold            INTEGER NOT NULL,
  amount               VARCHAR NOT NULL,
  memo                 VARCHAR NOT NULL,
  raw                  VARCHAR,
  hash                 VARCHAR,
  refs                 VARCHAR NOT NULL,
  sequence             INTEGER NOT NULL,
  compaction           BOOLEAN NOT NULL,
  storage              BOOLEAN NOT NULL,
  storage_trace_id     VARCHAR,
  request_id           VARCHAR,
  updated_at           TIMESTAMP NOT NULL,
  PRIMARY KEY ('trace_id')
);

CREATE UNIQUE INDEX IF NOT EXISTS transactions_by_hash ON transactions(hash) WHERE hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS transactions_by_state_sequence ON transactions(state, sequence);
CREATE INDEX IF NOT EXISTS transactions_by_asset_state_sequence ON transactions(asset_id, state, sequence);
