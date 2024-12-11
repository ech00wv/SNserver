-- +goose Up
CREATE TABLE users (
    id UUID DEFAULT gen_random_uuid(),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP NOT NULL,
    email TEXT UNIQUE NOT NULL,
    PRIMARY KEY (id)
);

-- +goose Down
DROP TABLE users;
