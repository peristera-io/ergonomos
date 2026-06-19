-- +goose Up
-- Initial ergonomos schema: accounts, tasks, authorization tuples, and the
-- transactional outbox (ADR-0003, ADR-0008).

CREATE TABLE users (
    id              TEXT        PRIMARY KEY,
    email           TEXT        NOT NULL UNIQUE,
    handle          TEXT        NOT NULL,
    instance_domain TEXT        NOT NULL,
    is_local        BOOLEAN     NOT NULL,
    password_hash   TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE tasks (
    id         TEXT        PRIMARY KEY,
    title      TEXT        NOT NULL,
    owner_id   TEXT        NOT NULL,
    done       BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX tasks_owner_id_idx ON tasks (owner_id);

-- Exact-tuple authorization store (ADR-0008 interim; OpenFGA replaces it).
CREATE TABLE authz_tuples (
    subject  TEXT NOT NULL,
    relation TEXT NOT NULL,
    object   TEXT NOT NULL,
    PRIMARY KEY (subject, relation, object)
);

-- Transactional outbox: events are appended in the same transaction as the
-- domain change that produced them. published_at supports a future dispatcher.
CREATE TABLE outbox (
    id           TEXT        PRIMARY KEY,
    type         TEXT        NOT NULL,
    subject      TEXT        NOT NULL,
    occurred_at  TIMESTAMPTZ NOT NULL,
    payload      JSONB       NOT NULL,
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_unpublished_idx ON outbox (occurred_at) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE outbox;
DROP TABLE authz_tuples;
DROP TABLE tasks;
DROP TABLE users;
