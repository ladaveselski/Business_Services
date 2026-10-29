CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS services (
    id SERIAL PRIMARY KEY,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL CHECK (status IN ('draft', 'published', 'deleted')),
    image_key VARCHAR(255),
    video_key VARCHAR(255),
    price INTEGER,
    deadline TIMESTAMPTZ,
    creator_id INTEGER NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS likes (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id),
    service_id INTEGER NOT NULL REFERENCES services(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, service_id)
);

-- Начальные данные
INSERT INTO users (name) VALUES ('Тестовый пользователь') ON CONFLICT DO NOTHING;

-- 3 услуги: черновик, удалён, опубликован
INSERT INTO services (title, description, status, image_key, video_key, price, deadline, creator_id)
VALUES
    ('Черновик', 'Описание черновика', 'draft', NULL, NULL, 0, NULL, 1),
    ('Удалённая', 'Описание удалённой', 'deleted', NULL, NULL, 0, NULL, 1),
    ('Опубликованная', 'Описание опубликованной', 'published', 'seed/plumber.jpg', 'seed/plumber.mp4', 4500, '2026-10-05 14:30:00+00', 1)
ON CONFLICT DO NOTHING;