CREATE TABLE IF NOT EXISTS properties (
  key           VARCHAR NOT NULL,
  value         VARCHAR NOT NULL,
  created_at    TIMESTAMP NOT NULL,
  updated_at    TIMESTAMP NOT NULL,
  PRIMARY KEY ('key')
);



CREATE TABLE IF NOT EXISTS iterations (
  action           INTEGER NOT NULL,
  node_id          VARCHAR NOT NULL,
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
  mask                 VARCHAR NOT NULL,
  keys                 VARCHAR NOT NULL,
  senders_threshold    INTEGER NOT NULL,
  senders              VARCHAR NOT NULL,
  receivers_threshold  INTEGER NOT NULL,
  receivers            VARCHAR NOT NULL,
  extra                VARCHAR NOT NULL,
  state                VARCHAR NOT NULL,
  sequence             INTEGER NOT NULL,
  created_at           TIMESTAMP NOT NULL,
  updated_at           TIMESTAMP NOT NULL,
  signers              VARCHAR NOT NULL,
  signed_by            VARCHAR NOT NULL,
  signed_tx            VARCHAR NOT NULL,
  trace_id             VARCHAR NOT NULL,
  group_id             VARCHAR NOT NULL,
  PRIMARY KEY ('output_id')
);

CREATE INDEX IF NOT EXISTS outputs_by_sequence ON outputs(sequence);
CREATE INDEX IF NOT EXISTS outputs_by_state_sequence ON outputs(state, sequence);
CREATE INDEX IF NOT EXISTS outputs_by_trace_sequence ON outputs(trace_id, sequence);
CREATE INDEX IF NOT EXISTS outputs_by_group_state_asset_sequence ON outputs(group_id, asset_id, sequence, state);



CREATE TABLE IF NOT EXISTS actions (
  output_id            VARCHAR NOT NULL,
  transaction_hash     VARCHAR NOT NULL,
  action_state         INTEGER NOT NULL,
  sequence             INTEGER NOT NULL,
  restore_sequence     INTEGER NOT NULL,
  PRIMARY KEY ('output_id')
);

CREATE INDEX IF NOT EXISTS actions_by_sequence ON actions(sequence);
CREATE INDEX IF NOT EXISTS actions_by_id_sequence ON actions(output_id, sequence);
CREATE INDEX IF NOT EXISTS actions_by_state_sequence ON actions(action_state, sequence);
CREATE INDEX IF NOT EXISTS actions_by_state_hash ON actions(action_state, transaction_hash);



CREATE TABLE IF NOT EXISTS transactions (
  group_id             VARCHAR NOT NULL,
  trace_id             VARCHAR NOT NULL,
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
  updated_at           TIMESTAMP NOT NULL,
  PRIMARY KEY ('trace_id')
);

CREATE INDEX IF NOT EXISTS transactions_by_hash ON transactions(hash);
CREATE INDEX IF NOT EXISTS transactions_by_state_sequence ON transactions(state, sequence);