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



CREATE TABLE IF NOT EXISTS actions (
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
  created_at           INTEGER NOT NULL,
  updated_at           INTEGER NOT NULL,
  signers              VARCHAR NOT NULL,
  signed_by            VARCHAR NOT NULL,
  trace_id            VARCHAR NOT NULL,
  group_id            VARCHAR NOT NULL,
  action_state         INTEGER NOT NULL,
  PRIMARY KEY ('output_id')
);