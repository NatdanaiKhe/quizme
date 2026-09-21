CREATE TABLE IF NOT EXISTS quiz_batches (
    id SERIAL PRIMARY KEY,
    batch_date DATE NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT now()
);
