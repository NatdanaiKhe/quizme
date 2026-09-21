CREATE TABLE IF NOT EXISTS generation_logs (
    id SERIAL PRIMARY KEY,
    batch_id INT REFERENCES quiz_batches(id),
    status TEXT NOT NULL,
    error_message TEXT,
    tokens_used INT,
    created_at TIMESTAMPTZ DEFAULT now()
);
