-- +goose Up
-- Пример таблицы для каркаса — можно удалить вместе с items-ресурсом.
CREATE TABLE items
(
    id         uuid PRIMARY KEY     DEFAULT gen_random_uuid(),
    title      text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE items;
