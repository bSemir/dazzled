-- Persist one-use game sessions to prevent score-token replay.
CREATE TABLE leaderboard_game_sessions (
    nonce TEXT PRIMARY KEY CHECK(length(nonce) <= 64),
    started_at_ns INTEGER NOT NULL,
    used_at_ns INTEGER
);

CREATE INDEX leaderboard_game_sessions_started_idx ON leaderboard_game_sessions(started_at_ns);
