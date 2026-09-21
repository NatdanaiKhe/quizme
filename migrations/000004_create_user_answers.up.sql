CREATE TABLE IF NOT EXISTS user_answers (
    id SERIAL PRIMARY KEY,
    question_id INT REFERENCES questions(id),
    selected_option TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL,
    answered_at TIMESTAMPTZ DEFAULT now()
);
