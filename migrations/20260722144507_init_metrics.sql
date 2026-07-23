-- +goose Up
CREATE TABLE metrics (
    id text NOT NULL,
    mtype text NOT NULL,
    delta bigint,
    value double precision,
    PRIMARY KEY (id, mtype)
);

-- +goose Down
DROP TABLE metrics;
