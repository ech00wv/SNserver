-- +goose Up
ALTER TABLE users
ADD is_premium boolean DEFAULT false;

-- +goose Down
ALTER TABLE users
DROP COLUMN is_premium;