CREATE TABLE teams (
    name text PRIMARY KEY
);

CREATE TABLE users (
    id         text PRIMARY KEY,
    name       text NOT NULL,
    team_name  text NOT NULL REFERENCES teams(name) ON DELETE RESTRICT,
    is_active  boolean NOT NULL DEFAULT true
);

CREATE TABLE pull_requests (
    id          text PRIMARY KEY,
    name        text NOT NULL,
    author_id   text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status      text NOT NULL,
    created_at  timestamptz NOT NULL,
    merged_at   timestamptz
);

CREATE TABLE pull_request_reviewers (
    pull_request_id text NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    user_id         text NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    PRIMARY KEY (pull_request_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_users_team_name_is_active
    ON users (team_name, is_active);

CREATE INDEX IF NOT EXISTS idx_pull_request_reviewers_user_id
    ON pull_request_reviewers (user_id);
