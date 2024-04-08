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