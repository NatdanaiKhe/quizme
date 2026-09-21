CREATE TABLE IF NOT EXISTS user_topic_stats (
    topic_id INT PRIMARY KEY REFERENCES topics(id),
    correct_count INT DEFAULT 0,
    wrong_count INT DEFAULT 0,
    last_practiced_at TIMESTAMPTZ,
    accuracy_rate NUMERIC(5,2) DEFAULT 0
);
