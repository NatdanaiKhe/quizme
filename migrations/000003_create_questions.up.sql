CREATE TABLE IF NOT EXISTS questions (
    id SERIAL PRIMARY KEY,
    batch_id INT REFERENCES quiz_batches(id),
    topic_id INT REFERENCES topics(id),
    prompt TEXT NOT NULL,
    options JSONB NOT NULL,
    correct_option TEXT NOT NULL,
    explanation TEXT,
    source TEXT DEFAULT 'ai_generated',
    created_at TIMESTAMPTZ DEFAULT now()
);
