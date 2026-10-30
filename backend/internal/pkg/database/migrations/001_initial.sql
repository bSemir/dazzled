CREATE TABLE incidents (
    id INTEGER PRIMARY KEY,
    vehicle_type TEXT NOT NULL CHECK(vehicle_type IN ('Hatchback','Sedan','SUV','Pickup / truck','Van','Motorbike','Bicycle')),
    brand TEXT NOT NULL CHECK(brand IN ('BMW','Audi','Mercedes-Benz','Tesla','Ford','Volkswagen','Unknown','Aftermarket ''xenon'' bulbs from a webshop')),
    severity INTEGER NOT NULL CHECK(severity BETWEEN 1 AND 5),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX incidents_created_at_idx ON incidents(created_at);
CREATE INDEX incidents_brand_created_idx ON incidents(brand, created_at);
CREATE INDEX incidents_vehicle_created_idx ON incidents(vehicle_type, created_at);

CREATE TABLE leaderboard_scores (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL CHECK(length(username) BETWEEN 2 AND 20),
    distance REAL NOT NULL CHECK(distance >= 0 AND distance <= 10000),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX leaderboard_scores_rank_idx ON leaderboard_scores(distance DESC, created_at ASC);
